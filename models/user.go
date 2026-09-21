package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

type UserType string

const (
	TypeAdmin          UserType = "ADMIN"
	TypeBuyer          UserType = "BUYER"
	TypePrivateSeller  UserType = "PRIVATE_SELLER"
	TypeBusinessSeller UserType = "BUSINESS_SELLER"
)

type Address struct {
	City    string `json:"City,omitempty"`
	State   string `json:"State,omitempty"`
	Zip     string `json:"Zip,omitempty"`
	Country string `json:"Country,omitempty"`
}

func (a Address) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *Address) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, a)
}

// UserStatus for seller account review
const (
	UserStatusActive    = "ACTIVE"
	UserStatusInReview  = "IN_REVIEW"
	UserStatusRejected  = "REJECTED"
	UserStatusSuspended = "SUSPENDED"
)

type User struct {
	ID                      string    `json:"id,omitempty"`
	FirstName               string    `json:"FirstName"`
	LastName                string    `json:"LastName"`
	Email                   string    `json:"Email"`
	Password                string    `json:"Password"`
	Type                    UserType  `json:"Type"`
	Contact                 string    `json:"Contact"`
	Address                 Address   `json:"Address"`
	Status                  string    `json:"status"` // ACTIVE, IN_REVIEW, REJECTED, SUSPENDED
	BusinessRegistrationURL string    `json:"businessRegistrationUrl,omitempty"`
	IdCardFrontURL          string    `json:"idCardFrontUrl,omitempty"`
	IdCardBackURL           string    `json:"idCardBackUrl,omitempty"`
	CreatedAt               time.Time `json:"createdAt,omitempty"`
	UpdatedAt               time.Time `json:"updatedAt,omitempty"`
}
