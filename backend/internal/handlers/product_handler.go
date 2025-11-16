package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/opentracing/opentracing-go"

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
		span.SetTag("error.message", "Invalid request body")
		ErrorResponse(w, "INVALID_REQUEST", "Invalid request body", http.StatusBadRequest)
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
		ErrorResponse(w, "INVALID_REQUEST", "Product ID required", http.StatusBadRequest)
		return
	}

	// Parse request
	var req pb.UpdateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.SetTag("error", true)
		ErrorResponse(w, "INVALID_REQUEST", "Invalid request body", http.StatusBadRequest)
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
		ErrorResponse(w, "INVALID_REQUEST", "Product ID required", http.StatusBadRequest)
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
		ErrorResponse(w, "INVALID_REQUEST", "Invalid request body", http.StatusBadRequest)
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

// HandleServiceError converts service errors to appropriate HTTP responses
func HandleServiceError(w http.ResponseWriter, err error) {
	errMsg := err.Error()

	// Check for specific error patterns
	switch {
	case strings.Contains(errMsg, "not found"), strings.Contains(errMsg, "record not found"):
		ErrorResponse(w, "NOT_FOUND", errMsg, http.StatusNotFound)
	case strings.Contains(errMsg, "duplicate"), strings.Contains(errMsg, "already exists"):
		ErrorResponse(w, "CONFLICT", errMsg, http.StatusConflict)
	case strings.Contains(errMsg, "required"), strings.Contains(errMsg, "invalid"), strings.Contains(errMsg, "validation"), strings.Contains(errMsg, "must be"), strings.Contains(errMsg, "cannot be"), strings.Contains(errMsg, "not in allowed options"):
		ErrorResponse(w, "VALIDATION_ERROR", errMsg, http.StatusBadRequest)
	default:
		ErrorResponse(w, "INTERNAL_ERROR", fmt.Sprintf("Internal server error: %s", errMsg), http.StatusInternalServerError)
	}
}

