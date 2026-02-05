package utils

import (
	"errors"
	"strings"
)

// ValidationError represents a validation error with field information
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// ValidateProduct validates product data
func ValidateProduct(title, description string, price float64, quantity int) []*ValidationError {
	var errs []*ValidationError

	title = strings.TrimSpace(title)
	if title == "" {
		errs = append(errs, &ValidationError{Field: "title", Message: "Product title is required"})
	} else if len(title) > 200 {
		errs = append(errs, &ValidationError{Field: "title", Message: "Product title must be 200 characters or less"})
	}

	description = strings.TrimSpace(description)
	if description == "" {
		errs = append(errs, &ValidationError{Field: "description", Message: "Product description is required"})
	} else if len(description) > 5000 {
		errs = append(errs, &ValidationError{Field: "description", Message: "Product description must be 5000 characters or less"})
	}

	if price < 0 {
		errs = append(errs, &ValidationError{Field: "price", Message: "Price must be non-negative"})
	} else if price == 0 {
		errs = append(errs, &ValidationError{Field: "price", Message: "Price must be greater than 0"})
	}

	if quantity < 0 {
		errs = append(errs, &ValidationError{Field: "quantity", Message: "Quantity must be non-negative"})
	}

	return errs
}

// FormatValidationErrors formats validation errors into a single error message
func FormatValidationErrors(errs []*ValidationError) error {
	if len(errs) == 0 {
		return nil
	}

	var messages []string
	for _, err := range errs {
		messages = append(messages, err.Message)
	}
	return errors.New(strings.Join(messages, "; "))
}
