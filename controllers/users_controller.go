package controllers

import (
	model "iox-service/models"
	services "iox-service/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CreateUser handles POST /users
func CreateUser(c *gin.Context) {
	var user model.User

	// Bind JSON to struct
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Call service layer
	if err := services.CreateUser(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// GetUsers handles GET /users
func GetUsers(c *gin.Context) {
	users, err := services.GetUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

func GetUserById(c *gin.Context) {
	id := c.Param("id")

	user, err := services.GetUserById(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"errors": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

func LoginUser(c *gin.Context) {
	var request struct {
		Email    string
		Password string
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := services.LoginUser(request.Email, request.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, token)

}
