// Package eventServer
package eventServer

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dronm/gobizapp/api"
	"github.com/dronm/session"

	"github.com/dronm/gobizapp/database"
	"github.com/dronm/gobizapp/logger"
)

const (
	dbAcquireConnWait    = 100
	dbMaxAcquireConnWait = 60000
	defLoopPause         = 100
)

//---- UniqEvents -----

// UniqEvents holds counters for events.
// This stucture is never cleanedup, as
// it holds event id`s and there number is finite.
// If an event is not needed any more (not used by any connection),
// it is removed from server.
type UniqEvents struct {
	mx sync.Mutex
	m  map[string]int // unique event counter
}

// AddEvent puts new event with the given ID to database channel.
// In pg it executes listen command.
func (e *UniqEvents) AddEvent(dbEventID string, qChan chan string) {
	e.mx.Lock()
	cnt, ok := e.m[dbEventID]
	if !ok {
		qChan <- `LISTEN "` + dbEventID + `"`
		e.m[dbEventID] = 1
	} else {
		e.m[dbEventID] = cnt + 1
	}
	e.mx.Unlock()
}

// RemoveEvent puts unlisten command to database channel.
func (e *UniqEvents) RemoveEvent(dbEventID string, qChan chan string) {
	e.mx.Lock()
	if cnt, ok := e.m[dbEventID]; ok {
		if cnt == 1 {
			qChan <- `UNLISTEN "` + dbEventID + `"`
			delete(e.m, dbEventID)
		} else {
			e.m[dbEventID] = cnt - 1
		}
	}
	e.mx.Unlock()
}

// EventCount returns count for a specific event ID.
// The second parameter is true, if the event exists.
// Count can be 0, in nobody is listening to this event.
func (e *UniqEvents) EventCount(eventID string) (int, bool) {
	e.mx.Lock()
	defer e.mx.Unlock()

	value, ok := e.m[eventID]

	return value, ok
}

// TotalEventCount return total event count in a map.
func (e *UniqEvents) TotalEventCount() int {
	e.mx.Lock()
	defer e.mx.Unlock()

	return len(e.m)
}

type SocketServer interface {
	PublishEvent(publisherID, eventID string, payload any) error
}

// EventServer is the main server structure.
type EventServer struct {
	DBPool       *pgxpool.Pool //
	DBQuery      chan string   // for notification queries
	SocketServer SocketServer
	Events       *UniqEvents // count of unique events for db
	LocalEvents  map[string]struct{}

	ctx        context.Context
	cancel     context.CancelFunc
	cancelDone chan struct{}

	sess session.Session

	loopPause time.Duration
	// ReconnectParams waitStrat.WaitStrategy
}

func NewEventServer(localEvents map[string]struct{}) *EventServer {
	return &EventServer{LocalEvents: localEvents}
}

type EurobetonNotif struct {
	Params json.RawMessage `json:"params"`
}

// OnNotification is called when there is a new event coming from pg. Pg channel name is in Service.Method format. Servce name and method name
// should be in PascelCase. Payload should contain an object of key-value pairs.
// Keys must correspond to the method signature. The order of keys does matter
// as method parameters are matched by order not by their names.
func (s *EventServer) OnNotification(_ *pgconn.PgConn, n *pgconn.Notification) {
	logger.Logger.Debugf("OnNotification Channel:%s, Payload:%s", n.Channel, n.Payload)
	srvMeth := strings.Split(n.Channel, ".")
	if len(srvMeth) < 2 {
		logger.Logger.Errorf("OnNotification invalid service.method signature for: %s", n.Channel)
		return
	}

	// local event
	if s.LocalEvents != nil {
		if _, ok := s.LocalEvents[n.Channel]; ok {
			// local cosumer, execute service function
			var params []string
			var err error
			if n.Payload != "" {
				params, err = api.UnmarshalParams([]byte(n.Payload))
				if err != nil {
					logger.Logger.Errorf("OnNotification api.UnmarshalParams: %v", err)
					return
				}
			}

			logger.Logger.Debugf("EventServer local service call %s.%s with params:%v", srvMeth[0], srvMeth[1], params)

			_, err = api.CallMethod(s.ctx, srvMeth[0], srvMeth[1], params,
				&api.ServiceContext{DB: database.DB, Session: s.sess},
			)
			if err != nil {
				logger.Logger.Errorf("EventServer api.CallMethod() %s.%s with params %v, failed: %v", srvMeth[0], srvMeth[1], params, err)
			}
			return
		}
	}

	// publish event for all client consumers
	if s.SocketServer == nil {
		logger.Logger.Errorf("EventSrv: OnNotification: can not publish event: socket server is undefined.")
		return
	}
	// send to all subscribed sockets
	if n.Payload == "" {
		n.Payload = "null"
	}

	var raw EurobetonNotif
	if err := json.Unmarshal([]byte(n.Payload), &raw); err != nil {
		logger.Logger.Errorf("EventSrv: invalid JSON payload from PG: %v", err)
		return
	}
	if err := s.SocketServer.PublishEvent("", n.Channel, raw.Params); err != nil {
		logger.Logger.Errorf("EventSrv: PublishEvent(): %v", err)
	}
}

