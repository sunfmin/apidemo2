package services

import "errors"

// Domain-specific sentinel errors for service layer
// These enable type-safe error checking with errors.Is() across wrapped errors
// Use with fmt.Errorf to wrap: fmt.Errorf("operation context: %w", ErrNotFound)
var (
	// Not Found errors
	ErrTemplateNotFound = errors.New("template not found")
	ErrProductNotFound  = errors.New("product not found")
	ErrVariantNotFound  = errors.New("variant not found")
	ErrMediaNotFound    = errors.New("media file not found")

	// Validation errors
	ErrInvalidSKU      = errors.New("invalid SKU format")
	ErrInvalidRequest  = errors.New("invalid request")
	ErrMissingRequired = errors.New("required field missing")
	ErrInvalidType     = errors.New("invalid type")
	ErrValueOutOfRange = errors.New("value out of range")

	// Conflict errors
	ErrDuplicateSKU  = errors.New("SKU already exists")
	ErrDuplicateName = errors.New("name already exists")
	ErrAlreadyExists = errors.New("resource already exists")

	// Business logic errors
	ErrHasProducts = errors.New("template has associated products")
)

