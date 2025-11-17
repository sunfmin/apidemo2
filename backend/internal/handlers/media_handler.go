package handlers

import (
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/opentracing/opentracing-go"

	pb "github.com/sunfmin/apidemo2/backend/api/gen/pim/v1"
	"github.com/sunfmin/apidemo2/backend/internal/services"
)

const (
	MaxUploadSize      = 100 * 1024 * 1024 // 100 MB
	MaxMultipartMemory = 32 << 20          // 32 MB
)

// MediaHandler handles HTTP requests for media files
type MediaHandler struct {
	service services.MediaService
}

// NewMediaHandler creates a new MediaHandler
func NewMediaHandler(service services.MediaService) *MediaHandler {
	return &MediaHandler{service: service}
}

// Upload handles POST /api/v1/media/upload
func (h *MediaHandler) Upload(w http.ResponseWriter, r *http.Request) {
	// Create child span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "MediaHandler.Upload")
	defer span.Finish()

	span.SetTag("http.method", r.Method)
	span.SetTag("http.url", r.URL.String())

	// Parse multipart form
	if err := r.ParseMultipartForm(MaxMultipartMemory); err != nil {
		span.SetTag("error", true)
		RespondWithErrorMessage(w, Errors.InvalidRequest, "Failed to parse multipart form")
		return
	}

	// Extract form fields
	entityType := r.FormValue("entity_type")
	entityID := r.FormValue("entity_id")
	attributeName := r.FormValue("attribute_name")

	if entityType == "" || entityID == "" || attributeName == "" {
		span.SetTag("error", true)
		RespondWithErrorMessage(w, Errors.InvalidRequest, "entity_type, entity_id, and attribute_name are required")
		return
	}

	// Get uploaded file
	file, header, err := r.FormFile("file")
	if err != nil {
		span.SetTag("error", true)
		RespondWithErrorMessage(w, Errors.InvalidRequest, "No file uploaded")
		return
	}
	defer file.Close()

	// Call service
	mediaFile, err := h.service.Upload(ctx, entityType, entityID, attributeName, file, header)
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
	json.NewEncoder(w).Encode(&pb.UploadMediaResponse{MediaFile: mediaFile})
}

// UploadBulk handles POST /api/v1/media/upload/bulk
func (h *MediaHandler) UploadBulk(w http.ResponseWriter, r *http.Request) {
	// Create child span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "MediaHandler.UploadBulk")
	defer span.Finish()

	span.SetTag("http.method", r.Method)
	span.SetTag("http.url", r.URL.String())

	// Parse multipart form
	if err := r.ParseMultipartForm(MaxMultipartMemory); err != nil {
		span.SetTag("error", true)
		RespondWithErrorMessage(w, Errors.InvalidRequest, "Failed to parse multipart form")
		return
	}

	// Extract form fields
	entityType := r.FormValue("entity_type")
	entityID := r.FormValue("entity_id")
	attributeName := r.FormValue("attribute_name")

	if entityType == "" || entityID == "" || attributeName == "" {
		span.SetTag("error", true)
		RespondWithErrorMessage(w, Errors.InvalidRequest, "entity_type, entity_id, and attribute_name are required")
		return
	}

	// Get uploaded files
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		span.SetTag("error", true)
		RespondWithErrorMessage(w, Errors.InvalidRequest, "No files uploaded")
		return
	}

	// Open all files
	openFiles := make([]multipart.File, 0, len(files))
	headers := make([]*multipart.FileHeader, 0, len(files))

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			span.SetTag("error", true)
			RespondWithErrorMessage(w, Errors.InvalidRequest, "Failed to open uploaded file")
			return
		}
		openFiles = append(openFiles, file)
		headers = append(headers, fileHeader)
	}

	// Ensure all files are closed
	defer func() {
		for _, f := range openFiles {
			f.Close()
		}
	}()

	// Call service
	response, err := h.service.UploadBulk(ctx, entityType, entityID, attributeName, openFiles, headers)
	if err != nil {
		span.SetTag("error", true)
		span.SetTag("error.message", err.Error())
		HandleServiceError(w, err)
		return
	}

	// Determine status code
	statusCode := http.StatusCreated
	if len(response.MediaFiles) == 0 {
		statusCode = http.StatusBadRequest
	} else if len(response.Errors) > 0 {
		statusCode = http.StatusMultiStatus
	}

	// Return response
	span.SetTag("http.status_code", statusCode)
	span.SetTag("files_uploaded", len(response.MediaFiles))
	span.SetTag("files_failed", len(response.Errors))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// Get handles GET /api/v1/media/{id}
func (h *MediaHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Create child span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "MediaHandler.Get")
	defer span.Finish()

	span.SetTag("http.method", r.Method)
	span.SetTag("http.url", r.URL.String())

	// Extract ID from URL path
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/media/")
	if id == "" {
		span.SetTag("error", true)
		RespondWithErrorMessage(w, Errors.InvalidRequest, "Media ID required")
		return
	}

	// Call service
	mediaFile, err := h.service.Get(ctx, id)
	if err != nil {
		span.SetTag("error", true)
		span.SetTag("error.message", err.Error())
		HandleServiceError(w, err)
		return
	}

	// Return response
	span.SetTag("http.status_code", http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&pb.GetMediaResponse{MediaFile: mediaFile})
}

