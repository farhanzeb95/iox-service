package controllers

import (
	"net/http"
	"iox-service/models"
	"iox-service/services"

	"github.com/gin-gonic/gin"
)

// GetMyReturns handles GET /returns - buyer sees their return requests, seller sees return requests for their orders
func GetMyReturns(c *gin.Context) {
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

	var list []models.OrderReturn
	var err error
	if userTypeStr == "BUYER" {
		list, err = services.GetReturnsByBuyer(emailStr)
	} else {
		list, err = services.GetReturnsBySeller(emailStr)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if list == nil {
		list = []models.OrderReturn{}
	}
	c.JSON(http.StatusOK, gin.H{"returns": list})
}

// UpdateReturnStatusInput is the request body for PATCH /returns/:id/status
type UpdateReturnStatusInput struct {
	Status string `json:"status" binding:"required"`
}

// UpdateReturnStatus handles PATCH /returns/:id/status (seller: APPROVED or REJECTED)
func UpdateReturnStatus(c *gin.Context) {
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
	returnID := c.Param("id")
	if returnID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "return id required"})
		return
	}
	var input UpdateReturnStatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status is required"})
		return
	}
	r, err := services.UpdateReturnStatus(returnID, emailStr, input.Status)
	if err != nil {
		if err.Error() == "return not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "forbidden" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		if err.Error() == "invalid status" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"return": r})
}
