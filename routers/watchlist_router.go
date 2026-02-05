package router

import (
	watchlist_controller "iox-service/controllers"
	"iox-service/middleware"

	"github.com/gin-gonic/gin"
)

// SetupWatchlistRoutes adds watchlist routes (buyer only, auth required)
func SetupWatchlistRoutes(rg *gin.RouterGroup) {
	watchlist := rg.Group("/watchlist")
	watchlist.Use(middleware.AuthMiddleware(), middleware.RequireRole("BUYER"))
	{
		watchlist.GET("", watchlist_controller.GetWatchlist)
		watchlist.POST("", watchlist_controller.AddToWatchlist)
		watchlist.DELETE("/:productId", watchlist_controller.RemoveFromWatchlist)
	}
}
