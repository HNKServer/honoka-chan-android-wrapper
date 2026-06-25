package router

import (
	"honoka-chan/config"
	"honoka-chan/handler"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AndroidRouter exposes a tiny localhost-only control API for the Android APK
// wrapper. It does not change the public game/client protocol and does not alter
// the config.json schema.
func AndroidRouter(r *gin.Engine) {
	g := r.Group("/__android")
	{
		g.GET("/health", func(ctx *gin.Context) {
			ctx.JSON(http.StatusOK, gin.H{"ok": true})
		})

		g.POST("/config/reload", func(ctx *gin.Context) {
			if err := config.ReloadConfigFileStrict("./config.json"); err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{
					"ok":    false,
					"error": err.Error(),
				})
				return
			}

			handler.ReloadConfigGlobals()
			ctx.JSON(http.StatusOK, gin.H{"ok": true})
		})
	}
}
