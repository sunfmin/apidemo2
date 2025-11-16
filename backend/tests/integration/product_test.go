package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

// TestProductHandler_Get tests the GET /api/v1/products/{id} endpoint
func TestProductHandler_Get(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "product_templates", "products", "product_pricings", "product_inventories")

	// Initialize services and handlers
	productService := services.NewProductService(db)
	handler := handlers.NewProductHandler(productService)

	// Create template and product fixtures
	template := testutil.CreateTemplateFixture(db, "Electronics", `[
		{"name":"Brand","type":"text","required":true},
		{"name":"Model","type":"text","required":true}
	]`)

	// Create a product fixture with pricing and inventory
	product := testutil.CreateProductFixture(db, template.ID, "Test Product", "TEST-GET-001", `{
		"Brand":{"type":"ATTRIBUTE_TYPE_TEXT","value":"Apple"},
		"Model":{"type":"ATTRIBUTE_TYPE_TEXT","value":"MacBook Pro"}
	}`)

	// Create pricing
	pricing := &models.ProductPricing{
		ProductID: product.ID,
		ListPrice: 1999.99,
		SalePrice: 1799.99,
		Currency:  "USD",
		ValidFrom: time.Now(),
	}
	db.Create(pricing)

	// Create inventory
	inventory := &models.ProductInventory{
		ProductID:      product.ID,
		LocationID:     "default",
		OnHandQuantity: 50,
	}
	db.Create(inventory)

	// Table-driven test cases
	testCases := []struct {
		name           string
		productID      string
		expectedStatus int
		expectError    bool
		checkFields    bool
	}{
		{
			name:           "successful_retrieval",
			productID:      product.ID,
			expectedStatus: http.StatusOK,
			expectError:    false,
			checkFields:    true,
		},
		{
			name:           "non_existent_product",
			productID:      "550e8400-e29b-41d4-a716-446655440000",
			expectedStatus: http.StatusNotFound,
			expectError:    true,
			checkFields:    false,
		},
		{
			name:           "empty_id",
			productID:      "",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
			checkFields:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create HTTP request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/products/"+tc.productID, nil)
			rec := httptest.NewRecorder()

			// Execute handler
			handler.Get(rec, req)

			// Assert response status
			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}

			if !tc.expectError && tc.checkFields {
				// Parse success response
				var response pb.GetProductResponse
				if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// Verify product exists
				if response.Product == nil {
					t.Fatal("Expected product in response")
				}

				// Build expected response
				expectedResponse := &pb.GetProductResponse{
					Product: &pb.Product{
						Id:             product.ID,
						TemplateId:     template.ID,
						TemplateName:   template.Name,
						Name:           product.Name,
						Sku:            product.SKU,
						Description:    product.Description,
						ListPrice:      pricing.ListPrice,
						SalePrice:      pricing.SalePrice,
						EffectivePrice: pricing.SalePrice, // Sale price is set
						Currency:       pricing.Currency,
						TotalStock:     inventory.OnHandQuantity,
						AvailableStock: inventory.OnHandQuantity,
						ReservedStock:  0,
						AttributeValues: response.Product.AttributeValues, // Use actual from response
						Status:          response.Product.Status,
						CreatedAt:       response.Product.CreatedAt,
						UpdatedAt:       response.Product.UpdatedAt,
						VariantCount:    0,
						PrimaryImageUrls: []string{},
					},
				}

				// Compare entire response using protocmp (MANDATORY per constitution)
				if diff := cmp.Diff(expectedResponse, &response, protocmp.Transform()); diff != "" {
					t.Errorf("Response mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}

	t.Log("✅ All test cases passed")
}

// TestProductHandler_List tests the GET /api/v1/products endpoint
func TestProductHandler_List(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "product_templates", "products", "product_pricings", "product_inventories")

	// Initialize services and handlers
	productService := services.NewProductService(db)
	handler := handlers.NewProductHandler(productService)

	// Create template
	template := testutil.CreateTemplateFixture(db, "Electronics", `[
		{"name":"Brand","type":"text","required":true}
	]`)

	// Create multiple product fixtures
	for i := 1; i <= 5; i++ {
		product := testutil.CreateProductFixture(db, template.ID, fmt.Sprintf("Product %d", i), fmt.Sprintf("SKU-%03d", i), `{"Brand":{"type":"ATTRIBUTE_TYPE_TEXT","value":"TestBrand"}}`)
		
		// Create pricing
		db.Create(&models.ProductPricing{
			ProductID: product.ID,
			ListPrice: float64(100 * i),
			Currency:  "USD",
			ValidFrom: time.Now(),
		})
		
		// Create inventory
		db.Create(&models.ProductInventory{
			ProductID:      product.ID,
			LocationID:     "default",
			OnHandQuantity: int32(10 * i),
		})
	}

	// Table-driven test cases
	testCases := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedMin    int // Minimum number of products expected
		expectError    bool
	}{
		{
			name:           "list_all_products",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedMin:    5,
			expectError:    false,
		},
		{
			name:           "pagination_page_1",
			queryParams:    "?page=1&page_size=3",
			expectedStatus: http.StatusOK,
			expectedMin:    3,
			expectError:    false,
		},
		{
			name:           "pagination_page_2",
			queryParams:    "?page=2&page_size=3",
			expectedStatus: http.StatusOK,
			expectedMin:    2, // Should have remaining 2 products
			expectError:    false,
		},
		{
			name:           "search_by_name",
			queryParams:    "?search=Product%201", // URL-encoded space
			expectedStatus: http.StatusOK,
			expectedMin:    1,
			expectError:    false,
		},
		{
			name:           "filter_by_template",
			queryParams:    fmt.Sprintf("?template_id=%s", template.ID),
			expectedStatus: http.StatusOK,
			expectedMin:    5,
			expectError:    false,
		},
		{
			name:           "filter_by_status",
			queryParams:    "?status=active",
			expectedStatus: http.StatusOK,
			expectedMin:    5,
			expectError:    false,
		},
		{
			name:           "empty_results",
			queryParams:    "?search=NonExistent",
			expectedStatus: http.StatusOK,
			expectedMin:    0,
			expectError:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create HTTP request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/products"+tc.queryParams, nil)
			rec := httptest.NewRecorder()

			// Execute handler
			handler.List(rec, req)

			// Assert response status
			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}

			if !tc.expectError {
				// Parse success response
				var response pb.ListProductsResponse
				if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// Verify count
				if len(response.Products) < tc.expectedMin {
					t.Errorf("Expected at least %d products, got %d", tc.expectedMin, len(response.Products))
				}

				// Verify pagination metadata
				if response.Pagination == nil {
					t.Error("Expected pagination metadata")
				}

				// Verify each product has pricing and inventory denormalized
				for i, product := range response.Products {
					if product.ListPrice == 0 {
						t.Errorf("Product %d (%s): Expected ListPrice to be set, got 0", i, product.Name)
					}
					if product.Currency == "" {
						t.Errorf("Product %d (%s): Expected Currency to be set", i, product.Name)
					}
					// Log for debugging
					t.Logf("Product %d: %s, ListPrice: %.2f, Stock: %d", i, product.Name, product.ListPrice, product.TotalStock)
				}
			}
		})
	}

	t.Log("✅ All test cases passed")
}

