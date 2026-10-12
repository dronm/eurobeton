package models

import (
	fields "github.com/dronm/crudifier/v2/fields"
)

const orderRealation = "orders"

type Order struct {
	CreateDateTime fields.FieldDateTime `json:"create_date_time"`
}

type OrderKey struct {
	ID fields.FieldInt `json:"id" required:"true"`
}

func (m OrderKey) Relation() string {
	return orderRealation
}
