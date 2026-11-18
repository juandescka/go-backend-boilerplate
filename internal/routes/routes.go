package routes

import (
	"net/http"

	"hris-core-service/internal/handlers"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(router *gin.Engine, operational *handlers.OperationalHandler) {
	router.GET("/health", operational.Health)
	router.GET("/ready", operational.Ready)
}

func SetupMetricsRoute(router *gin.Engine, path string, handler http.Handler) {
	router.GET(path, gin.WrapH(handler))
}
