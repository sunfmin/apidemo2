package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/disintegration/imaging"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	pb "github.com/sunfmin/apidemo2/backend/api/gen/pim/v1"
	"github.com/sunfmin/apidemo2/backend/handlers"
	"github.com/sunfmin/apidemo2/backend/internal/models"
	"github.com/sunfmin/apidemo2/backend/services"
	"github.com/sunfmin/apidemo2/backend/tests/testutil"
)

// Helper function to create a test image file using imaging library
func createTestImageFile(t *testing.T) (string, func()) {
	tmpDir := t.TempDir()
	imagePath := filepath.Join(tmpDir, "test-image.jpg")

	// Create a simple 100x100 red square image
	img := imaging.New(100, 100, color.RGBA{255, 0, 0, 255})

	// Save as JPEG
	if err := imaging.Save(img, imagePath); err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return imagePath, cleanup
}

// Helper function to create multipart form with file upload
func createMultipartFormWithFile(fields map[string]string, fileFieldName, filePath string) (*bytes.Buffer, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add form fields
	for key, val := range fields {
		if err := writer.WriteField(key, val); err != nil {
			return nil, "", err
		}
	}

	// Add file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	part, err := writer.CreateFormFile(fileFieldName, filepath.Base(filePath))
	if err != nil {
		return nil, "", err
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, "", err
	}

	contentType := writer.FormDataContentType()
	writer.Close()

	return body, contentType, nil
}

// TestMediaHandler_Upload tests the POST /api/v1/media/upload endpoint
func TestMediaHandler_Upload(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "media_files", "product_variants", "variant_pricings", "variant_inventories", "products", "product_pricings", "product_inventories", "product_templates")

	// Create temporary storage directory
	storageDir := t.TempDir()

	// Initialize services and handlers
	localStorage, err := services.NewLocalStorage(storageDir, "http://localhost:8080/media")
	if err != nil {
		t.Fatalf("Failed to create local storage: %v", err)
	}
	imageProcessor := services.NewImageProcessor(300, 300)
	videoProcessor := services.NewVideoProcessor()

	mediaService := services.NewMediaService(db, localStorage, imageProcessor, videoProcessor, storageDir)
	handler := handlers.NewMediaHandler(mediaService)

	// Create test fixtures
	template := testutil.CreateTemplateFixture(db, "Electronics", `[
		{"name":"ProductImage","type":"image","required":true}
	]`)
	product := testutil.CreateProductFixture(db, template.ID, "Laptop", "LAPTOP-001", `{}`)
	db.Create(&models.ProductPricing{ProductID: product.ID, ListPrice: 999.99, Currency: "USD", ValidFrom: time.Now()})
	db.Create(&models.ProductInventory{ProductID: product.ID, LocationID: "default", OnHandQuantity: 10})

	// Create test image
	imagePath, cleanupImage := createTestImageFile(t)
	defer cleanupImage()

	// Table-driven test cases
	testCases := []struct {
		name           string
		fields         map[string]string
		filePath       string
		expectedStatus int
		expectError    bool
	}{
		{
			name: "successful_image_upload",
			fields: map[string]string{
				"entity_type":    "product",
				"entity_id":      product.ID,
				"attribute_name": "ProductImage",
			},
			filePath:       imagePath,
			expectedStatus: http.StatusCreated,
			expectError:    false,
		},
		{
			name: "missing_entity_type",
			fields: map[string]string{
				"entity_id":      product.ID,
				"attribute_name": "ProductImage",
			},
			filePath:       imagePath,
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "invalid_entity_type",
			fields: map[string]string{
				"entity_type":    "invalid",
				"entity_id":      product.ID,
				"attribute_name": "ProductImage",
			},
			filePath:       imagePath,
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "non_existent_entity",
			fields: map[string]string{
				"entity_type":    "product",
				"entity_id":      "550e8400-e29b-41d4-a716-446655440000",
				"attribute_name": "ProductImage",
			},
			filePath:       imagePath,
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create multipart form
			body, contentType, err := createMultipartFormWithFile(tc.fields, "file", tc.filePath)
			if err != nil {
				t.Fatalf("Failed to create multipart form: %v", err)
			}

			// Create HTTP request
			req := httptest.NewRequest(http.MethodPost, "/api/v1/media/upload", body)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()

			// Execute handler
			handler.Upload(rec, req)

			// Assert response status
			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}

			if !tc.expectError {
				// Parse success response
				var response pb.UploadMediaResponse
				if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// Verify response fields
				if response.MediaFile == nil {
					t.Fatal("Expected media file in response")
				}
				if response.MediaFile.Id == "" {
					t.Error("Expected media file ID")
				}
				if response.MediaFile.EntityType != tc.fields["entity_type"] {
					t.Errorf("Expected entity_type %s, got %s", tc.fields["entity_type"], response.MediaFile.EntityType)
				}
				if response.MediaFile.FileType != pb.MediaFileType_MEDIA_FILE_TYPE_IMAGE {
					t.Errorf("Expected file type IMAGE, got %v", response.MediaFile.FileType)
				}
				if response.MediaFile.Url == "" {
					t.Error("Expected URL to be set")
				}
				if response.MediaFile.ThumbnailUrl == "" {
					t.Error("Expected thumbnail URL to be set")
				}

				// Verify database record
				var mediaFile models.MediaFile
				if err := db.First(&mediaFile, "id = ?", response.MediaFile.Id).Error; err != nil {
					t.Errorf("Media file not found in database: %v", err)
				}
			}
		})
	}

	t.Log("✅ All test cases passed")
}