// List handles GET /api/v1/media
func (h *MediaHandler) List(w http.ResponseWriter, r *http.Request) {
	// Create child span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "MediaHandler.List")
	defer span.Finish()

	span.SetTag("http.method", r.Method)
	span.SetTag("http.url", r.URL.String())

	// Parse query parameters
	query := r.URL.Query()
	entityType := query.Get("entity_type")
	entityID := query.Get("entity_id")
	attributeName := query.Get("attribute_name")
	fileType := query.Get("file_type")

	if entityType == "" || entityID == "" {
		span.SetTag("error", true)
		RespondWithErrorMessage(w, Errors.InvalidRequest, "entity_type and entity_id are required")
		return
	}

	// Call service
	mediaFiles, err := h.service.List(ctx, entityType, entityID, attributeName, fileType)
	if err != nil {
		span.SetTag("error", true)
		span.SetTag("error.message", err.Error())
		HandleServiceError(w, err)
		return
	}

	// Return response
	span.SetTag("http.status_code", http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&pb.ListMediaResponse{MediaFiles: mediaFiles})
}

// Update handles PUT /api/v1/media/{id}
func (h *MediaHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Create child span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "MediaHandler.Update")
	defer span.Finish()

	span.SetTag("http.method", r.Method)
	span.SetTag("http.url", r.URL.String())

	// Extract ID from URL path
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/media/")
	if id == "" {
		span.SetTag("error", true)
		RespondWithErrorMessage(w, Errors.InvalidRequest, "Media ID required")
		return
	}

	// Parse request
	var req pb.UpdateMediaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.SetTag("error", true)
		RespondWithError(w, Errors.InvalidRequest)
		return
	}

	// Prepare pointers for optional fields
	var displayOrder *int32
	var fileName *string

	if req.DisplayOrder != 0 {
		displayOrder = &req.DisplayOrder
	}
	if req.FileName != "" {
		fileName = &req.FileName
	}

	// Call service
	mediaFile, err := h.service.Update(ctx, id, displayOrder, fileName)
	if err != nil {
		span.SetTag("error", true)
		span.SetTag("error.message", err.Error())
		HandleServiceError(w, err)
		return
	}

	// Return response
	span.SetTag("http.status_code", http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&pb.UpdateMediaResponse{MediaFile: mediaFile})
}

// Delete handles DELETE /api/v1/media/{id}
func (h *MediaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Create child span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "MediaHandler.Delete")
	defer span.Finish()

	span.SetTag("http.method", r.Method)
	span.SetTag("http.url", r.URL.String())

	// Extract ID from URL path
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/media/")
	if id == "" {
		span.SetTag("error", true)
		RespondWithErrorMessage(w, Errors.InvalidRequest, "Media ID required")
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
	json.NewEncoder(w).Encode(&pb.DeleteMediaResponse{Success: true})
}

// Reorder handles POST /api/v1/media/reorder
func (h *MediaHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	// Create child span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "MediaHandler.Reorder")
	defer span.Finish()

	span.SetTag("http.method", r.Method)
	span.SetTag("http.url", r.URL.String())

	// Parse request
	var req pb.ReorderMediaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.SetTag("error", true)
		RespondWithError(w, Errors.InvalidRequest)
		return
	}

	// Call service
	mediaFiles, err := h.service.Reorder(ctx, req.EntityType, req.EntityId, req.AttributeName, req.MediaIds)
	if err != nil {
		span.SetTag("error", true)
		span.SetTag("error.message", err.Error())
		HandleServiceError(w, err)
		return
	}

	// Return response
	span.SetTag("http.status_code", http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&pb.ReorderMediaResponse{MediaFiles: mediaFiles})
}

// ServeFile handles GET /api/v1/media/file/{id}
func (h *MediaHandler) ServeFile(w http.ResponseWriter, r *http.Request) {
	// Create child span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "MediaHandler.ServeFile")
	defer span.Finish()

	span.SetTag("http.method", r.Method)
	span.SetTag("http.url", r.URL.String())

	// Extract ID from URL path
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/media/file/")
	if id == "" {
		span.SetTag("error", true)
		http.Error(w, "Media ID required", http.StatusBadRequest)
		return
	}

	// Get media file
	mediaFile, err := h.service.Get(ctx, id)
	if err != nil {
		span.SetTag("error", true)
		http.Error(w, "Media file not found", http.StatusNotFound)
		return
	}

	// Redirect to the file URL
	span.SetTag("http.status_code", http.StatusFound)
	http.Redirect(w, r, mediaFile.Url, http.StatusFound)
}

// ServeThumbnail handles GET /api/v1/media/thumbnail/{id}
func (h *MediaHandler) ServeThumbnail(w http.ResponseWriter, r *http.Request) {
	// Create child span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "MediaHandler.ServeThumbnail")
	defer span.Finish()

	span.SetTag("http.method", r.Method)
	span.SetTag("http.url", r.URL.String())

	// Extract ID from URL path
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/media/thumbnail/")
	if id == "" {
		span.SetTag("error", true)
		http.Error(w, "Media ID required", http.StatusBadRequest)
		return
	}

	// Get media file
	mediaFile, err := h.service.Get(ctx, id)
	if err != nil {
		span.SetTag("error", true)
		http.Error(w, "Media file not found", http.StatusNotFound)
		return
	}

	// Redirect to the thumbnail URL
	if mediaFile.ThumbnailUrl == "" {
		span.SetTag("error", true)
		http.Error(w, "No thumbnail available", http.StatusNotFound)
		return
	}

	span.SetTag("http.status_code", http.StatusFound)
	http.Redirect(w, r, mediaFile.ThumbnailUrl, http.StatusFound)
}

// ParseDisplayOrder is a helper to parse display_order from form value
func ParseDisplayOrder(value string) int32 {
	if value == "" {
		return 0
	}
	order, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0
	}
	return int32(order)
}
