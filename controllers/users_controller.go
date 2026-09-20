package controllers

import (
	model "iox-service/models"
	services "iox-service/services"
	"net/http"
	"os"

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

	user.Password = ""
	c.JSON(http.StatusCreated, user)
}

// GetUsers handles GET /users
func GetUsers(c *gin.Context) {
	users, err := services.GetUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for i := range users {
		users[i].Password = ""
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

	user.Password = ""
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

	c.JSON(http.StatusOK, gin.H{"token": token})
}

// GetCurrentUser handles GET /users/me (auth required). Returns current user without password.
func GetCurrentUser(c *gin.Context) {
	emailVal, ok := c.Get("user_email")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	email, ok := emailVal.(string)
	if !ok || email == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	user, err := services.GetUserByEmail(email)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	user.Password = ""
	c.JSON(http.StatusOK, user)
}

// UpdateCurrentUser handles PATCH /users/:id (auth required). User can only update their own profile.
func UpdateCurrentUser(c *gin.Context) {
	emailVal, ok := c.Get("user_email")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	email, ok := emailVal.(string)
	if !ok || email == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	id := c.Param("id")
	user, err := services.GetUserById(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if user.Email != email {
		c.JSON(http.StatusForbidden, gin.H{"error": "can only update your own profile"})
		return
	}
	var body struct {
		FirstName string         `json:"FirstName"`
		LastName  string         `json:"LastName"`
		Contact   string         `json:"Contact"`
		Address   *model.Address `json:"Address"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	firstName := body.FirstName
	if firstName == "" {
		firstName = user.FirstName
	}
	lastName := body.LastName
	if lastName == "" {
		lastName = user.LastName
	}
	contact := body.Contact
	addr := body.Address
	if addr == nil {
		addr = &user.Address
	}
	if err := services.UpdateUser(id, firstName, lastName, contact, addr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	updated, _ := services.GetUserById(id)
	updated.Password = ""
	c.JSON(http.StatusOK, updated)
}

// UploadSignupDocument handles POST /users/upload-signup-document (no auth).
// Accepts a single file (form key "document") for identity/business docs; returns public URL.
func UploadSignupDocument(c *gin.Context) {
	bucket := os.Getenv("SUPABASE_STORAGE_BUCKET_USERS")
	if bucket == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Document upload not configured (SUPABASE_STORAGE_BUCKET_USERS)"})
		return
	}
	file, header, err := c.Request.FormFile("document")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing or invalid file: use form key 'document'"})
		return
	}
	defer file.Close()
	url, err := services.UploadImageToBucket(bucket, "identity-docs", file, header)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": url})
}
