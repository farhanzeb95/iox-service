package middleware

import (
	"net/http"
	"os"
	"strings"

	"iox-service/database"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware validates JWT token and extracts user info
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
		if secret == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "JWT secret is not configured"})
			c.Abort()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Parse and validate token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Extract claims
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			// Store user info in context
			c.Set("user_email", claims["user_id"])
			c.Set("user_type", claims["type"])
			if claims["type"] != "ADMIN" {
				var status string
				if err := database.Pool.QueryRow(c.Request.Context(), `SELECT status FROM users WHERE email = $1`, claims["user_id"]).Scan(&status); err != nil {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "user account could not be verified"})
					c.Abort()
					return
				}
				if status == "SUSPENDED" {
					c.JSON(http.StatusForbidden, gin.H{"error": "user account is temporarily suspended"})
					c.Abort()
					return
				}
			}
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireRole middleware checks if user has required role
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userType, exists := c.Get("user_type")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User type not found"})
			c.Abort()
			return
		}

		userTypeStr, ok := userType.(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user type"})
			c.Abort()
			return
		}

		// Check if user type is in allowed roles
		for _, role := range allowedRoles {
			if userTypeStr == role {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		c.Abort()
	}
}
