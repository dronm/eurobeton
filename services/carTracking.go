package services

import (
	"context"
	"fmt"

	"github.com/dronm/ds/v4"
	"github.com/dronm/eurobeton/models"
	"github.com/dronm/session"
)

type CarTrackingService struct {
	DB      ds.Provider
	Session session.Session
	QueryID string
}

func NewCarTrackingService(db ds.Provider, sess session.Session) *CarTrackingService {
	return &CarTrackingService{DB: db, Session: sess}
}

func (s *CarTrackingService) SetDB(db ds.Provider) {
	s.DB = db
}

func (s *CarTrackingService) SetSession(sess session.Session) {
	s.Session = sess
}

func (s *CarTrackingService) SetQueryID(queryID string) {
	s.QueryID = queryID
}

func (s *CarTrackingService) ToKonkrid(ctx context.Context, model models.CarTrackingKey) error {
	poolConn, connID, err := s.DB.GetPrimary(ctx)
	if err != nil {
		return fmt.Errorf("GetPrimary(): %w", err)
	}
	defer s.DB.Release(poolConn, connID)
	conn := poolConn.Conn()

	if _, err := conn.Exec(ctx,
		`INSERT INTO
			public.car_tracking
		SELECT
			* 
		FROM
			beton.car_tracking
		WHERE
			car_id = $1 AND period = $2
		ON CONFLICT (car_id, period)
		DO NOTHING`,
		model.CarID, model.Period,
	); err != nil {
		return fmt.Errorf("car_tracking insert Exec(): %v", err)
	}

	return nil
}
