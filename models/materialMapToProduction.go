package models

import (
	fields "github.com/dronm/crudifier/v2/fields"

	gmodels "github.com/dronm/gobizapp/models"
)

const (
	materialMapToProductionRelation = "material_map_to_production"
)

// MaterialMapToProduction is an object model for insert/update.
type MaterialMapToProduction struct {
	ID               fields.FieldInt        `json:"id" primaryKey:"true" srvCalc:"true"`
	DateTime         fields.FieldDateTimeTZ `json:"date_time" required:"true"`
	MaterialID       fields.FieldInt        `json:"material_id"`
	ProductionDescr  fields.FieldText       `json:"production_descr" required:"true" max:"100"`
	OrderID          fields.FieldInt        `json:"order_id"`
	ProductionSiteID fields.FieldInt        `json:"production_site_id"`
}

func (m MaterialMapToProduction) Relation() string {
	return materialMapToProductionRelation
}

func (m MaterialMapToProduction) CollectionAgg() any {
	return &gmodels.TotCount{TotCount: 0}
}

// MaterialMapToProductionKey is an object key model.
type MaterialMapToProductionKey struct {
	ID fields.FieldInt `json:"id" required:"true"`
}

func (m MaterialMapToProductionKey) Relation() string {
	return materialMapToProductionRelation
}

// MaterialMapToProductionUpdate is an update model.
type MaterialMapToProductionUpdate struct {
	Keys  MaterialMapToProductionKey `json:"keys"`
	Model MaterialMapToProduction    `json:"model"`
}

// MaterialMapToProductionDelete is a delete model.
type MaterialMapToProductionDelete struct {
	Keys []MaterialMapToProductionKey `json:"keys"`
}
