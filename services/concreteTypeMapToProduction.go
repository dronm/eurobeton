// Package services
package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/dronm/eurobeton/models"

	crud "github.com/dronm/crudifier/v2"
	fields "github.com/dronm/crudifier/v2/fields"
	crudTypes "github.com/dronm/crudifier/v2/types"
	"github.com/dronm/ds/v4"
	gmodels "github.com/dronm/gobizapp/models"
	gsrv "github.com/dronm/gobizapp/services"
	"github.com/dronm/session"
)

// for external upload fast access
type concreteTypeMapToProduction struct {
	mx     sync.RWMutex
	idList map[string]int // key is concrete description in external app, value is concrete ID in database
}

var ConcreteTypeMapToProductionList concreteTypeMapToProduction

// ConcreteTypeMapToProductionService is a service for ConcreteTypeMapToProduction database object.
type ConcreteTypeMapToProductionService struct {
	DB      ds.Provider
	Session session.Session
}

// NewConcreteTypeMapToProductionService is a service constructor.
func NewConcreteTypeMapToProductionService(db ds.Provider, sess session.Session) *ConcreteTypeMapToProductionService {
	return &ConcreteTypeMapToProductionService{DB: db, Session: sess}
}

// Insert inserts one row into models.ConcreteTypeMapToProduction
func (s *ConcreteTypeMapToProductionService) Insert(ctx context.Context, model models.ConcreteTypeMapToProduction) (map[string]any, error) {
	retFields, err := gsrv.InsertModel(ctx, s.DB, &model, nil)
	if err != nil {
		return nil, err
	}

	PublishEvent(ctx, s.Session, "ConcreteTypeMapToProduction.insert", retFields)

	return retFields, nil
}

func (s *ConcreteTypeMapToProductionService) Update(ctx context.Context, keyModel models.ConcreteTypeMapToProductionKey, model models.ConcreteTypeMapToProduction) (int64, error) {
	cnt, err := gsrv.UpdateModel(ctx, s.DB, keyModel, &model)
	if err != nil {
		return 0, err
	}

	PublishEvent(ctx, s.Session, "ConcreteTypeMapToProduction.update", keyModel)

	return cnt, nil
}

// Delete removes row from models.ConcreteTypeMapToProduction
func (s *ConcreteTypeMapToProductionService) Delete(ctx context.Context, keyModels []models.ConcreteTypeMapToProductionKey) (int64, error) {
	models := make([]crudTypes.DBModel, len(keyModels))
	for i, m := range keyModels {
		models[i] = m
	}
	cnt, err := gsrv.DeleteModel(ctx, s.DB, models, nil)
	if err != nil {
		return 0, err
	}

	PublishEvent(ctx, s.Session, "ConcreteTypeMapToProduction.delete", keyModels)

	return cnt, nil
}

// FetchDetail retrieves one row from models.ConcreteTypeMapToProduction
func (s *ConcreteTypeMapToProductionService) FetchDetail(ctx context.Context, id int) (*models.ConcreteTypeMapToProductionList, error) {
	model := models.ConcreteTypeMapToProductionList{}
	if err := gsrv.FetchModel(ctx, s.DB, &models.ConcreteTypeMapToProductionKey{ID: fields.NewFieldInt(int64(id), true, false)}, &model); err != nil {
		return nil, err
	}
	return &model, nil
}

// FetchList retrieves rows from models.ConcreteTypeMapToProduction. Returns two models: data and aggregates.
func (s *ConcreteTypeMapToProductionService) FetchList(ctx context.Context, params crud.CollectionParams) ([]*models.ConcreteTypeMapToProductionList, *gmodels.TotCount, error) {
	return gsrv.FetchCollectionModel(ctx, s.DB, &models.ConcreteTypeMapToProductionList{}, &gmodels.TotCount{}, params)
}

func (s *ConcreteTypeMapToProductionService) InitMapToProductionList(ctx context.Context) error {
	poolConn, connID, err := s.DB.GetPrimary(ctx)
	if err != nil {
		return fmt.Errorf("GetPrimary(): %w", err)
	}
	defer s.DB.Release(poolConn, connID)

	ConcreteTypeMapToProductionList.mx.Lock()
	defer ConcreteTypeMapToProductionList.mx.Unlock()
	return s.initMapToProductionListWithConn(ctx, poolConn.Conn())
}

// The caller holds the map's write lock and owns the supplied connection.
func (s *ConcreteTypeMapToProductionService) initMapToProductionListWithConn(ctx context.Context, conn ds.Conn) error {
	rows, err := conn.Query(ctx,
		`SELECT
			production_descr,
			concrete_type_id
		FROM concrete_type_map_to_production
		WHERE concrete_type_id IS NOT NULL AND production_descr IS NOT NULL`,
	)
	if err != nil {
		return fmt.Errorf("Query(): %w", err)
	}
	defer rows.Close()

	idList := make(map[string]int)
	for rows.Next() {
		var productionDescr string
		var concreteTypeID int
		if err := rows.Scan(&productionDescr, &concreteTypeID); err != nil {
			return fmt.Errorf("Scan(): %w", err)
		}
		idList[productionDescr] = concreteTypeID
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("Err(): %w", err)
	}
	ConcreteTypeMapToProductionList.idList = idList
	return nil
}

func (s *ConcreteTypeMapToProductionService) Change(ctx context.Context) error {
	return s.InitMapToProductionList(ctx)
}

// MapToProduction returns a concrete type ID for the external description.
func (s *ConcreteTypeMapToProductionService) MapToProduction(ctx context.Context, productionDescr string) (int, error) {
	ConcreteTypeMapToProductionList.mx.RLock()
	id, ok := ConcreteTypeMapToProductionList.idList[productionDescr]
	ConcreteTypeMapToProductionList.mx.RUnlock()
	if ok {
		return id, nil
	}

	// Acquire before locking so uploads that already own a lease can use the map.
	poolConn, connID, err := s.DB.GetPrimary(ctx)
	if err != nil {
		return 0, fmt.Errorf("GetPrimary(): %w", err)
	}
	defer s.DB.Release(poolConn, connID)
	return s.mapToProductionWithConn(ctx, poolConn.Conn(), productionDescr)
}

func (s *ConcreteTypeMapToProductionService) mapToProductionWithConn(ctx context.Context, conn ds.Conn, productionDescr string) (int, error) {
	ConcreteTypeMapToProductionList.mx.RLock()
	id, ok := ConcreteTypeMapToProductionList.idList[productionDescr]
	ConcreteTypeMapToProductionList.mx.RUnlock()
	if ok {
		return id, nil
	}

	ConcreteTypeMapToProductionList.mx.Lock()
	defer ConcreteTypeMapToProductionList.mx.Unlock()
	if ConcreteTypeMapToProductionList.idList == nil {
		if err := s.initMapToProductionListWithConn(ctx, conn); err != nil {
			return 0, err
		}
	}
	if id, ok := ConcreteTypeMapToProductionList.idList[productionDescr]; ok {
		return id, nil
	}

	if err := conn.QueryRow(ctx, `SELECT material_fact_consumptions_add_concrete_type($1)`, productionDescr).Scan(&id); err != nil {
		return 0, fmt.Errorf("SELECT material_fact_consumptions_add_concrete_type: %w", err)
	}
	ConcreteTypeMapToProductionList.idList[productionDescr] = id
	return id, nil
}