// TestMediaHandler_Get tests the GET /api/v1/media/{id} endpoint
func TestMediaHandler_Get(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "media_files", "products", "product_pricings", "product_inventories", "product_templates")

	// Create temporary storage directory
	storageDir := t.TempDir()

	// Initialize services and handlers
	localStorage, _ := services.NewLocalStorage(storageDir, "http://localhost:8080/media")
	imageProcessor := services.NewImageProcessor(300, 300)
	videoProcessor := services.NewVideoProcessor()
	mediaService := services.NewMediaService(db, localStorage, imageProcessor, videoProcessor, storageDir)
	handler := handlers.NewMediaHandler(mediaService)

	// Create test fixtures
	template := testutil.CreateTemplateFixture(db, "Electronics", `[{"name":"ProductImage","type":"image","required":true}]`)
	product := testutil.CreateProductFixture(db, template.ID, "Laptop", "LAPTOP-001", `{}`)

	// Create media file record
	mediaFile := &models.MediaFile{
		EntityType:    "product",
		EntityID:      product.ID,
		AttributeName: "ProductImage",
		FileType:      "image",
		MimeType:      "image/jpeg",
		FileName:      "test.jpg",
		FilePath:      "images/originals/test.jpg",
		FileSize:      1024,
		Width:         100,
		Height:        100,
		DisplayOrder:  0,
	}
	db.Create(mediaFile)

	// Table-driven test cases
	testCases := []struct {
		name           string
		mediaID        string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "successful_get",
			mediaID:        mediaFile.ID,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "non_existent_media",
			mediaID:        "550e8400-e29b-41d4-a716-446655440000",
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name:           "empty_id",
			mediaID:        "",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create HTTP request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/media/"+tc.mediaID, nil)
			rec := httptest.NewRecorder()

			// Execute handler
			handler.Get(rec, req)

			// Assert response status
			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}

			if !tc.expectError {
				// Parse success response
				var response pb.GetMediaResponse
				if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// Verify response
				if response.MediaFile == nil {
					t.Fatal("Expected media file in response")
				}
				if response.MediaFile.Id != tc.mediaID {
					t.Errorf("Expected ID %s, got %s", tc.mediaID, response.MediaFile.Id)
				}
			}
		})
	}

	t.Log("✅ All test cases passed")
}

