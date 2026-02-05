package controllers

import (
	model "iox-service/models"
	services "iox-service/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetProductReviews handles GET /products/:id/reviews
func GetProductReviews(c *gin.Context) {
	productID := c.Param("id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product id required"})
		return
	}
	reviews, err := services.GetReviewsByProductID(productID)
	if err != nil {
		if err.Error() == "invalid product id" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	if reviews == nil {
		reviews = []model.Review{}
	}
	c.JSON(http.StatusOK, gin.H{"reviews": reviews})
}

// CanReviewProduct handles GET /products/:id/can-review (auth, buyer) - returns whether the buyer can review (has received this product)
func CanReviewProduct(c *gin.Context) {
	productID := c.Param("id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product id required"})
		return
	}
	userEmail := ""
	if email, exists := c.Get("user_email"); exists {
		if emailStr, ok := email.(string); ok {
			userEmail = emailStr
		}
	}
	canReview, err := services.CanBuyerReviewProduct(userEmail, productID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"canReview": canReview})
}

// CreateReviewInput is the request body for creating a review (author comes from user collection)
type CreateReviewInput struct {
	Rating  int    `json:"rating" binding:"required"`
	Comment string `json:"comment"`
}

// CreateReview handles POST /products/:id/reviews (buyer only, auth required)
func CreateReview(c *gin.Context) {
	productID := c.Param("id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product id required"})
		return
	}
	var input CreateReviewInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userEmail := ""
	if email, exists := c.Get("user_email"); exists {
		if emailStr, ok := email.(string); ok {
			userEmail = emailStr
		}
	}
	review, err := services.CreateReview(productID, userEmail, input.Rating, input.Comment)
	if err != nil {
		if err.Error() == "invalid product id" || err.Error() == "rating must be between 1 and 5" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else if err.Error() == "product not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else if err.Error() == "you can only review products you have received (delivered orders)" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusCreated, review)
}
