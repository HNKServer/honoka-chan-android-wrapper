package main

import (
	"honoka-chan/config"
	"honoka-chan/router"
	_ "honoka-chan/tools"

	"github.com/gin-gonic/gin"
)

func main() {
	gin.SetMode(gin.ReleaseMode)

	// Router
	r := gin.Default()
	router.SifRouter(r)
	router.AsRouter(r)

	// Android wrapper internal API. This only adds /__android/* endpoints and
	// does not change the original client-facing API. Optional Android helper fields are backward compatible.
	router.AndroidRouter(r)

	r.Run(":" + config.Conf.Settings.ServerPort)
}
