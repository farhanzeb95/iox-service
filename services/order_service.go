package services

import (
	"context"
	"encoding/json"
	"errors"
	"iox-service/database"
	model "iox-service/models"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type PlaceOrderInput struct {
	ShippingAddress model.Address `json:"shippingAddress" binding:"required"`
	PaymentMethod   string        `json:"paymentMethod" binding:"required"`
}

func CreateOrdersFromCart(userEmail string, shippingAddress model.Address, paymentMethod string) ([]model.Order, error) {
	validPayment := map[string]bool{
		model.PaymentCOD: true, model.PaymentBankTransfer: true,
		model.PaymentJazzCash: true, model.PaymentEasyPaisa: true,
		model.PaymentRaast: true, model.PaymentCard: true,
	}
	if !validPayment[paymentMethod] {
		return nil, errors.New("invalid payment method")
	}

	userID, err := GetUserIDByEmail(userEmail)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	tx, err := database.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var buyerName string
	err = tx.QueryRow(ctx, `SELECT first_name || ' ' || last_name FROM users WHERE id = $1`, userID).Scan(&buyerName)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	cart, err := GetCart(userEmail)
	if err != nil {
		return nil, err
	}
	if cart == nil || len(cart.Items) == 0 {
		return nil, errors.New("cart is empty")
	}

	type sellerGroup struct {
		sellerID   string
		sellerName string
		lines      []model.OrderLine
	}
	groups := make(map[string]*sellerGroup)

	for _, it := range cart.Items {
		var price float64
		var stock int
		var title, sellerID, sellerName string
		var imagesJSON []byte
		err := tx.QueryRow(ctx,
			`SELECT title, price, quantity, seller_id, seller_name, images FROM products WHERE id = $1`, it.ProductID,
		).Scan(&title, &price, &stock, &sellerID, &sellerName, &imagesJSON)
		if err != nil {
			if err == pgx.ErrNoRows {
				return nil, errors.New("product not found: " + it.ProductID)
			}
			return nil, err
		}
		if stock < it.Quantity {
			return nil, errors.New("insufficient stock for: " + title)
		}
		img := ""
		var imgs []string
		_ = json.Unmarshal(imagesJSON, &imgs)
		if len(imgs) > 0 {
			img = imgs[0]
		}
		line := model.OrderLine{
			ProductID: it.ProductID,
			Title:     title,
			Price:     price,
			Quantity:  it.Quantity,
			ImageURL:  img,
		}
		if g, ok := groups[sellerID]; ok {
			g.lines = append(g.lines, line)
		} else {
			groups[sellerID] = &sellerGroup{sellerID: sellerID, sellerName: sellerName, lines: []model.OrderLine{line}}
		}
	}

	addrJSON, _ := json.Marshal(shippingAddress)
	now := time.Now()
	var created []model.Order

	for _, g := range groups {
		subTotal := 0.0
		for _, l := range g.lines {
			subTotal += l.Price * float64(l.Quantity)
		}

		orderID := uuid.New().String()
		_, err := tx.Exec(ctx,
			`INSERT INTO orders (id, buyer_id, seller_id, buyer_email, buyer_name, seller_name, sub_total, status, payment_method, payment_status, shipping_address, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
			orderID, userID, g.sellerID, userEmail, buyerName, g.sellerName, subTotal, "PENDING", paymentMethod, "PENDING", string(addrJSON), now, now,
		)
		if err != nil {
			log.Printf("Error creating order: %v", err)
			return nil, errors.New("failed to create order")
		}

		for _, l := range g.lines {
			_, err = tx.Exec(ctx,
				`INSERT INTO order_lines (id, order_id, product_id, title, price, quantity, image_url) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
				uuid.New(), orderID, l.ProductID, l.Title, l.Price, l.Quantity, l.ImageURL,
			)
			if err != nil {
				return nil, err
			}
			updateResult, err := tx.Exec(ctx, `UPDATE products SET quantity = quantity - $1, updated_at = $2 WHERE id = $3 AND quantity >= $1`, l.Quantity, now, l.ProductID)
			if err != nil {
				return nil, err
			}
			if updateResult.RowsAffected() == 0 {
				return nil, errors.New("insufficient stock for: " + l.Title)
			}
		}

		created = append(created, model.Order{
			ID:              orderID,
			BuyerID:         userID,
			BuyerEmail:      userEmail,
			BuyerName:       buyerName,
			SellerID:        g.sellerID,
			SellerName:      g.sellerName,
			Items:           g.lines,
			SubTotal:        subTotal,
			Status:          "PENDING",
			PaymentMethod:   paymentMethod,
			PaymentStatus:   "PENDING",
			ShippingAddress: shippingAddress,
			CreatedAt:       now,
			UpdatedAt:       now,
		})
	}

	var cartID string
	err = tx.QueryRow(ctx, `SELECT id FROM carts WHERE user_id = $1`, userID).Scan(&cartID)
	if err != nil && err != pgx.ErrNoRows {
		return nil, err
	}
	if err == nil {
		if _, err := tx.Exec(ctx, `DELETE FROM cart_items WHERE cart_id = $1`, cartID); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `UPDATE carts SET updated_at = $1 WHERE id = $2`, now, cartID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return created, nil
}

func GetOrdersByBuyer(userEmail string) ([]model.Order, error) {
	userID, err := GetUserIDByEmail(userEmail)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := database.Pool.Query(ctx,
		`SELECT id, buyer_id, seller_id, buyer_email, buyer_name, seller_name, sub_total, status, payment_method, payment_status, shipping_address, tracking_number, carrier, created_at, updated_at FROM orders WHERE buyer_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []model.Order
	for rows.Next() {
		var o model.Order
		var addrJSON []byte
		var trackingNumber, carrier *string
		if err := rows.Scan(&o.ID, &o.BuyerID, &o.SellerID, &o.BuyerEmail, &o.BuyerName, &o.SellerName, &o.SubTotal, &o.Status, &o.PaymentMethod, &o.PaymentStatus, &addrJSON, &trackingNumber, &carrier, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(addrJSON, &o.ShippingAddress)
		if trackingNumber != nil {
			o.TrackingNumber = *trackingNumber
		}
		if carrier != nil {
			o.Carrier = *carrier
		}
		o.Items = nil
		lineRows, _ := database.Pool.Query(ctx, `SELECT product_id, title, price, quantity, image_url FROM order_lines WHERE order_id = $1`, o.ID)
		for lineRows.Next() {
			var l model.OrderLine
			_ = lineRows.Scan(&l.ProductID, &l.Title, &l.Price, &l.Quantity, &l.ImageURL)
			o.Items = append(o.Items, l)
		}
		lineRows.Close()
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func GetOrdersBySeller(userEmail string) ([]model.Order, error) {
	sellerID, err := GetUserIDByEmail(userEmail)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := database.Pool.Query(ctx,
		`SELECT id, buyer_id, seller_id, buyer_email, buyer_name, seller_name, sub_total, status, payment_method, payment_status, shipping_address, tracking_number, carrier, created_at, updated_at FROM orders WHERE seller_id = $1 ORDER BY created_at DESC`,
		sellerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []model.Order
	for rows.Next() {
		var o model.Order
		var addrJSON []byte
		var trackingNumber, carrier *string
		if err := rows.Scan(&o.ID, &o.BuyerID, &o.SellerID, &o.BuyerEmail, &o.BuyerName, &o.SellerName, &o.SubTotal, &o.Status, &o.PaymentMethod, &o.PaymentStatus, &addrJSON, &trackingNumber, &carrier, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(addrJSON, &o.ShippingAddress)
		if trackingNumber != nil {
			o.TrackingNumber = *trackingNumber
		}
		if carrier != nil {
			o.Carrier = *carrier
		}
		o.Items = nil
		lineRows, _ := database.Pool.Query(ctx, `SELECT product_id, title, price, quantity, image_url FROM order_lines WHERE order_id = $1`, o.ID)
		for lineRows.Next() {
			var l model.OrderLine
			_ = lineRows.Scan(&l.ProductID, &l.Title, &l.Price, &l.Quantity, &l.ImageURL)
			o.Items = append(o.Items, l)
		}
		lineRows.Close()
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

// GetOrderByID returns a single order by ID if the requester is the buyer or the seller.
func GetOrderByID(orderID string, userEmail string, isBuyer bool) (*model.Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var o model.Order
	var addrJSON []byte
	var trackingNumber, carrier *string
	err := database.Pool.QueryRow(ctx,
		`SELECT id, buyer_id, seller_id, buyer_email, buyer_name, seller_name, sub_total, status, payment_method, payment_status, shipping_address, tracking_number, carrier, created_at, updated_at FROM orders WHERE id = $1`,
		orderID,
	).Scan(&o.ID, &o.BuyerID, &o.SellerID, &o.BuyerEmail, &o.BuyerName, &o.SellerName, &o.SubTotal, &o.Status, &o.PaymentMethod, &o.PaymentStatus, &addrJSON, &trackingNumber, &carrier, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New("order not found")
		}
		return nil, err
	}
	_ = json.Unmarshal(addrJSON, &o.ShippingAddress)
	if trackingNumber != nil {
		o.TrackingNumber = *trackingNumber
	}
	if carrier != nil {
		o.Carrier = *carrier
	}

	// Load order lines
	lineRows, err := database.Pool.Query(ctx, `SELECT product_id, title, price, quantity, image_url FROM order_lines WHERE order_id = $1`, o.ID)
	if err != nil {
		return nil, err
	}
	defer lineRows.Close()
	for lineRows.Next() {
		var l model.OrderLine
		_ = lineRows.Scan(&l.ProductID, &l.Title, &l.Price, &l.Quantity, &l.ImageURL)
		o.Items = append(o.Items, l)
	}

	if isBuyer {
		if o.BuyerEmail != userEmail {
			return nil, errors.New("forbidden")
		}
	} else {
		sellerID, err := GetUserIDByEmail(userEmail)
		if err != nil || sellerID != o.SellerID {
			return nil, errors.New("forbidden")
		}
	}
	return &o, nil
}

// Valid order statuses and allowed transitions:
// - PENDING -> CONFIRMED (seller), CANCELLED (buyer or seller)
// - CONFIRMED -> SHIPPED (seller)
// - SHIPPED -> DELIVERED (seller)
// - DELIVERED, CANCELLED -> no further changes
const (
	StatusPending   = "PENDING"
	StatusConfirmed = "CONFIRMED"
	StatusShipped   = "SHIPPED"
	StatusDelivered = "DELIVERED"
	StatusCancelled = "CANCELLED"
)

func UpdateOrderStatus(orderID string, userEmail string, isBuyer bool, newStatus string, trackingNumber, carrier string) (*model.Order, error) {
	order, err := GetOrderByID(orderID, userEmail, isBuyer)
	if err != nil {
		return nil, err
	}
	current := order.Status
	newStatus = strings.TrimSpace(strings.ToUpper(newStatus))
	validStatus := map[string]bool{StatusPending: true, StatusConfirmed: true, StatusShipped: true, StatusDelivered: true, StatusCancelled: true}
	if !validStatus[newStatus] {
		return nil, errors.New("invalid status")
	}
	if current == StatusDelivered || current == StatusCancelled {
		return nil, errors.New("order cannot be updated")
	}
	if newStatus == StatusCancelled {
		if current != StatusPending {
			return nil, errors.New("only pending orders can be cancelled")
		}
	} else if isBuyer {
		return nil, errors.New("only seller can update order status")
	} else {
		switch current {
		case StatusPending:
			if newStatus != StatusConfirmed {
				return nil, errors.New("pending orders can only be confirmed or cancelled")
			}
		case StatusConfirmed:
			if newStatus != StatusShipped {
				return nil, errors.New("confirmed orders can only be marked shipped")
			}
		case StatusShipped:
			if newStatus != StatusDelivered {
				return nil, errors.New("shipped orders can only be marked delivered")
			}
		default:
			return nil, errors.New("invalid transition")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	now := time.Now()
	if newStatus == StatusConfirmed || newStatus == StatusShipped {
		_, err = database.Pool.Exec(ctx,
			`UPDATE orders SET status = $1, updated_at = $2, tracking_number = NULLIF(TRIM($3), ''), carrier = NULLIF(TRIM($4), '') WHERE id = $5`,
			newStatus, now, strings.TrimSpace(trackingNumber), strings.TrimSpace(carrier), orderID,
		)
	} else {
		_, err = database.Pool.Exec(ctx, `UPDATE orders SET status = $1, updated_at = $2 WHERE id = $3`, newStatus, now, orderID)
	}
	if err != nil {
		return nil, err
	}
	order.Status = newStatus
	order.UpdatedAt = now
	if newStatus == StatusConfirmed || newStatus == StatusShipped {
		if strings.TrimSpace(trackingNumber) != "" {
			order.TrackingNumber = strings.TrimSpace(trackingNumber)
		}
		if strings.TrimSpace(carrier) != "" {
			order.Carrier = strings.TrimSpace(carrier)
		}
	}
	return order, nil
}
