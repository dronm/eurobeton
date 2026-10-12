package services

import (
	"github.com/dronm/gobizapp/api"
)

func RegisterServices() {
	api.RegisterMethods("Event", &EventService{})
	api.RegisterMethods("Distrib", &DistribService{})
	api.RegisterMethods("ElkonProduction", &ElkonProductionService{})
	api.RegisterMethods("ElkonProductionCmd", &ElkonProductionCmdService{})
	api.RegisterMethods("Vehicle", &VehicleService{})
	api.RegisterMethods("VehicleSchedule", &VehicleScheduleService{})
	api.RegisterMethods("Shipment", &ShipmentService{})
	api.RegisterMethods("Order", &OrderService{})
	api.RegisterMethods("CarTracking", &CarTrackingService{})

	// NEW SERVICES Do not modify this comment. It is a new service mark used by autogenerator
}
