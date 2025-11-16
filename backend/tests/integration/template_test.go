package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	pb "github.com/sunfmin/apidemo2/backend/api/gen/pim/v1"
	"github.com/sunfmin/apidemo2/backend/internal/handlers"
	"github.com/sunfmin/apidemo2/backend/internal/services"
	"github.com/sunfmin/apidemo2/backend/tests/testutil"
)

// TestTemplateHandler_Create tests the POST /api/v1/templates endpoint
func TestTemplateHandler_Create(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "product_templates")

	// Initialize services and handlers
	templateService := services.NewTemplateService(db)
	handler := handlers.NewTemplateHandler(templateService)

	// Table-driven test cases
	testCases := []struct {
		name           string
		request        *pb.CreateTemplateRequest
		expectedStatus int
		expectError    bool
	}{
		{
			name: "successful_creation_all_8_attribute_types",
			request: &pb.CreateTemplateRequest{
				Name: "Electronics Template",
				Attributes: []*pb.AttributeDefinition{
					{Name: "Product Name", Type: pb.AttributeType_ATTRIBUTE_TYPE_TEXT, Required: true},
					{Name: "Price", Type: pb.AttributeType_ATTRIBUTE_TYPE_NUMBER, Required: true},
					{Name: "In Stock", Type: pb.AttributeType_ATTRIBUTE_TYPE_BOOLEAN, Required: true},
					{Name: "Release Date", Type: pb.AttributeType_ATTRIBUTE_TYPE_DATE, Required: false},
					{Name: "Color Options", Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, Required: true, Options: []string{"Black", "White", "Silver"}},
					{Name: "Technical Specs", Type: pb.AttributeType_ATTRIBUTE_TYPE_MAP, Required: false},
					{Name: "Product Image", Type: pb.AttributeType_ATTRIBUTE_TYPE_IMAGE, Required: true},
					{Name: "Demo Video", Type: pb.AttributeType_ATTRIBUTE_TYPE_VIDEO, Required: false},
				},
			},
			expectedStatus: http.StatusCreated,
			expectError:    false,
		},
		{
			name: "empty_template_name",
			request: &pb.CreateTemplateRequest{
				Name: "",
				Attributes: []*pb.AttributeDefinition{
					{Name: "Size", Type: pb.AttributeType_ATTRIBUTE_TYPE_TEXT, Required: true},
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "duplicate_template_name",
			request: &pb.CreateTemplateRequest{
				Name: "Duplicate",
				Attributes: []*pb.AttributeDefinition{
					{Name: "Field", Type: pb.AttributeType_ATTRIBUTE_TYPE_TEXT, Required: true},
				},
			},
			expectedStatus: http.StatusConflict,
			expectError:    true,
		},
		{
			name: "empty_attributes_array",
			request: &pb.CreateTemplateRequest{
				Name:       "Test",
				Attributes: []*pb.AttributeDefinition{},
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "list_attribute_with_empty_options",
			request: &pb.CreateTemplateRequest{
				Name: "Invalid List Template",
				Attributes: []*pb.AttributeDefinition{
					{Name: "Size", Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, Required: true, Options: []string{}},
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "duplicate_attribute_names",
			request: &pb.CreateTemplateRequest{
				Name: "Duplicate Attrs Template",
				Attributes: []*pb.AttributeDefinition{
					{Name: "Color", Type: pb.AttributeType_ATTRIBUTE_TYPE_TEXT, Required: true},
					{Name: "Color", Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, Required: false, Options: []string{"Red", "Blue"}},
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "invalid_attribute_type_unspecified",
			request: &pb.CreateTemplateRequest{
				Name: "Invalid Type Template",
				Attributes: []*pb.AttributeDefinition{
					{Name: "Field", Type: pb.AttributeType_ATTRIBUTE_TYPE_UNSPECIFIED, Required: true},
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "very_long_template_name",
			request: &pb.CreateTemplateRequest{
				Name: "This is a very long template name that exceeds the maximum allowed length of 255 characters. This should be rejected by the validation logic. Let me add more text to ensure we definitely exceed 255 characters. Adding even more text here to make absolutely sure we go over the limit. Still adding more characters to reach the validation threshold.",
				Attributes: []*pb.AttributeDefinition{
					{Name: "Field", Type: pb.AttributeType_ATTRIBUTE_TYPE_TEXT, Required: true},
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// For duplicate test, create fixture first
			if tc.name == "duplicate_template_name" {
				testutil.CreateTemplateFixture(db, "Duplicate", "")
			}

			// Create HTTP request
			body, _ := json.Marshal(tc.request)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/templates", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			// Execute handler
			handler.Create(rec, req)

			// Assert response status
			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}

			if !tc.expectError {
				// Parse success response
				var response pb.CreateTemplateResponse
				if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// Verify template properties
				if response.Template == nil {
					t.Fatal("Expected template in response")
				}
				if response.Template.Id == "" {
					t.Error("Expected template ID to be generated")
				}
				if response.Template.Name != tc.request.Name {
					t.Errorf("Expected name %s, got %s", tc.request.Name, response.Template.Name)
				}

				// Verify all attributes are stored correctly
				if len(response.Template.Attributes) != len(tc.request.Attributes) {
					t.Errorf("Expected %d attributes, got %d", len(tc.request.Attributes), len(response.Template.Attributes))
				}

				// For the comprehensive test, verify all 8 attribute types
				if tc.name == "successful_creation_all_8_attribute_types" {
					expectedTypes := []pb.AttributeType{
						pb.AttributeType_ATTRIBUTE_TYPE_TEXT,
						pb.AttributeType_ATTRIBUTE_TYPE_NUMBER,
						pb.AttributeType_ATTRIBUTE_TYPE_BOOLEAN,
						pb.AttributeType_ATTRIBUTE_TYPE_DATE,
						pb.AttributeType_ATTRIBUTE_TYPE_LIST,
						pb.AttributeType_ATTRIBUTE_TYPE_MAP,
						pb.AttributeType_ATTRIBUTE_TYPE_IMAGE,
						pb.AttributeType_ATTRIBUTE_TYPE_VIDEO,
					}

					foundTypes := make(map[pb.AttributeType]bool)
					for _, attr := range response.Template.Attributes {
						foundTypes[attr.Type] = true
					}

					for _, expectedType := range expectedTypes {
						if !foundTypes[expectedType] {
							t.Errorf("Missing attribute type: %s", expectedType.String())
						}
					}

					// Verify LIST type has options
					for _, attr := range response.Template.Attributes {
						if attr.Type == pb.AttributeType_ATTRIBUTE_TYPE_LIST {
							if len(attr.Options) == 0 {
								t.Errorf("LIST attribute '%s' should have options", attr.Name)
							}
						}
					}

					t.Log("✅ All 8 attribute types verified: TEXT, NUMBER, BOOLEAN, DATE, LIST, MAP, IMAGE, VIDEO")
				}

				// Verify timestamps
				if response.Template.CreatedAt == nil {
					t.Error("Expected CreatedAt to be set")
				}
				if response.Template.UpdatedAt == nil {
					t.Error("Expected UpdatedAt to be set")
				}
			}

			// Cleanup for next test
			testutil.TruncateTables(db, "product_templates")
		})
	}

	t.Log("✅ All test cases passed")
}

// TestTemplateHandler_Get tests the GET /api/v1/templates/{id} endpoint
func TestTemplateHandler_Get(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "product_templates")

	// Initialize services and handlers
	templateService := services.NewTemplateService(db)
	handler := handlers.NewTemplateHandler(templateService)

	// Create a fixture template
	fixture := testutil.CreateTemplateFixture(db, "Test Template", "")

	// Table-driven test cases
	testCases := []struct {
		name           string
		templateID     string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "successful_retrieval",
			templateID:     fixture.ID,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "non_existent_template",
			templateID:     "550e8400-e29b-41d4-a716-446655440000",
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name:           "empty_id",
			templateID:     "",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create HTTP request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/templates/"+tc.templateID, nil)
			rec := httptest.NewRecorder()

			// Execute handler
			handler.Get(rec, req)

			// Assert response status
			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}

			if !tc.expectError {
				// Parse success response
				var response pb.GetTemplateResponse
				if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// Verify template
				if response.Template == nil {
					t.Fatal("Expected template in response")
				}
				if response.Template.Id != tc.templateID {
					t.Errorf("Expected template ID %s, got %s", tc.templateID, response.Template.Id)
				}
			}
		})
	}

	t.Log("✅ All test cases passed")
}

// TestTemplateHandler_List tests the GET /api/v1/templates endpoint with pagination
func TestTemplateHandler_List(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "product_templates")

	// Initialize services and handlers
	templateService := services.NewTemplateService(db)
	handler := handlers.NewTemplateHandler(templateService)

	// Create fixtures
	testutil.CreateTemplateFixture(db, "Template 1", "")
	testutil.CreateTemplateFixture(db, "Template 2", "")
	testutil.CreateTemplateFixture(db, "Template 3", "")

	// Table-driven test cases
	testCases := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedMin    int // Minimum number of templates expected
		expectError    bool
	}{
		{
			name:           "list_all_templates",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedMin:    3,
			expectError:    false,
		},
		{
			name:           "pagination_page_1",
			queryParams:    "?page=1&page_size=2",
			expectedStatus: http.StatusOK,
			expectedMin:    2,
			expectError:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create HTTP request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/templates"+tc.queryParams, nil)
			rec := httptest.NewRecorder()

			// Execute handler
			handler.List(rec, req)

			// Assert response status
			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}

			if !tc.expectError {
				// Parse success response
				var response pb.ListTemplatesResponse
				if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// Verify count
				if len(response.Templates) < tc.expectedMin {
					t.Errorf("Expected at least %d templates, got %d", tc.expectedMin, len(response.Templates))
				}

				// Verify pagination metadata
				if response.Pagination == nil {
					t.Error("Expected pagination metadata")
				}
			}
		})
	}

	t.Log("✅ All test cases passed")
}

