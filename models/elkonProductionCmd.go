package models

type ElkonProductionCmdFetchLog struct {
	ProductionSiteID int `json:"production_site_id"`
	Count            int `json:"count"`
}

type ElkonProductionCmdSetLastProduction struct {
	ProductionSiteID int `json:"production_site_id"`
	ProductionID     int `json:"production_id"`
}

type ElkonProductionCmdFetchProduction struct {
	ProductionSiteID int `json:"production_site_id"`
	ProductionID     int `json:"production_id"`
}

type ElkonProductionCmdSetConfigParams struct {
	ProductionSiteID int            `json:"production_site_id"`
	Params           map[string]any `json:"params"`
}

type ElkonProductionCmd struct {
	ProductionSiteID int `json:"production_site_id"`
}
