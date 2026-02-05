package router

import (
	cart_controller "iox-service/controllers"
	"iox-service/middleware"

	"github.com/gin-gonic/gin"
)

// SetupCartRoutes adds cart routes (buyer only, auth required)
func SetupCartRoutes(rg *gin.RouterGroup) {
	cart := rg.Group("/cart")
	cart.Use(middleware.AuthMiddleware(), middleware.RequireRole("BUYER"))
	{
		cart.GET("", cart_controller.GetCart)
		cart.POST("", cart_controller.AddToCart)
		cart.PATCH("", cart_controller.UpdateQuantity)
		cart.DELETE("/:productId", cart_controller.RemoveFromCart)
		cart.DELETE("", cart_controller.ClearCart)
	}
}
