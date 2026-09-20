package router

import (
	seller_fee_controller "iox-service/controllers"
	"iox-service/middleware"
	model "iox-service/models"

	"github.com/gin-gonic/gin"
)

func SetupSellerStoreFeeRoutes(rg *gin.RouterGroup) {
	fees := rg.Group("/seller-fees")
	fees.Use(middleware.AuthMiddleware())
	{
		fees.GET("/me", middleware.RequireRole(string(model.TypePrivateSeller), string(model.TypeBusinessSeller)), seller_fee_controller.GetMySellerStoreFee)
		fees.POST("", middleware.RequireRole(string(model.TypePrivateSeller), string(model.TypeBusinessSeller)), seller_fee_controller.SubmitMySellerStoreFee)
		fees.GET("", middleware.RequireRole(string(model.TypeAdmin)), seller_fee_controller.ListSellerStoreFees)
		fees.PATCH("/:id/status", middleware.RequireRole(string(model.TypeAdmin)), seller_fee_controller.ReviewSellerStoreFee)
	}
}
