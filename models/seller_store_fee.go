package models

import "time"

const (
	SellerFeePaymentJazzCash     = "JAZZCASH"
	SellerFeePaymentEasyPaisa    = "EASYPAISA"
	SellerFeePaymentRaast        = "RAAST"
	SellerFeePaymentBankTransfer = "BANK_TRANSFER"

	SellerFeePending  = "PENDING"
	SellerFeePaid     = "PAID"
	SellerFeeRejected = "REJECTED"
)

type SellerStoreFee struct {
	ID               string    `json:"id"`
	SellerID         string    `json:"sellerId"`
	Amount           float64   `json:"amount"`
	PaymentMethod    string    `json:"paymentMethod"`
	PaymentReference string    `json:"paymentReference"`
	Status           string    `json:"status"`
	ReviewNote       string    `json:"reviewNote,omitempty"`
	ReviewedBy       string    `json:"reviewedBy,omitempty"`
	SubmittedAt      time.Time `json:"submittedAt"`
	ReviewedAt       time.Time `json:"reviewedAt,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}
