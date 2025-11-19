package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/opentracing/opentracing-go"
	"gorm.io/gorm"

	pb "github.com/sunfmin/apidemo2/backend/api/gen/pim/v1"
	"github.com/sunfmin/apidemo2/backend/internal/services"
)

// ProductHandler handles HTTP requests for products
type ProductHandler struct {
	service services.ProductService
}

// NewProductHandler creates a new ProductHandler
func NewProductHandler(service services.ProductService) *ProductHandler {
	return &ProductHandler{service: service}
}

// Create handles POST /api/v1/products
func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Create child span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "ProductHandler.Create")
	defer span.Finish()

	span.SetTag("http.method", r.Method)
	span.SetTag("http.url", r.URL.String())

	// Parse request
	var req pb.CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.SetTag("error", true)
		span.SetTag("error.message", Errors.InvalidRequest.Message)
		RespondWithError(w, Errors.InvalidRequest)
		return
	}

	// Call service
	product, err := h.service.Create(ctx, &req)
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
	json.NewEncoder(w).Encode(&pb.CreateProductResponse{Product: product})
}

// Get handles GET /api/v1/products/{id}
func (h *ProductHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Create child span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "ProductHandler.Get")
	defer span.Finish()

	span.SetTag("http.method", r.Method)
	span.SetTag("http.url", r.URL.String())

	// Extract ID from URL path
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/products/")
	if id == "" {
		span.SetTag("error", true)
		ErrorResponse(w, "INVALID_REQUEST", "Product ID required", http.StatusBadRequest)
		return
	}

	// Parse query parameters
	includeVariants := r.URL.Query().Get("include_variants") == "true"
	includeMedia := r.URL.Query().Get("include_media") == "true"

	// Build request
	req := &pb.GetProductRequest{
		Id:              id,
		IncludeVariants: includeVariants,
		IncludeMedia:    includeMedia,
	}

	// Call service
	response, err := h.service.Get(ctx, req)
	if err != nil {
		span.SetTag("error", true)
		span.SetTag("error.message", err.Error())
		HandleServiceError(w, err)
		return
	}

	// Return response
	span.SetTag("http.status_code", http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// List handles GET /api/v1/products
func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	// Create child span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "ProductHandler.List")
	defer span.Finish()

	span.SetTag("http.method", r.Method)
	span.SetTag("http.url", r.URL.String())

	// Parse query parameters
	query := r.URL.Query()

	page, _ := strconv.ParseInt(query.Get("page"), 10, 32)
	pageSize, _ := strconv.ParseInt(query.Get("page_size"), 10, 32)

	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Parse status from string to enum
	var status pb.ProductStatus
	statusStr := query.Get("status")
	switch statusStr {
	case "draft":
		status = pb.ProductStatus_PRODUCT_STATUS_DRAFT
	case "active":
		status = pb.ProductStatus_PRODUCT_STATUS_ACTIVE
	case "inactive":
		status = pb.ProductStatus_PRODUCT_STATUS_INACTIVE
	default:
		status = pb.ProductStatus_PRODUCT_STATUS_UNSPECIFIED
	}

	// Build request
	req := &pb.ListProductsRequest{
		Pagination: &pb.PaginationRequest{
			Page:     int32(page),
			PageSize: int32(pageSize),
		},
		TemplateId: query.Get("template_id"),
		Status:     status,
		Search:     query.Get("search"),
		SortBy:     query.Get("sort_by"),
		SortOrder:  query.Get("sort_order"),
	}

	// Call service
	products, pagination, err := h.service.List(ctx, req)
	if err != nil {
		span.SetTag("error", true)
		span.SetTag("error.message", err.Error())
		HandleServiceError(w, err)
		return
	}

	// Return response
	span.SetTag("http.status_code", http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&pb.ListProductsResponse{
		Products:   products,
		Pagination: pagination,
	})
}

// Update handles PUT /api/v1/products/{id}
func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Create child span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "ProductHandler.Update")
	defer span.Finish()

	span.SetTag("http.method", r.Method)
	span.SetTag("http.url", r.URL.String())

	// Extract ID from URL path
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/products/")
	if id == "" {
		span.SetTag("error", true)
		RespondWithErrorMessage(w, Errors.InvalidRequest, "Product ID required")
		return
	}

	// Parse request
	var req pb.UpdateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.SetTag("error", true)
		RespondWithError(w, Errors.InvalidRequest)
		return
	}

	// Set ID from URL
	req.Id = id

	// Call service
	product, err := h.service.Update(ctx, &req)
	if err != nil {
		span.SetTag("error", true)
		span.SetTag("error.message", err.Error())
		HandleServiceError(w, err)
		return
	}

	// Return response
	span.SetTag("http.status_code", http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&pb.UpdateProductResponse{Product: product})
}

