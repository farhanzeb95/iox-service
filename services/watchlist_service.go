package services

import (
	"context"
	"errors"
	"iox-service/database"
	"log"
	"time"
)

func GetWatchlist(userEmail string) ([]string, error) {
	userID, err := GetUserIDByEmail(userEmail)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := database.Pool.Query(ctx,
		`SELECT product_id FROM watchlist WHERE user_id = $1 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var pid string
		if err := rows.Scan(&pid); err != nil {
			return nil, err
		}
		ids = append(ids, pid)
	}
	return ids, rows.Err()
}

func AddToWatchlist(userEmail string, productID string) error {
	userID, err := GetUserIDByEmail(userEmail)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = database.Pool.Exec(ctx,
		`INSERT INTO watchlist (user_id, product_id, created_at) VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, product_id) DO NOTHING`,
		userID, productID, time.Now(),
	)
	if err != nil {
		log.Printf("Error adding to watchlist: %v", err)
		return errors.New("failed to add to watchlist")
	}
	return nil
}

func RemoveFromWatchlist(userEmail string, productID string) error {
	userID, err := GetUserIDByEmail(userEmail)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = database.Pool.Exec(ctx, `DELETE FROM watchlist WHERE user_id = $1 AND product_id = $2`, userID, productID)
	if err != nil {
		return err
	}
	return nil
}
