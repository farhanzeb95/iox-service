package router

import (
	returns_controller "iox-service/controllers"
	"iox-service/middleware"

	"github.com/gin-gonic/gin"
)

// SetupReturnsRoutes adds return routes (auth required)
func SetupReturnsRoutes(rg *gin.RouterGroup) {
	returns := rg.Group("/returns")
	returns.Use(middleware.AuthMiddleware())
	{
		returns.GET("", returns_controller.GetMyReturns)
		returns.PATCH("/:id/status", returns_controller.UpdateReturnStatus)
	}
}
