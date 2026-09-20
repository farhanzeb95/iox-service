package router

import (
	user_controller "iox-service/controllers"
	"iox-service/middleware"
	model "iox-service/models"

	"github.com/gin-gonic/gin"
)

// SetupUserRoutes adds all user-related routes to a router group
func SetupUserRoutes(rg *gin.RouterGroup) {
	users := rg.Group("/users")
	{
		users.POST("", user_controller.CreateUser)
		users.POST("/upload-signup-document", user_controller.UploadSignupDocument)
		users.GET("", middleware.AuthMiddleware(), middleware.RequireRole(string(model.TypeAdmin)), user_controller.GetUsers)
		users.POST("/login", user_controller.LoginUser)
		users.GET("/me", middleware.AuthMiddleware(), user_controller.GetCurrentUser)
		users.PATCH("/:id", middleware.AuthMiddleware(), user_controller.UpdateCurrentUser)
		users.GET("/:id", middleware.AuthMiddleware(), middleware.RequireRole(string(model.TypeAdmin)), user_controller.GetUserById)
		users.PATCH("/:id/status", middleware.AuthMiddleware(), middleware.RequireRole(string(model.TypeAdmin)), user_controller.UpdateSellerStatus)
	}
}