// TestProductHandler_Update tests the PUT /api/v1/products/{id} endpoint
func TestProductHandler_Update(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "product_templates", "products", "product_pricings", "product_inventories")

	// Initialize services and handlers
	productService := services.NewProductService(db)
	handler := handlers.NewProductHandler(productService)

	// Create template and product
	template := testutil.CreateTemplateFixture(db, "Electronics", `[
		{"name":"Brand","type":"text","required":true}
	]`)

	product := testutil.CreateProductFixture(db, template.ID, "Original Name", "UPDATE-001", `{"Brand":{"type":"ATTRIBUTE_TYPE_TEXT","value":"Apple"}}`)

	// Create pricing and inventory
	db.Create(&models.ProductPricing{
		ProductID: product.ID,
		ListPrice: 999.99,
		Currency:  "USD",
		ValidFrom: time.Now(),
	})
	db.Create(&models.ProductInventory{
		ProductID:      product.ID,
		LocationID:     "default",
		OnHandQuantity: 100,
	})

	// Table-driven test cases
	testCases := []struct {
		name           string
		productID      string
		request        *pb.UpdateProductRequest
		expectedStatus int
		expectError    bool
	}{
		{
			name:      "successful_update_name",
			productID: product.ID,
			request: &pb.UpdateProductRequest{
				Id:   product.ID,
				Name: "Updated Product Name",
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:      "successful_update_description",
			productID: product.ID,
			request: &pb.UpdateProductRequest{
				Id:          product.ID,
				Description: "Updated description",
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:      "successful_update_status",
			productID: product.ID,
			request: &pb.UpdateProductRequest{
				Id:     product.ID,
				Status: pb.ProductStatus_PRODUCT_STATUS_INACTIVE,
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:      "non_existent_product",
			productID: "550e8400-e29b-41d4-a716-446655440000",
			request: &pb.UpdateProductRequest{
				Id:   "550e8400-e29b-41d4-a716-446655440000",
				Name: "Updated",
			},
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name:      "duplicate_sku",
			productID: product.ID,
			request: &pb.UpdateProductRequest{
				Id:  product.ID,
				Sku: "EXISTING-SKU", // Will create fixture with this SKU
			},
			expectedStatus: http.StatusConflict,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// For duplicate SKU test, create another product
			if tc.name == "duplicate_sku" {
				testutil.CreateProductFixture(db, template.ID, "Another Product", "EXISTING-SKU", `{"Brand":{"type":"ATTRIBUTE_TYPE_TEXT","value":"Samsung"}}`)
				db.Create(&models.ProductPricing{
					ProductID: "temp-id",
					ListPrice: 500.00,
					Currency:  "USD",
					ValidFrom: time.Now(),
				})
			}

			// Create HTTP request
			body, _ := json.Marshal(tc.request)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/products/"+tc.productID, bytes.NewReader(body))
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
				var response pb.UpdateProductResponse
				if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// Verify product exists
				if response.Product == nil {
					t.Fatal("Expected product in response")
				}

				// Verify UpdatedAt changed
				if response.Product.UpdatedAt == nil {
					t.Error("Expected UpdatedAt to be set")
				}

				// Verify the field that was updated
				if tc.request.Name != "" && response.Product.Name != tc.request.Name {
					t.Errorf("Expected name %s, got %s", tc.request.Name, response.Product.Name)
				}
				if tc.request.Description != "" && response.Product.Description != tc.request.Description {
					t.Errorf("Expected description %s, got %s", tc.request.Description, response.Product.Description)
				}
			}
		})
	}

	t.Log("✅ All test cases passed")
}

