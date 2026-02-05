package models

import "time"

type OrderReturn struct {
	ID          string    `json:"id"`
	OrderID     string    `json:"orderId"`
	BuyerID     string    `json:"buyerId"`
	Reason      string    `json:"reason"`
	Status      string    `json:"status"`
	RequestedAt time.Time `json:"requestedAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
