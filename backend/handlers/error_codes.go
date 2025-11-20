package handlers

import (
	"net/http"

	"github.com/sunfmin/apidemo2/backend/services"
)

// ErrorCode represents a typed error with code, message, HTTP status, and optional service error mapping
type ErrorCode struct {
	Code       string
	Message    string
	HTTPStatus int
	ServiceErr error // Optional: Maps to service sentinel error for automatic checking
}

// Errors is a singleton struct containing all error definitions for type safety
// Use these instead of hardcoded error strings throughout the codebase
var Errors = struct {
	// Validation errors (400)
	InvalidRequest   ErrorCode
	ValidationFailed ErrorCode
	MissingRequired  ErrorCode
	InvalidFormat    ErrorCode
	InvalidType      ErrorCode
	ValueOutOfRange  ErrorCode

	// Not found errors (404)
	NotFound         ErrorCode
	TemplateNotFound ErrorCode
	ProductNotFound  ErrorCode
	VariantNotFound  ErrorCode
	MediaNotFound    ErrorCode

	// Conflict errors (409)
	Conflict      ErrorCode
	DuplicateSKU  ErrorCode
	DuplicateName ErrorCode
	AlreadyExists ErrorCode
	HasProducts   ErrorCode

	// Internal errors (500)
	InternalError ErrorCode
	DatabaseError ErrorCode
	StorageError  ErrorCode
}{
	// Validation errors (400) - mapped to service errors
	InvalidRequest:   ErrorCode{"INVALID_REQUEST", "Invalid request body", http.StatusBadRequest, services.ErrInvalidRequest},
	ValidationFailed: ErrorCode{"VALIDATION_ERROR", "Validation failed", http.StatusBadRequest, services.ErrInvalidSKU},
	MissingRequired:  ErrorCode{"MISSING_REQUIRED", "Required field missing", http.StatusBadRequest, services.ErrMissingRequired},
	InvalidFormat:    ErrorCode{"INVALID_FORMAT", "Invalid format", http.StatusBadRequest, nil},
	InvalidType:      ErrorCode{"INVALID_TYPE", "Invalid type", http.StatusBadRequest, services.ErrInvalidType},
	ValueOutOfRange:  ErrorCode{"VALUE_OUT_OF_RANGE", "Value out of acceptable range", http.StatusBadRequest, services.ErrValueOutOfRange},

	// Not found errors (404) - mapped to service errors
	NotFound:         ErrorCode{"NOT_FOUND", "Resource not found", http.StatusNotFound, nil},
	TemplateNotFound: ErrorCode{"TEMPLATE_NOT_FOUND", "Template not found", http.StatusNotFound, services.ErrTemplateNotFound},
	ProductNotFound:  ErrorCode{"PRODUCT_NOT_FOUND", "Product not found", http.StatusNotFound, services.ErrProductNotFound},
	VariantNotFound:  ErrorCode{"VARIANT_NOT_FOUND", "Variant not found", http.StatusNotFound, services.ErrVariantNotFound},
	MediaNotFound:    ErrorCode{"MEDIA_NOT_FOUND", "Media file not found", http.StatusNotFound, services.ErrMediaNotFound},

	// Conflict errors (409) - mapped to service errors
	Conflict:      ErrorCode{"CONFLICT", "Resource conflict", http.StatusConflict, nil},
	DuplicateSKU:  ErrorCode{"DUPLICATE_SKU", "SKU already exists", http.StatusConflict, services.ErrDuplicateSKU},
	DuplicateName: ErrorCode{"DUPLICATE_NAME", "Name already exists", http.StatusConflict, services.ErrDuplicateName},
	AlreadyExists: ErrorCode{"ALREADY_EXISTS", "Resource already exists", http.StatusConflict, services.ErrAlreadyExists},
	HasProducts:   ErrorCode{"HAS_PRODUCTS", "Template has associated products", http.StatusConflict, services.ErrHasProducts},

	// Internal errors (500) - no service mapping (catch-all)
	InternalError: ErrorCode{"INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError, nil},
	DatabaseError: ErrorCode{"DATABASE_ERROR", "Database operation failed", http.StatusInternalServerError, nil},
	StorageError:  ErrorCode{"STORAGE_ERROR", "Storage operation failed", http.StatusInternalServerError, nil},
}

// RespondWithError writes an error response using a typed ErrorCode
func RespondWithError(w http.ResponseWriter, errCode ErrorCode) {
	ErrorResponse(w, errCode.Code, errCode.Message, errCode.HTTPStatus)
}

// RespondWithErrorMessage writes an error response with a custom message
func RespondWithErrorMessage(w http.ResponseWriter, errCode ErrorCode, customMessage string) {
	ErrorResponse(w, errCode.Code, customMessage, errCode.HTTPStatus)
}

// AllErrors returns a slice of all error codes for iteration
func AllErrors() []ErrorCode {
	return []ErrorCode{
		Errors.InvalidRequest,
		Errors.ValidationFailed,
		Errors.MissingRequired,
		Errors.InvalidFormat,
		Errors.InvalidType,
		Errors.ValueOutOfRange,
		Errors.NotFound,
		Errors.TemplateNotFound,
		Errors.ProductNotFound,
		Errors.VariantNotFound,
		Errors.MediaNotFound,
		Errors.Conflict,
		Errors.DuplicateSKU,
		Errors.DuplicateName,
		Errors.AlreadyExists,
		Errors.HasProducts,
		Errors.InternalError,
		Errors.DatabaseError,
		Errors.StorageError,
	}
}
