package router

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func configuredOrigins() (map[string]bool, bool) {
	value := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if value == "" {
		return nil, true
	}
	if value == "*" {
		return nil, true
	}

	origins := make(map[string]bool)
	for _, origin := range strings.Split(value, ",") {
		origin = strings.TrimRight(strings.TrimSpace(origin), "/")
		if origin != "" {
			origins[origin] = true
		}
	}
	return origins, false
}

func corsMiddleware() gin.HandlerFunc {
	allowedOrigins, allowAllOrigins := configuredOrigins()
	return func(c *gin.Context) {
		origin := strings.TrimRight(strings.TrimSpace(c.GetHeader("Origin")), "/")
		if origin != "" {
			if !allowAllOrigins && !allowedOrigins[origin] {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "origin is not allowed"})
				return
			}
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Vary", "Origin")
		}
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, Origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func Setup() *gin.Engine {
	r := gin.Default()
	r.Use(corsMiddleware())

	health := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
	r.GET("/health", health)

	api := r.Group("/api/v1")
	api.GET("/health", health)

	SetupUserRoutes(api)
	SetupProductRoutes(api)
	SetupWatchlistRoutes(api)
	SetupCartRoutes(api)
	SetupOrderRoutes(api)
	SetupReturnsRoutes(api)
	SetupSellerStoreFeeRoutes(api)

	return r
}
