package services

import (
	"context"
	"encoding/json"
	"errors"
	"iox-service/database"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type CartItemResponse struct {
	ProductID         string   `json:"productId"`
	Quantity          int      `json:"quantity"`
	Title             string   `json:"title"`
	Price             float64  `json:"price"`
	Images            []string `json:"images,omitempty"`
	SellerId          string   `json:"sellerId,omitempty"`
	SellerName        string   `json:"seller,omitempty"`
	QuantityAvailable int      `json:"quantityAvailable,omitempty"`
}

type CartResponse struct {
	ID        string             `json:"id,omitempty"`
	UserID    string             `json:"userId,omitempty"`
	Items     []CartItemResponse `json:"items"`
	UpdatedAt time.Time          `json:"updatedAt,omitempty"`
}

func GetCart(userEmail string) (*CartResponse, error) {
	userID, err := GetUserIDByEmail(userEmail)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var cartID string
	var updatedAt time.Time
	err = database.Pool.QueryRow(ctx, `SELECT id, updated_at FROM carts WHERE user_id = $1`, userID).Scan(&cartID, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return &CartResponse{Items: []CartItemResponse{}, UpdatedAt: time.Now()}, nil
		}
		return nil, err
	}

	rows, err := database.Pool.Query(ctx,
		`SELECT ci.product_id, ci.quantity, p.title, p.price, p.images, p.seller_id, p.seller_name, p.quantity as qty_avail
		 FROM cart_items ci JOIN products p ON p.id = ci.product_id WHERE ci.cart_id = $1`,
		cartID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []CartItemResponse
	for rows.Next() {
		var it CartItemResponse
		var imagesJSON []byte
		var sellerID string
		var qtyAvail int
		if err := rows.Scan(&it.ProductID, &it.Quantity, &it.Title, &it.Price, &imagesJSON, &sellerID, &it.SellerName, &qtyAvail); err != nil {
			return nil, err
		}
		it.SellerId = sellerID
		it.QuantityAvailable = qtyAvail
		_ = json.Unmarshal(imagesJSON, &it.Images)
		items = append(items, it)
	}

	return &CartResponse{ID: cartID, UserID: userID, Items: items, UpdatedAt: updatedAt}, nil
}

func AddToCart(userEmail string, productID string, quantity int) error {
	if quantity < 1 {
		return errors.New("quantity must be at least 1")
	}
	userID, err := GetUserIDByEmail(userEmail)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var stock int
	err = database.Pool.QueryRow(ctx, `SELECT quantity FROM products WHERE id = $1`, productID).Scan(&stock)
	if err != nil {
		if err == pgx.ErrNoRows {
			return errors.New("product not found")
		}
		return err
	}
	if stock < quantity {
		return errors.New("insufficient stock")
	}

	var cartID string
	err = database.Pool.QueryRow(ctx, `SELECT id FROM carts WHERE user_id = $1`, userID).Scan(&cartID)
	if err != nil {
		if err == pgx.ErrNoRows {
			cartID = uuid.New().String()
			now := time.Now()
			_, err = database.Pool.Exec(ctx, `INSERT INTO carts (id, user_id, updated_at) VALUES ($1, $2, $3)`, cartID, userID, now)
			if err != nil {
				log.Printf("Error creating cart: %v", err)
				return errors.New("failed to add to cart")
			}
		} else {
			return err
		}
	}

	_, err = database.Pool.Exec(ctx,
		`INSERT INTO cart_items (cart_id, product_id, quantity) VALUES ($1, $2, $3)
		 ON CONFLICT (cart_id, product_id) DO UPDATE SET quantity = cart_items.quantity + $3`,
		cartID, productID, quantity,
	)
	if err != nil {
		return err
	}
	_, _ = database.Pool.Exec(ctx, `UPDATE carts SET updated_at = $1 WHERE id = $2`, time.Now(), cartID)

	var newQty int
	_ = database.Pool.QueryRow(ctx, `SELECT quantity FROM cart_items WHERE cart_id = $1 AND product_id = $2`, cartID, productID).Scan(&newQty)
	if newQty > stock {
		_, _ = database.Pool.Exec(ctx, `UPDATE cart_items SET quantity = $1 WHERE cart_id = $2 AND product_id = $3`, stock, cartID, productID)
		return errors.New("insufficient stock")
	}
	return nil
}

func UpdateQuantity(userEmail string, productID string, quantity int) error {
	userID, err := GetUserIDByEmail(userEmail)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var cartID string
	err = database.Pool.QueryRow(ctx, `SELECT id FROM carts WHERE user_id = $1`, userID).Scan(&cartID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil
		}
		return err
	}

	if quantity <= 0 {
		_, err = database.Pool.Exec(ctx, `DELETE FROM cart_items WHERE cart_id = $1 AND product_id = $2`, cartID, productID)
	} else {
		var stock int
		if err := database.Pool.QueryRow(ctx, `SELECT quantity FROM products WHERE id = $1`, productID).Scan(&stock); err != nil {
			if err == pgx.ErrNoRows {
				return errors.New("product not found")
			}
			return err
		}
		if stock < quantity {
			return errors.New("insufficient stock")
		}
		_, err = database.Pool.Exec(ctx,
			`INSERT INTO cart_items (cart_id, product_id, quantity) VALUES ($1, $2, $3)
			 ON CONFLICT (cart_id, product_id) DO UPDATE SET quantity = $3`,
			cartID, productID, quantity,
		)
	}
	if err != nil {
		return err
	}
	_, _ = database.Pool.Exec(ctx, `UPDATE carts SET updated_at = $1 WHERE id = $2`, time.Now(), cartID)
	return nil
}

func RemoveFromCart(userEmail string, productID string) error {
	return UpdateQuantity(userEmail, productID, 0)
}

func ClearCart(userEmail string) error {
	userID, err := GetUserIDByEmail(userEmail)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var cartID string
	err = database.Pool.QueryRow(ctx, `SELECT id FROM carts WHERE user_id = $1`, userID).Scan(&cartID)
	if err == pgx.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = database.Pool.Exec(ctx, `DELETE FROM cart_items WHERE cart_id = $1`, cartID)
	if err != nil {
		return err
	}
	_, _ = database.Pool.Exec(ctx, `UPDATE carts SET updated_at = $1 WHERE id = $2`, time.Now(), cartID)
	return nil
}
