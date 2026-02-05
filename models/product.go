package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

type Product struct {
	ID          string    `json:"id,omitempty"`
	Title       string    `json:"title"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Quantity    int       `json:"quantity"`
	IsAvailable bool      `json:"isAvailable"`
	Images      []string  `json:"images"`
	Category    string    `json:"category"`
	SellerId    string    `json:"sellerId,omitempty"`
	SellerName  string    `json:"seller"`
	SKU         string    `json:"sku,omitempty"`
	Condition   string    `json:"condition,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	Views       int       `json:"views,omitempty"`
	Rating      float64   `json:"rating,omitempty"`
	ReviewCount int       `json:"reviewCount,omitempty"`
	CreatedAt   time.Time `json:"createdAt,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt,omitempty"`
}

// StringArray for PostgreSQL text[] / JSONB array scan
type StringArray []string

func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, s)
}

func (s StringArray) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "[]", nil
	}
	return json.Marshal(s)
}
