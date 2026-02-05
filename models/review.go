package models

import "time"

type Review struct {
	ID        string    `json:"id,omitempty"`
	ProductID string    `json:"productId"`
	Author    string    `json:"author"`
	Rating    int       `json:"rating"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"createdAt"`
}
