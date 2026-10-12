package services

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dronm/crudifier/v2/fields"
	"github.com/dronm/ds/v4"
	"github.com/dronm/eurobeton/models"
	"github.com/dronm/gobizapp/logger"
	"github.com/dronm/session"
	"github.com/sirupsen/logrus"
)

type productionTestSession struct {
	session.Session
}

func (productionTestSession) SessionID() string { return "elkon-test" }
func (productionTestSession) GetString(key string) string {
	if key == "user_id" {
		return "1"
	}
	return ""
}

type productionTestProvider struct {
	ds.Provider
	conn     *productionTestConn
	ctx      context.Context
	acquires int
	releases int
	acquire  func(context.Context) error
}

func (p *productionTestProvider) GetPrimary(ctx context.Context) (ds.PoolConn, ds.ServerID, error) {
	p.acquires++
	if ctx != p.ctx {
		return nil, "", errors.New("acquisition lost caller context")
	}
	if p.acquire != nil {
		if err := p.acquire(ctx); err != nil {
			return nil, "", err
		}
	}
	if p.acquires > 1 {
		return nil, "", errors.New("second acquisition would exhaust this one-connection pool")
	}
	return &productionTestLease{p: p}, "primary", nil
}

func (p *productionTestProvider) Release(pc ds.PoolConn, _ ds.ServerID) { pc.Release() }

type productionTestLease struct{ p *productionTestProvider }

func (l *productionTestLease) Conn() ds.Conn { return l.p.conn }
func (l *productionTestLease) Release()      { l.p.releases++ }

type productionTestResult struct{}

func (productionTestResult) RowsAffected() int64 { return 1 }

type productionTestRow struct{ scan func(...any) error }

func (r productionTestRow) Scan(dest ...any) error { return r.scan(dest...) }

type productionTestRows struct {
	scan func(...any) error
	next bool
}

func (*productionTestRows) Close() error { return nil }
func (*productionTestRows) Err() error   { return nil }
func (r *productionTestRows) Next() bool {
	next := r.next
	r.next = false
	return next
}
func (r *productionTestRows) Scan(dest ...any) error { return r.scan(dest...) }

type productionTestConn struct {
	ds.Conn
	ctx          context.Context
	commands     []string
	materialArgs [][]any
	failUpdate   error
	failInsert   bool
	committed    bool
	rolledBack   bool
}

func (c *productionTestConn) record(ctx context.Context, sql string) error {
	if ctx != c.ctx {
		return errors.New("database call lost caller context")
	}
	c.commands = append(c.commands, sql)
	return nil
}

func (c *productionTestConn) Exec(ctx context.Context, sql string, args ...any) (ds.ExecResult, error) {
	if err := c.record(ctx, sql); err != nil {
		return nil, err
	}
	if strings.HasPrefix(sql, "UPDATE production_sites") && c.failUpdate != nil {
		return nil, c.failUpdate
	}
	if sql == "materialInsertQuery" {
		c.materialArgs = append(c.materialArgs, args)
		if c.failInsert {
			return nil, errors.New("material insert failed")
		}
	}
	return productionTestResult{}, nil
}

func (c *productionTestConn) Query(ctx context.Context, sql string, args ...any) (ds.Rows, error) {
	if err := c.record(ctx, sql); err != nil {
		return nil, err
	}
	if strings.Contains(sql, "FROM cement_silos") {
		return &productionTestRows{next: true, scan: func(dest ...any) error {
			*dest[0].(*fields.FieldInt) = fields.NewFieldIntVal(9)
			*dest[1].(*fields.FieldText) = fields.NewFieldTextVal("Cement")
			*dest[2].(*fields.FieldInt) = fields.NewFieldIntVal(3)
			return nil
		}}, nil
	}
	return &productionTestRows{}, nil
}

func (c *productionTestConn) QueryRow(ctx context.Context, sql string, args ...any) ds.Row {
	err := c.record(ctx, sql)
	return productionTestRow{scan: func(dest ...any) error {
		if err != nil {
			return err
		}
		if sql == "productionInsertQuery" {
			for _, d := range dest {
				*d.(**int) = nil
			}
			return nil
		}
		if strings.Contains(sql, "material_fact_consumptions_add_") {
			*dest[0].(*int) = 42
			return nil
		}
		return errors.New("unexpected row query: " + sql)
	}}
}

func (c *productionTestConn) Prepare(ctx context.Context, name string, sql string) (ds.PreparedStatement, error) {
	if err := c.record(ctx, "prepare:"+name); err != nil {
		return nil, err
	}
	return productionTestStatement{conn: c, name: name}, nil
}

func (c *productionTestConn) Begin(ctx context.Context) (ds.Tx, error) {
	if err := c.record(ctx, "begin"); err != nil {
		return nil, err
	}
	return productionTestTx{productionTestConn: c}, nil
}

type productionTestStatement struct {
	conn *productionTestConn
	name string
}

func (s productionTestStatement) Name() string { return s.name }
func (s productionTestStatement) Exec(ctx context.Context, args ...any) (ds.ExecResult, error) {
	return s.conn.Exec(ctx, s.name, args...)
}
func (s productionTestStatement) Query(ctx context.Context, args ...any) (ds.Rows, error) {
	return s.conn.Query(ctx, s.name, args...)
}
func (s productionTestStatement) QueryRow(ctx context.Context, args ...any) ds.Row {
	return s.conn.QueryRow(ctx, s.name, args...)
}

type productionTestTx struct{ *productionTestConn }