// Delete handles DELETE /api/v1/products/{id}
func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Create child span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "ProductHandler.Delete")
	defer span.Finish()

	span.SetTag("http.method", r.Method)
	span.SetTag("http.url", r.URL.String())

	// Extract ID from URL path
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/products/")
	if id == "" {
		span.SetTag("error", true)
		RespondWithErrorMessage(w, Errors.InvalidRequest, "Product ID required")
		return
	}

	// Build request
	req := &pb.DeleteProductRequest{Id: id}

	// Call service
	response, err := h.service.Delete(ctx, req)
	if err != nil {
		span.SetTag("error", true)
		span.SetTag("error.message", err.Error())
		HandleServiceError(w, err)
		return
	}

	// Return response
	span.SetTag("http.status_code", http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// BulkUpdateStatus handles POST /api/v1/products/bulk/status
func (h *ProductHandler) BulkUpdateStatus(w http.ResponseWriter, r *http.Request) {
	// Create child span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "ProductHandler.BulkUpdateStatus")
	defer span.Finish()

	span.SetTag("http.method", r.Method)
	span.SetTag("http.url", r.URL.String())

	// Parse request
	var req pb.BulkUpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.SetTag("error", true)
		RespondWithError(w, Errors.InvalidRequest)
		return
	}

	// Call service
	response, err := h.service.BulkUpdateStatus(ctx, &req)
	if err != nil {
		span.SetTag("error", true)
		span.SetTag("error.message", err.Error())
		HandleServiceError(w, err)
		return
	}

	// Return response
	span.SetTag("http.status_code", http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleServiceError converts service errors to appropriate HTTP responses using typed error codes
// Uses errors.Is() and errors.As() for type-safe error checking (Principle XI)
func HandleServiceError(w http.ResponseWriter, err error) {
	// Use errors.Is() to check for specific error types (works with wrapped errors)
	switch {
	case errors.Is(err, services.ErrTemplateNotFound):
		RespondWithErrorMessage(w, Errors.TemplateNotFound, err.Error())
	case errors.Is(err, services.ErrProductNotFound):
		RespondWithErrorMessage(w, Errors.ProductNotFound, err.Error())
	case errors.Is(err, services.ErrVariantNotFound):
		RespondWithErrorMessage(w, Errors.VariantNotFound, err.Error())
	case errors.Is(err, services.ErrMediaNotFound):
		RespondWithErrorMessage(w, Errors.MediaNotFound, err.Error())
	case errors.Is(err, services.ErrDuplicateSKU):
		RespondWithErrorMessage(w, Errors.DuplicateSKU, err.Error())
	case errors.Is(err, services.ErrDuplicateName):
		RespondWithErrorMessage(w, Errors.DuplicateName, err.Error())
	case errors.Is(err, services.ErrAlreadyExists):
		RespondWithErrorMessage(w, Errors.AlreadyExists, err.Error())
	case errors.Is(err, services.ErrInvalidSKU):
		RespondWithErrorMessage(w, Errors.ValidationFailed, err.Error())
	case errors.Is(err, services.ErrMissingRequired):
		RespondWithErrorMessage(w, Errors.MissingRequired, err.Error())
	case errors.Is(err, services.ErrInvalidRequest):
		RespondWithErrorMessage(w, Errors.InvalidRequest, err.Error())
	case errors.Is(err, context.Canceled):
		http.Error(w, "Request cancelled", 499) // Client closed connection
	case errors.Is(err, context.DeadlineExceeded):
		http.Error(w, "Request timeout", 504) // Gateway timeout
	case errors.Is(err, gorm.ErrRecordNotFound):
		RespondWithErrorMessage(w, Errors.NotFound, err.Error())
	case errors.Is(err, services.ErrValueOutOfRange):
		RespondWithErrorMessage(w, Errors.ValueOutOfRange, err.Error())
	case errors.Is(err, services.ErrInvalidType):
		RespondWithErrorMessage(w, Errors.InvalidType, err.Error())
	default:
		// True internal errors - don't expose details to client
		RespondWithErrorMessage(w, Errors.InternalError, "Internal server error")
	}
}
