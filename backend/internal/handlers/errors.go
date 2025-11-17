package handlers

import (
	"encoding/json"
	"net/http"

	pb "github.com/sunfmin/apidemo2/backend/api/gen/pim/v1"
)

// ErrorResponse writes a JSON error response using protobuf ErrorResponse message
func ErrorResponse(w http.ResponseWriter, code string, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := &pb.ErrorResponse{
		Code:    code,
		Message: message,
	}

	json.NewEncoder(w).Encode(resp)
}

// ValidationErrorResponse writes a JSON error response with field-specific errors
func ValidationErrorResponse(w http.ResponseWriter, message string, fieldErrors []*pb.FieldError, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := &pb.ErrorResponse{
		Code:        "VALIDATION_ERROR",
		Message:     message,
		FieldErrors: fieldErrors,
	}

	json.NewEncoder(w).Encode(resp)
}

// JSONResponse writes a successful JSON response
func JSONResponse(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
