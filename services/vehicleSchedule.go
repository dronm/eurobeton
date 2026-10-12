package services

import (
	"context"
	"fmt"

	"github.com/dronm/ds/v4"
	"github.com/dronm/eurobeton/models"
	"github.com/dronm/session"
)

type VehicleScheduleService struct {
	DB      ds.Provider
	Session session.Session
	QueryID string
}

func NewVehicleScheduleService(db ds.Provider, sess session.Session) *VehicleScheduleService {
	return &VehicleScheduleService{DB: db, Session: sess}
}

func (s *VehicleScheduleService) SetDB(db ds.Provider) {
	s.DB = db
}

func (s *VehicleScheduleService) SetSession(sess session.Session) {
	s.Session = sess
}

func (s *VehicleScheduleService) SetQueryID(queryID string) {
	s.QueryID = queryID
}

func (s *VehicleScheduleService) ToKonkridDelete(ctx context.Context, model models.VehicleScheduleKey) error {
	poolConn, connID, err := s.DB.GetPrimary(ctx)
	if err != nil {
		return fmt.Errorf("GetPrimary(): %w", err)
	}
	defer s.DB.Release(poolConn, connID)
	conn := poolConn.Conn()

	q := `DELETE FROM vehicle_schedules WHERE id = (
		WITH beton_data AS (
			SELECT 
				bsch.schedule_date,
				bv.plate
			FROM beton.vehicle_schedules AS bsch
			LEFT JOIN beton.vehicles AS bv ON bv.id = bsch.vehicle_id
			WHERE bsch.id = $1 
		)
		SELECT
			sch.id
		FROM vehicle_schedules AS sch
		WHERE sch.schedule_date = (SELECT schedule_date FROM beton_data)
				AND sch.vehicle_id = (SELECT v.id FROM vehicles AS v WHERE v.plate = (SELECT plate FROM beton_data))
		)`

	if _, err := conn.Exec(ctx, q, model.ID); err != nil {
		return fmt.Errorf("delete Exec(): %v", err)
	}

	evnt := fmt.Sprintf(`SELECT pg_notify('VehicleScheduleState.update',
				json_build_object('params', json_build_object('keys', json_build_object('id', %d)))::text)`, model.ID.GetValue())

	PublishEvent(ctx, s.Session, evnt, nil)

	return nil
}

func (s *VehicleScheduleService) ToKonkridInsert(ctx context.Context, model models.VehicleScheduleKey) error {
	return s.ToKonkridUpsert(ctx, model)
}

func (s *VehicleScheduleService) ToKonkridUpdate(ctx context.Context, model models.VehicleScheduleKey) error {
	return s.ToKonkridUpsert(ctx, model)
}

func (s *VehicleScheduleService) ToKonkridUpsert(ctx context.Context, model models.VehicleScheduleKey) error {
	/*
		poolConn, connID, err := s.DB.GetPrimary(ctx)
		if err != nil {
			return fmt.Errorf("GetPrimary(): %w", err)
		}
		defer s.DB.Release(poolConn, connID)
		conn := poolConn.Conn()

		q := `INSERT INTO vehicle_schedules
			(schedule_date, vehicle_id, driver_id, auto_gen, production_base_id, edit_user_id, edit_date_time)
			SELECT
				bsch.schedule_date,
				(SELECT v.id FROM vehicles AS v WHERE v.plate = bv.plate),
				(SELECT dr.id FROM drivers AS dr WHERE dr.name = bdr.name),
				bsch.auto_gen,
				bsch.production_base_id,
				(const_reglament_user_val()->'keys'->>'id')::int,
				NOW()
			FROM beton.vehicle_schedules AS bsch
			LEFT JOIN beton.vehicles AS bv ON bv.id = bsch.vehicle_id
			LEFT JOIN beton.drivers AS bdr ON bdr.id = bsch.driver_id
			WHERE bsch.id = $1
			ON CONFLICT (schedule_date, vehicle_id) DO NOTHING
		`
	*/

	return nil
}
