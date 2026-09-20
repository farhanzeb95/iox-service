package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iox-service/database"
	model "iox-service/models"
	"iox-service/utils"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func CreateProduct(product *model.Product, sellerEmail string) error {
	if validationErrs := utils.ValidateProduct(product.Title, product.Description, product.Price, product.Quantity); len(validationErrs) > 0 {
		return utils.FormatValidationErrors(validationErrs)
	}
	if len(product.Images) > 10 {
		return errors.New("maximum 10 images allowed per product")
	}
	if len(product.Tags) > 20 {
		return errors.New("maximum 20 tags allowed per product")
	}
	for _, tag := range product.Tags {
		if len(strings.TrimSpace(tag)) > 50 {
			return errors.New("each tag must be 50 characters or less")
		}
	}
	if product.Condition != "" {
		valid := map[string]bool{"new": true, "used": true, "refurbished": true}
		if !valid[product.Condition] {
			return errors.New("condition must be one of: new, used, refurbished")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var seller model.User
	var feeStatus string
	periodStart := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.Now().Location())
	err := database.Pool.QueryRow(ctx, `
		SELECT u.id, u.first_name, u.last_name, u.type, u.status,
		       COALESCE(f.status, 'NOT_SUBMITTED')
		FROM users u
		LEFT JOIN seller_store_fees f ON f.seller_id = u.id
		WHERE u.email = $1 AND (f.billing_period_start = $2 OR f.id IS NULL)`, sellerEmail, periodStart).
		Scan(&seller.ID, &seller.FirstName, &seller.LastName, &seller.Type, &seller.Status, &feeStatus)
	if err != nil {
		if err == pgx.ErrNoRows {
			return errors.New("seller not found")
		}
		return err
	}
	if seller.Type != model.TypePrivateSeller && seller.Type != model.TypeBusinessSeller {
		return errors.New("only sellers can create products")
	}
	if seller.Status != model.UserStatusActive {
		return errors.New("seller account must be approved before listing products")
	}
	if feeStatus != model.SellerFeePaid {
		return errors.New("seller store fee must be paid before listing products")
	}

	product.ID = uuid.New().String()
	product.SellerId = seller.ID
	product.SellerName = seller.FirstName + " " + seller.LastName
	product.Name = product.Title
	if product.Images == nil {
		product.Images = []string{}
	}
	if product.Tags == nil {
		product.Tags = []string{}
	}
	if product.Condition == "" {
		product.Condition = "new"
	}
	product.Views = 0
	product.Rating = 0
	product.ReviewCount = 0
	product.CreatedAt = time.Now()
	product.UpdatedAt = time.Now()

	imagesJSON, _ := json.Marshal(product.Images)
	tagsJSON, _ := json.Marshal(product.Tags)

	_, err = database.Pool.Exec(ctx,
		`INSERT INTO products (id, seller_id, seller_name, title, name, description, price, quantity, is_available, images, category, sku, condition, tags, views, rating, review_count, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)`,
		product.ID, product.SellerId, product.SellerName, product.Title, product.Name, product.Description, product.Price, product.Quantity, product.IsAvailable,
		string(imagesJSON), product.Category, product.SKU, product.Condition, string(tagsJSON), product.Views, product.Rating, product.ReviewCount, product.CreatedAt, product.UpdatedAt,
	)
	if err != nil {
		log.Printf("Error creating product: %v", err)
		return errors.New("failed to create product")
	}
	return nil
}

func GetProducts(filters map[string]interface{}, page, limit int, sortBy string) ([]model.Product, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var args []interface{}
	where := []string{"1=1"}
	n := 1
	if category, ok := filters["category"].(string); ok && category != "" {
		where = append(where, "category = $"+fmt.Sprintf("%d", n))
		args = append(args, category)
		n++
	}
	if sellerId, ok := filters["sellerId"].(string); ok && sellerId != "" {
		where = append(where, "seller_id = $"+fmt.Sprintf("%d", n))
		args = append(args, sellerId)
		n++
	}
	if search, ok := filters["search"].(string); ok && search != "" {
		where = append(where, "(title ILIKE $"+fmt.Sprintf("%d", n)+" OR description ILIKE $"+fmt.Sprintf("%d", n)+")")
		args = append(args, "%"+search+"%")
		n++
	}
	if isAvailable, ok := filters["isAvailable"].(bool); ok {
		where = append(where, "is_available = $"+fmt.Sprintf("%d", n))
		args = append(args, isAvailable)
		n++
	}
	if condition, ok := filters["condition"].(string); ok && condition != "" {
		where = append(where, "condition = $"+fmt.Sprintf("%d", n))
		args = append(args, condition)
		n++
	}
	if minPrice, ok := filters["minPrice"].(float64); ok && minPrice > 0 {
		where = append(where, "price >= $"+fmt.Sprintf("%d", n))
		args = append(args, minPrice)
		n++
	}
	if maxPrice, ok := filters["maxPrice"].(float64); ok && maxPrice > 0 {
		where = append(where, "price <= $"+fmt.Sprintf("%d", n))
		args = append(args, maxPrice)
		n++
	}

	whereClause := strings.Join(where, " AND ")

	var total int64
	err := database.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM products WHERE "+whereClause, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	if sortBy == "" {
		sortBy = "created_at"
	}
	sortOrder := "DESC"
	if strings.Contains(sortBy, ":") {
		parts := strings.Split(sortBy, ":")
		sortBy = parts[0]
		if len(parts) > 1 && parts[1] == "asc" {
			sortOrder = "ASC"
		}
	}
	validSort := map[string]bool{"created_at": true, "updated_at": true, "price": true, "title": true, "views": true, "rating": true}
	if !validSort[sortBy] {
		sortBy = "created_at"
	}
	orderClause := sortBy + " " + sortOrder

	q := `SELECT id, seller_id, seller_name, title, name, description, price, quantity, is_available, images, category, sku, condition, tags, views, rating, review_count, created_at, updated_at FROM products WHERE ` + whereClause + ` ORDER BY ` + orderClause
	if limit > 0 {
		offset := (page - 1) * limit
		args = append(args, limit, offset)
		q += ` LIMIT $` + fmt.Sprintf("%d", len(args)-1) + ` OFFSET $` + fmt.Sprintf("%d", len(args))
	}

	rows, err := database.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var products []model.Product
	for rows.Next() {
		var p model.Product
		var imagesJSON, tagsJSON []byte
		err := rows.Scan(&p.ID, &p.SellerId, &p.SellerName, &p.Title, &p.Name, &p.Description, &p.Price, &p.Quantity, &p.IsAvailable, &imagesJSON, &p.Category, &p.SKU, &p.Condition, &tagsJSON, &p.Views, &p.Rating, &p.ReviewCount, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, 0, err
		}
		_ = json.Unmarshal(imagesJSON, &p.Images)
		_ = json.Unmarshal(tagsJSON, &p.Tags)
		if p.Name == "" {
			p.Name = p.Title
		}
		products = append(products, p)
	}
	return products, total, rows.Err()
}

