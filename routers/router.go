package router

import (
	"github.com/gin-gonic/gin"
)

// CORS middleware
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func Setup() *gin.Engine {
	r := gin.Default()

	// Add CORS middleware
	r.Use(corsMiddleware())

	api := r.Group("/api/v1")

	SetupUserRoutes(api)
	SetupProductRoutes(api)
	SetupWatchlistRoutes(api)
	SetupCartRoutes(api)
	SetupOrderRoutes(api)
	SetupReturnsRoutes(api)

	return r
}
