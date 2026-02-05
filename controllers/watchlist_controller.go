package controllers

import (
	services "iox-service/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetWatchlist handles GET /watchlist (auth required)
func GetWatchlist(c *gin.Context) {
	userEmail, exists := c.Get("user_email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
		return
	}
	emailStr, ok := userEmail.(string)
	if !ok || emailStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return
	}

	ids, err := services.GetWatchlist(emailStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ids == nil {
		ids = []string{}
	}
	c.JSON(http.StatusOK, gin.H{"productIds": ids})
}

// AddToWatchlistInput is the request body for adding to watchlist
type AddToWatchlistInput struct {
	ProductID string `json:"productId" binding:"required"`
}

// AddToWatchlist handles POST /watchlist (auth required)
func AddToWatchlist(c *gin.Context) {
	userEmail, exists := c.Get("user_email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
		return
	}
	emailStr, ok := userEmail.(string)
	if !ok || emailStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return
	}

	var input AddToWatchlistInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "productId is required"})
		return
	}

	if err := services.AddToWatchlist(emailStr, input.ProductID); err != nil {
		if err.Error() == "invalid product id" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Added to watchlist"})
}

// RemoveFromWatchlist handles DELETE /watchlist/:productId (auth required)
func RemoveFromWatchlist(c *gin.Context) {
	productID := c.Param("productId")
	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "productId is required"})
		return
	}

	userEmail, exists := c.Get("user_email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
		return
	}
	emailStr, ok := userEmail.(string)
	if !ok || emailStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return
	}

	if err := services.RemoveFromWatchlist(emailStr, productID); err != nil {
		if err.Error() == "invalid product id" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Removed from watchlist"})
}
