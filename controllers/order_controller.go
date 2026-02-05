package controllers

import (
	"net/http"
	"strings"

	"iox-service/models"
	"iox-service/services"

	"github.com/gin-gonic/gin"
)

// PlaceOrderInput is the request body for placing an order
type PlaceOrderInput struct {
	ShippingAddress models.Address `json:"shippingAddress" binding:"required"`
	PaymentMethod   string         `json:"paymentMethod" binding:"required"`
}

// PlaceOrder handles POST /orders (auth, buyer only)
func PlaceOrder(c *gin.Context) {
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

	var input PlaceOrderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "shippingAddress and paymentMethod are required"})
		return
	}

	orders, err := services.CreateOrdersFromCart(emailStr, input.ShippingAddress, input.PaymentMethod)
	if err != nil {
		if err.Error() == "cart is empty" || err.Error() == "invalid payment method" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusCreated, gin.H{"orders": orders, "message": "Orders placed successfully"})
}

// GetMyOrders handles GET /orders - buyer sees their orders, seller sees orders for them
func GetMyOrders(c *gin.Context) {
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
	userType, _ := c.Get("user_type")
	userTypeStr, _ := userType.(string)

	var orders []models.Order
	var err error
	if userTypeStr == "BUYER" {
		orders, err = services.GetOrdersByBuyer(emailStr)
	} else {
		orders, err = services.GetOrdersBySeller(emailStr)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if orders == nil {
		orders = []models.Order{}
	}
	c.JSON(http.StatusOK, gin.H{"orders": orders})
}

// GetOrderByID handles GET /orders/:id - returns order detail if requester is buyer or seller
func GetOrderByID(c *gin.Context) {
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
	userType, _ := c.Get("user_type")
	userTypeStr, _ := userType.(string)
	isBuyer := userTypeStr == "BUYER"

	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order id required"})
		return
	}

	order, err := services.GetOrderByID(orderID, emailStr, isBuyer)
	if err != nil {
		if err.Error() == "order not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else if err.Error() == "forbidden" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": order})
}

// UpdateOrderStatusInput is the request body for PATCH /orders/:id/status
type UpdateOrderStatusInput struct {
	Status string `json:"status" binding:"required"`
}

// UpdateOrderStatus handles PATCH /orders/:id/status - seller: CONFIRMED/SHIPPED/DELIVERED; buyer/seller: CANCELLED when PENDING
func UpdateOrderStatus(c *gin.Context) {
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
	userType, _ := c.Get("user_type")
	userTypeStr, _ := userType.(string)
	isBuyer := userTypeStr == "BUYER"

	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order id required"})
		return
	}
	var input UpdateOrderStatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status is required"})
		return
	}

	order, err := services.UpdateOrderStatus(orderID, emailStr, isBuyer, input.Status)
	if err != nil {
		if err.Error() == "order not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "forbidden" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		if err.Error() == "invalid status" || err.Error() == "order cannot be updated" ||
			err.Error() == "only pending orders can be cancelled" || err.Error() == "only seller can update order status" ||
			strings.Contains(err.Error(), "can only be") || err.Error() == "invalid transition" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": order})
}

// RequestReturnInput is the request body for POST /orders/:id/return
type RequestReturnInput struct {
	Reason string `json:"reason"`
}

// RequestReturn handles POST /orders/:id/return (buyer only, order must be DELIVERED)
func RequestReturn(c *gin.Context) {
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
	userType, _ := c.Get("user_type")
	if userTypeStr, _ := userType.(string); userTypeStr != "BUYER" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Buyers only"})
		return
	}
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order id required"})
		return
	}
	var input RequestReturnInput
	_ = c.ShouldBindJSON(&input)
	r, err := services.CreateReturn(orderID, emailStr, input.Reason)
	if err != nil {
		if err.Error() == "order not found" || err.Error() == "forbidden" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "only delivered orders can be returned" || err.Error() == "return already requested for this order" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"return": r})
}