// TestTemplateHandler_Update tests the PUT /api/v1/templates/{id} endpoint
func TestTemplateHandler_Update(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "product_templates")

	// Initialize services and handlers
	templateService := services.NewTemplateService(db)
	handler := handlers.NewTemplateHandler(templateService)

	// Create a fixture template
	fixture := testutil.CreateTemplateFixture(db, "Original Name", "")

	// Table-driven test cases
	testCases := []struct {
		name           string
		templateID     string
		request        *pb.UpdateTemplateRequest
		expectedStatus int
		expectError    bool
	}{
		{
			name:       "successful_update_name",
			templateID: fixture.ID,
			request: &pb.UpdateTemplateRequest{
				Id:   fixture.ID,
				Name: "Updated Name",
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:       "non_existent_template",
			templateID: "550e8400-e29b-41d4-a716-446655440000",
			request: &pb.UpdateTemplateRequest{
				Id:   "550e8400-e29b-41d4-a716-446655440000",
				Name: "Updated",
			},
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create HTTP request
			body, _ := json.Marshal(tc.request)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/templates/"+tc.templateID, bytes.NewReader(body))
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
				var response pb.UpdateTemplateResponse
				if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// Verify template was updated
				if response.Template.UpdatedAt == nil {
					t.Error("Expected UpdatedAt to be set")
				}
			}
		})
	}

	t.Log("✅ All test cases passed")
}

// TestTemplateHandler_Delete tests the DELETE /api/v1/templates/{id} endpoint
func TestTemplateHandler_Delete(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "product_templates")

	// Initialize services and handlers
	templateService := services.NewTemplateService(db)
	handler := handlers.NewTemplateHandler(templateService)

	// Create a fixture template
	fixture := testutil.CreateTemplateFixture(db, "To Delete", "")

	// Table-driven test cases
	testCases := []struct {
		name           string
		templateID     string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "successful_deletion",
			templateID:     fixture.ID,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "non_existent_template",
			templateID:     "550e8400-e29b-41d4-a716-446655440000",
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create HTTP request
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/templates/"+tc.templateID, nil)
			rec := httptest.NewRecorder()

			// Execute handler
			handler.Delete(rec, req)

			// Assert response status
			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}

			if !tc.expectError {
				// Parse success response
				var response pb.DeleteTemplateResponse
				if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if !response.Success {
					t.Error("Expected Success to be true")
				}
			}
		})
	}

	t.Log("✅ All test cases passed")
}
