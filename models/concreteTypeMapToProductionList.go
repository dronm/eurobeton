package models

import (
	gmodels "github.com/dronm/gobizapp/models"
)

const (
	concreteTypeMapToProductionListRelation = "concrete_type_map_to_production_list"
)

// ConcreteTypeMapToProductionList is a virtual table model.
type ConcreteTypeMapToProductionList struct {
	ID              int         `json:"id" primaryKey:"true"`
	ConcreteTypeID  int         `json:"concrete_type_id"`
	ProductionDescr string      `json:"production_descr" required:"true" max:"100"`
	ConcreteType    gmodels.Ref `json:"concrete_types_ref"`
}

func (m ConcreteTypeMapToProductionList) Relation() string {
	return concreteTypeMapToProductionListRelation
}

func (m ConcreteTypeMapToProductionList) CollectionAgg() any {
	return &gmodels.TotCount{TotCount: 0}
}