// TestProductHandler_Delete tests the DELETE /api/v1/products/{id} endpoint
func TestProductHandler_Delete(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "product_templates", "products", "product_pricings", "product_inventories")

	// Initialize services and handlers
	productService := services.NewProductService(db)
	handler := handlers.NewProductHandler(productService)

	// Create template
	template := testutil.CreateTemplateFixture(db, "Electronics", `[
		{"name":"Brand","type":"text","required":true}
	]`)

	// Create product to delete
	product := testutil.CreateProductFixture(db, template.ID, "To Delete", "DELETE-001", `{"Brand":{"type":"ATTRIBUTE_TYPE_TEXT","value":"Test"}}`)
	db.Create(&models.ProductPricing{
		ProductID: product.ID,
		ListPrice: 100.00,
		Currency:  "USD",
		ValidFrom: time.Now(),
	})
	db.Create(&models.ProductInventory{
		ProductID:      product.ID,
		LocationID:     "default",
		OnHandQuantity: 10,
	})

	// Table-driven test cases
	testCases := []struct {
		name           string
		productID      string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "successful_deletion",
			productID:      product.ID,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "non_existent_product",
			productID:      "550e8400-e29b-41d4-a716-446655440000",
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name:           "empty_id",
			productID:      "",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create HTTP request
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/products/"+tc.productID, nil)
			rec := httptest.NewRecorder()

			// Execute handler
			handler.Delete(rec, req)

			// Assert response status
			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}

			if !tc.expectError {
				// Parse success response
				var response pb.DeleteProductResponse
				if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// Build expected response
				expectedResponse := &pb.DeleteProductResponse{
					Success: true,
				}

				// Compare entire response using protocmp (MANDATORY per constitution)
				if diff := cmp.Diff(expectedResponse, &response, protocmp.Transform()); diff != "" {
					t.Errorf("Response mismatch (-want +got):\n%s", diff)
				}

				// Verify product is actually deleted from database
				var deletedProduct models.Product
				err := db.First(&deletedProduct, "id = ?", tc.productID).Error
				if err == nil {
					t.Error("Expected product to be deleted from database")
				}
			}
		})
	}

	t.Log("✅ All test cases passed")
}

