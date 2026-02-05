package services

import (
	"context"
	"errors"
	"iox-service/database"
	model "iox-service/models"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func CreateReturn(orderID string, userEmail string, reason string) (*model.OrderReturn, error) {
	order, err := GetOrderByID(orderID, userEmail, true)
	if err != nil {
		return nil, err
	}
	if order.Status != StatusDelivered {
		return nil, errors.New("only delivered orders can be returned")
	}
	buyerID := order.BuyerID
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var existing string
	err = database.Pool.QueryRow(ctx, `SELECT id FROM order_returns WHERE order_id = $1`, orderID).Scan(&existing)
	if err == nil {
		return nil, errors.New("return already requested for this order")
	}
	if err != nil && err != pgx.ErrNoRows {
		return nil, err
	}
	id := uuid.New().String()
	now := time.Now()
	_, err = database.Pool.Exec(ctx,
		`INSERT INTO order_returns (id, order_id, buyer_id, reason, status, requested_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		id, orderID, buyerID, reason, "PENDING", now, now,
	)
	if err != nil {
		return nil, err
	}
	return &model.OrderReturn{
		ID:          id,
		OrderID:     orderID,
		BuyerID:     buyerID,
		Reason:      reason,
		Status:      "PENDING",
		RequestedAt: now,
		UpdatedAt:   now,
	}, nil
}

func GetReturnsByBuyer(userEmail string) ([]model.OrderReturn, error) {
	userID, err := GetUserIDByEmail(userEmail)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rows, err := database.Pool.Query(ctx,
		`SELECT id, order_id, buyer_id, reason, status, requested_at, updated_at FROM order_returns WHERE buyer_id = $1 ORDER BY requested_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.OrderReturn
	for rows.Next() {
		var r model.OrderReturn
		if err := rows.Scan(&r.ID, &r.OrderID, &r.BuyerID, &r.Reason, &r.Status, &r.RequestedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func GetReturnsBySeller(userEmail string) ([]model.OrderReturn, error) {
	sellerID, err := GetUserIDByEmail(userEmail)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rows, err := database.Pool.Query(ctx,
		`SELECT r.id, r.order_id, r.buyer_id, r.reason, r.status, r.requested_at, r.updated_at
		 FROM order_returns r JOIN orders o ON o.id = r.order_id WHERE o.seller_id = $1 ORDER BY r.requested_at DESC`,
		sellerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.OrderReturn
	for rows.Next() {
		var r model.OrderReturn
		if err := rows.Scan(&r.ID, &r.OrderID, &r.BuyerID, &r.Reason, &r.Status, &r.RequestedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func UpdateReturnStatus(returnID string, userEmail string, newStatus string) (*model.OrderReturn, error) {
	if newStatus != "APPROVED" && newStatus != "REJECTED" {
		return nil, errors.New("invalid status")
	}
	sellerID, err := GetUserIDByEmail(userEmail)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var orderID string
	err = database.Pool.QueryRow(ctx, `SELECT order_id FROM order_returns WHERE id = $1`, returnID).Scan(&orderID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New("return not found")
		}
		return nil, err
	}
	var sid string
	err = database.Pool.QueryRow(ctx, `SELECT seller_id FROM orders WHERE id = $1`, orderID).Scan(&sid)
	if err != nil || sid != sellerID {
		return nil, errors.New("forbidden")
	}
	now := time.Now()
	_, err = database.Pool.Exec(ctx, `UPDATE order_returns SET status = $1, updated_at = $2 WHERE id = $3`, newStatus, now, returnID)
	if err != nil {
		return nil, err
	}
	var r model.OrderReturn
	err = database.Pool.QueryRow(ctx,
		`SELECT id, order_id, buyer_id, reason, status, requested_at, updated_at FROM order_returns WHERE id = $1`,
		returnID,
	).Scan(&r.ID, &r.OrderID, &r.BuyerID, &r.Reason, &r.Status, &r.RequestedAt, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func HasReturnForOrder(orderID string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var id string
	err := database.Pool.QueryRow(ctx, `SELECT id FROM order_returns WHERE order_id = $1`, orderID).Scan(&id)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
