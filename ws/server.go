// Package ws is a websocket server implementation.
package ws

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	sess "github.com/dronm/session"

	"github.com/dronm/gobizapp/logger"

	"github.com/gin-gonic/gin"

	"github.com/gorilla/websocket"

	"github.com/dronm/eurobeton/beregmw"
)

var Server *WSServer

const defMaxMethodCallDuration = time.Duration(1) * time.Minute

type EventPubSub interface {
	AddEvent(ID string)
	RemoveEvent(ID string)
}

type WSServer struct {
	Addr                  string
	MaxMethodCallDuration time.Duration
	IsProduction          bool
	EventServer           EventPubSub
	server                *http.Server
	clientsMx             sync.RWMutex

	// clients is a client connections holder with mutex protection.
	// keys is a client session ID
	clients map[string][]*Client // clients is a client connections holder with mutex protection.

	checkPermission CheckPermission
	isMethodAllowed IsMethodAllowed
}

type SessionManager interface {
	SessionStart(string) (sess.Session, error)
	GetMaxLifeTime() int64
}

// These functions are used for checking if ws method is allowed.
type (
	CheckPermission = func(method string) gin.HandlerFunc
	IsMethodAllowed = func(userSess sess.Session, method string) error
)

type WSInit struct {
	Addr            string
	EventServer     EventPubSub
	SessManager     SessionManager
	CheckPermission CheckPermission
	IsMethodAllowed IsMethodAllowed
	IsProduction    bool
	URL             string
	SessCookieKey   string
}

func NewWSServer(wsInit WSInit) *WSServer {
	router := gin.Default()

	var ginMode string
	if wsInit.IsProduction {
		ginMode = gin.ReleaseMode
	} else {
		ginMode = gin.DebugMode
	}
	gin.SetMode(ginMode)

	srv := &WSServer{
		Addr:         wsInit.Addr,
		IsProduction: wsInit.IsProduction,
		EventServer:  wsInit.EventServer,
		server: &http.Server{
			Addr:    wsInit.Addr,
			Handler: router,
		},
		clients:         map[string][]*Client{},
		checkPermission: wsInit.CheckPermission,
		isMethodAllowed: wsInit.IsMethodAllowed,
	}

	router.Use(beregmw.SessionMiddleware(wsInit.SessManager, wsInit.SessCookieKey, wsInit.IsProduction))

	if wsInit.URL == "" {
		wsInit.URL = "/"
	}
	router.GET(wsInit.URL, srv.checkPermission("WS.Init"), srv.Init)
	// router.GET(wsInit.URL, srv.Init)

	return srv
}

func (s *WSServer) Serve(cleanupConnInterval int) {
	go func() {
		logger.Logger.Info("WSServer is running on", s.Addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Logger.Fatalf("WSServer server.ListenAndServe(): %s\n", err)
		}
	}()

	// if cleanupConnInterval > 0 {
	// 	go s.CleanupConnections(time.Duration(cleanupConnInterval) * time.Millisecond)
	// }
}

func (s *WSServer) Shutdown(ctx context.Context) {
	s.ShutdownWebSockets(ctx)
	// Attempt to gracefully shut down the server
	if err := s.server.Shutdown(ctx); err != nil {
		logger.Logger.Fatalf("WSServer forced to shutdown: %s\n", err)
	}
	logger.Logger.Info("WSServer gracefully shutdown")
}

func (s *WSServer) CleanupConnections(interval time.Duration) {
	logger.Logger.Infof("WSServer starting cleanup connections with interval: %v", interval)

	for {
		time.Sleep(interval)

		logger.Logger.Warn("WSServer running cleanup connections")

		s.clientsMx.Lock()
		for sessionID, clientList := range s.clients {
			var activeClients []*Client

			for _, client := range clientList {
				if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					logger.Logger.Warnf("WSServer closing stale socket for session %s", sessionID)
					client.Conn.Close()
					client.RemoveAllEvents()
				} else {
					activeClients = append(activeClients, client)
				}
			}

			if len(activeClients) > 0 {
				s.clients[sessionID] = activeClients
			} else {
				delete(s.clients, sessionID)
			}
		}
		s.clientsMx.Unlock()
	}
}

func (s *WSServer) ShutdownWebSockets(ctx context.Context) {
	s.clientsMx.Lock()
	conns := make([]*websocket.Conn, 0)
	for _, clientList := range s.clients {
		for _, client := range clientList {
			conns = append(conns, client.Conn)
		}
	}
	s.clientsMx.Unlock()

	for _, conn := range conns {
		if ctx.Err() != nil {
			break
		}

		// Add per-connection timeout
		connCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		errChan := make(chan error, 1)

		go func(c *websocket.Conn) {
			err := c.WriteMessage(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, "WSServer shutting down"),
			)
			logger.Logger.Warnf("WSServer closing socket on shutdown")
			c.Close()
			errChan <- err
		}(conn)

		select {
		case <-connCtx.Done():
			// Timeout or shutdown
		case <-ctx.Done():
			// Global context canceled
		case <-errChan:
			// Success or error from WriteMessage
		}

		cancel()
	}
}

func (s *WSServer) SubscribeToEvent(sessionID, eventID string) error {
	s.clientsMx.RLock()
	defer s.clientsMx.RUnlock()

	clients, ok := s.clients[sessionID]
	if !ok {
		return fmt.Errorf("WSServer SubscribeToEvent(): session not found by ID")
	}

	for _, client := range clients {
		client.AddEvent(eventID)
	}

	return nil
}

func (s *WSServer) UnsubscribeFromEvent(sessionID, eventID string) error {
	s.clientsMx.RLock()
	defer s.clientsMx.RUnlock()

	clients, ok := s.clients[sessionID]
	if !ok {
		return fmt.Errorf("WSServer UnsubscribeFromEvent(): session not found by ID")
	}

	for _, client := range clients {
		client.RemoveEvent(eventID)
	}

	return nil
}

func (s *WSServer) removeConn(clientID string, target *Client) {
    s.clientsMx.Lock()
    defer s.clientsMx.Unlock()

    list := s.clients[clientID]
    out := list[:0]
    for _, c := range list {
        if c == target {
            c.Conn.Close()
        } else {
            out = append(out, c)
        }
    }

    if len(out) == 0 {
        delete(s.clients, clientID)
    } else {
        s.clients[clientID] = out
    }
}

