package models

import (
	fields "github.com/dronm/crudifier/v2/fields"

	gmodels "github.com/dronm/gobizapp/models"
)

const (
	concreteTypeMapToProductionRelation = "concrete_type_map_to_production"
)

// ConcreteTypeMapToProduction is an object model for insert/update.
type ConcreteTypeMapToProduction struct {
	ID              fields.FieldInt  `json:"id" primaryKey:"true" srvCalc:"true"`
	ConcreteTypeID  fields.FieldInt  `json:"concrete_type_id"`
	ProductionDescr fields.FieldText `json:"production_descr" required:"true" max:"100"`
}

func (m ConcreteTypeMapToProduction) Relation() string {
	return concreteTypeMapToProductionRelation
}

func (m ConcreteTypeMapToProduction) CollectionAgg() any {
	return &gmodels.TotCount{TotCount: 0}
}

// ConcreteTypeMapToProductionKey is an object key model.
type ConcreteTypeMapToProductionKey struct {
	ID fields.FieldInt `json:"id" required:"true"`
}

func (m ConcreteTypeMapToProductionKey) Relation() string {
	return concreteTypeMapToProductionRelation
}

// ConcreteTypeMapToProductionUpdate is an update model.
type ConcreteTypeMapToProductionUpdate struct {
	Keys  ConcreteTypeMapToProductionKey `json:"keys"`
	Model ConcreteTypeMapToProduction    `json:"model"`
}

// ConcreteTypeMapToProductionDelete is a delete model.
type ConcreteTypeMapToProductionDelete struct {
	Keys []ConcreteTypeMapToProductionKey `json:"keys"`
}

