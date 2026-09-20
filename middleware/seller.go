package middleware

import (
	"net/http"

	"iox-service/database"

	"github.com/gin-gonic/gin"
)

// RequireActiveSeller blocks seller management operations until an admin
// approves the account. Buyers continue through unchanged on shared routes.
func RequireActiveSeller() gin.HandlerFunc {
	return func(c *gin.Context) {
		userType, _ := c.Get("user_type")
		if userType == "BUYER" || userType == "ADMIN" {
			c.Next()
			return
		}

		email, ok := c.Get("user_email")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user identity is missing"})
			return
		}
		var status string
		if err := database.Pool.QueryRow(c.Request.Context(), `SELECT status FROM users WHERE email = $1`, email).Scan(&status); err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "seller account could not be verified"})
			return
		}
		if status != "ACTIVE" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "seller account must be approved before accessing seller tools"})
			return
		}
		c.Next()
	}
}
