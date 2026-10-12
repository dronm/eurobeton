package services

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/dronm/ds/v4"
)

type failoverWaitingProvider struct {
	ds.Provider
	started  chan struct{}
	canceled chan struct{}
}

func (p *failoverWaitingProvider) GetPrimary(ctx context.Context) (ds.PoolConn, ds.ServerID, error) {
	close(p.started)
	<-ctx.Done()
	close(p.canceled)
	return nil, "", ctx.Err()
}

func TestElkonFailoverShutdownCancelsReplayAndWaits(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = newProductionTestProvider(t, ctx) // initialize a test logger
	p := &failoverWaitingProvider{started: make(chan struct{}), canceled: make(chan struct{})}
	workerCtx, stop := context.WithCancel(context.Background())
	f := &ElkonFailoverService{
		DB: p, filename: t.TempDir() + "/failover.dat", interval: time.Hour,
		workerCtx: workerCtx, cancel: stop, doneCh: make(chan struct{}),
	}
	if err := os.WriteFile(f.filename, []byte("{\"production_site_id\":3,\"packet\":[]}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f.Start()
	select {
	case <-p.started:
	case <-ctx.Done():
		t.Fatal("replay did not acquire a connection")
	}
	f.Shutdown(ctx)
	select {
	case <-p.canceled:
	default:
		t.Fatal("replay acquisition was not canceled")
	}
	select {
	case <-f.doneCh:
	default:
		t.Fatal("shutdown returned before replay exited")
	}
	if _, err := os.Stat(f.filename); err != nil {
		t.Fatalf("unfinished failover packet must survive shutdown: %v", err)
	}
	f.Shutdown(ctx) // repeat shutdown must not panic
}

func TestElkonFailoverShutdownBeforeStart(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = newProductionTestProvider(t, ctx)
	workerCtx, stop := context.WithCancel(context.Background())
	f := &ElkonFailoverService{
		interval: time.Hour, workerCtx: workerCtx, cancel: stop,
		doneCh: make(chan struct{}),
	}
	f.Shutdown(ctx)
	f.Start()
	select {
	case <-f.doneCh:
	default:
		t.Fatal("unstarted worker was not marked complete")
	}
}
