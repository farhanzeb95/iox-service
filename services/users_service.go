package services

import (
	"context"
	"errors"
	"iox-service/database"
	model "iox-service/models"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

// CreateUser inserts a new user into MongoDB
func CreateUser(user *model.User) error {
	if user.FirstName == "" || user.Email == "" {
		return errors.New("first name and email are required")
	}

	hashedPassword, err := hashPassword(user.Password)
	if err != nil {
		return err
	}

	user.Password = hashedPassword
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	collection := database.Client.Database("iox").Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = collection.InsertOne(ctx, user)
	return err
}

// GetUsers fetches all users from MongoDB
func GetUsers() ([]model.User, error) {
	collection := database.Client.Database("iox").Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []model.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}

	return users, nil
}

func GetUserById(id string) (*model.User, error) {
	collection := database.Client.Database("iox").Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objectId, err := primitive.ObjectIDFromHex((id))

	if err != nil {
		return nil, errors.New("invalid user id")
	}

	var user model.User
	err = collection.FindOne(ctx, bson.M{"_id": objectId}).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func LoginUser(email, password string) (string, error) {
	collection := database.Client.Database("iox").Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user model.User

	err := collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		return "", errors.New("user does not eist")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))

	if err != nil {
		return "", errors.New("invalid login credentials")
	}

	token, err := generateToken(user.Email, int(user.Type))
	if err != nil {
		return "", err
	}

	return token, nil
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func generateToken(email string, userType int) (string, error) {
	var jwtSecret = []byte("super-secret-key")
	claims := jwt.MapClaims{
		"user_id": email,
		"type":    userType,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}
