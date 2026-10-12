package models

import (
	fields "github.com/dronm/crudifier/v2/fields"
)

type VehicleRebuildRoute struct {
	TrackerID    fields.FieldInt  `json:"tracker_id"`
	ShipmentID   fields.FieldInt  `json:"shipment_id"`
	VehicleState fields.FieldText `json:"vehicle_state"`
}
