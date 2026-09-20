package router

import (
	product_controller "iox-service/controllers"
	upload_controller "iox-service/controllers"
	"iox-service/middleware"

	"github.com/gin-gonic/gin"
)

// SetupProductRoutes adds all product-related routes to a router group
func SetupProductRoutes(rg *gin.RouterGroup) {
	products := rg.Group("/products")
	{
		// Public routes (/:id/reviews and /:id/can-review must be before /:id)
		products.GET("", product_controller.GetProducts)
		products.GET("/seller/:sellerId", product_controller.GetProductsBySeller)
		products.GET("/:id/reviews", product_controller.GetProductReviews)
		products.GET("/:id/can-review", middleware.AuthMiddleware(), middleware.RequireRole("BUYER"), product_controller.CanReviewProduct)
		products.POST("/:id/reviews", middleware.AuthMiddleware(), middleware.RequireRole("BUYER"), product_controller.CreateReview)
		products.GET("/:id", product_controller.GetProductById)

		// Protected routes (require authentication)
		protected := products.Group("")
		protected.Use(middleware.AuthMiddleware())
		{
			activeSeller := middleware.RequireActiveSeller()
			protected.GET("/my-products", activeSeller, product_controller.GetMyProducts)
			protected.POST("", middleware.RequireRole("PRIVATE_SELLER", "BUSINESS_SELLER"), activeSeller, product_controller.CreateProduct)
			protected.POST("/upload", activeSeller, upload_controller.UploadProductImages)
			protected.PUT("/:id", activeSeller, product_controller.UpdateProduct)
			protected.DELETE("/:id", activeSeller, product_controller.DeleteProduct)
		}
	}
}
