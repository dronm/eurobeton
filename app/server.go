// Package app
package app

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dronm/crudifier/v2/metadata"
	"github.com/dronm/ds/v4/pgds"
	"github.com/dronm/session"

	"github.com/dronm/gobizapp/database"
	"github.com/dronm/gobizapp/logger"
	"github.com/dronm/gobizapp/middleware"
	gsrv "github.com/dronm/gobizapp/services"

	"github.com/dronm/eurobeton/config"
	"github.com/dronm/eurobeton/constants"
	"github.com/dronm/eurobeton/controllers"
	"github.com/dronm/eurobeton/enums"
	sess "github.com/dronm/eurobeton/session"

	"github.com/dronm/eurobeton/beregmw"
	"github.com/dronm/eurobeton/eventServer"
	"github.com/dronm/eurobeton/services"
	"github.com/dronm/eurobeton/ws"
)

var PermissionList map[string]struct{} // key is a method + role

var (
	EvntSrv *eventServer.EventServer
	WSSrv   *ws.WSServer
)

const defMiddlewareTimeout = 5000 // default timeout in milliseconds

// InitApp initializes application:
// reads config file, setup logger, database connection,
// initializes session support, sets up routes and returns a Gin engine instance
func InitApp(configFileName string) *gin.Engine {
	config.Config = config.AppConfig{}
	err := config.Config.ReadConf(configFileName)
	if err != nil {
		panic(fmt.Sprintf("config.ReadConf() failed: %v", err))
	}

	if err := logger.Initialize(logger.LoggerLogLevel(config.Config.LogLevel)); err != nil {
		panic(fmt.Sprintf("logger.Initialize() failed: %v", err))
	}

	if config.Config.DebugQueries {
		logger.Logger.Info("Debug queries is set")
	}

	master, err := IsClusterMaster()
	if err != nil {
		panic(fmt.Errorf("IsClusterMaster(): %v", err))
	}

	var localEvents map[string]struct{}
	if config.Config.AppID == "beton" {
		if master {
			localEvents = map[string]struct{}{
				"Vehicle.rebuild_route": {},
				"Order.to_bereg_insert": {},
				"Order.to_bereg_update": {},
				"Order.to_bereg_delete": {},
				"Distrib.Command":       {},
			}
		} else {
			localEvents = map[string]struct{}{
				"Distrib.Command": {},
			}
		}
	} else if config.Config.AppID == "concrete1" {
		if master {
			localEvents = map[string]struct{}{
				"Vehicle.rebuild_route":             {},
				"CarTracking.to_konkrid":            {},
				"Shipment.to_konkrid":               {},
				"Order.to_konkrid_insert":           {},
				"Order.to_konkrid_update":           {},
				"Order.to_konkrid_delete":           {},
				"VehicleSchedule.to_konkrid_insert": {},
				"VehicleSchedule.to_konkrid_delete": {},
				"VehicleSchedule.to_konkrid_update": {},
				"Distrib.Command":                   {},
			}
		} else {
			localEvents = map[string]struct{}{
				"Distrib.Command": {},
			}
		}
	}
	// localEvents["CementSilo.MapToProductionChange"] = struct{}{}
	// localEvents["ConcreteTypeMapToProduction.Change"] = struct{}{}
	EvntSrv = eventServer.NewEventServer(localEvents)

	gsrv.Configer = &config.Config

	// database
	startupCtx, cancelStartup := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelStartup()
	if err := database.Initialize(startupCtx, config.Config.DB, EvntSrv.OnNotification); err != nil {
		panic(fmt.Sprintf("database.Initialize() failed: %v", err))
	}

	pgProvider, ok := database.DB.(*pgds.Provider)
	if !ok {
		panic("database provider must support PostgreSQL sessions and notifications")
	}
	primaryPool, err := pgProvider.PrimaryPool()
	if err != nil {
		panic(fmt.Sprintf("pgProvider.PrimaryPool() failed: %v", err))
	}
	if err := sess.Initialize(config.Config.Session, primaryPool, config.Config.LogLevel); err != nil {
		panic(fmt.Sprintf("session.Initialize() failed: %v", err))
	}

	if config.Config.WSServer.Start {
		wsInit := ws.WSInit{
			Addr:            config.Config.WSServer.Address,
			EventServer:     EvntSrv,
			SessManager:     sess.SessManager,
			CheckPermission: CheckPermission,
			IsProduction:    config.Config.Production,
			URL:             "/beton/*sessID",
			SessCookieKey:   "token",
		}
		WSSrv = ws.NewWSServer(wsInit)
		gsrv.EvHandler = WSSrv
	}

	EvntSrv.DBPool = primaryPool
	EvntSrv.SocketServer = WSSrv

	services.EvHandler = WSSrv

	controllers.ReportErrors = config.Config.ReportErrors

	// constants specific validations
	constants.Init()

	metadata.ControledTags = []string{"export"} // for excel exclude fields

	// permissions
	perm := services.NewPermissionService(database.DB)
	PermissionList, err = perm.ReadList(startupCtx)
	if err != nil {
		panic(fmt.Sprintf("perm.ReadList(): %v", err))
	}

	router := gin.Default()

	var ginMode string
	if config.Config.Production {
		ginMode = gin.ReleaseMode
	} else {
		ginMode = gin.DebugMode
	}
	gin.SetMode(ginMode)

	router.Use(func(c *gin.Context) {
		logger.Logger.Debugf("Path: %s | UA: %s", c.Request.URL.Path, c.Request.UserAgent())
		c.Next()
	})

	middlewareTimeout := config.Config.MiddlewareTimeout
	if middlewareTimeout == 0 {
		middlewareTimeout = defMiddlewareTimeout
	}
	router.Use(middleware.TimeoutMiddleware(time.Duration(middlewareTimeout) * time.Millisecond))

	router.Use(middleware.CorsMiddleware(config.Config.BaseURL))

	router.Use(beregmw.SessionMiddleware(
		sess.SessManager,
		sess.SessCookieKey,
		config.Config.Production,
	))

	// Define routes
	RegisterRoutes(router)

	enums.InitEnums()

	return router
}

func IsMethodAllowed(userSess session.Session, method string) error {
	_, err := services.LoggedUserID(userSess)
	if err != nil {
		return fmt.Errorf("IsMethodAllowed: LoggedUserID(): %v", err)
	}
	roleID := userSess.GetString("role_id")

	//all methods are allowed
	if roleID == enums.RoleTypesOwner ||
		roleID == enums.RoleTypesBoss ||
		method == "WS.Init" {

		return nil
	}

	// if empty - for all users.
	if method != "" {
		_, ok := PermissionList[method+"."+roleID]
		if !ok {
			logger.Logger.Errorf("method '%s' not allowed for role '%s'", method, roleID)
			return fmt.Errorf("method %s not allowed", method)
		}
	}
	return nil
}

// CheckPermission is called for protected routes, where user is already logged,
// there is a structure UserLogin saved in session.
func CheckPermission(method string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userSess := controllers.GetSession(c, "CheckPermission")
		if err := IsMethodAllowed(userSess, method); err != nil {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}

// IsClusterMaster returns true if we are on main cluster server (for running certain commands)
func IsClusterMaster() (bool, error) {
	out, err := exec.Command(config.Config.IsClusterMasterScript).Output()
	if err != nil {
		return false, err
	}
	if len(out) == 0 {
		return false, fmt.Errorf("IsClusterMaster() returned 0 length result")
	}
	return out[0] == '1', nil
}
