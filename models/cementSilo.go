package models

import (
	fields "github.com/dronm/crudifier/v2/fields"

	gmodels "github.com/dronm/gobizapp/models"
)

const (
	cementSiloRelation = "cement_silos"
)

// CementSilo is an object model for insert/update.
type CementSilo struct {
	ID               fields.FieldInt   `json:"id" primaryKey:"true" srvCalc:"true"`
	ProductionSiteID fields.FieldInt   `json:"production_site_id" required:"true" `
	ProductionDescr  fields.FieldText  `json:"production_descr" required:"true" max:"100"`
	Name             fields.FieldText  `json:"name" required:"true" max:"100"`
	WeighAppName     fields.FieldText  `json:"weigh_app_name" max:"100"`
	LoadCapacity     fields.FieldFloat `json:"load_capacity"`
	Visible          fields.FieldBool  `json:"visible"`
}

func (m CementSilo) Relation() string {
	return cementSiloRelation
}

func (m CementSilo) CollectionAgg() any {
	return &gmodels.TotCount{TotCount: 0}
}

// CementSiloKey is an object key model.
type CementSiloKey struct {
	ID fields.FieldInt `json:"id" required:"true"`
}

func (m CementSiloKey) Relation() string {
	return cementSiloRelation
}

// CementSiloUpdate is an update model.
type CementSiloUpdate struct {
	Keys  CementSiloKey `json:"keys"`
	Model CementSilo    `json:"model"`
}

// CementSiloDelete is a delete model.
type CementSiloDelete struct {
	Keys []CementSiloKey `json:"keys"`
}

