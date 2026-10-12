// Package services
package services

import (
	"context"
	"fmt"
	"time"

	"github.com/dronm/eurobeton/models"

	crud "github.com/dronm/crudifier/v2"
	fields "github.com/dronm/crudifier/v2/fields"
	crudTypes "github.com/dronm/crudifier/v2/types"
	"github.com/dronm/ds/v4"
	"github.com/dronm/session"

	gmodels "github.com/dronm/gobizapp/models"
	gsrv "github.com/dronm/gobizapp/services"
)

// MaterialMapToProductionService is a service for MaterialMapToProduction database object.
type MaterialMapToProductionService struct {
	DB      ds.Provider
	Session session.Session
}

// NewMaterialMapToProductionService is a service constructor.
func NewMaterialMapToProductionService(db ds.Provider, sess session.Session) *MaterialMapToProductionService {
	return &MaterialMapToProductionService{DB: db, Session: sess}
}

// Insert inserts one row into models.MaterialMapToProduction
func (s *MaterialMapToProductionService) Insert(ctx context.Context, model models.MaterialMapToProduction) (map[string]any, error) {
	retFields, err := gsrv.InsertModel(ctx, s.DB, &model, nil)
	if err != nil {
		return nil, err
	}

	PublishEvent(ctx, s.Session, "MaterialMapToProduction.Insert", retFields)

	return retFields, nil
}

func (s *MaterialMapToProductionService) Update(ctx context.Context, keyModel models.MaterialMapToProductionKey, model models.MaterialMapToProduction) (int64, error) {
	cnt, err := gsrv.UpdateModel(ctx, s.DB, keyModel, &model)
	if err != nil {
		return 0, err
	}

	PublishEvent(ctx, s.Session, "MaterialMapToProduction.Update", keyModel)

	return cnt, nil
}

// Delete removes row from models.MaterialMapToProduction
func (s *MaterialMapToProductionService) Delete(ctx context.Context, keyModels []models.MaterialMapToProductionKey) (int64, error) {
	models := make([]crudTypes.DBModel, len(keyModels))
	for i, m := range keyModels {
		models[i] = m
	}
	cnt, err := gsrv.DeleteModel(ctx, s.DB, models, nil)
	if err != nil {
		return 0, err
	}

	PublishEvent(ctx, s.Session, "MaterialMapToProduction.delete", keyModels)

	return cnt, nil
}

// FetchDetail retrieves one row from models.MaterialMapToProduction
func (s *MaterialMapToProductionService) FetchDetail(ctx context.Context, id int) (*models.MaterialMapToProductionList, error) {
	model := models.MaterialMapToProductionList{}
	if err := gsrv.FetchModel(ctx, s.DB, &models.MaterialMapToProductionKey{ID: fields.NewFieldInt(int64(id), true, false)}, &model); err != nil {
		return nil, err
	}
	return &model, nil
}

// FetchList retrieves rows from models.MaterialMapToProduction. Returns two models: data and aggregates.
func (s *MaterialMapToProductionService) FetchList(ctx context.Context, params crud.CollectionParams) ([]*models.MaterialMapToProductionList, *gmodels.TotCount, error) {
	return gsrv.FetchCollectionModel(ctx, s.DB, &models.MaterialMapToProductionList{}, &gmodels.TotCount{}, params)
}

func (s *MaterialMapToProductionService) Upsert(ctx context.Context, model models.MaterialMapToProduction, productionEnd time.Time) (map[string]any, error) {
	poolConn, connID, err := s.DB.GetPrimary(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetPrimary(): %w", err)
	}
	defer s.DB.Release(poolConn, connID)
	return s.upsertWithConn(ctx, poolConn.Conn(), model, productionEnd)
}

func (s *MaterialMapToProductionService) upsertWithConn(ctx context.Context, conn ds.Conn, model models.MaterialMapToProduction, productionEnd time.Time) (map[string]any, error) {
	var id int
	if err := conn.QueryRow(ctx,
		`SELECT material_fact_consumptions_add_material($1::int, $2::text, $3)`,
		model.ProductionSiteID.GetValue(),
		model.ProductionDescr.GetValue(),
		productionEnd,
	).Scan(&id); err != nil {
		return nil, fmt.Errorf("QueryRow(): %w", err)
	}
	retFields := map[string]any{"id": id}

	if s.Session != nil {
		if err := gsrv.PublishEventWithConn(ctx, conn, s.Session.SessionID(), "MaterialMapToProduction.Insert", retFields); err != nil {
			return nil, fmt.Errorf("PublishEventWithConn(): %w", err)
		}
	}

	return retFields, nil
}
