package models

import (
	fields "github.com/dronm/crudifier/v2/fields"
)

const carTrackingRealtion = "car_tracking"

type CarTrackingKey struct {
	CarID  fields.FieldText     `json:"car_id" required:"true"`
	Period fields.FieldDateTime `json:"period" required:"true"`
}

func (m CarTrackingKey) Relation() string {
	return carTrackingRealtion
}
