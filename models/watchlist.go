package models

import "time"

type WatchlistItem struct {
	UserID    string    `json:"userId"`
	ProductID string    `json:"productId"`
	CreatedAt time.Time `json:"createdAt"`
}