// TestMediaHandler_List tests the GET /api/v1/media endpoint
func TestMediaHandler_List(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "media_files", "products", "product_pricings", "product_inventories", "product_templates")

	// Create temporary storage directory
	storageDir := t.TempDir()

	// Initialize services and handlers
	localStorage, _ := services.NewLocalStorage(storageDir, "http://localhost:8080/media")
	imageProcessor := services.NewImageProcessor(300, 300)
	videoProcessor := services.NewVideoProcessor()
	mediaService := services.NewMediaService(db, localStorage, imageProcessor, videoProcessor, storageDir)
	handler := handlers.NewMediaHandler(mediaService)

	// Create test fixtures
	template := testutil.CreateTemplateFixture(db, "Electronics", `[
		{"name":"ProductImage","type":"image","required":true},
		{"name":"ProductVideo","type":"video","required":false}
	]`)
	product := testutil.CreateProductFixture(db, template.ID, "Laptop", "LAPTOP-001", `{}`)

	// Create multiple media files
	mediaFiles := []*models.MediaFile{
		{
			EntityType:    "product",
			EntityID:      product.ID,
			AttributeName: "ProductImage",
			FileType:      "image",
			MimeType:      "image/jpeg",
			FileName:      "image1.jpg",
			FilePath:      "images/originals/image1.jpg",
			FileSize:      1024,
			DisplayOrder:  0,
		},
		{
			EntityType:    "product",
			EntityID:      product.ID,
			AttributeName: "ProductImage",
			FileType:      "image",
			MimeType:      "image/png",
			FileName:      "image2.png",
			FilePath:      "images/originals/image2.png",
			FileSize:      2048,
			DisplayOrder:  1,
		},
		{
			EntityType:    "product",
			EntityID:      product.ID,
			AttributeName: "ProductVideo",
			FileType:      "video",
			MimeType:      "video/mp4",
			FileName:      "video1.mp4",
			FilePath:      "videos/originals/video1.mp4",
			FileSize:      10240,
			DisplayOrder:  0,
		},
	}
	for _, mf := range mediaFiles {
		db.Create(mf)
	}

	// Table-driven test cases
	testCases := []struct {
		name           string
		queryParams    map[string]string
		expectedStatus int
		expectedCount  int
		expectError    bool
	}{
		{
			name: "list_all_media_for_product",
			queryParams: map[string]string{
				"entity_type": "product",
				"entity_id":   product.ID,
			},
			expectedStatus: http.StatusOK,
			expectedCount:  3,
			expectError:    false,
		},
		{
			name: "filter_by_attribute",
			queryParams: map[string]string{
				"entity_type":    "product",
				"entity_id":      product.ID,
				"attribute_name": "ProductImage",
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
			expectError:    false,
		},
		{
			name: "filter_by_file_type",
			queryParams: map[string]string{
				"entity_type": "product",
				"entity_id":   product.ID,
				"file_type":   "video",
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
			expectError:    false,
		},
		{
			name: "missing_entity_type",
			queryParams: map[string]string{
				"entity_id": product.ID,
			},
			expectedStatus: http.StatusBadRequest,
			expectedCount:  0,
			expectError:    true,
		},
		{
			name: "empty_results",
			queryParams: map[string]string{
				"entity_type": "product",
				"entity_id":   "550e8400-e29b-41d4-a716-446655440000",
			},
			expectedStatus: http.StatusOK,
			expectedCount:  0,
			expectError:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Build query string
			query := ""
			for key, val := range tc.queryParams {
				if query != "" {
					query += "&"
				}
				query += fmt.Sprintf("%s=%s", key, val)
			}

			// Create HTTP request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/media?"+query, nil)
			rec := httptest.NewRecorder()

			// Execute handler
			handler.List(rec, req)

			// Assert response status
			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}

			if !tc.expectError {
				// Parse success response
				var response pb.ListMediaResponse
				if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// Verify count
				if len(response.MediaFiles) != tc.expectedCount {
					t.Errorf("Expected %d media files, got %d", tc.expectedCount, len(response.MediaFiles))
				}

				// Verify display order
				for i := 0; i < len(response.MediaFiles)-1; i++ {
					if response.MediaFiles[i].DisplayOrder > response.MediaFiles[i+1].DisplayOrder {
						t.Error("Media files not sorted by display_order")
					}
				}
			}
		})
	}

	t.Log("✅ All test cases passed")
}