func (s *EventServer) Serve() {
	go func() {
		s.ctx, s.cancel = context.WithCancel(context.Background())

		s.cancelDone = make(chan struct{})
		defer close(s.cancelDone)

		s.DBQuery = make(chan string)
		s.Events = &UniqEvents{m: make(map[string]int, 0)}

		if s.LocalEvents != nil {
			s.Events.mx.Lock()
			for evntID := range s.LocalEvents {
				s.Events.m[evntID] = 1 // one instance only
			}
			s.Events.mx.Unlock()
		}

		if s.loopPause == 0 {
			s.loopPause = defLoopPause
		}
		logger.Logger.Infof("EventServer: started, loop pause: %v", s.loopPause)

		dbAcquireWait := dbAcquireConnWait

		for {
			var conn *pgxpool.Conn

			select {
			case <-s.ctx.Done():
				logger.Logger.Debug("EventServer breaking loop on stop request")
				return
			default:
				var err error
				conn, err = s.DBPool.Acquire(s.ctx)
				if err != nil {
					if dbAcquireWait > dbMaxAcquireConnWait {
						dbAcquireWait = dbMaxAcquireConnWait
					}
					logger.Logger.Errorf("EventServer DbPool.Acquire(): %v", err)

					time.Sleep(time.Duration(dbAcquireWait) * time.Millisecond)
					dbAcquireWait = dbAcquireWait * 2
					continue
				}
			}

			for evnt := range s.Events.m {
				logger.Logger.Debugf("EventSrv LocalEvent: %s", evnt)
				conn.Exec(s.ctx, `LISTEN "`+evnt+`"`)
			}

			logger.Logger.Debug("EventSrv acquired connection")

			dbAcquireWait = dbAcquireConnWait

			var q string
			for {
				select {
				case <-s.ctx.Done():
					conn.Release()
					return
				case q = <-s.DBQuery:
					logger.Logger.Debugf("EventSrv: query: %s", q)
				default:
					q = ";" // emty query
				}

				if _, err := conn.Exec(s.ctx, q); err != nil {
					if s.ctx.Err() == context.Canceled {
						conn.Release()
						return
					}
					logger.Logger.Errorf("EventSrv conn.Exec(): %v on query: %s", err, q)

					conn.Release()
					break
				}

				// pause
				select {
				case <-s.ctx.Done():
				case <-time.After(s.loopPause * time.Millisecond):
				}
			}
		}
	}()
}

func (s *EventServer) Shutdown(ctx context.Context) {
	if s.cancel == nil {
		return
	}
	logger.Logger.Debug("EventServer stopping on request...")
	s.cancel()

	select {
	case <-ctx.Done():
	case <-s.cancelDone:
	}
	logger.Logger.Info("EventServer stopped")
}

func (s *EventServer) AddEvent(ID string) {
	s.Events.AddEvent(ID, s.DBQuery)
}

func (s *EventServer) RemoveEvent(ID string) {
	s.Events.RemoveEvent(ID, s.DBQuery)
}

func SnakeToPascal(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) == 0 {
			continue
		}

		// Convert the whole part to lowercase first
		part = strings.ToLower(part)

		// Uppercase the first letter if it's a lowercase ASCII letter
		if part[0] >= 'a' && part[0] <= 'z' {
			part = string(part[0]-'a'+'A') + part[1:]
		}

		parts[i] = part
	}
	return strings.Join(parts, "")
}
