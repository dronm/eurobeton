package models

import (
	fields "github.com/dronm/crudifier/v2/fields"
)

const vehicleSchduleRealtion = "vehicle_schedules"

type VehicleScheduleKey struct {
	ID fields.FieldInt `json:"id" required:"true"`
}

func (m VehicleScheduleKey) Relation() string {
	return vehicleSchduleRealtion
}
