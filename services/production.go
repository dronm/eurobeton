// Package services
package services

import (
	"context"

	crud "github.com/dronm/crudifier/v2"
	fields "github.com/dronm/crudifier/v2/fields"
	crudTypes "github.com/dronm/crudifier/v2/types"
	"github.com/dronm/ds/v4"
	"github.com/dronm/eurobeton/models"
	gmodels "github.com/dronm/gobizapp/models"
	gsrv "github.com/dronm/gobizapp/services"
	"github.com/dronm/session"
)

// ProductionService is a service for Production database object.
type ProductionService struct {
	DB      ds.Provider
	Session session.Session
}

// NewProductionService is a service constructor.
func NewProductionService(db ds.Provider, sess session.Session) *ProductionService {
	return &ProductionService{DB: db, Session: sess}
}

// Insert inserts one row into models.Production
func (s *ProductionService) Insert(ctx context.Context, model models.Production) (map[string]any, error) {
	retFields, err := gsrv.InsertModel(ctx, s.DB, &model, nil)
	if err != nil {
		return nil, err
	}

	PublishEvent(ctx, s.Session, "Production.insert", retFields)

	return retFields, nil
}

func (s *ProductionService) Update(ctx context.Context, keyModel models.ProductionKey, model models.Production) (int64, error) {
	cnt, err := gsrv.UpdateModel(ctx, s.DB, keyModel, &model)
	if err != nil {
		return 0, err
	}

	PublishEvent(ctx, s.Session, "Production.update", keyModel)

	return cnt, nil
}

// Delete removes row from models.Production
func (s *ProductionService) Delete(ctx context.Context, keyModels []models.ProductionKey) (int64, error) {
	models := make([]crudTypes.DBModel, len(keyModels))
	for i, m := range keyModels {
		models[i] = m
	}
	cnt, err := gsrv.DeleteModel(ctx, s.DB, models, nil)
	if err != nil {
		return 0, err
	}

	PublishEvent(ctx, s.Session, "Production.delete", keyModels)

	return cnt, nil
}

// FetchDetail retrieves one row from models.Production
func (s *ProductionService) FetchDetail(ctx context.Context, id int) (*models.ProductionList, error) {
	model := models.ProductionList{}
	if err := gsrv.FetchModel(ctx, s.DB, &models.ProductionKey{ID: fields.NewFieldInt(int64(id), true, false)}, &model); err != nil {
		return nil, err
	}
	return &model, nil
}

// FetchList retrieves rows from models.Production. Returns two models: data and aggregates.
func (s *ProductionService) FetchList(ctx context.Context, params crud.CollectionParams) ([]*models.ProductionList, *gmodels.TotCount, error) {
	return gsrv.FetchCollectionModel(ctx, s.DB, &models.ProductionList{}, &gmodels.TotCount{}, params)
}
