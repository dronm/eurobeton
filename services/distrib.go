package services

import (
	"context"
	"fmt"
	"time"

	"github.com/dronm/ds/v4"
	"github.com/dronm/gobizapp/logger"

	"github.com/dronm/session"
)

const cmdExecMaxTime = time.Duration(10) * time.Minute

// DistribService is a service.
type DistribService struct {
	DB      ds.Provider
	Session session.Session
	QueryID string
}

// NewDistribService is a service constructor.
func NewDistribService(db ds.Provider, sess session.Session) *DistribService {
	return &DistribService{DB: db, Session: sess}
}

func (s *DistribService) SetDB(db ds.Provider) {
	s.DB = db
}

func (s *DistribService) SetSession(sess session.Session) {
	s.Session = sess
}

func (s *DistribService) SetQueryID(queryID string) {
	s.QueryID = queryID
}

// Command is a an event executer. This method is run from event. Returning errors
// may not have effect, as they are received by event service, log them immediately.
// If a client is found on this server, it gets the payload.
func (s *DistribService) Command(ctx context.Context, recipientSessionID string, payload any) error {
	// Check if client is connected to this server
	logger.Logger.Debugf("DistribService: DistribService.Command() recipientSessionID:%s", recipientSessionID)

	if WSSrv == nil {
		logger.Logger.Errorf("DistribService.Command: WSSrv is nil")
		return fmt.Errorf("WSSrv is nil")
	}

	if !WSSrv.HasClientID(recipientSessionID) {
		logger.Logger.Warnf("DistribService.Command: client session not found: %s (payload: %v)", recipientSessionID, payload)
		return fmt.Errorf("client session not found: %s", recipientSessionID)
	}

	if err := WSSrv.SendMessageToClientID(recipientSessionID, payload); err != nil {
		logger.Logger.Errorf("DistribService.Command WSSrv.SendMessageToClientID(): %v", err)
		return fmt.Errorf("DistribService.Command WSSrv.SendMessageToClientID(): %v", err)
	}

	return nil
}

// AddQuery adds a query-session pair.
func (s *DistribService) AddQuery(ctx context.Context, queryID, sessionID string) error {
	logger.Logger.Debugf("DistribService: DistribService.AddQuery() queryID:%s, sessionID:%s", queryID, sessionID)

	poolConn, connID, err := s.DB.GetPrimary(ctx)
	if err != nil {
		return fmt.Errorf("GetPrimary() failed: %w", err)
	}
	defer s.DB.Release(poolConn, connID)
	conn := poolConn.Conn()

	if _, err := conn.Exec(
		ctx,
		`INSERT INTO distrib_commands 
			(query_id, session_id)
			VALUES ($1, $2) 
		ON CONFLICT (query_id) DO UPDATE SET 
			session_id = EXCLUDED.session_id`,
		queryID, sessionID,
	); err != nil {
		return fmt.Errorf("Exec(): %v", err)
	}

	return nil
}

// SessionID retrieves a sessionID value for a given queryID.
func (s *DistribService) SessionID(ctx context.Context, queryID string) (string, error) {
	logger.Logger.Debugf("DistribService: DistribService.SessionID() queryID:%s", queryID)

	poolConn, connID, err := s.DB.GetPrimary(ctx)
	if err != nil {
		return "", fmt.Errorf("GetPrimary() failed: %w", err)
	}
	defer s.DB.Release(poolConn, connID)
	conn := poolConn.Conn()

	var sessionID string
	if err := conn.QueryRow(
		ctx,
		`SELECT 
			session_id 
		FROM distrib_commands 
		WHERE query_id = $1`,
		queryID,
	).Scan(&sessionID); err != nil {
		return "", fmt.Errorf("QueryRow(): %v", err)
	}

	if err := s.CollectGarbageWithConn(ctx, conn); err != nil {
		logger.Logger.Errorf("CollectGarbageWithConn(): %v", err)
	}

	return sessionID, nil
}

func (s *DistribService) CollectGarbage(ctx context.Context) error {
	poolConn, connID, err := s.DB.GetPrimary(ctx)
	if err != nil {
		return fmt.Errorf("GetPrimary() failed: %w", err)
	}
	defer s.DB.Release(poolConn, connID)
	conn := poolConn.Conn()

	return s.CollectGarbageWithConn(ctx, conn)
}

func (s *DistribService) CollectGarbageWithConn(ctx context.Context, conn ds.Conn) error {
	if _, err := conn.Exec(
		ctx,
		`DELETE 
		FROM distrib_commands 
		WHERE created_at < NOW() - ($1 * INTERVAL '1 minute')`,
		cmdExecMaxTime.Minutes(),
	); err != nil {
		return fmt.Errorf("Exec(): %v", err)
	}

	return nil
}
