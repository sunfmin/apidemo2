package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	pb "github.com/sunfmin/apidemo2/backend/api/gen/pim/v1"
	"github.com/sunfmin/apidemo2/backend/internal/handlers"
	"github.com/sunfmin/apidemo2/backend/internal/models"
	"github.com/sunfmin/apidemo2/backend/internal/services"
	"github.com/sunfmin/apidemo2/backend/tests/testutil"
)

// TestProductHandler_Create tests the POST /api/v1/products endpoint
func TestProductHandler_Create(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "products", "product_templates")

	// Initialize services and handlers
	productService := services.NewProductService(db)
	handler := handlers.NewProductHandler(productService)

	// Table-driven test cases
	testCases := []struct {
		name           string
		request        *pb.CreateProductRequest
		expectedStatus int
		expectError    bool
	}{
		{
			name: "successful_creation_all_attribute_types",
			request: &pb.CreateProductRequest{
				TemplateId:       "", // Will be set in test loop
				Name:             "Laptop Pro",
				Sku:              "LAPTOP-001",
				Description:      "High-performance laptop",
				InitialListPrice: 1299.99,
				InitialSalePrice: 0,
				InitialStock:     100,
				AttributeValues: map[string]*pb.AttributeValue{
					"Product Name": {Type: pb.AttributeType_ATTRIBUTE_TYPE_TEXT, TextValue: "Laptop Pro"},
					"Color":        {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"Silver"}},
					"Specs":        {Type: pb.AttributeType_ATTRIBUTE_TYPE_MAP, MapValue: map[string]string{"CPU": "Intel i7", "RAM": "16GB"}},
				},
				Status: pb.ProductStatus_PRODUCT_STATUS_DRAFT,
			},
			expectedStatus: http.StatusCreated,
			expectError:    false,
		},
		{
			name: "empty_product_name",
			request: &pb.CreateProductRequest{
				TemplateId:   "", // Will be set in test loop
				Name:         "",
				Sku:          "TEST-001",
				InitialListPrice: 99.99,
				InitialStock:     50,
				AttributeValues: map[string]*pb.AttributeValue{
					"Product Name": {Type: pb.AttributeType_ATTRIBUTE_TYPE_TEXT, TextValue: "Test"},
					"Color":        {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"Black"}},
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "duplicate_sku",
			request: &pb.CreateProductRequest{
				TemplateId:   "", // Will be set in test loop
				Name:         "Duplicate Product",
				Sku:          "DUPLICATE-SKU",
				InitialListPrice: 99.99,
				InitialStock:     50,
				AttributeValues: map[string]*pb.AttributeValue{
					"Product Name": {Type: pb.AttributeType_ATTRIBUTE_TYPE_TEXT, TextValue: "Test"},
					"Color":        {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"Black"}},
				},
			},
			expectedStatus: http.StatusConflict,
			expectError:    true,
		},
		{
			name: "invalid_template_reference",
			request: &pb.CreateProductRequest{
				TemplateId: "550e8400-e29b-41d4-a716-446655440000",
				Name:       "Test Product",
				Sku:        "TEST-002",
				AttributeValues: map[string]*pb.AttributeValue{},
			},
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name: "missing_required_attributes",
			request: &pb.CreateProductRequest{
				TemplateId:   "", // Will be set in test loop
				Name:         "Incomplete Product",
				Sku:          "TEST-003",
				InitialListPrice: 99.99,
				InitialStock: 10,
				AttributeValues: map[string]*pb.AttributeValue{
					"Product Name": {Type: pb.AttributeType_ATTRIBUTE_TYPE_TEXT, TextValue: "Test"},
					// Missing "Color" (required from template)
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "list_value_not_in_options",
			request: &pb.CreateProductRequest{
				TemplateId:   "", // Will be set in test loop
				Name:         "Invalid Color Product",
				Sku:          "TEST-004",
				InitialListPrice: 99.99,
				InitialStock: 25,
				AttributeValues: map[string]*pb.AttributeValue{
					"Product Name": {Type: pb.AttributeType_ATTRIBUTE_TYPE_TEXT, TextValue: "Test"},
					"Color":        {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"Red"}}, // Red not in options
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "sku_too_long",
			request: &pb.CreateProductRequest{
				TemplateId:   "", // Will be set in test loop
				Name:         "Test Product",
				Sku:          "THIS-IS-A-VERY-LONG-SKU-THAT-EXCEEDS-THE-MAXIMUM-LENGTH-OF-100-CHARACTERS-AND-SHOULD-BE-REJECTED-BY-VALIDATION",
				InitialListPrice: 99.99,
				InitialStock: 10,
				AttributeValues: map[string]*pb.AttributeValue{
					"Product Name": {Type: pb.AttributeType_ATTRIBUTE_TYPE_TEXT, TextValue: "Test"},
					"Color":        {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"Black"}},
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a fresh template fixture for this test case (unless it's the invalid template test)
			var template *models.ProductTemplate
			if tc.name != "invalid_template_reference" {
				template = testutil.CreateTemplateFixture(db, "Electronics", `[
					{"name":"Product Name","type":"text","required":true},
					{"name":"Color","type":"list","required":true,"options":["Black","White","Silver"]},
					{"name":"Specs","type":"map","required":false}
				]`)

				// Update request with actual template ID
				tc.request.TemplateId = template.ID
			}

			// For duplicate test, create fixture first
			if tc.name == "duplicate_sku" && template != nil {
				testutil.CreateProductFixture(db, template.ID, "Duplicate Product", "DUPLICATE-SKU", `{}`)
			}

			// Create HTTP request
			body, _ := json.Marshal(tc.request)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(body))
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
				var response pb.CreateProductResponse
				if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// Verify product exists
				if response.Product == nil {
					t.Fatal("Expected product in response")
				}

				// Verify generated fields exist (ID and timestamps)
				if response.Product.Id == "" {
					t.Error("Expected product ID to be generated")
				}
				if response.Product.CreatedAt == nil {
					t.Error("Expected CreatedAt to be set")
				}
				if response.Product.UpdatedAt == nil {
					t.Error("Expected UpdatedAt to be set")
				}

				// Build expected attribute values with Type field properly set
				expectedAttributeValues := make(map[string]*pb.AttributeValue)
				for name, value := range tc.request.AttributeValues {
					expectedAttributeValues[name] = &pb.AttributeValue{
						Type:         value.Type,
						TextValue:    value.TextValue,
						NumberValue:  value.NumberValue,
						BooleanValue: value.BooleanValue,
						DateValue:    value.DateValue,
						ListValue:    value.ListValue,
						MapValue:     value.MapValue,
						MediaIds:     value.MediaIds,
					}
				}

				// Build expected response
				expectedResponse := &pb.CreateProductResponse{
					Product: &pb.Product{
						Id:              response.Product.Id,              // Use actual generated ID
						TemplateId:      tc.request.TemplateId,
						TemplateName:    response.Product.TemplateName,    // Use actual template name
						Name:            tc.request.Name,
						Sku:             tc.request.Sku,
						Description:     tc.request.Description,
						ListPrice:       tc.request.InitialListPrice,
						SalePrice:       tc.request.InitialSalePrice,
						EffectivePrice:  tc.request.InitialListPrice, // No sale price set
						Currency:        "USD",
						TotalStock:      tc.request.InitialStock,
						AvailableStock:  tc.request.InitialStock, // Initially no reserved stock
						ReservedStock:   0,
						AttributeValues: expectedAttributeValues, // Expected attributes with Type field
						Status:          tc.request.Status,
						CreatedAt:       response.Product.CreatedAt,       // Use actual timestamp
						UpdatedAt:       response.Product.UpdatedAt,       // Use actual timestamp
						VariantCount:    0,                                // No variants yet
						PrimaryImageUrls: []string{},                      // No images yet
					},
				}

				// Compare entire response using protocmp (MANDATORY per constitution)
				if diff := cmp.Diff(expectedResponse, &response, protocmp.Transform()); diff != "" {
					t.Errorf("Response mismatch (-want +got):\n%s", diff)
				}
			}

			// Cleanup for next test
			testutil.TruncateTables(db, "products", "product_templates")
		})
	}

	t.Log("✅ All test cases passed")
}