// TestMediaHandler_Update tests the PUT /api/v1/media/{id} endpoint
func TestMediaHandler_Update(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "media_files", "products", "product_pricings", "product_inventories", "product_templates")

	// Create temporary storage directory
	storageDir := t.TempDir()

	// Initialize services and handlers
	localStorage, _ := services.NewLocalStorage(storageDir, "http://localhost:8080/media")
	imageProcessor := services.NewImageProcessor(300, 300)
	videoProcessor := services.NewVideoProcessor()
	mediaService := services.NewMediaService(db, localStorage, imageProcessor, videoProcessor, storageDir)
	handler := handlers.NewMediaHandler(mediaService)

	// Create test fixtures
	template := testutil.CreateTemplateFixture(db, "Electronics", `[{"name":"ProductImage","type":"image","required":true}]`)
	product := testutil.CreateProductFixture(db, template.ID, "Laptop", "LAPTOP-001", `{}`)

	// Create media file
	mediaFile := &models.MediaFile{
		EntityType:    "product",
		EntityID:      product.ID,
		AttributeName: "ProductImage",
		FileType:      "image",
		MimeType:      "image/jpeg",
		FileName:      "original.jpg",
		FilePath:      "images/originals/test.jpg",
		FileSize:      1024,
		DisplayOrder:  0,
	}
	db.Create(mediaFile)

	// Table-driven test cases
	testCases := []struct {
		name           string
		mediaID        string
		request        *pb.UpdateMediaRequest
		expectedStatus int
		expectError    bool
	}{
		{
			name:    "update_display_order",
			mediaID: mediaFile.ID,
			request: &pb.UpdateMediaRequest{
				DisplayOrder: 5,
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:    "update_file_name",
			mediaID: mediaFile.ID,
			request: &pb.UpdateMediaRequest{
				FileName: "renamed.jpg",
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:    "non_existent_media",
			mediaID: "550e8400-e29b-41d4-a716-446655440000",
			request: &pb.UpdateMediaRequest{
				DisplayOrder: 1,
			},
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name:    "invalid_display_order",
			mediaID: mediaFile.ID,
			request: &pb.UpdateMediaRequest{
				DisplayOrder: -1,
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Serialize request
			reqBody, err := json.Marshal(tc.request)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			// Create HTTP request
			req := httptest.NewRequest(http.MethodPut, "/api/v1/media/"+tc.mediaID, bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			// Execute handler
			handler.Update(rec, req)

			// Assert response status
			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}

			if !tc.expectError {
				// Parse success response
				var response pb.UpdateMediaResponse
				if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// Verify updates
				if response.MediaFile == nil {
					t.Fatal("Expected media file in response")
				}
				if tc.request.DisplayOrder != 0 && response.MediaFile.DisplayOrder != tc.request.DisplayOrder {
					t.Errorf("Expected display_order %d, got %d", tc.request.DisplayOrder, response.MediaFile.DisplayOrder)
				}
				if tc.request.FileName != "" && response.MediaFile.FileName != tc.request.FileName {
					t.Errorf("Expected file_name %s, got %s", tc.request.FileName, response.MediaFile.FileName)
				}
			}
		})
	}

	t.Log("✅ All test cases passed")
}

// TestMediaHandler_Delete tests the DELETE /api/v1/media/{id} endpoint
func TestMediaHandler_Delete(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "media_files", "products", "product_pricings", "product_inventories", "product_templates")

	// Create temporary storage directory
	storageDir := t.TempDir()

	// Initialize services and handlers
	localStorage, _ := services.NewLocalStorage(storageDir, "http://localhost:8080/media")
	imageProcessor := services.NewImageProcessor(300, 300)
	videoProcessor := services.NewVideoProcessor()
	mediaService := services.NewMediaService(db, localStorage, imageProcessor, videoProcessor, storageDir)
	handler := handlers.NewMediaHandler(mediaService)

	// Create test fixtures
	template := testutil.CreateTemplateFixture(db, "Electronics", `[{"name":"ProductImage","type":"image","required":true}]`)
	product := testutil.CreateProductFixture(db, template.ID, "Laptop", "LAPTOP-001", `{}`)

	// Create media file
	mediaFile := &models.MediaFile{
		EntityType:    "product",
		EntityID:      product.ID,
		AttributeName: "ProductImage",
		FileType:      "image",
		MimeType:      "image/jpeg",
		FileName:      "test.jpg",
		FilePath:      "images/originals/test.jpg",
		FileSize:      1024,
		DisplayOrder:  0,
	}
	db.Create(mediaFile)

	// Table-driven test cases
	testCases := []struct {
		name           string
		mediaID        string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "successful_deletion",
			mediaID:        mediaFile.ID,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "non_existent_media",
			mediaID:        "550e8400-e29b-41d4-a716-446655440000",
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name:           "empty_id",
			mediaID:        "",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create HTTP request
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/media/"+tc.mediaID, nil)
			rec := httptest.NewRecorder()

			// Execute handler
			handler.Delete(rec, req)

			// Assert response status
			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}

			if !tc.expectError {
				// Parse success response
				var response pb.DeleteMediaResponse
				if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// Build expected response
				expectedResponse := &pb.DeleteMediaResponse{
					Success: true,
				}

				// Compare using protocmp
				if diff := cmp.Diff(expectedResponse, &response, protocmp.Transform()); diff != "" {
					t.Errorf("Response mismatch (-want +got):\n%s", diff)
				}

				// Verify deletion in database
				var deletedMedia models.MediaFile
				err := db.First(&deletedMedia, "id = ?", tc.mediaID).Error
				if err == nil {
					t.Error("Expected media file to be deleted from database")
				}
			}
		})
	}

	t.Log("✅ All test cases passed")
}

