package controllers

import (
	"net/http"
	"strings"

	"iox-service/services"

	"github.com/gin-gonic/gin"
)

type SubmitSellerStoreFeeInput struct {
	PaymentMethod    string `json:"paymentMethod" binding:"required"`
	PaymentReference string `json:"paymentReference" binding:"required"`
}

// GetMySellerStoreFee returns the seller's required fee and current submission.
func GetMySellerStoreFee(c *gin.Context) {
	email, ok := c.Get("user_email")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
		return
	}
	fee, amount, err := services.GetSellerStoreFee(email.(string))
	if err != nil {
		if strings.Contains(err.Error(), "only sellers") {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"amount": amount, "fee": fee})
}

// SubmitMySellerStoreFee creates or resubmits a seller fee for manual verification.
func SubmitMySellerStoreFee(c *gin.Context) {
	email, ok := c.Get("user_email")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
		return
	}
	var input SubmitSellerStoreFeeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "paymentMethod and paymentReference are required"})
		return
	}
	fee, err := services.SubmitSellerStoreFee(email.(string), input.PaymentMethod, input.PaymentReference)
	if err != nil {
		if strings.Contains(err.Error(), "only sellers") || strings.Contains(err.Error(), "already paid") {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"fee": fee, "message": "Store fee submitted for review"})
}

type ReviewSellerStoreFeeInput struct {
	Status string `json:"status" binding:"required"`
	Note   string `json:"note"`
}

// ListSellerStoreFees returns seller fee submissions for admin review.
func ListSellerStoreFees(c *gin.Context) {
	fees, err := services.ListSellerStoreFees()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"fees": fees})
}

// ReviewSellerStoreFee is an admin-only manual payment verification action.
func ReviewSellerStoreFee(c *gin.Context) {
	email, ok := c.Get("user_email")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
		return
	}
	var input ReviewSellerStoreFeeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status is required"})
		return
	}
	fee, err := services.ReviewSellerStoreFee(c.Param("id"), email.(string), input.Status, input.Note)
	if err != nil {
		if err.Error() == "seller store fee not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"fee": fee})
}
