package models

import (
	gmodels "github.com/dronm/gobizapp/models"
	"time"
)

const (
	materialMapToProductionListRelation = "material_map_to_production_list"
)

// MaterialMapToProductionList is a virtual table model.
type MaterialMapToProductionList struct {
	ID                 int         `json:"id" primaryKey:"true"`
	DateTime           time.Time   `json:"date_time"`
	MaterialsRef       gmodels.Ref `json:"materials_ref"`
	ProductionDescr    string      `json:"production_descr" required:"true" max:"100"`
	OrderID            *int        `json:"order_id"`
	MaterialID         *int        `json:"material_id"`
	MaterialOrd        *int        `json:"material_ord"`
	ProductionSitesRef gmodels.Ref `json:"production_sites_ref"`
	ProductionSiteID   *int        `json:"production_site_id"`
}

func (m MaterialMapToProductionList) Relation() string {
	return materialMapToProductionListRelation
}

func (m MaterialMapToProductionList) CollectionAgg() any {
	return &gmodels.TotCount{TotCount: 0}
}
