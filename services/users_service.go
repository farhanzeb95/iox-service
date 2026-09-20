package services

import (
	"context"
	"encoding/json"
	"errors"
	"iox-service/database"
	model "iox-service/models"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func CreateUser(user *model.User) error {
	if user.FirstName == "" || user.Email == "" {
		return errors.New("first name and email are required")
	}
	if user.Type == model.TypeBusinessSeller && user.BusinessRegistrationURL == "" {
		return errors.New("business registration document is required for business seller")
	}
	if user.Type == model.TypePrivateSeller {
		if user.IdCardFrontURL == "" || user.IdCardBackURL == "" {
			return errors.New("ID card front and back images are required for private seller")
		}
	}

	hashedPassword, err := hashPassword(user.Password)
	if err != nil {
		return err
	}

	user.Password = hashedPassword
	user.ID = uuid.New().String()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	status := model.UserStatusActive
	if user.Type == model.TypePrivateSeller || user.Type == model.TypeBusinessSeller {
		status = model.UserStatusInReview
	}
	user.Status = status

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = database.Pool.Exec(ctx,
		`INSERT INTO users (id, email, first_name, last_name, password, type, contact, address, status, business_registration_url, id_card_front_url, id_card_back_url, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		user.ID, user.Email, user.FirstName, user.LastName, user.Password, string(user.Type),
		user.Contact, mustMarshalAddress(user.Address), status,
		nullIfEmpty(user.BusinessRegistrationURL), nullIfEmpty(user.IdCardFrontURL), nullIfEmpty(user.IdCardBackURL),
		user.CreatedAt, user.UpdatedAt,
	)
	return err
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func GetUsers() ([]model.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := database.Pool.Query(ctx, `SELECT id, email, first_name, last_name, password, type, contact, address, status, business_registration_url, id_card_front_url, id_card_back_url, created_at, updated_at FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		var addr []byte
		var bizReg, idFront, idBack *string
		err := rows.Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.Password, &u.Type, &u.Contact, &addr, &u.Status, &bizReg, &idFront, &idBack, &u.CreatedAt, &u.UpdatedAt)
		if err != nil {
			return nil, err
		}
		_ = unmarshalAddress(addr, &u.Address)
		if bizReg != nil {
			u.BusinessRegistrationURL = *bizReg
		}
		if idFront != nil {
			u.IdCardFrontURL = *idFront
		}
		if idBack != nil {
			u.IdCardBackURL = *idBack
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func GetUserById(id string) (*model.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var u model.User
	var addr []byte
	var bizReg, idFront, idBack *string
	err := database.Pool.QueryRow(ctx,
		`SELECT id, email, first_name, last_name, password, type, contact, address, status, business_registration_url, id_card_front_url, id_card_back_url, created_at, updated_at FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.Password, &u.Type, &u.Contact, &addr, &u.Status, &bizReg, &idFront, &idBack, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	_ = unmarshalAddress(addr, &u.Address)
	if bizReg != nil {
		u.BusinessRegistrationURL = *bizReg
	}
	if idFront != nil {
		u.IdCardFrontURL = *idFront
	}
	if idBack != nil {
		u.IdCardBackURL = *idBack
	}
	return &u, nil
}

func LoginUser(email, password string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user model.User
	err := database.Pool.QueryRow(ctx,
		`SELECT id, email, first_name, last_name, password, type FROM users WHERE email = $1`, email,
	).Scan(&user.ID, &user.Email, &user.FirstName, &user.LastName, &user.Password, &user.Type)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", errors.New("user does not exist")
		}
		return "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", errors.New("invalid login credentials")
	}

	return generateToken(user.Email, user.Type)
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func generateToken(email string, userType model.UserType) (string, error) {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		return "", errors.New("JWT secret is not configured")
	}

	claims := jwt.MapClaims{
		"user_id": email,
		"type":    string(userType),
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func mustMarshalAddress(a model.Address) string {
	b, _ := json.Marshal(a)
	return string(b)
}

func unmarshalAddress(b []byte, a *model.Address) error {
	if len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, a)
}

// GetUserIDByEmail returns user ID for the given email (used by cart, watchlist, order services)
func GetUserIDByEmail(email string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var id string
	err := database.Pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, email).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", errors.New("user not found")
		}
		return "", err
	}
	return id, nil
}

// GetUserByEmail returns the user by email (for /users/me).
func GetUserByEmail(email string) (*model.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var u model.User
	var addr []byte
	var bizReg, idFront, idBack *string
	err := database.Pool.QueryRow(ctx,
		`SELECT id, email, first_name, last_name, password, type, contact, address, status, business_registration_url, id_card_front_url, id_card_back_url, created_at, updated_at FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.Password, &u.Type, &u.Contact, &addr, &u.Status, &bizReg, &idFront, &idBack, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	_ = unmarshalAddress(addr, &u.Address)
	if bizReg != nil {
		u.BusinessRegistrationURL = *bizReg
	}
	if idFront != nil {
		u.IdCardFrontURL = *idFront
	}
	if idBack != nil {
		u.IdCardBackURL = *idBack
	}
	return &u, nil
}

// UpdateUser updates first name, last name, contact, address. Does not change email, type, or password.
func UpdateUser(id string, firstName, lastName, contact string, address *model.Address) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var addrBytes string
	if address != nil {
		addrBytes = mustMarshalAddress(*address)
	} else {
		addrBytes = mustMarshalAddress(model.Address{})
	}
	_, err := database.Pool.Exec(ctx,
		`UPDATE users SET first_name = $1, last_name = $2, contact = $3, address = $4, updated_at = $5 WHERE id = $6`,
		firstName, lastName, contact, addrBytes, time.Now(), id,
	)
	return err
}
