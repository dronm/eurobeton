package models

import (
	fields "github.com/dronm/crudifier/v2/fields"
)

const shipmentRealtion = "shipments"

type ShipmentKey struct {
	ID fields.FieldInt `json:"id" required:"true"`
}

func (m ShipmentKey) Relation() string {
	return shipmentRealtion
}

