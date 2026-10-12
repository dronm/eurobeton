package app

import (
	"path/filepath"

	"github.com/gin-gonic/gin"

	"github.com/dronm/eurobeton/config"
	"github.com/dronm/eurobeton/controllers"

	gcontr "github.com/dronm/gobizapp/controllers"
)

func RegisterRoutes(r *gin.Engine) {

	r.StaticFile("/favicon.ico", filepath.Join(config.Config.PublicStoreDir, "favicon.ico"))

	r.POST("/session/bootstrap", controllers.SessionBootstrap)

	r.GET("/api-get/:service/:method", gcontr.APIGet)
	r.POST("/api-post/:service/:method", gcontr.APIPost)

	// constant permissions are handled with roles_set, roles_get arrays
	r.GET("/constant/get", CheckPermission("Constant.get"), gcontr.ConstantGet)
	r.GET("/constant/list", CheckPermission("Constant.list"), gcontr.ConstantList)
	r.POST("/constant/set/:id", CheckPermission("Constant.set"), gcontr.ConstantSet)


	//NEW ROUTES DO not remove this line. It is a marker to add new routes.
}