func (tx productionTestTx) Commit(ctx context.Context) error {
	tx.committed = true
	return tx.record(ctx, "commit")
}
func (tx productionTestTx) Rollback(ctx context.Context) error {
	tx.rolledBack = true
	return tx.record(ctx, "rollback")
}

func newProductionTestProvider(t *testing.T, ctx context.Context) *productionTestProvider {
	t.Helper()
	oldLogger := logger.Logger
	logger.Logger = logrus.New()
	t.Cleanup(func() { logger.Logger = oldLogger })
	return &productionTestProvider{ctx: ctx, conn: &productionTestConn{ctx: ctx}}
}

func TestElkonPingUsesOneLeaseAndPropagatesUpdateError(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "failed site update"}[fail], func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			p := newProductionTestProvider(t, ctx)
			if fail {
				p.conn.failUpdate = errors.New("site update failed")
			}
			err := NewElkonProductionService(p, productionTestSession{}).Ping(ctx, 3, nil)
			if fail && !errors.Is(err, p.conn.failUpdate) {
				t.Fatalf("expected site update error, got %v", err)
			}
			if !fail && err != nil {
				t.Fatal(err)
			}
			if p.acquires != 1 || p.releases != 1 {
				t.Fatalf("leases: acquired %d, released %d", p.acquires, p.releases)
			}
			wantCommands := 3 // update, log, notify on the same connection
			if fail {
				wantCommands = 1
			}
			if len(p.conn.commands) != wantCommands {
				t.Fatalf("unexpected commands: %v", p.conn.commands)
			}
		})
	}
}

func TestElkonUploadUsesOneLeaseWithColdMappingsAndAtomicMaterials(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "commit", true: "rollback"}[fail], func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			p := newProductionTestProvider(t, ctx)
			p.conn.failInsert = fail
			ConcreteTypeMapToProductionList.mx.Lock()
			oldMap := ConcreteTypeMapToProductionList.idList
			ConcreteTypeMapToProductionList.idList = nil
			ConcreteTypeMapToProductionList.mx.Unlock()
			CementSiloMapToProductionList.mx.Lock()
			oldSiloMap := CementSiloMapToProductionList.idList
			CementSiloMapToProductionList.idList = nil
			CementSiloMapToProductionList.mx.Unlock()
			t.Cleanup(func() {
				ConcreteTypeMapToProductionList.mx.Lock()
				ConcreteTypeMapToProductionList.idList = oldMap
				ConcreteTypeMapToProductionList.mx.Unlock()
				CementSiloMapToProductionList.mx.Lock()
				CementSiloMapToProductionList.idList = oldSiloMap
				CementSiloMapToProductionList.mx.Unlock()
			})
			oldFailover := ElkonFailoverSrv
			ElkonFailoverSrv = &ElkonFailoverService{filename: t.TempDir() + "/failover.dat"}
			t.Cleanup(func() { ElkonFailoverSrv = oldFailover })

			now := time.Now()
			msg := models.ElkonProductionList{{
				ID: 58697, ProductionDtStart: &now, ProductionDtEnd: &now,
				ConcreteDescr: "B25", ConcreteQuant: 2,
				Mat1Descr: "Sand", Mat1Quant: 1200, Mat1QuantReq: 1300,
				Mat2Descr: "Cement", Mat2Quant: 600, Mat2QuantReq: 650, Mat2Cement: true,
			}}
			err := NewElkonProductionService(p, productionTestSession{}).Upload(ctx, 3, msg)
			if (err != nil) != fail {
				t.Fatalf("error = %v, expected failure = %v", err, fail)
			}
			if p.acquires != 1 || p.releases != 1 {
				t.Fatalf("leases: acquired %d, released %d", p.acquires, p.releases)
			}
			if p.conn.committed != !fail || p.conn.rolledBack != fail {
				t.Fatalf("transaction state: committed %v, rolled back %v", p.conn.committed, p.conn.rolledBack)
			}
			wantInserts := 2
			if fail {
				wantInserts = 1
			}
			if len(p.conn.materialArgs) != wantInserts {
				t.Fatalf("material inserts = %d", len(p.conn.materialArgs))
			}
			args := p.conn.materialArgs[0]
			if args[12] != 1.2 || args[13] != 1.3 || args[15] != "58697" {
				t.Fatalf("incorrect material quantities/production ID: %v", args)
			}
			if !fail && *p.conn.materialArgs[1][14].(*int) != 9 {
				t.Fatalf("incorrect cement silo: %v", p.conn.materialArgs[1])
			}
			var txCommands []string
			for _, cmd := range p.conn.commands {
				if cmd == "begin" || cmd == "materialDeleteQuery" || cmd == "materialInsertQuery" || cmd == "commit" || cmd == "rollback" {
					txCommands = append(txCommands, cmd)
				}
			}
			last := "commit"
			if fail {
				last = "rollback"
			}
			want := []string{"begin", "materialDeleteQuery", "materialInsertQuery"}
			if !fail {
				want = append(want, "materialInsertQuery")
			}
			want = append(want, last)
			if !reflect.DeepEqual(txCommands, want) {
				t.Fatalf("transaction command order = %v, want %v", txCommands, want)
			}
		})
	}
}

func TestElkonAcquisitionUsesRequestCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := newProductionTestProvider(t, ctx)
	p.acquire = func(ctx context.Context) error {
		<-ctx.Done() // model an exhausted pool waiting for a request cancellation
		return ctx.Err()
	}
	err := NewElkonProductionService(p, productionTestSession{}).Ping(ctx, 3, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	if p.acquires != 1 || p.releases != 0 || len(p.conn.commands) != 0 {
		t.Fatalf("unexpected activity after failed acquire: %+v", p)
	}
}
