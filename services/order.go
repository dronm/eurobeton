package services

import (
	"context"
	"fmt"
	"time"

	"github.com/dronm/ds/v4"
	"github.com/dronm/eurobeton/models"
	"github.com/dronm/session"
)

type OrderService struct {
	DB      ds.Provider
	Session session.Session
	QueryID string
}

func NewOrderService(db ds.Provider, sess session.Session) *OrderService {
	return &OrderService{DB: db, Session: sess}
}

func (s *OrderService) SetDB(db ds.Provider) {
	s.DB = db
}

func (s *OrderService) SetSession(sess session.Session) {
	s.Session = sess
}

func (s *OrderService) SetQueryID(queryID string) {
	s.QueryID = queryID
}

func (s *OrderService) ToBeregInsert(ctx context.Context, model models.OrderKey) error {
	return s.ToBeregUpsert(ctx, model)
}

func (s *OrderService) ToBeregUpdate(ctx context.Context, model models.OrderKey) error {
	return s.ToBeregUpsert(ctx, model)
}

func (s *OrderService) ToBeregDelete(ctx context.Context, model models.Order) error {
	return s.Delete(ctx, model)
}

func (s *OrderService) ToBeregUpsert(ctx context.Context, model models.OrderKey) error {
	poolConn, connID, err := s.DB.GetPrimary(ctx)
	if err != nil {
		return fmt.Errorf("GetPrimary(): %w", err)
	}
	defer s.DB.Release(poolConn, connID)
	conn := poolConn.Conn()

	searchSchema := "konkrid"
	q := fmt.Sprintf(`INSERT INTO orders
		(client_id, destination_id, concrete_type_id, unload_type, comment_text, descr, date_time, time_to,
		quant, phone_cel, unload_speed, user_id, client_mark, date_time_to,
		lang_id, total, concrete_price, destination_price, unload_price, pump_vehicle_id,
		pay_cash, total_edit, payed, under_control, create_date_time, last_modif_user_id, last_modif_date_time,
		f_val, w_val)
	SELECT
		(const_konkrid_client_val()->'keys'->>'id')::int,
		(const_self_ship_dest_val()->'keys'->>'id')::int,
		(SELECT c.id FROM concrete_types as c WHERE c.name = ct.name::text),	
		o.unload_type,
		case
			WHEN o.pump_vehicle_id IS NOT NULL THEN
				coalesce(vh.plate::text,'')||' '||coalesce(vh.make::text,'')||coalesce(' ('||pvh.pump_length::text||coalesce(', '||person_init(pvh_own.name)::text,'')||')','')
			ELSE o.comment_text
		end,
		'Антон Александрович',
		o.date_time,
		o.time_to,
		o.quant,
		'9829440903',
		o.unload_speed,
		(const_reglament_user_val()->'keys'->>'id')::int,
		o.client_mark,
		o.date_time_to,
		(const_def_lang_val()->'keys'->>'id')::int,
		o.total, o.concrete_price, o.destination_price, o.unload_price,
		NULL, -- pump_vehicle_id in comment!!!	
		o.pay_cash, o.total_edit, o.payed, o.under_control, o.create_date_time,
		(const_reglament_user_val()->'keys'->>'id')::int,
		now(),
		o.f_val, o.w_val
	FROM
		%s.orders AS o
	LEFT JOIN %s.concrete_types AS ct ON ct.id = o.concrete_type_id
	LEFT JOIN %s.pump_vehicles AS pvh ON pvh.id = o.pump_vehicle_id
	LEFT JOIN %s.vehicles AS vh ON vh.id = pvh.vehicle_id
	LEFT JOIN %s.vehicle_owners pvh_own ON pvh_own.id = vh.vehicle_owner_id
	WHERE
		o.id = $1
	ON CONFLICT (create_date_time)
	DO UPDATE SET
		unload_type = excluded.unload_type,
		comment_text = excluded.comment_text,
		date_time = excluded.date_time,
		time_to = excluded.time_to,
		quant = excluded.quant,
		concrete_type_id = excluded.concrete_type_id,
		unload_speed = excluded.unload_speed,
		client_mark = excluded.client_mark,
		date_time_to = excluded.date_time_to,
		total = excluded.total,
		concrete_price = excluded.concrete_price,
		destination_price = excluded.destination_price,
		unload_price = excluded.unload_price,
		pay_cash = excluded.pay_cash, 
		total_edit = excluded.total_edit,
		payed = excluded.payed,
		under_control = excluded.under_control,
		last_modif_date_time = now(),
		f_val = excluded.f_val,
		w_val = excluded.w_val
	RETURNING date_time::date
	`, searchSchema, searchSchema, searchSchema, searchSchema, searchSchema)

	var orderDate time.Time
	if err := conn.QueryRow(ctx, q, model.ID.GetValue()).Scan(&orderDate); err != nil {
		return fmt.Errorf("QueryRow(): %v", err)
	}

	evnt := fmt.Sprintf(
		`SELECT pg_notify('Graph.change', json_build_object('params', json_build_object('cond_date', '%s'))::text)`,
		orderDate.Format("2006-01-02"),
	)
	PublishEvent(ctx, s.Session, evnt, nil)

	return nil
}

