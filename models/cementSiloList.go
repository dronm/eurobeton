package models

import (
	gmodels "github.com/dronm/gobizapp/models"
)

const (
	cementSiloListRelation = "cement_silos_list"
)

// CementSiloList is a virtual table model.
type CementSiloList struct {
	ID               int         `json:"id" primaryKey:"true" srvCalc:"true"`
	ProductionSiteID int         `json:"production_site_id"`
	ProductionSite   gmodels.Ref `json:"production_sites_ref"`
	ProductionDescr  string      `json:"production_descr" required:"true" max:"100"`
	Name             string      `json:"name" required:"true" max:"100"`
	WeighAppName     *string     `json:"weigh_app_name" required:"true" max:"100"`
	LoadCapacity     *float64    `json:"load_capacity"`
	Visible          *bool       `json:"visible"`
}

func (m CementSiloList) Relation() string {
	return cementSiloListRelation
}

func (m CementSiloList) CollectionAgg() any {
	return &gmodels.TotCount{TotCount: 0}
}

