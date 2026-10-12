package services

import (
	"context"
	"fmt"
	"time"

	"github.com/dronm/ds/v4"
	"github.com/dronm/eurobeton/models"
	"github.com/dronm/session"
)

type ShipmentService struct {
	DB      ds.Provider
	Session session.Session
	QueryID string
}

func NewShipmentService(db ds.Provider, sess session.Session) *ShipmentService {
	return &ShipmentService{DB: db, Session: sess}
}

func (s *ShipmentService) SetDB(db ds.Provider) {
	s.DB = db
}

func (s *ShipmentService) SetSession(sess session.Session) {
	s.Session = sess
}

func (s *ShipmentService) SetQueryID(queryID string) {
	s.QueryID = queryID
}

func (s *ShipmentService) ToKonkridUpsert(ctx context.Context, model models.ShipmentKey) error {
	poolConn, connID, err := s.DB.GetPrimary(ctx)
	if err != nil {
		return fmt.Errorf("GetPrimary(): %w", err)
	}
	defer s.DB.Release(poolConn, connID)
	conn := poolConn.Conn()

	q := `INSERT INTO public.shipments
			(date_time, order_id, vehicle_schedule_id, quant, user_id, shipped, client_mark,
			ship_date_time, demurrage, blanks_exist, production_site_id)
		SELECT 
			sh.date_time,
			(SELECT kd_o.id FROM orders AS kd_o WHERE kd_o.create_date_time = o.create_date_time LIMIT 1),
			konkrid_ship_get_schedule(sh.date_time::date, sh_v.plate::text),
			sh.quant,
			(const_reglament_user_val()->'keys'->>'id')::int,
			sh.shipped,
			sh.client_mark,
			sh.ship_date_time,
			sh.demurrage,
			sh.blanks_exist,
			sh.production_site_id
		FROM beton.shipments as sh
		LEFT JOIN beton.orders as o on o.id = sh.order_id
		LEFT JOIN beton.concrete_types as o_ct on o_ct.id = o.concrete_type_id
		LEFT JOIN beton.vehicle_schedules as sh_sched on sh_sched.id = sh.vehicle_schedule_id
		LEFT JOIN beton.vehicles as sh_v on sh_v.id = sh_sched.vehicle_id 
		WHERE sh.id = $1
		ON CONFLICT (order_id, date_time) 
		DO UPDATE
		SET 
			vehicle_schedule_id = excluded.vehicle_schedule_id, 
			quant = excluded.quant,
			shipped = excluded.shipped,
			client_mark = excluded.client_mark,
			ship_date_time = excluded.ship_date_time,
			demurrage = excluded.demurrage,
			blanks_exist = excluded.blanks_exist,
			production_site_id = excluded.production_site_id
	RETURNING date_time::date
	`

	var shipDate time.Time
	if err := conn.QueryRow(ctx, q, model.ID.GetValue()).Scan(&shipDate); err != nil {
		return fmt.Errorf("QueryRow(): %v", err)
	}

	evnt := fmt.Sprintf(
		`SELECT pg_notify('Graph.change', json_build_object('params', json_build_object('cond_date', '%s'))::text)`,
		shipDate.Format("2006-01-02"),
	)
	PublishEvent(ctx, s.Session, evnt, nil)

	return nil
}