func GetProductById(id string) (*model.Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var p model.Product
	var imagesJSON, tagsJSON []byte
	err := database.Pool.QueryRow(ctx,
		`SELECT id, seller_id, seller_name, title, name, description, price, quantity, is_available, images, category, sku, condition, tags, views, rating, review_count, created_at, updated_at FROM products WHERE id = $1`, id,
	).Scan(&p.ID, &p.SellerId, &p.SellerName, &p.Title, &p.Name, &p.Description, &p.Price, &p.Quantity, &p.IsAvailable, &imagesJSON, &p.Category, &p.SKU, &p.Condition, &tagsJSON, &p.Views, &p.Rating, &p.ReviewCount, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New("product not found")
		}
		return nil, err
	}
	_ = json.Unmarshal(imagesJSON, &p.Images)
	_ = json.Unmarshal(tagsJSON, &p.Tags)
	if p.Name == "" {
		p.Name = p.Title
	}
	_, _ = database.Pool.Exec(ctx, `UPDATE products SET views = views + 1, updated_at = $1 WHERE id = $2`, time.Now(), id)
	return &p, nil
}

func UpdateProduct(id string, updates *model.Product, userEmail string, userType model.UserType) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var existing model.Product
	var imagesJSON, tagsJSON []byte
	err := database.Pool.QueryRow(ctx, `SELECT id, seller_id, seller_name, title, name, description, price, quantity, is_available, images, category, sku, condition, tags FROM products WHERE id = $1`, id).
		Scan(&existing.ID, &existing.SellerId, &existing.SellerName, &existing.Title, &existing.Name, &existing.Description, &existing.Price, &existing.Quantity, &existing.IsAvailable, &imagesJSON, &existing.Category, &existing.SKU, &existing.Condition, &tagsJSON)
	if err != nil {
		if err == pgx.ErrNoRows {
			return errors.New("product not found")
		}
		return err
	}
	_ = json.Unmarshal(imagesJSON, &existing.Images)
	_ = json.Unmarshal(tagsJSON, &existing.Tags)

	if userType != model.TypeAdmin {
		userID, err := GetUserIDByEmail(userEmail)
		if err != nil {
			return errors.New("user not found")
		}
		if existing.SellerId != userID {
			return errors.New("unauthorized: you can only update your own products")
		}
	}

	title := existing.Title
	if updates.Title != "" {
		title = updates.Title
	}
	desc := existing.Description
	if updates.Description != "" {
		desc = updates.Description
	}
	price := existing.Price
	if updates.Price > 0 {
		price = updates.Price
	}
	quantity := existing.Quantity
	if updates.Quantity >= 0 {
		quantity = updates.Quantity
	}
	if validationErrs := utils.ValidateProduct(title, desc, price, quantity); len(validationErrs) > 0 {
		return utils.FormatValidationErrors(validationErrs)
	}

	images := existing.Images
	if len(updates.Images) > 0 {
		images = updates.Images
	}
	tags := existing.Tags
	if len(updates.Tags) > 0 {
		tags = updates.Tags
	}
	imagesValue, _ := json.Marshal(images)
	tagsValue, _ := json.Marshal(tags)
	category := existing.Category
	if updates.Category != "" {
		category = updates.Category
	}
	sku := existing.SKU
	if updates.SKU != "" {
		sku = updates.SKU
	}
	condition := existing.Condition
	if updates.Condition != "" {
		condition = updates.Condition
	}

	_, err = database.Pool.Exec(ctx,
		`UPDATE products SET title = $1, name = $1, description = $2, price = $3, quantity = $4, images = $5, category = $6, sku = $7, condition = $8, tags = $9, is_available = $10, updated_at = $11 WHERE id = $12`,
		title, desc, price, quantity, string(imagesValue), category, sku, condition, string(tagsValue), updates.IsAvailable, time.Now(), id,
	)
	if err != nil {
		return errors.New("failed to update product")
	}
	return nil
}

