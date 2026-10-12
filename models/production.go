package models

import (
	fields "github.com/dronm/crudifier/v2/fields"

	gmodels "github.com/dronm/gobizapp/models"
)

const (
	productionRelation = "productions"
)

// Production is an object model for insert/update.
type Production struct {
	ID                          fields.FieldInt        `json:"id" primaryKey:"true" srvCalc:"true"`
	ProductionID                fields.FieldText       `json:"production_id" required:"true" alias:"External production ID"`
	ProductionDtStart           fields.FieldDateTimeTZ `json:"production_dt_start" alias:"Начало производства в ExtProd"`
	ProductionDtEnd             fields.FieldDateTimeTZ `json:"production_dt_end" alias:"Окончание производства в ExtProg"`
	ProductionUser              fields.FieldText       `json:"production_user" alias:"Пользователь ExternalProd" max:"150"`
	ProductionVehicleDescr      fields.FieldText       `json:"production_vehicle_descr" alias:"ТС в External" max:"5"`
	DtStartSet                  fields.FieldDateTimeTZ `json:"dt_start_set"`
	DtEndSet                    fields.FieldDateTimeTZ `json:"dt_end_set"`
	ProductionSiteID            fields.FieldInt        `json:"production_site_id" required:"true"`
	ShipmentID                  fields.FieldInt        `json:"shipment_id"`
	ConcreteTypeID              fields.FieldInt        `json:"concrete_type_id"`
	VehicleScheduleStateID      fields.FieldInt        `json:"vehicle_schedule_state_id"`
	VehicleID                   fields.FieldInt        `json:"vehicle_id"`
	ProductionConcreteTypeDescr fields.FieldText       `json:"production_concrete_type_descr"`
	MaterialToleranceViolated   fields.FieldBool       `json:"material_tolerance_violated"`
	ConcreteQuant               fields.FieldFloat      `json:"concrete_quant"`
	ManualCorrection            fields.FieldBool       `json:"manual_correction" alias:"Ручное исправление"`
}

func (m Production) Relation() string {
	return productionRelation
}

func (m Production) CollectionAgg() any {
	return &gmodels.TotCount{TotCount: 0}
}

// ProductionKey is an object key model.
type ProductionKey struct {
	ID fields.FieldInt `json:"id" required:"true"`
}

func (m ProductionKey) Relation() string {
	return productionRelation
}

// ProductionUpdate is an update model.
type ProductionUpdate struct {
	Keys  ProductionKey `json:"keys"`
	Model Production    `json:"model"`
}

// ProductionDelete is a delete model.
type ProductionDelete struct {
	Keys []ProductionKey `json:"keys"`
}
