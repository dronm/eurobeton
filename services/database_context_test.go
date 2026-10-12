package services

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/dronm/ds/v4"
	"github.com/dronm/eurobeton/models"
	"github.com/dronm/gobizapp/errs"
)

type requestContextProvider struct {
	ds.Provider
	ctx      context.Context
	poolConn ds.PoolConn
}

func (p *requestContextProvider) GetPrimary(ctx context.Context) (ds.PoolConn, ds.ServerID, error) {
	p.ctx = ctx
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	return p.poolConn, "primary", nil
}

func (p *requestContextProvider) Release(conn ds.PoolConn, _ ds.ServerID) {
	conn.Release()
}

func TestServiceDatabaseAcquisitionUsesRequestContext(t *testing.T) {
	tests := []struct {
		name string
		run  func(context.Context, ds.Provider) error
	}{
		{"user password", func(ctx context.Context, db ds.Provider) error {
			return NewUserService(db, nil).UpdatePwd(ctx, 1, "password")
		}},
		{"password recovery", func(ctx context.Context, db ds.Provider) error {
			return NewUserService(db, nil).RecoverPassword(ctx, "user@example.com")
		}},
		{"order", func(ctx context.Context, db ds.Provider) error {
			return NewOrderService(db, nil).ToKonkridUpsert(ctx, models.OrderKey{})
		}},
		{"shipment", func(ctx context.Context, db ds.Provider) error {
			return NewShipmentService(db, nil).ToKonkridUpsert(ctx, models.ShipmentKey{})
		}},
		{"car tracking", func(ctx context.Context, db ds.Provider) error {
			return NewCarTrackingService(db, nil).ToKonkrid(ctx, models.CarTrackingKey{})
		}},
		{"vehicle schedule", func(ctx context.Context, db ds.Provider) error {
			return NewVehicleScheduleService(db, nil).ToKonkridDelete(ctx, models.VehicleScheduleKey{})
		}},
		{"distribution cleanup", func(ctx context.Context, db ds.Provider) error {
			return NewDistribService(db, nil).CollectGarbage(ctx)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			db := &requestContextProvider{}
			err := tt.run(ctx, db)
			if db.ctx != ctx {
				t.Fatal("database acquisition did not receive the request context")
			}
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("expected a cancellation error, got %v", err)
			}
		})
	}
}

type missingUserConn struct {
	ds.Conn
	contexts []context.Context
}

func (c *missingUserConn) Prepare(ctx context.Context, _, _ string) (ds.PreparedStatement, error) {
	c.contexts = append(c.contexts, ctx)
	return nil, nil
}

func (c *missingUserConn) QueryRow(ctx context.Context, _ string, _ ...any) ds.Row {
	c.contexts = append(c.contexts, ctx)
	return missingUserRow{}
}

type missingUserRow struct{}

func (missingUserRow) Scan(...any) error {
	return ds.ErrNoRows
}

type serviceTestPoolConn struct {
	ds.PoolConn
	conn     ds.Conn
	released bool
}

func (c *serviceTestPoolConn) Conn() ds.Conn {
	return c.conn
}

func (c *serviceTestPoolConn) Release() {
	c.released = true
}

func TestLoginHandlesDSNoRowsAndPropagatesContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	conn := &missingUserConn{}
	poolConn := &serviceTestPoolConn{conn: conn}
	db := &requestContextProvider{poolConn: poolConn}

	result, err := NewUserService(db, nil).Login(ctx, UserLoginInf{}, models.User_login{})
	var publicErr errs.PublicError
	if result != nil || !errors.As(err, &publicErr) {
		t.Fatalf("expected a public login error for ds.ErrNoRows, got result=%v, err=%v", result, err)
	}
	if publicErr.HTTPStatus() != http.StatusForbidden || publicErr.Code() != errs.UserCredIncorrect {
		t.Fatalf("unexpected login error: status=%d, code=%v", publicErr.HTTPStatus(), publicErr.Code())
	}
	if db.ctx != ctx || len(conn.contexts) != 2 {
		t.Fatal("login did not acquire, prepare, and query using the request context")
	}
	for _, queryCtx := range conn.contexts {
		if queryCtx != ctx {
			t.Fatal("login query discarded the request context")
		}
	}
	if !poolConn.released {
		t.Fatal("login did not release its connection after missing credentials")
	}
}