func (s *OrderService) ToKonkridInsert(ctx context.Context, model models.OrderKey) error {
	return s.ToKonkridUpsert(ctx, model)
}

func (s *OrderService) ToKonkridUpdate(ctx context.Context, model models.OrderKey) error {
	return s.ToKonkridUpsert(ctx, model)
}

func (s *OrderService) ToKonkridDelete(ctx context.Context, model models.Order) error {
	return s.Delete(ctx, model)
}

func (s *OrderService) ToKonkridUpsert(ctx context.Context, model models.OrderKey) error {
	poolConn, connID, err := s.DB.GetPrimary(ctx)
	if err != nil {
		return fmt.Errorf("GetPrimary(): %w", err)
	}
	defer s.DB.Release(poolConn, connID)
	conn := poolConn.Conn()

	searchSchema := "beton"
	q := fmt.Sprintf(`INSERT INTO orders
		(client_id, destination_id, concrete_type_id, unload_type, comment_text, descr, date_time, time_to,
		quant, phone_cel, unload_speed, user_id, client_mark, number, date_time_to,
		lang_id, total, concrete_price, destination_price, unload_price, pump_vehicle_id,
		pay_cash, total_edit, payed, under_control, create_date_time, last_modif_user_id, last_modif_date_time,
		f_val, w_val)
	SELECT
		(const_konkrid_client_val()->'keys'->>'id')::int,
		(const_self_ship_dest_val()->'keys'->>'id')::int,
		(SELECT c.id FROM concrete_types as c WHERE c.name = ct.name::text),	
		o.unload_type,
		o.comment_text,
		o.descr,
		o.date_time,
		o.time_to,
		o.quant,
		o.phone_cel,
		o.unload_speed,
		(const_reglament_user_val()->'keys'->>'id')::int,
		o.client_mark,
		o.number,
		o.date_time_to,
		(const_def_lang_val()->'keys'->>'id')::int,
		o.total, o.concrete_price, o.destination_price, o.unload_price,
		konkrid_get_pump_veh_id(vh.plate),
		o.pay_cash, o.total_edit, o.payed, o.under_control, o.create_date_time,
		(const_reglament_user_val()->'keys'->>'id')::int,
		now(),
		o.f_val, o.w_val
	FROM
		%s.orders AS o
	LEFT JOIN %s.concrete_types AS ct ON ct.id = o.concrete_type_id
	LEFT JOIN %s.pump_vehicles AS pvh ON pvh.id = o.pump_vehicle_id
	LEFT JOIN %s.vehicles AS vh ON vh.id = pvh.vehicle_id
	WHERE
		o.id = $1
	ON CONFLICT (create_date_time)
	DO UPDATE SET
		unload_type = excluded.unload_type,
		date_time = excluded.date_time,
		time_to = excluded.time_to,
		quant = excluded.quant,
		concrete_type_id = excluded.concrete_type_id,
		unload_speed = excluded.unload_speed,
		date_time_to = excluded.date_time_to,
		total = excluded.total,
		concrete_price = excluded.concrete_price,
		destination_price = excluded.destination_price,
		unload_price = excluded.unload_price,
		pump_vehicle_id = excluded.pump_vehicle_id, 
		pay_cash = excluded.pay_cash,
		total_edit = excluded.total_edit,
		payed = excluded.payed,
		under_control = excluded.under_control,
		last_modif_date_time = now(),
		f_val = excluded.f_val,
		w_val = excluded.w_val
	RETURNING date_time::date
	`, searchSchema, searchSchema, searchSchema, searchSchema)

	var orderDate time.Time
	if err := conn.QueryRow(ctx, q, model.ID.GetValue()).Scan(&orderDate); err != nil {
		return fmt.Errorf("QueryRow(): %v", err)
	}

	evnt := fmt.Sprintf(
		`SELECT pg_notify('Graph.change', json_build_object('params', json_build_object('cond_date', '%s'))::text)`,
		orderDate.Format("2006-01-02"),
	)
	PublishEvent(ctx, s.Session, evnt, nil)

	return nil
}

func (s *OrderService) Delete(ctx context.Context, model models.Order) error {
	poolConn, connID, err := s.DB.GetPrimary(ctx)
	if err != nil {
		return fmt.Errorf("GetPrimary(): %w", err)
	}
	defer s.DB.Release(poolConn, connID)
	conn := poolConn.Conn()

	q := `DELETE FROM orders WHERE create_date_time = $1::timestamp without time zone 
	RETURNING date_time::date`

	var orderDate time.Time
	if err := conn.QueryRow(ctx, q, model.CreateDateTime.GetValue()).Scan(&orderDate); err != nil {
		return fmt.Errorf("QueryRow(): %v", err)
	}

	evnt := fmt.Sprintf(
		`SELECT pg_notify('Graph.change', json_build_object('params', json_build_object('cond_date', '%s'))::text)`,
		orderDate.Format("2006-01-02"),
	)
	PublishEvent(ctx, s.Session, evnt, nil)

	return nil
}
