package routes

import (
	"github.com/gin-gonic/gin"
	"hris-core-service/internal/handlers"
)

func SetupAuthenticationRoutes(router *gin.Engine, handler *handlers.AuthenticationHandler, bearer gin.HandlerFunc) {
	group := router.Group("/v1/auth")
	group.POST("/login", handler.Login)
	group.GET("/me", bearer, handler.Me)
	group.POST("/logout", bearer, handler.Logout)
	group.POST("/logout-all", bearer, handler.LogoutAll)
}