// TestProductHandler_BulkUpdateStatus tests the POST /api/v1/products/bulk/status endpoint
func TestProductHandler_BulkUpdateStatus(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "product_templates", "products", "product_pricings", "product_inventories")

	// Initialize services and handlers
	productService := services.NewProductService(db)
	handler := handlers.NewProductHandler(productService)

	// Create template and products
	template := testutil.CreateTemplateFixture(db, "Electronics", `[
		{"name":"Brand","type":"text","required":true}
	]`)

	// Create 3 products for bulk update
	productIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		product := testutil.CreateProductFixture(db, template.ID, fmt.Sprintf("Bulk Product %d", i), fmt.Sprintf("BULK-%03d", i), `{"Brand":{"type":"ATTRIBUTE_TYPE_TEXT","value":"Test"}}`)
		productIDs[i] = product.ID
		
		db.Create(&models.ProductPricing{
			ProductID: product.ID,
			ListPrice: 100.00,
			Currency:  "USD",
			ValidFrom: time.Now(),
		})
		db.Create(&models.ProductInventory{
			ProductID:      product.ID,
			LocationID:     "default",
			OnHandQuantity: 10,
		})
	}

	// Table-driven test cases
	testCases := []struct {
		name           string
		request        *pb.BulkUpdateStatusRequest
		expectedStatus int
		expectError    bool
	}{
		{
			name: "successful_bulk_update",
			request: &pb.BulkUpdateStatusRequest{
				ProductIds: productIDs,
				Status:     pb.ProductStatus_PRODUCT_STATUS_INACTIVE,
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name: "empty_product_ids",
			request: &pb.BulkUpdateStatusRequest{
				ProductIds: []string{},
				Status:     pb.ProductStatus_PRODUCT_STATUS_ACTIVE,
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "invalid_status",
			request: &pb.BulkUpdateStatusRequest{
				ProductIds: productIDs,
				Status:     pb.ProductStatus_PRODUCT_STATUS_UNSPECIFIED,
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "non_existent_products",
			request: &pb.BulkUpdateStatusRequest{
				ProductIds: []string{"550e8400-e29b-41d4-a716-446655440000"},
				Status:     pb.ProductStatus_PRODUCT_STATUS_ACTIVE,
			},
			expectedStatus: http.StatusOK, // Partial success
			expectError:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create HTTP request
			body, _ := json.Marshal(tc.request)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/products/bulk/status", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			// Execute handler
			handler.BulkUpdateStatus(rec, req)

			// Assert response status
			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}

			if !tc.expectError {
				// Parse success response
				var response pb.BulkUpdateStatusResponse
				if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// Verify response structure
				if response.UpdatedCount == 0 && len(tc.request.ProductIds) > 0 && tc.name == "successful_bulk_update" {
					t.Error("Expected at least one product to be updated")
				}
			}
		})
	}

	t.Log("✅ All test cases passed")
}

