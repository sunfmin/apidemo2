package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/opentracing/opentracing-go"

	pb "github.com/sunfmin/apidemo2/backend/api/gen/pim/v1"
	"github.com/sunfmin/apidemo2/backend/internal/services"
)

// VariantHandler handles HTTP requests for product variants
type VariantHandler struct {
	service services.VariantService
}

// NewVariantHandler creates a new VariantHandler
func NewVariantHandler(service services.VariantService) *VariantHandler {
	return &VariantHandler{service: service}
}

// Create handles POST /api/v1/products/{product_id}/variants
func (h *VariantHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Create child span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "VariantHandler.Create")
	defer span.Finish()

	span.SetTag("http.method", r.Method)
	span.SetTag("http.url", r.URL.String())

	// Parse request
	var req pb.CreateVariantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.SetTag("error", true)
		RespondWithError(w, Errors.InvalidRequest)
		return
	}

	// Call service
	variant, err := h.service.Create(ctx, &req)
	if err != nil {
		span.SetTag("error", true)
		span.SetTag("error.message", err.Error())
		HandleServiceError(w, err)
		return
	}

	// Return response
	span.SetTag("http.status_code", http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(&pb.CreateVariantResponse{Variant: variant})
}

// Get handles GET /api/v1/variants/{id}
func (h *VariantHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Create child span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "VariantHandler.Get")
	defer span.Finish()

	span.SetTag("http.method", r.Method)
	span.SetTag("http.url", r.URL.String())

	// Extract ID from URL path
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/variants/")
	if id == "" {
		span.SetTag("error", true)
		RespondWithErrorMessage(w, Errors.InvalidRequest, "Variant ID required")
		return
	}

	// Call service
	variant, err := h.service.Get(ctx, id)
	if err != nil {
		span.SetTag("error", true)
		span.SetTag("error.message", err.Error())
		HandleServiceError(w, err)
		return
	}

	// Return response
	span.SetTag("http.status_code", http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&pb.GetVariantResponse{Variant: variant})
}

// List handles GET /api/v1/products/{product_id}/variants
func (h *VariantHandler) List(w http.ResponseWriter, r *http.Request) {
	// Create child span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "VariantHandler.List")
	defer span.Finish()

	span.SetTag("http.method", r.Method)
	span.SetTag("http.url", r.URL.String())

	// Extract product ID from URL path
	productID := strings.TrimPrefix(r.URL.Path, "/api/v1/products/")
	productID = strings.TrimSuffix(productID, "/variants")
	if productID == "" {
		span.SetTag("error", true)
		RespondWithErrorMessage(w, Errors.InvalidRequest, "Product ID required")
		return
	}

	// Call service
	variants, err := h.service.List(ctx, productID)
	if err != nil {
		span.SetTag("error", true)
		span.SetTag("error.message", err.Error())
		HandleServiceError(w, err)
		return
	}

	// Return response
	span.SetTag("http.status_code", http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&pb.ListVariantsResponse{Variants: variants})
}

// Update handles PUT /api/v1/variants/{id}
func (h *VariantHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Create child span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "VariantHandler.Update")
	defer span.Finish()

	span.SetTag("http.method", r.Method)
	span.SetTag("http.url", r.URL.String())

	// Extract ID from URL path
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/variants/")
	if id == "" {
		span.SetTag("error", true)
		RespondWithErrorMessage(w, Errors.InvalidRequest, "Variant ID required")
		return
	}

	// Parse request
	var req pb.UpdateVariantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.SetTag("error", true)
		RespondWithError(w, Errors.InvalidRequest)
		return
	}

	// Set ID from URL
	req.Id = id

	// Call service
	variant, err := h.service.Update(ctx, &req)
	if err != nil {
		span.SetTag("error", true)
		span.SetTag("error.message", err.Error())
		HandleServiceError(w, err)
		return
	}

	// Return response
	span.SetTag("http.status_code", http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&pb.UpdateVariantResponse{Variant: variant})
}

// Delete handles DELETE /api/v1/variants/{id}
func (h *VariantHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Create child span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "VariantHandler.Delete")
	defer span.Finish()

	span.SetTag("http.method", r.Method)
	span.SetTag("http.url", r.URL.String())

	// Extract ID from URL path
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/variants/")
	if id == "" {
		span.SetTag("error", true)
		RespondWithErrorMessage(w, Errors.InvalidRequest, "Variant ID required")
		return
	}

	// Call service
	err := h.service.Delete(ctx, id)
	if err != nil {
		span.SetTag("error", true)
		span.SetTag("error.message", err.Error())
		HandleServiceError(w, err)
		return
	}

	// Return response
	span.SetTag("http.status_code", http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&pb.DeleteVariantResponse{Success: true})
}

// BulkCreate handles POST /api/v1/products/{product_id}/variants/bulk
func (h *VariantHandler) BulkCreate(w http.ResponseWriter, r *http.Request) {
	// Create child span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "VariantHandler.BulkCreate")
	defer span.Finish()

	span.SetTag("http.method", r.Method)
	span.SetTag("http.url", r.URL.String())

	// Parse request
	var req pb.BulkCreateVariantsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.SetTag("error", true)
		RespondWithError(w, Errors.InvalidRequest)
		return
	}

	// Call service
	response, err := h.service.BulkCreate(ctx, &req)
	if err != nil {
		span.SetTag("error", true)
		span.SetTag("error.message", err.Error())
		HandleServiceError(w, err)
		return
	}

	// Determine HTTP status code based on results
	statusCode := http.StatusCreated
	if len(response.Variants) == 0 {
		// All variants failed
		statusCode = http.StatusBadRequest
	} else if len(response.Errors) > 0 {
		// Partial success
		statusCode = http.StatusMultiStatus
	}

	// Return response
	span.SetTag("http.status_code", statusCode)
	span.SetTag("variants_created", len(response.Variants))
	span.SetTag("variants_failed", len(response.Errors))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}
