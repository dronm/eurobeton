package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dronm/gobizapp/database"
	"github.com/dronm/gobizapp/logger"

	"github.com/dronm/eurobeton/app"
	"github.com/dronm/eurobeton/config"
	"github.com/dronm/eurobeton/services"
)

var (
	Version   = "dev" // default
	Commit    = "none"
	BuildDate = "unknown"
)

func main() {
	// Configuration file: first argument or PROG_FILE_NAME.json
	var iniFile string
	if len(os.Args) >= 2 {
		iniFile = os.Args[1]
	} else {
		iniFile = os.Args[0] + ".json"
	}

	srv := app.InitApp(iniFile)
	defer func() {
		if err := database.Close(); err != nil {
			logger.Logger.Errorf("database.Close(): %v", err)
		}
	}()

	services.ProgAbout.Name = "eurobeton"
	services.ProgAbout.Author.Name = "Andrey Mikhalevich"
	services.ProgAbout.Author.Email = "katrenplus@mail.ru"
	services.ProgAbout.Backend.Version = Version
	services.ProgAbout.Backend.Commit = Commit
	services.ProgAbout.Backend.BuildDate = BuildDate

	server := &http.Server{
		Addr:    config.Config.HTTPServer,
		Handler: srv,
	}

	// register services for dynamic calls with API
	services.RegisterServices()

	if config.Config.WSServer.Start {
		app.WSSrv.Serve(config.Config.WSServer.CleanupInterval)
		services.WSSrv = app.WSSrv
	}

	// event server
	app.EvntSrv.Serve()

	//elkon
	services.InitElkonFailoverService(database.DB, nil)
	services.ElkonFailoverSrv.Start()

	// Run server in a goroutine to avoid blocking
	go func() {
		logger.Logger.Info("Server is running on", config.Config.HTTPServer)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Logger.Fatalf("Server failed: %s\n", err)
		}
	}()

	// Channel to listen for termination signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Block until a termination signal is received
	<-quit
	logger.Logger.Info("Shutting down server...")

	// Context with timeout to allow ongoing requests to complete
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Config.AppShutdownTimeout)*time.Second)
	defer cancel()

	services.ElkonFailoverSrv.Shutdown(ctx)

	app.EvntSrv.Shutdown(ctx)

	if config.Config.WSServer.Start {
		app.WSSrv.Shutdown(ctx)
	}

	// Attempt to gracefully shut down the server
	if err := server.Shutdown(ctx); err != nil {
		logger.Logger.Fatalf("Server forced to shutdown: %s\n", err)
	}

	logger.Logger.Info("Server gracefully shutdown")
}
