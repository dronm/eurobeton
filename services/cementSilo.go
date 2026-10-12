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
	"github.com/dronm/session"

	gmodels "github.com/dronm/gobizapp/models"
	gsrv "github.com/dronm/gobizapp/services"
)

// for external upload fast access
type cementSiloMapToProduction struct {
	mx     sync.RWMutex
	idList map[string]int // key is production_site_id + material description, value is cementSiloID
}

var CementSiloMapToProductionList cementSiloMapToProduction

// CementSiloService is a service for CementSilo database object.
type CementSiloService struct {
	DB      ds.Provider
	Session session.Session
}

// NewCementSiloService is a service constructor.
func NewCementSiloService(db ds.Provider, sess session.Session) *CementSiloService {
	return &CementSiloService{DB: db, Session: sess}
}

// Insert inserts one row into models.CementSilo
func (s *CementSiloService) Insert(ctx context.Context, model models.CementSilo) (map[string]any, error) {
	retFields, err := gsrv.InsertModel(ctx, s.DB, &model, nil)
	if err != nil {
		return nil, err
	}

	PublishEvent(ctx, s.Session, "CementSilo.insert", retFields)

	return retFields, nil
}

func (s *CementSiloService) Update(ctx context.Context, keyModel models.CementSiloKey, model models.CementSilo) (int64, error) {
	cnt, err := gsrv.UpdateModel(ctx, s.DB, keyModel, &model)
	if err != nil {
		return 0, err
	}

	PublishEvent(ctx, s.Session, "CementSilo.update", keyModel)

	return cnt, nil
}

// Delete removes row from models.CementSilo
func (s *CementSiloService) Delete(ctx context.Context, keyModels []models.CementSiloKey) (int64, error) {
	models := make([]crudTypes.DBModel, len(keyModels))
	for i, m := range keyModels {
		models[i] = m
	}
	cnt, err := gsrv.DeleteModel(ctx, s.DB, models, nil)
	if err != nil {
		return 0, err
	}

	PublishEvent(ctx, s.Session, "CementSilo.delete", keyModels)

	return cnt, nil
}

// FetchDetail retrieves one row from models.CementSilo
func (s *CementSiloService) FetchDetail(ctx context.Context, id int) (*models.CementSiloList, error) {
	model := models.CementSiloList{}
	if err := gsrv.FetchModel(ctx, s.DB, &models.CementSiloKey{ID: fields.NewFieldInt(int64(id), true, false)}, &model); err != nil {
		return nil, err
	}
	return &model, nil
}

// FetchList retrieves rows from models.CementSilo. Returns two models: data and aggregates.
func (s *CementSiloService) FetchList(ctx context.Context, params crud.CollectionParams) ([]*models.CementSiloList, *gmodels.TotCount, error) {
	return gsrv.FetchCollectionModel(ctx, s.DB, &models.CementSiloList{}, &gmodels.TotCount{}, params)
}

// MapToProduction returns a silo ID for a production site and external description.
func (s *CementSiloService) MapToProduction(ctx context.Context, productionSiteID int, productionDescr string) (int, error) {
	key := fmt.Sprintf("%d%s", productionSiteID, productionDescr)
	CementSiloMapToProductionList.mx.RLock()
	id, ok := CementSiloMapToProductionList.idList[key]
	CementSiloMapToProductionList.mx.RUnlock()
	if ok {
		return id, nil
	}

	poolConn, connID, err := s.DB.GetPrimary(ctx)
	if err != nil {
		return 0, fmt.Errorf("GetPrimary(): %w", err)
	}
	defer s.DB.Release(poolConn, connID)
	return s.mapToProductionWithConn(ctx, poolConn.Conn(), productionSiteID, productionDescr)
}

func (s *CementSiloService) mapToProductionWithConn(ctx context.Context, conn ds.Conn, productionSiteID int, productionDescr string) (int, error) {
	key := fmt.Sprintf("%d%s", productionSiteID, productionDescr)
	CementSiloMapToProductionList.mx.RLock()
	id, ok := CementSiloMapToProductionList.idList[key]
	CementSiloMapToProductionList.mx.RUnlock()
	if ok {
		return id, nil
	}

	CementSiloMapToProductionList.mx.Lock()
	defer CementSiloMapToProductionList.mx.Unlock()
	if CementSiloMapToProductionList.idList == nil {
		if err := s.initMapToProductionListWithConn(ctx, conn); err != nil {
			return 0, err
		}
	}
	if id, ok := CementSiloMapToProductionList.idList[key]; ok {
		return id, nil
	}
	return 0, fmt.Errorf("map not found for descr: %s", productionDescr)
}

func (s *CementSiloService) MapToProductionChange(ctx context.Context) error {
	return s.InitMapToProductionList(ctx)
}

func (s *CementSiloService) InitMapToProductionList(ctx context.Context) error {
	poolConn, connID, err := s.DB.GetPrimary(ctx)
	if err != nil {
		return fmt.Errorf("GetPrimary(): %w", err)
	}
	defer s.DB.Release(poolConn, connID)

	CementSiloMapToProductionList.mx.Lock()
	defer CementSiloMapToProductionList.mx.Unlock()
	return s.initMapToProductionListWithConn(ctx, poolConn.Conn())
}

// The caller holds the map's write lock and owns the supplied connection.
func (s *CementSiloService) initMapToProductionListWithConn(ctx context.Context, conn ds.Conn) error {
	rows, err := conn.Query(ctx,
		`SELECT
			id,
			production_descr,
			production_site_id
		FROM cement_silos`,
	)
	if err != nil {
		return fmt.Errorf("Query(): %w", err)
	}
	defer rows.Close()

	idList := make(map[string]int)
	for rows.Next() {
		cemSilo := models.CementSilo{}
		if err := rows.Scan(&cemSilo.ID, &cemSilo.ProductionDescr, &cemSilo.ProductionSiteID); err != nil {
			return fmt.Errorf("Scan(): %w", err)
		}
		idList[fmt.Sprintf("%d%s",
			cemSilo.ProductionSiteID.GetValue(), cemSilo.ProductionDescr.GetValue(),
		)] = int(cemSilo.ID.GetValue())
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("Err(): %w", err)
	}
	CementSiloMapToProductionList.idList = idList
	return nil
}
