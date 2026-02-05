package services

import (
	"context"
	"errors"
	"iox-service/database"
	model "iox-service/models"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

const minRating, maxRating = 1, 5

func GetReviewsByProductID(productID string) ([]model.Review, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := database.Pool.Query(ctx,
		`SELECT id, product_id, author, rating, comment, created_at FROM reviews WHERE product_id = $1 ORDER BY created_at DESC`, productID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []model.Review
	for rows.Next() {
		var r model.Review
		if err := rows.Scan(&r.ID, &r.ProductID, &r.Author, &r.Rating, &r.Comment, &r.CreatedAt); err != nil {
			return nil, err
		}
		reviews = append(reviews, r)
	}
	return reviews, rows.Err()
}

// CanBuyerReviewProduct returns true if the buyer has at least one DELIVERED order containing this product.
func CanBuyerReviewProduct(buyerEmail string, productID string) (bool, error) {
	if buyerEmail == "" || productID == "" {
		return false, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var exists bool
	err := database.Pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM orders o
			JOIN order_lines ol ON ol.order_id = o.id
			WHERE o.buyer_email = $1 AND o.status = $2 AND ol.product_id = $3
		)`,
		buyerEmail, "DELIVERED", productID,
	).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func CreateReview(productID string, userEmail string, rating int, comment string) (*model.Review, error) {
	if rating < minRating || rating > maxRating {
		return nil, errors.New("rating must be between 1 and 5")
	}
	canReview, err := CanBuyerReviewProduct(userEmail, productID)
	if err != nil {
		return nil, err
	}
	if !canReview {
		return nil, errors.New("you can only review products you have received (delivered orders)")
	}
	author := "Guest"
	if userEmail != "" {
		var firstName, lastName string
		err := database.Pool.QueryRow(context.Background(), `SELECT first_name, last_name FROM users WHERE email = $1`, userEmail).
			Scan(&firstName, &lastName)
		if err == nil {
			author = strings.TrimSpace(firstName + " " + lastName)
			if author == "" {
				author = userEmail
			}
		} else {
			author = userEmail
		}
	}
	author = trimTo(author, 100)
	comment = trimTo(comment, 2000)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var exists bool
	err = database.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM products WHERE id = $1)`, productID).Scan(&exists)
	if err != nil || !exists {
		return nil, errors.New("product not found")
	}

	review := model.Review{
		ID:        uuid.New().String(),
		ProductID: productID,
		Author:    author,
		Rating:    rating,
		Comment:   comment,
		CreatedAt: time.Now(),
	}

	_, err = database.Pool.Exec(ctx,
		`INSERT INTO reviews (id, product_id, author, rating, comment, created_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		review.ID, review.ProductID, review.Author, review.Rating, review.Comment, review.CreatedAt,
	)
	if err != nil {
		log.Printf("Error creating review: %v", err)
		return nil, errors.New("failed to create review")
	}

	if err := updateProductRating(ctx, productID); err != nil {
		log.Printf("Warning: failed to update product rating: %v", err)
	}
	return &review, nil
}

func trimTo(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

func updateProductRating(ctx context.Context, productID string) error {
	var avg float64
	var count int
	err := database.Pool.QueryRow(ctx,
		`SELECT COALESCE(AVG(rating), 0), COUNT(*) FROM reviews WHERE product_id = $1`, productID,
	).Scan(&avg, &count)
	if err != nil {
		return err
	}
	_, err = database.Pool.Exec(ctx,
		`UPDATE products SET rating = $1, review_count = $2, updated_at = $3 WHERE id = $4`,
		avg, count, time.Now(), productID,
	)
	return err
}