func DeleteProduct(id string, userEmail string, userType model.UserType) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var sellerID string
	err := database.Pool.QueryRow(ctx, `SELECT seller_id FROM products WHERE id = $1`, id).Scan(&sellerID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return errors.New("product not found")
		}
		return err
	}

	if userType != model.TypeAdmin {
		userID, err := GetUserIDByEmail(userEmail)
		if err != nil {
			return errors.New("user not found")
		}
		if sellerID != userID {
			return errors.New("unauthorized: you can only delete your own products")
		}
	}

	_, err = database.Pool.Exec(ctx, `DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		return errors.New("failed to delete product")
	}
	return nil
}

func GetProductsBySeller(sellerId string) ([]model.Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := database.Pool.Query(ctx,
		`SELECT id, seller_id, seller_name, title, name, description, price, quantity, is_available, images, category, sku, condition, tags, views, rating, review_count, created_at, updated_at FROM products WHERE seller_id = $1 ORDER BY created_at DESC`, sellerId,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []model.Product
	for rows.Next() {
		var p model.Product
		var imagesJSON, tagsJSON []byte
		if err := rows.Scan(&p.ID, &p.SellerId, &p.SellerName, &p.Title, &p.Name, &p.Description, &p.Price, &p.Quantity, &p.IsAvailable, &imagesJSON, &p.Category, &p.SKU, &p.Condition, &tagsJSON, &p.Views, &p.Rating, &p.ReviewCount, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(imagesJSON, &p.Images)
		_ = json.Unmarshal(tagsJSON, &p.Tags)
		if p.Name == "" {
			p.Name = p.Title
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func GetProductsBySellerEmail(sellerEmail string) ([]model.Product, error) {
	userID, err := GetUserIDByEmail(sellerEmail)
	if err != nil {
		return nil, errors.New("seller not found")
	}
	return GetProductsBySeller(userID)
}
