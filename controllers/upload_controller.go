package controllers

import (
	"net/http"
	"iox-service/services"

	"github.com/gin-gonic/gin"
)

// UploadProductImages handles POST /api/v1/products/upload
func UploadProductImages(c *gin.Context) {
	// Get user from context (set by auth middleware)
	userEmail, exists := c.Get("user_email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	_ = userEmail // User email available for logging if needed

	// Parse multipart form (max 50MB)
	err := c.Request.ParseMultipartForm(50 << 20) // 50MB
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form: " + err.Error()})
		return
	}

	files := c.Request.MultipartForm.File["images"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No files uploaded. Please use 'images' as the form field name."})
		return
	}

	// Limit to 5 images
	if len(files) > 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Maximum 5 images allowed"})
		return
	}

	// Upload to R2
	urls, err := services.UploadMultipleImages(files, "products")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"urls":  urls,
		"count": len(urls),
	})
}
