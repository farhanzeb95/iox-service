package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"iox-service/database"
	model "iox-service/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	PrivateSellerStoreFee  = 500.0
	BusinessSellerStoreFee = 1000.0
)

var validSellerFeePaymentMethods = map[string]bool{
	model.SellerFeePaymentJazzCash:     true,
	model.SellerFeePaymentEasyPaisa:    true,
	model.SellerFeePaymentRaast:        true,
	model.SellerFeePaymentBankTransfer: true,
}

func sellerStoreFeeForType(userType model.UserType) (float64, error) {
	switch userType {
	case model.TypePrivateSeller:
		return PrivateSellerStoreFee, nil
	case model.TypeBusinessSeller:
		return BusinessSellerStoreFee, nil
	default:
		return 0, errors.New("only sellers can submit a store fee")
	}
}

func currentBillingPeriod(now time.Time) (time.Time, time.Time) {
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	return start, start.AddDate(0, 1, 0)
}

const sellerStoreFeeColumns = `id, seller_id, amount, payment_method, payment_reference, status,
	COALESCE(review_note, ''), COALESCE(reviewed_by::text, ''), submitted_at,
	COALESCE(reviewed_at, 'epoch'::timestamptz), billing_period_start,
	billing_period_end, created_at, updated_at`

func GetSellerStoreFee(email string) (*model.SellerStoreFee, float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var userID string
	var userType model.UserType
	if err := database.Pool.QueryRow(ctx, `SELECT id, type FROM users WHERE email = $1`, email).Scan(&userID, &userType); err != nil {
		if err == pgx.ErrNoRows {
			return nil, 0, errors.New("user not found")
		}
		return nil, 0, err
	}
	amount, err := sellerStoreFeeForType(userType)
	if err != nil {
		return nil, 0, err
	}

	periodStart, _ := currentBillingPeriod(time.Now())
	fee, err := scanSellerStoreFee(database.Pool.QueryRow(ctx, `SELECT `+sellerStoreFeeColumns+`
		FROM seller_store_fees WHERE seller_id = $1 AND billing_period_start = $2`, userID, periodStart))
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, 0, err
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, amount, nil
	}
	return fee, amount, nil
}

func SubmitSellerStoreFee(email, paymentMethod, paymentReference string) (*model.SellerStoreFee, error) {
	paymentMethod = strings.ToUpper(strings.TrimSpace(paymentMethod))
	paymentReference = strings.TrimSpace(paymentReference)
	if !validSellerFeePaymentMethods[paymentMethod] {
		return nil, errors.New("invalid seller fee payment method")
	}
	if paymentReference == "" || len(paymentReference) > 120 {
		return nil, errors.New("payment reference is required and must be 120 characters or fewer")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var userID string
	var userType model.UserType
	if err := database.Pool.QueryRow(ctx, `SELECT id, type FROM users WHERE email = $1`, email).Scan(&userID, &userType); err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	amount, err := sellerStoreFeeForType(userType)
	if err != nil {
		return nil, err
	}

	periodStart, periodEnd := currentBillingPeriod(time.Now())
	var existingStatus string
	err = database.Pool.QueryRow(ctx, `SELECT status FROM seller_store_fees WHERE seller_id = $1 AND billing_period_start = $2`, userID, periodStart).Scan(&existingStatus)
	if err == nil && existingStatus == model.SellerFeePaid {
		return nil, errors.New("seller store fee is already paid")
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	now := time.Now()
	var fee *model.SellerStoreFee
	if errors.Is(err, pgx.ErrNoRows) {
		fee, err = scanSellerStoreFee(database.Pool.QueryRow(ctx, `
			INSERT INTO seller_store_fees (id, seller_id, amount, payment_method, payment_reference, status, billing_period_start, billing_period_end, submitted_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9, $9)
			RETURNING `+sellerStoreFeeColumns,
			uuid.New().String(), userID, amount, paymentMethod, paymentReference, model.SellerFeePending, periodStart, periodEnd, now))
	} else {
		fee, err = scanSellerStoreFee(database.Pool.QueryRow(ctx, `
			UPDATE seller_store_fees
			SET amount = $1, payment_method = $2, payment_reference = $3, status = $4,
			    review_note = NULL, reviewed_by = NULL, reviewed_at = NULL, submitted_at = $5, updated_at = $5
			WHERE seller_id = $6 AND billing_period_start = $7
			RETURNING `+sellerStoreFeeColumns,
			amount, paymentMethod, paymentReference, model.SellerFeePending, now, userID, periodStart))
	}
	if err != nil {
		return nil, err
	}
	return fee, nil
}

func ReviewSellerStoreFee(feeID, adminEmail, status, note string) (*model.SellerStoreFee, error) {
	status = strings.ToUpper(strings.TrimSpace(status))
	if status != model.SellerFeePaid && status != model.SellerFeeRejected {
		return nil, errors.New("status must be PAID or REJECTED")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var adminID string
	if err := database.Pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, adminEmail).Scan(&adminID); err != nil {
		return nil, err
	}
	now := time.Now()
	fee, err := scanSellerStoreFee(database.Pool.QueryRow(ctx, `
		UPDATE seller_store_fees
		SET status = $1, review_note = $2, reviewed_by = $3, reviewed_at = $4, updated_at = $4
		WHERE id = $5
		RETURNING `+sellerStoreFeeColumns,
		status, strings.TrimSpace(note), adminID, now, feeID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("seller store fee not found")
	}
	if err != nil {
		return nil, err
	}
	return fee, nil
}

func ListSellerStoreFees() ([]model.SellerStoreFee, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := database.Pool.Query(ctx, `
		SELECT `+sellerStoreFeeColumns+`
		FROM seller_store_fees ORDER BY submitted_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fees := make([]model.SellerStoreFee, 0)
	for rows.Next() {
		fee, scanErr := scanSellerStoreFee(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		fees = append(fees, *fee)
	}
	return fees, rows.Err()
}

type rowScanner interface{ Scan(dest ...any) error }

func scanSellerStoreFee(row rowScanner) (*model.SellerStoreFee, error) {
	var fee model.SellerStoreFee
	return &fee, row.Scan(
		&fee.ID, &fee.SellerID, &fee.Amount, &fee.PaymentMethod, &fee.PaymentReference,
		&fee.Status, &fee.ReviewNote, &fee.ReviewedBy, &fee.SubmittedAt, &fee.ReviewedAt,
		&fee.BillingPeriodStart, &fee.BillingPeriodEnd, &fee.CreatedAt, &fee.UpdatedAt,
	)
}
