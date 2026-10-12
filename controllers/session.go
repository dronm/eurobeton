package controllers

import (
	"net/http"

	gcontr "github.com/dronm/gobizapp/controllers"

	"github.com/gin-gonic/gin"
)

func SessionBootstrap(c *gin.Context) {
	//start session if not exists yet
	_ = gcontr.GetSession(c, "SessionBootstrap")
	c.Status(http.StatusNoContent)
}
