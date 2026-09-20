package router

import (
	order_controller "iox-service/controllers"
	"iox-service/middleware"

	"github.com/gin-gonic/gin"
)

// SetupOrderRoutes adds order routes (auth required)
func SetupOrderRoutes(rg *gin.RouterGroup) {
	orders := rg.Group("/orders")
	orders.Use(middleware.AuthMiddleware())
	{
		orders.POST("", middleware.RequireRole("BUYER"), order_controller.PlaceOrder)
		orders.GET("", middleware.RequireActiveSeller(), order_controller.GetMyOrders)
		orders.GET("/:id", middleware.RequireActiveSeller(), order_controller.GetOrderByID)
		orders.PATCH("/:id/status", middleware.RequireActiveSeller(), order_controller.UpdateOrderStatus)
		orders.POST("/:id/return", middleware.RequireRole("BUYER"), order_controller.RequestReturn)
	}
}
