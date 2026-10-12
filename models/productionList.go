package models

import (
	fields "github.com/dronm/crudifier/v2/fields"

	gmodels "github.com/dronm/gobizapp/models"
)

const (
	productionListRelation = "productions_list"
)

// ProductionList is an object model for insert/update.
type ProductionList struct {
	ID                          fields.FieldInt        `json:"id" primaryKey:"true"`
	ProductionID                fields.FieldText       `json:"production_id"`
	ProductionSiteID            fields.FieldInt        `json:"production_site_id"`
	ProductionDtStart           fields.FieldDateTimeTZ `json:"production_dt_start"`
	ProductionDtEnd             fields.FieldDateTimeTZ `json:"production_dt_end"`
	ProductionUser              fields.FieldText       `json:"production_user"`
	ProductionVehicleDescr      fields.FieldText       `json:"production_vehicle_descr"`
	DtStartSet                  fields.FieldDateTimeTZ `json:"dt_start_set"`
	DtEndSet                    fields.FieldDateTimeTZ `json:"dt_end_set"`
	ProductionSitesRef          gmodels.Ref                    `json:"production_sites_ref"`
	ShipmentsRef                gmodels.Ref                    `json:"shipments_ref"`
	VehicleSchedulesRef         gmodels.Ref                    `json:"vehicle_schedules_ref"`
	OrdersRef                   gmodels.Ref                    `json:"orders_ref"`
	ConcreteTypesRef            gmodels.Ref                    `json:"concrete_types_ref"`
	ProductionConcreteTypeDescr fields.FieldText       `json:"production_concrete_type_descr"`
	VehicleID                   fields.FieldInt        `json:"vehicle_id"`
	ConcreteTypeID              fields.FieldInt        `json:"concrete_type_id"`
	OrderID                     fields.FieldInt        `json:"order_id"`
	MaterialToleranceViolated   fields.FieldBool       `json:"material_tolerance_violated"`
	ConcreteQuant               fields.FieldFloat      `json:"concrete_quant"`
}

func (m ProductionList) Relation() string {
	return productionListRelation
}

func (m ProductionList) CollectionAgg() any {
	return &gmodels.TotCount{TotCount: 0}
}

// ProductionListKey is an object key model.
type ProductionListKey struct {
	ID fields.FieldInt `json:"id" required:"true"`
}

func (m ProductionListKey) Relation() string {
	return productionListRelation
}

// ProductionListUpdate is an update model.
type ProductionListUpdate struct {
	Keys  ProductionListKey `json:"keys"`
	Model ProductionList    `json:"model"`
}

// ProductionListDelete is a delete model.
type ProductionListDelete struct {
	Keys []ProductionListKey `json:"keys"`
}
