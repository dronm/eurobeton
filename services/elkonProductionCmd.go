package services

import (
	"context"
	"fmt"

	"github.com/dronm/eurobeton/models"

	"github.com/dronm/ds/v4"
	"github.com/dronm/gobizapp/logger"
	"github.com/dronm/session"
)

type ElkonProductionCmdService struct {
	DB      ds.Provider
	Session session.Session
	QueryID string
}

func NewElkonProductionCmdService(db ds.Provider, sess session.Session) *ElkonProductionCmdService {
	return &ElkonProductionCmdService{DB: db, Session: sess}
}

func (s *ElkonProductionCmdService) SetDB(db ds.Provider) {
	s.DB = db
}

func (s *ElkonProductionCmdService) SetSession(sess session.Session) {
	s.Session = sess
}

func (s *ElkonProductionCmdService) SetQueryID(queryID string) {
	s.QueryID = queryID
}

// These methods are called from browser, transmitted to weigh app
// through events.

// SetConfigParams sends command to retrieve elkon parameter
func (s *ElkonProductionCmdService) SetConfigParams(ctx context.Context, productionSiteID int, configParams map[string]any) error {
	msg := models.ElkonCommand{Cmd: "set", QueryID: s.QueryID, Params: configParams}
	return s.SendElkonCommand(ctx, productionSiteID, &msg)
}

func (s *ElkonProductionCmdService) SetLastProduction(ctx context.Context, productionSiteID int, productionID int) error {
	msg := models.ElkonCommand{Cmd: "set_last_production", QueryID: s.QueryID, Params: map[string]any{"production_id": productionID}}
	return s.SendElkonCommand(ctx, productionSiteID, &msg)
}

// FetchProduction sends command to retrieve elkon production by its ID.
func (s *ElkonProductionCmdService) FetchProduction(ctx context.Context, productionSiteID int, productionID int) error {
	msg := models.ElkonCommand{Cmd: "fetch_production", QueryID: s.QueryID, Params: map[string]any{"production_id": productionID}}
	return s.SendElkonCommand(ctx, productionSiteID, &msg)
}

func (s *ElkonProductionCmdService) FetchConfig(ctx context.Context, productionSiteID int) error {
	msg := models.ElkonCommand{Cmd: "fetch_config", QueryID: s.QueryID}
	return s.SendElkonCommand(ctx, productionSiteID, &msg)
}

// FetchLog sends command to retrieve elkon N log lines.
func (s *ElkonProductionCmdService) FetchLog(ctx context.Context, productionSiteID int, lineCount int) error {
	msg := models.ElkonCommand{Cmd: "fetch_log", QueryID: s.QueryID, Params: map[string]any{"count": lineCount}}
	return s.SendElkonCommand(ctx, productionSiteID, &msg)
}

// SaveConfig sends command to save elkon configuration to a file.
func (s *ElkonProductionCmdService) SaveConfig(ctx context.Context, productionSiteID int) error {
	msg := models.ElkonCommand{Cmd: "save", QueryID: s.QueryID}
	return s.SendElkonCommand(ctx, productionSiteID, &msg)
}

// ReloadConfig sends command to save elkon configuration to a file.
func (s *ElkonProductionCmdService) ReloadConfig(ctx context.Context, productionSiteID int) error {
	msg := models.ElkonCommand{Cmd: "reload", QueryID: s.QueryID}
	return s.SendElkonCommand(ctx, productionSiteID, &msg)
}

// Stop sends command to save elkon configuration to a file.
func (s *ElkonProductionCmdService) Stop(ctx context.Context, productionSiteID int) error {
	msg := models.ElkonCommand{Cmd: "stop", QueryID: s.QueryID}
	return s.SendElkonCommand(ctx, productionSiteID, &msg)
}

func (s *ElkonProductionCmdService) SendElkonCommand(ctx context.Context, productionSiteID int, msg *models.ElkonCommand) error {
	// recipient
	elkonSrv := NewElkonProductionService(s.DB, s.Session)
	sessID, err := elkonSrv.ElkonProductionSiteSessionID(ctx, productionSiteID)
	if err != nil {
		return fmt.Errorf("ElkonProductionService.ElkonProductionSiteSessionID(): %v", err)
	}

	distrServ := NewDistribService(s.DB, s.Session)
	if err := distrServ.AddQuery(ctx, msg.QueryID, s.Session.SessionID()); err != nil {
		return fmt.Errorf("AddQuery(): %v", err)
	}

	logger.Logger.Debugf("ElkonProductionService: sending command to elkon with PublishEvent Distrib.Command, params: %v, recipient site session:%s", msg, sessID)
	PublishEvent(ctx, s.Session, "Distrib.Command", models.DistribCommand{
		RecipientSessionID: sessID,
		Payload:            msg,
	})

	return nil
}
