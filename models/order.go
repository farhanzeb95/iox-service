package models

import (
	"time"
)

const (
	PaymentCOD          = "COD"
	PaymentBankTransfer = "BANK_TRANSFER"
	PaymentJazzCash     = "JAZZCASH"
	PaymentEasyPaisa    = "EASYPAISA"
	PaymentRaast        = "RAAST"
	PaymentCard         = "CARD"
)

type OrderLine struct {
	ProductID string  `json:"productId"`
	Title     string  `json:"title"`
	Price     float64 `json:"price"`
	Quantity  int     `json:"quantity"`
	ImageURL  string  `json:"imageUrl,omitempty"`
}

type Order struct {
	ID              string      `json:"id,omitempty"`
	BuyerID         string      `json:"buyerId"`
	BuyerEmail      string      `json:"buyerEmail"`
	BuyerName       string      `json:"buyerName"`
	SellerID        string      `json:"sellerId"`
	SellerName      string      `json:"sellerName"`
	Items           []OrderLine `json:"items"`
	SubTotal        float64     `json:"subTotal"`
	Status          string      `json:"status"`
	PaymentMethod   string      `json:"paymentMethod"`
	PaymentStatus   string      `json:"paymentStatus"`
	ShippingAddress Address     `json:"shippingAddress"`
	TrackingNumber  string      `json:"trackingNumber"`
	Carrier         string      `json:"carrier"`
	CreatedAt       time.Time   `json:"createdAt"`
	UpdatedAt       time.Time   `json:"updatedAt"`
}
