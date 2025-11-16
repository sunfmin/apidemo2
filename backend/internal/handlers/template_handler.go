package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/opentracing/opentracing-go"

	pb "github.com/sunfmin/apidemo2/backend/api/gen/pim/v1"
	"github.com/sunfmin/apidemo2/backend/internal/services"
)

// Note: Error codes singleton defined in error_codes.go

// TemplateHandler handles HTTP requests for product templates
type TemplateHandler struct {
	service services.TemplateService
}

// NewTemplateHandler creates a new TemplateHandler
func NewTemplateHandler(service services.TemplateService) *TemplateHandler {
	return &TemplateHandler{service: service}
}

// Create handles POST /api/v1/templates
func (h *TemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Create child span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "TemplateHandler.Create")
	defer span.Finish()

	// Parse request
	var req pb.CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.SetTag("error", true)
		RespondWithError(w, Errors.InvalidRequest)
		return
	}

	// Call service
	template, err := h.service.Create(ctx, &req)
	if err != nil {
		span.SetTag("error", true)
		span.SetTag("error.message", err.Error())

		// Determine error type based on error message
		if strings.Contains(err.Error(), "already exists") {
			RespondWithErrorMessage(w, Errors.DuplicateName, err.Error())
			return
		}
		if strings.Contains(err.Error(), "required") || 
		   strings.Contains(err.Error(), "invalid") || 
		   strings.Contains(err.Error(), "must") || 
		   strings.Contains(err.Error(), "duplicate") {
			RespondWithErrorMessage(w, Errors.ValidationFailed, err.Error())
			return
		}

		RespondWithErrorMessage(w, Errors.InternalError, "Failed to create template")
		return
	}

	// Return success response
	response := &pb.CreateTemplateResponse{Template: template}
	JSONResponse(w, response, http.StatusCreated)
}

// Get handles GET /api/v1/templates/{id}
func (h *TemplateHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Create child span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "TemplateHandler.Get")
	defer span.Finish()

	// Extract ID from URL path
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/templates/")
	if id == "" || id == "/api/v1/templates" {
		span.SetTag("error", true)
		RespondWithErrorMessage(w, Errors.InvalidRequest, "Template ID is required")
		return
	}

	// Call service
	template, err := h.service.Get(ctx, id)
	if err != nil {
		span.SetTag("error", true)
		span.SetTag("error.message", err.Error())

		if strings.Contains(err.Error(), "not found") {
			RespondWithErrorMessage(w, Errors.TemplateNotFound, err.Error())
			return
		}

		RespondWithErrorMessage(w, Errors.InternalError, "Failed to get template")
		return
	}

	// Return success response
	response := &pb.GetTemplateResponse{Template: template}
	JSONResponse(w, response, http.StatusOK)
}

// List handles GET /api/v1/templates
func (h *TemplateHandler) List(w http.ResponseWriter, r *http.Request) {
	// Create child span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "TemplateHandler.List")
	defer span.Finish()

	// Parse query parameters
	query := r.URL.Query()
	
	// Build request
	req := &pb.ListTemplatesRequest{
		Pagination: &pb.PaginationRequest{},
		Search:     query.Get("search"),
		SortBy:     query.Get("sort_by"),
		SortOrder:  query.Get("sort_order"),
	}

	// Parse pagination parameters
	if pageStr := query.Get("page"); pageStr != "" {
		var page int32
		if _, err := fmt.Sscanf(pageStr, "%d", &page); err == nil && page > 0 {
			req.Pagination.Page = page
		}
	}

	if pageSizeStr := query.Get("page_size"); pageSizeStr != "" {
		var pageSize int32
		if _, err := fmt.Sscanf(pageSizeStr, "%d", &pageSize); err == nil && pageSize > 0 {
			req.Pagination.PageSize = pageSize
		}
	}

	// Call service
	templates, pagination, err := h.service.List(ctx, req)
	if err != nil {
		span.SetTag("error", true)
		span.SetTag("error.message", err.Error())
		RespondWithErrorMessage(w, Errors.InternalError, "Failed to list templates")
		return
	}

	// Return success response
	response := &pb.ListTemplatesResponse{
		Templates:  templates,
		Pagination: pagination,
	}
	JSONResponse(w, response, http.StatusOK)
}

// Update handles PUT /api/v1/templates/{id}
func (h *TemplateHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Create child span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "TemplateHandler.Update")
	defer span.Finish()

	// Extract ID from URL path
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/templates/")
	if id == "" || id == "/api/v1/templates" {
		span.SetTag("error", true)
		RespondWithErrorMessage(w, Errors.InvalidRequest, "Template ID is required")
		return
	}

	// Parse request
	var req pb.UpdateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.SetTag("error", true)
		RespondWithError(w, Errors.InvalidRequest)
		return
	}

	// Set ID from URL
	req.Id = id

	// Call service
	template, err := h.service.Update(ctx, &req)
	if err != nil {
		span.SetTag("error", true)
		span.SetTag("error.message", err.Error())

		if strings.Contains(err.Error(), "not found") {
			RespondWithErrorMessage(w, Errors.TemplateNotFound, err.Error())
			return
		}
		if strings.Contains(err.Error(), "already exists") {
			RespondWithErrorMessage(w, Errors.DuplicateName, err.Error())
			return
		}
		if strings.Contains(err.Error(), "required") || 
		   strings.Contains(err.Error(), "invalid") || 
		   strings.Contains(err.Error(), "must") || 
		   strings.Contains(err.Error(), "duplicate") {
			RespondWithErrorMessage(w, Errors.ValidationFailed, err.Error())
			return
		}

		RespondWithErrorMessage(w, Errors.InternalError, "Failed to update template")
		return
	}

	// Return success response
	response := &pb.UpdateTemplateResponse{Template: template}
	JSONResponse(w, response, http.StatusOK)
}

// Delete handles DELETE /api/v1/templates/{id}
func (h *TemplateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Create child span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "TemplateHandler.Delete")
	defer span.Finish()

	// Extract ID from URL path
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/templates/")
	if id == "" || id == "/api/v1/templates" {
		span.SetTag("error", true)
		RespondWithErrorMessage(w, Errors.InvalidRequest, "Template ID is required")
		return
	}

	// Call service
	err := h.service.Delete(ctx, id)
	if err != nil {
		span.SetTag("error", true)
		span.SetTag("error.message", err.Error())

		if strings.Contains(err.Error(), "not found") {
			RespondWithErrorMessage(w, Errors.TemplateNotFound, err.Error())
			return
		}
		if strings.Contains(err.Error(), "has products") {
			RespondWithErrorMessage(w, Errors.Conflict, err.Error())
			return
		}

		RespondWithErrorMessage(w, Errors.InternalError, "Failed to delete template")
		return
	}

	// Return success response
	response := &pb.DeleteTemplateResponse{
		Success: true,
		Message: "Template deleted successfully",
	}
	JSONResponse(w, response, http.StatusOK)
}

