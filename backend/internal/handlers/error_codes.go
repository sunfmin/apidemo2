package handlers

import "net/http"

// ErrorCode represents a typed error with code, message, and HTTP status
type ErrorCode struct {
	Code       string
	Message    string
	HTTPStatus int
}

// Errors is a singleton struct containing all error definitions for type safety
// Use these instead of hardcoded error strings throughout the codebase
var Errors = struct {
	// Validation errors (400)
	InvalidRequest     ErrorCode
	ValidationFailed   ErrorCode
	MissingRequired    ErrorCode
	InvalidFormat      ErrorCode
	InvalidType        ErrorCode
	ValueOutOfRange    ErrorCode

	// Not found errors (404)
	NotFound         ErrorCode
	TemplateNotFound ErrorCode
	ProductNotFound  ErrorCode
	VariantNotFound  ErrorCode
	MediaNotFound    ErrorCode

	// Conflict errors (409)
	Conflict       ErrorCode
	DuplicateSKU   ErrorCode
	DuplicateName  ErrorCode
	AlreadyExists  ErrorCode

	// Internal errors (500)
	InternalError ErrorCode
	DatabaseError ErrorCode
	StorageError  ErrorCode
}{
	// Validation errors (400)
	InvalidRequest:   ErrorCode{"INVALID_REQUEST", "Invalid request body", http.StatusBadRequest},
	ValidationFailed: ErrorCode{"VALIDATION_ERROR", "Validation failed", http.StatusBadRequest},
	MissingRequired:  ErrorCode{"MISSING_REQUIRED", "Required field missing", http.StatusBadRequest},
	InvalidFormat:    ErrorCode{"INVALID_FORMAT", "Invalid format", http.StatusBadRequest},
	InvalidType:      ErrorCode{"INVALID_TYPE", "Invalid type", http.StatusBadRequest},
	ValueOutOfRange:  ErrorCode{"VALUE_OUT_OF_RANGE", "Value out of acceptable range", http.StatusBadRequest},

	// Not found errors (404)
	NotFound:         ErrorCode{"NOT_FOUND", "Resource not found", http.StatusNotFound},
	TemplateNotFound: ErrorCode{"TEMPLATE_NOT_FOUND", "Template not found", http.StatusNotFound},
	ProductNotFound:  ErrorCode{"PRODUCT_NOT_FOUND", "Product not found", http.StatusNotFound},
	VariantNotFound:  ErrorCode{"VARIANT_NOT_FOUND", "Variant not found", http.StatusNotFound},
	MediaNotFound:    ErrorCode{"MEDIA_NOT_FOUND", "Media file not found", http.StatusNotFound},

	// Conflict errors (409)
	Conflict:      ErrorCode{"CONFLICT", "Resource conflict", http.StatusConflict},
	DuplicateSKU:  ErrorCode{"DUPLICATE_SKU", "SKU already exists", http.StatusConflict},
	DuplicateName: ErrorCode{"DUPLICATE_NAME", "Name already exists", http.StatusConflict},
	AlreadyExists: ErrorCode{"ALREADY_EXISTS", "Resource already exists", http.StatusConflict},

	// Internal errors (500)
	InternalError: ErrorCode{"INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError},
	DatabaseError: ErrorCode{"DATABASE_ERROR", "Database operation failed", http.StatusInternalServerError},
	StorageError:  ErrorCode{"STORAGE_ERROR", "Storage operation failed", http.StatusInternalServerError},
}

// RespondWithError writes an error response using a typed ErrorCode
func RespondWithError(w http.ResponseWriter, errCode ErrorCode) {
	ErrorResponse(w, errCode.Code, errCode.Message, errCode.HTTPStatus)
}

// RespondWithErrorMessage writes an error response with a custom message
func RespondWithErrorMessage(w http.ResponseWriter, errCode ErrorCode, customMessage string) {
	ErrorResponse(w, errCode.Code, customMessage, errCode.HTTPStatus)
}