// TestMediaHandler_Reorder tests the POST /api/v1/media/reorder endpoint
func TestMediaHandler_Reorder(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "media_files", "products", "product_pricings", "product_inventories", "product_templates")

	// Create temporary storage directory
	storageDir := t.TempDir()

	// Initialize services and handlers
	localStorage, _ := services.NewLocalStorage(storageDir, "http://localhost:8080/media")
	imageProcessor := services.NewImageProcessor(300, 300)
	videoProcessor := services.NewVideoProcessor()
	mediaService := services.NewMediaService(db, localStorage, imageProcessor, videoProcessor, storageDir)
	handler := handlers.NewMediaHandler(mediaService)

	// Create test fixtures
	template := testutil.CreateTemplateFixture(db, "Electronics", `[{"name":"ProductImage","type":"image","required":true}]`)
	product := testutil.CreateProductFixture(db, template.ID, "Laptop", "LAPTOP-001", `{}`)

	// Create multiple media files
	media1 := &models.MediaFile{
		EntityType:    "product",
		EntityID:      product.ID,
		AttributeName: "ProductImage",
		FileType:      "image",
		MimeType:      "image/jpeg",
		FileName:      "image1.jpg",
		FilePath:      "images/originals/image1.jpg",
		FileSize:      1024,
		DisplayOrder:  0,
	}
	media2 := &models.MediaFile{
		EntityType:    "product",
		EntityID:      product.ID,
		AttributeName: "ProductImage",
		FileType:      "image",
		MimeType:      "image/jpeg",
		FileName:      "image2.jpg",
		FilePath:      "images/originals/image2.jpg",
		FileSize:      1024,
		DisplayOrder:  1,
	}
	media3 := &models.MediaFile{
		EntityType:    "product",
		EntityID:      product.ID,
		AttributeName: "ProductImage",
		FileType:      "image",
		MimeType:      "image/jpeg",
		FileName:      "image3.jpg",
		FilePath:      "images/originals/image3.jpg",
		FileSize:      1024,
		DisplayOrder:  2,
	}
	db.Create(media1)
	db.Create(media2)
	db.Create(media3)

	// Table-driven test cases
	testCases := []struct {
		name           string
		request        *pb.ReorderMediaRequest
		expectedStatus int
		expectError    bool
		expectedOrder  []string
	}{
		{
			name: "successful_reorder",
			request: &pb.ReorderMediaRequest{
				EntityType:    "product",
				EntityId:      product.ID,
				AttributeName: "ProductImage",
				MediaIds:      []string{media3.ID, media1.ID, media2.ID},
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
			expectedOrder:  []string{media3.ID, media1.ID, media2.ID},
		},
		{
			name: "empty_media_ids",
			request: &pb.ReorderMediaRequest{
				EntityType:    "product",
				EntityId:      product.ID,
				AttributeName: "ProductImage",
				MediaIds:      []string{},
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "duplicate_ids",
			request: &pb.ReorderMediaRequest{
				EntityType:    "product",
				EntityId:      product.ID,
				AttributeName: "ProductImage",
				MediaIds:      []string{media1.ID, media1.ID, media2.ID},
			},
			expectedStatus: http.StatusConflict, // Duplicate ID returns 409
			expectError:    true,
		},
		{
			name: "invalid_media_id",
			request: &pb.ReorderMediaRequest{
				EntityType:    "product",
				EntityId:      product.ID,
				AttributeName: "ProductImage",
				MediaIds:      []string{media1.ID, "550e8400-e29b-41d4-a716-446655440000"},
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Serialize request
			reqBody, err := json.Marshal(tc.request)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			// Create HTTP request
			req := httptest.NewRequest(http.MethodPost, "/api/v1/media/reorder", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			// Execute handler
			handler.Reorder(rec, req)

			// Assert response status
			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}

			if !tc.expectError {
				// Parse success response
				var response pb.ReorderMediaResponse
				if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// Verify order
				if len(response.MediaFiles) != len(tc.expectedOrder) {
					t.Errorf("Expected %d media files, got %d", len(tc.expectedOrder), len(response.MediaFiles))
				}

				for i, expectedID := range tc.expectedOrder {
					if response.MediaFiles[i].Id != expectedID {
						t.Errorf("Expected ID %s at position %d, got %s", expectedID, i, response.MediaFiles[i].Id)
					}
					if response.MediaFiles[i].DisplayOrder != int32(i) {
						t.Errorf("Expected display_order %d, got %d", i, response.MediaFiles[i].DisplayOrder)
					}
				}
			}
		})
	}

	t.Log("✅ All test cases passed")
}

// TestMediaHandler_BulkUpload tests the POST /api/v1/media/upload/bulk endpoint
func TestMediaHandler_BulkUpload(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "media_files", "products", "product_pricings", "product_inventories", "product_templates")

	// Create temporary storage directory
	storageDir := t.TempDir()

	// Initialize services and handlers
	localStorage, _ := services.NewLocalStorage(storageDir, "http://localhost:8080/media")
	imageProcessor := services.NewImageProcessor(300, 300)
	videoProcessor := services.NewVideoProcessor()
	mediaService := services.NewMediaService(db, localStorage, imageProcessor, videoProcessor, storageDir)
	handler := handlers.NewMediaHandler(mediaService)

	// Create test fixtures
	template := testutil.CreateTemplateFixture(db, "Electronics", `[{"name":"ProductImage","type":"image","required":true}]`)
	product := testutil.CreateProductFixture(db, template.ID, "Laptop", "LAPTOP-001", `{}`)

	// Create test images
	imagePath1, cleanup1 := createTestImageFile(t)
	defer cleanup1()
	imagePath2, cleanup2 := createTestImageFile(t)
	defer cleanup2()

	t.Run("successful_bulk_upload", func(t *testing.T) {
		// Create multipart form with multiple files
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Add form fields
		writer.WriteField("entity_type", "product")
		writer.WriteField("entity_id", product.ID)
		writer.WriteField("attribute_name", "ProductImage")

		// Add first file
		file1, _ := os.Open(imagePath1)
		defer file1.Close()
		part1, _ := writer.CreateFormFile("files", "image1.jpg")
		io.Copy(part1, file1)

		// Add second file
		file2, _ := os.Open(imagePath2)
		defer file2.Close()
		part2, _ := writer.CreateFormFile("files", "image2.jpg")
		io.Copy(part2, file2)

		contentType := writer.FormDataContentType()
		writer.Close()

		// Create HTTP request
		req := httptest.NewRequest(http.MethodPost, "/api/v1/media/upload/bulk", body)
		req.Header.Set("Content-Type", contentType)
		rec := httptest.NewRecorder()

		// Execute handler
		handler.UploadBulk(rec, req)

		// Assert response status
		if rec.Code != http.StatusCreated {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusCreated, rec.Code, rec.Body.String())
		}

		// Parse response
		var response pb.BulkUploadMediaResponse
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Verify results
		if len(response.MediaFiles) != 2 {
			t.Errorf("Expected 2 media files uploaded, got %d", len(response.MediaFiles))
		}
		if len(response.Errors) != 0 {
			t.Errorf("Expected 0 errors, got %d", len(response.Errors))
		}
	})

	t.Log("✅ All test cases passed")
}
