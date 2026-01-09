package models

import (
	"time"
)

type UserType int

const (
	Admin  UserType = 1
	Seller UserType = 2
	Buyer  UserType = 3
)

type User struct {
	FirstName string
	LastName  string
	Email     string
	Password  string
	Type      UserType
	Contact   string
	City      string
	State     string
	Zip       string
	Country   string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time
}
