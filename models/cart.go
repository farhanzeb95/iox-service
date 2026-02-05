package models

import "time"

type CartItem struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
}

type Cart struct {
	ID        string    `json:"id,omitempty"`
	UserID    string    `json:"userId"`
	Items     []CartItem `json:"items"`
	UpdatedAt time.Time `json:"updatedAt"`
}
