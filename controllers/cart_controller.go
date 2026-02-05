package controllers

import (
	"net/http"
	"iox-service/services"

	"github.com/gin-gonic/gin"
)

// GetCart handles GET /cart (auth required)
func GetCart(c *gin.Context) {
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

	cart, err := services.GetCart(emailStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if cart == nil {
		cart = &services.CartResponse{Items: []services.CartItemResponse{}}
	}
	c.JSON(http.StatusOK, cart)
}

// AddToCartInput is the request body for adding to cart
type AddToCartInput struct {
	ProductID string `json:"productId" binding:"required"`
	Quantity  int    `json:"quantity"`
}

// AddToCart handles POST /cart (auth required)
func AddToCart(c *gin.Context) {
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

	var input AddToCartInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "productId is required"})
		return
	}
	if input.Quantity < 1 {
		input.Quantity = 1
	}

	if err := services.AddToCart(emailStr, input.ProductID, input.Quantity); err != nil {
		if err.Error() == "invalid product id" || err.Error() == "product not found" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else if err.Error() == "insufficient stock" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Added to cart"})
}

// UpdateCartInput is the request body for updating quantity
type UpdateCartInput struct {
	ProductID string `json:"productId" binding:"required"`
	Quantity  int    `json:"quantity"`
}

// UpdateQuantity handles PATCH /cart (auth required)
func UpdateQuantity(c *gin.Context) {
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

	var input UpdateCartInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "productId and quantity required"})
		return
	}

	if err := services.UpdateQuantity(emailStr, input.ProductID, input.Quantity); err != nil {
		if err.Error() == "invalid product id" || err.Error() == "product not found" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Cart updated"})
}

// RemoveFromCart handles DELETE /cart/:productId (auth required)
func RemoveFromCart(c *gin.Context) {
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

	if err := services.RemoveFromCart(emailStr, productID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Removed from cart"})
}

// ClearCart handles DELETE /cart (auth required)
func ClearCart(c *gin.Context) {
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

	if err := services.ClearCart(emailStr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Cart cleared"})
}
