package router

import (
	user_controller "iox-service/controllers"

	"github.com/gin-gonic/gin"
)

// SetupUserRoutes adds all user-related routes to a router group
func SetupUserRoutes(rg *gin.RouterGroup) {
	users := rg.Group("/users")
	{
		users.POST("", user_controller.CreateUser)
		users.GET("", user_controller.GetUsers) // optional: list all users
		users.GET("/:id", user_controller.GetUserById)
		users.POST("/login", user_controller.LoginUser)
		// you can add PUT, DELETE routes here
	}
}
