package services

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/dronm/eurobeton/config"
	"github.com/dronm/eurobeton/models"

	"github.com/dronm/ds/v4"
	"github.com/dronm/session"

	"github.com/dronm/gobizapp/logger"
)

type ElkonPacket struct {
	ProductionSiteID int                         `json:"production_site_id"`
	Packet           *models.ElkonProductionList `json:"packet"`
}

type ElkonFailoverService struct {
	DB      ds.Provider
	Session session.Session

	filename  string
	mu        sync.Mutex
	interval  time.Duration
	workerCtx context.Context
	cancel    context.CancelFunc
	doneCh    chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once
}

var ElkonFailoverSrv *ElkonFailoverService

func InitElkonFailoverService(db ds.Provider, sess session.Session) {
	interval := config.Config.Elkon.FailoverCheckInterval
	if interval == 0 {
		interval = 60000
	}
	workerCtx, cancel := context.WithCancel(context.Background())
	ElkonFailoverSrv = &ElkonFailoverService{
		DB:        db,
		Session:   sess,
		filename:  "elkonFailover.dat",
		interval:  time.Duration(interval) * time.Millisecond,
		workerCtx: workerCtx,
		cancel:    cancel,
		doneCh:    make(chan struct{}),
	}
}

func (f *ElkonFailoverService) Save(packet *ElkonPacket) error {
	if f == nil {
		return fmt.Errorf("ElkonFailoverSrv is not initialized")
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	file, err := os.OpenFile(f.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("os.OpenFile(): %w", err)
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	if err := enc.Encode(packet); err != nil {
		return fmt.Errorf("enc.Encode(): %w", err)
	}

	return nil
}

func (f *ElkonFailoverService) Start() {
	f.startOnce.Do(func() {
		go func() {
			defer close(f.doneCh)
			ticker := time.NewTicker(f.interval)
			defer ticker.Stop()

			f.replay(f.workerCtx)

			for {
				select {
				case <-ticker.C:
					f.replay(f.workerCtx)
				case <-f.workerCtx.Done():
					return
				}
			}
		}()
		logger.Logger.Infof("started elkon failover service with interval: %v", f.interval)
	})
}

func (f *ElkonFailoverService) Shutdown(ctx context.Context) {
	if f == nil {
		return
	}
	f.stopOnce.Do(f.cancel)
	// Mark an unstarted service done, preventing a later Start after shutdown.
	f.startOnce.Do(func() { close(f.doneCh) })
	select {
	case <-f.doneCh:
		logger.Logger.Info("ElkonFailoverService gracefully shutdown")
	case <-ctx.Done():
		logger.Logger.Errorf("ElkonFailoverService shutdown: %v", ctx.Err())
	}
}

func (f *ElkonFailoverService) replay(workerCtx context.Context) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if workerCtx.Err() != nil {
		return
	}

	file, err := os.Open(f.filename)
	if errors.Is(err, os.ErrNotExist) {
		return // nothing to process
	} else if err != nil {
		logger.Logger.Errorf("ElkonFailoverService error opening failover file: %v", err)
		return
	}
	defer file.Close()

	logger.Logger.Warnf("ElkonFailoverService: processing saved file")

	timeout := config.Config.ClientQueryTimeout
	if timeout == 0 {
		timeout = 120000
	}

	ctx, cancel := context.WithTimeout(workerCtx, time.Duration(timeout)*time.Millisecond)
	defer cancel()

	elkSrv := NewElkonProductionService(f.DB, f.Session)
	elkSrv.FromFailoverFile = true

	// var packets []*ElkonPacket
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		var p ElkonPacket
		if err := json.Unmarshal(scanner.Bytes(), &p); err != nil {
			logger.Logger.Errorf("ElkonFailoverService: Unmarshal(): %v", err)
			continue
		}
		if p.Packet == nil {
			logger.Logger.Error("ElkonFailoverService: saved packet is missing")
			return
		}

		if err := elkSrv.Upload(ctx, p.ProductionSiteID, *p.Packet); err != nil {
			logger.Logger.Errorf("ElkonFailoverService.Upload(): %v", err)
			return // leave file for retry
		}
	}
	if err := scanner.Err(); err != nil {
		logger.Logger.Errorf("ElkonFailoverService: reading saved packets: %v", err)
		return
	}
	if ctx.Err() != nil {
		return
	}

	logger.Logger.Debug("ElkonFailoverService: all packets processed, removing file")

	// If all ok, remove file
	if err := os.Remove(f.filename); err != nil {
		logger.Logger.Errorf("ElkonFailoverService os.Remove(): %v", err)
	}
}
