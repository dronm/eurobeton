package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/dronm/eurobeton/config"
	"github.com/dronm/eurobeton/models"

	"github.com/dronm/ds/v4"
	"github.com/dronm/session"
)

// VehicleService is a service for CementSilo database object.
type VehicleService struct {
	DB      ds.Provider
	Session session.Session
	QueryID string
}

func NewVehicleService(db ds.Provider, sess session.Session) *VehicleService {
	return &VehicleService{DB: db, Session: sess}
}

func (s *VehicleService) SetDB(db ds.Provider) {
	s.DB = db
}

func (s *VehicleService) SetSession(sess session.Session) {
	s.Session = sess
}

func (s *VehicleService) SetQueryID(queryID string) {
	s.QueryID = queryID
}

func (s *VehicleService) RebuildRoute(ctx context.Context, model models.VehicleRebuildRoute) error {
	if config.Config.HTTPProgDir == "" {
		return fmt.Errorf("HTTPProgDir not defined")
	}

	script := filepath.Join(config.Config.HTTPProgDir, "functions", "vehicle_rebuild_route.php")
	if _, err := os.Stat(script); os.IsNotExist(err) {
		return fmt.Errorf("vehicle_rebuild_route.php not found")
	}

	//cmd := "php "+script+" "+args.TrackerID.String()+" "+args.ShipmentID.String()+" "+args.VehicleState.String()
	//wsapp.DebugLogger.Println("Executing:",cmd)
	//_ = exec.Command("/bin/sh", cmd)
	cmd := exec.Command(config.Config.PHPDir, script, model.TrackerID.String(), model.ShipmentID.String(), model.VehicleState.String())
	if _, err := cmd.Output(); err != nil {
		return fmt.Errorf("output: %s, cmd: %s, trackerID: %d, shipmentID: %d, vehicleState: %s",
			err.Error(), script, model.TrackerID.GetValue(), model.ShipmentID.GetValue(), model.VehicleState.GetValue(),
		)
	}

	return nil
}
