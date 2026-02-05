package controllers

import (
	model "iox-service/models"
	services "iox-service/services"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// CreateProduct handles POST /products
func CreateProduct(c *gin.Context) {
	var product model.Product

	// Bind JSON to struct
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user email from context (set by auth middleware)
	userEmail, exists := c.Get("user_email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	emailStr, ok := userEmail.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user email"})
		return
	}

	// Call service layer
	if err := services.CreateProduct(&product, emailStr); err != nil {
		// Check if it's a validation error
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "must be") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// Set name alias for frontend
	if product.Name == "" {
		product.Name = product.Title
	}

	c.JSON(http.StatusCreated, product)
}

// GetProducts handles GET /products
func GetProducts(c *gin.Context) {
	// Parse query parameters
	filters := make(map[string]interface{})

	if category := c.Query("category"); category != "" {
		filters["category"] = category
	}
	if sellerId := c.Query("sellerId"); sellerId != "" {
		filters["sellerId"] = sellerId
	}
	if search := c.Query("search"); search != "" {
		filters["search"] = search
	}
	if isAvailable := c.Query("isAvailable"); isAvailable != "" {
		if isAvail, err := strconv.ParseBool(isAvailable); err == nil {
			filters["isAvailable"] = isAvail
		}
	}
	if condition := c.Query("condition"); condition != "" {
		filters["condition"] = condition
	}
	if minPrice := c.Query("minPrice"); minPrice != "" {
		if mp, err := strconv.ParseFloat(minPrice, 64); err == nil && mp >= 0 {
			filters["minPrice"] = mp
		}
	}
	if maxPrice := c.Query("maxPrice"); maxPrice != "" {
		if mp, err := strconv.ParseFloat(maxPrice, 64); err == nil && mp >= 0 {
			filters["maxPrice"] = mp
		}
	}

	// Parse pagination
	page := 1
	limit := 0 // 0 means no limit
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Parse sorting
	sortBy := c.DefaultQuery("sortBy", "created_at:desc")

	products, total, err := services.GetProducts(filters, page, limit, sortBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"products": products,
		"total":    total,
		"page":     page,
		"limit":    limit,
	})
}

// GetProductById handles GET /products/:id
func GetProductById(c *gin.Context) {
	id := c.Param("id")

	product, err := services.GetProductById(id)
	if err != nil {
		if err.Error() == "product not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, product)
}

// UpdateProduct handles PUT /products/:id
func UpdateProduct(c *gin.Context) {
	id := c.Param("id")

	var product model.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user info from context
	userEmail, exists := c.Get("user_email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userType, exists := c.Get("user_type")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User type not found"})
		return
	}

	emailStr, ok := userEmail.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user email"})
		return
	}

	typeStr, ok := userType.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user type"})
		return
	}

	userTypeEnum := model.UserType(typeStr)

	// Get existing product to check for removed images
	existingProduct, err := services.GetProductById(id)
	if err == nil && existingProduct != nil && len(product.Images) > 0 {
		// Find and delete removed images
		for _, existingImg := range existingProduct.Images {
			found := false
			for _, newImg := range product.Images {
				if existingImg == newImg {
					found = true
					break
				}
			}
			if !found {
				// Image was removed, delete from storage
				if err := services.DeleteImage(existingImg); err != nil {
					// Log but continue update
					c.Header("X-Warning", "Some images could not be deleted from storage")
				}
			}
		}
	}

	if err := services.UpdateProduct(id, &product, emailStr, userTypeEnum); err != nil {
		if err.Error() == "product not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else if err.Error() == "unauthorized: you can only update your own products" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		} else if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "must be") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product updated successfully"})
}

// DeleteProduct handles DELETE /products/:id
func DeleteProduct(c *gin.Context) {
	id := c.Param("id")

	// Get user info from context
	userEmail, exists := c.Get("user_email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userType, exists := c.Get("user_type")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User type not found"})
		return
	}

	emailStr, ok := userEmail.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user email"})
		return
	}

	typeStr, ok := userType.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user type"})
		return
	}

	userTypeEnum := model.UserType(typeStr)

	// Get product first to clean up images
	existingProduct, err := services.GetProductById(id)
	if err == nil && existingProduct != nil && len(existingProduct.Images) > 0 {
		// Clean up images from storage
		for _, imageURL := range existingProduct.Images {
			if err := services.DeleteImage(imageURL); err != nil {
				// Log but continue deletion
				c.Header("X-Warning", "Some images could not be deleted from storage")
			}
		}
	}

	if err := services.DeleteProduct(id, emailStr, userTypeEnum); err != nil {
		if err.Error() == "product not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else if err.Error() == "unauthorized: you can only delete your own products" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product deleted successfully"})
}

// GetProductsBySeller handles GET /products/seller/:sellerId
func GetProductsBySeller(c *gin.Context) {
	sellerId := c.Param("sellerId")

	products, err := services.GetProductsBySeller(sellerId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, products)
}

// GetMyProducts handles GET /products/my-products (for authenticated sellers)
func GetMyProducts(c *gin.Context) {
	// Get user email from context (set by auth middleware)
	userEmail, exists := c.Get("user_email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	emailStr, ok := userEmail.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user email"})
		return
	}

	products, err := services.GetProductsBySellerEmail(emailStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, products)
}
