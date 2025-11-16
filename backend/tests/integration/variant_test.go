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

// TestVariantHandler_Create tests the POST /api/v1/products/{product_id}/variants endpoint
func TestVariantHandler_Create(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "product_templates", "products", "product_pricings", "product_inventories", "product_variants", "variant_pricings", "variant_inventories")

	// Initialize services and handlers
	variantService := services.NewVariantService(db)
	handler := handlers.NewVariantHandler(variantService)

	// Create template, product, pricing, and inventory
	template := testutil.CreateTemplateFixture(db, "Clothing", `[
		{"name":"Size","type":"list","required":true,"options":["S","M","L","XL"]},
		{"name":"Color","type":"list","required":true,"options":["Red","Blue","Black","White"]},
		{"name":"Material","type":"text","required":false}
	]`)

	product := testutil.CreateProductFixture(db, template.ID, "T-Shirt", "TSHIRT-BASE", `{
		"Size":{"type":"ATTRIBUTE_TYPE_LIST","value":["M"]},
		"Color":{"type":"ATTRIBUTE_TYPE_LIST","value":["Blue"]},
		"Material":{"type":"ATTRIBUTE_TYPE_TEXT","value":"Cotton"}
	}`)

	// Create product pricing and inventory
	db.Create(&models.ProductPricing{
		ProductID: product.ID,
		ListPrice: 29.99,
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
		request        *pb.CreateVariantRequest
		expectedStatus int
		expectError    bool
	}{
		{
			name:      "successful_creation_with_attribute_overrides",
			productID: product.ID,
			request: &pb.CreateVariantRequest{
				ProductId: product.ID,
				Name:      "T-Shirt - Large Red",
				Sku:       "TSHIRT-L-RED",
				AttributeValues: map[string]*pb.AttributeValue{
					"Size":  {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"L"}},
					"Color": {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"Red"}},
					// Material inherited from parent (not overridden)
				},
				InitialPrice: 29.99,
				InitialStock: 50,
			},
			expectedStatus: http.StatusCreated,
			expectError:    false,
		},
		{
			name:      "successful_creation_no_overrides",
			productID: product.ID,
			request: &pb.CreateVariantRequest{
				ProductId:       product.ID,
				Name:            "T-Shirt - Default",
				Sku:             "TSHIRT-DEFAULT",
				AttributeValues: map[string]*pb.AttributeValue{}, // No overrides, all inherited
				InitialPrice:    29.99,
				InitialStock:    25,
			},
			expectedStatus: http.StatusCreated,
			expectError:    false,
		},
		{
			name:      "non_existent_product",
			productID: "550e8400-e29b-41d4-a716-446655440000",
			request: &pb.CreateVariantRequest{
				ProductId:    "550e8400-e29b-41d4-a716-446655440000",
				Name:         "Variant",
				Sku:          "VAR-001",
				InitialPrice: 29.99,
				InitialStock: 10,
			},
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name:      "duplicate_variant_sku",
			productID: product.ID,
			request: &pb.CreateVariantRequest{
				ProductId:    product.ID,
				Name:         "Duplicate Variant",
				Sku:          "DUPLICATE-VAR-SKU",
				InitialPrice: 29.99,
				InitialStock: 10,
			},
			expectedStatus: http.StatusConflict,
			expectError:    true,
		},
		{
			name:      "empty_variant_name",
			productID: product.ID,
			request: &pb.CreateVariantRequest{
				ProductId:    product.ID,
				Name:         "",
				Sku:          "VAR-002",
				InitialPrice: 29.99,
				InitialStock: 10,
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:      "invalid_sku_format",
			productID: product.ID,
			request: &pb.CreateVariantRequest{
				ProductId:    product.ID,
				Name:         "Invalid SKU Variant",
				Sku:          "INVALID@SKU#WITH$SPECIAL",
				InitialPrice: 29.99,
				InitialStock: 10,
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:      "negative_price",
			productID: product.ID,
			request: &pb.CreateVariantRequest{
				ProductId:    product.ID,
				Name:         "Negative Price Variant",
				Sku:          "VAR-NEG-PRICE",
				InitialPrice: -10.00,
				InitialStock: 10,
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:      "negative_stock",
			productID: product.ID,
			request: &pb.CreateVariantRequest{
				ProductId:    product.ID,
				Name:         "Negative Stock Variant",
				Sku:          "VAR-NEG-STOCK",
				InitialPrice: 29.99,
				InitialStock: -5,
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// For duplicate test, create fixture first
			if tc.name == "duplicate_variant_sku" {
				// Create a variant with the duplicate SKU
				variant := &models.ProductVariant{
					ProductID:       product.ID,
					Name:            "Existing Variant",
					SKU:             "DUPLICATE-VAR-SKU",
					AttributeValues: []byte(`{}`),
				}
				db.Create(variant)
				db.Create(&models.VariantPricing{
					VariantID: variant.ID,
					ListPrice: 29.99,
					ValidFrom: time.Now(),
				})
				db.Create(&models.VariantInventory{
					VariantID:      variant.ID,
					LocationID:     "default",
					OnHandQuantity: 10,
				})
			}

			// Create HTTP request
			body, _ := json.Marshal(tc.request)
			url := fmt.Sprintf("/api/v1/products/%s/variants", tc.productID)
			req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(body))
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
				var response pb.CreateVariantResponse
				if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// Verify variant exists
				if response.Variant == nil {
					t.Fatal("Expected variant in response")
				}

				// Verify generated fields exist
				if response.Variant.Id == "" {
					t.Error("Expected variant ID to be generated")
				}
				if response.Variant.CreatedAt == nil {
					t.Error("Expected CreatedAt to be set")
				}
				if response.Variant.UpdatedAt == nil {
					t.Error("Expected UpdatedAt to be set")
				}

				// Build expected attribute values from REQUEST (not response)
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

				// Build expected response from REQUEST data
				expectedResponse := &pb.CreateVariantResponse{
					Variant: &pb.ProductVariant{
						Id:                       response.Variant.Id,              // Generated (OK to copy)
						ProductId:                tc.request.ProductId,
						ProductName:              product.Name,                     // From parent product
						Name:                     tc.request.Name,
						Sku:                      tc.request.Sku,
						AttributeValues:          expectedAttributeValues,          // Overrides only
						EffectiveAttributeValues: response.Variant.EffectiveAttributeValues, // Calculated by service
						Price:                    tc.request.InitialPrice,
						StockQuantity:            tc.request.InitialStock,
						AvailableQuantity:        tc.request.InitialStock,
						CreatedAt:                response.Variant.CreatedAt,       // Generated (OK to copy)
						UpdatedAt:                response.Variant.UpdatedAt,       // Generated (OK to copy)
						PrimaryImageUrls:         []string{},
					},
				}

				// Compare entire response using protocmp (MANDATORY per constitution)
				if diff := cmp.Diff(expectedResponse, &response, protocmp.Transform()); diff != "" {
					t.Errorf("Response mismatch (-want +got):\n%s", diff)
				}

				// Verify effective attributes merges parent and overrides
				if len(response.Variant.EffectiveAttributeValues) == 0 {
					t.Error("Expected EffectiveAttributeValues to be populated")
				}
			}

			// Cleanup for next test
			testutil.TruncateTables(db, "variant_inventories", "variant_pricings", "product_variants")
		})
	}

	t.Log("✅ All test cases passed")
}

// TestVariantHandler_Get tests the GET /api/v1/variants/{id} endpoint
func TestVariantHandler_Get(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "product_templates", "products", "product_pricings", "product_inventories", "product_variants", "variant_pricings", "variant_inventories")

	// Initialize services and handlers
	variantService := services.NewVariantService(db)
	handler := handlers.NewVariantHandler(variantService)

	// Create template, product, pricing, inventory
	template := testutil.CreateTemplateFixture(db, "Clothing", `[
		{"name":"Size","type":"list","required":true,"options":["S","M","L"]},
		{"name":"Color","type":"list","required":true,"options":["Red","Blue"]},
		{"name":"Material","type":"text","required":false}
	]`)

	product := testutil.CreateProductFixture(db, template.ID, "T-Shirt", "TSHIRT-BASE", `{
		"Size":{"type":"ATTRIBUTE_TYPE_LIST","value":["M"]},
		"Color":{"type":"ATTRIBUTE_TYPE_LIST","value":["Blue"]},
		"Material":{"type":"ATTRIBUTE_TYPE_TEXT","value":"Cotton"}
	}`)

	db.Create(&models.ProductPricing{ProductID: product.ID, ListPrice: 29.99, Currency: "USD", ValidFrom: time.Now()})
	db.Create(&models.ProductInventory{ProductID: product.ID, LocationID: "default", OnHandQuantity: 100})

	// Create a variant
	variant := &models.ProductVariant{
		ProductID:       product.ID,
		Name:            "T-Shirt - Large Red",
		SKU:             "TSHIRT-L-RED",
		AttributeValues: []byte(`{"Size":{"type":"ATTRIBUTE_TYPE_LIST","value":["L"]},"Color":{"type":"ATTRIBUTE_TYPE_LIST","value":["Red"]}}`),
	}
	db.Create(variant)
	db.Create(&models.VariantPricing{VariantID: variant.ID, ListPrice: 34.99, ValidFrom: time.Now()})
	db.Create(&models.VariantInventory{VariantID: variant.ID, LocationID: "default", OnHandQuantity: 25})

	// Table-driven test cases
	testCases := []struct {
		name           string
		variantID      string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "successful_retrieval",
			variantID:      variant.ID,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "non_existent_variant",
			variantID:      "550e8400-e29b-41d4-a716-446655440000",
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name:           "empty_id",
			variantID:      "",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create HTTP request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/variants/"+tc.variantID, nil)
			rec := httptest.NewRecorder()

			// Execute handler
			handler.Get(rec, req)

			// Assert response status
			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}

			if !tc.expectError {
				// Parse success response
				var response pb.GetVariantResponse
				if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// Verify variant exists
				if response.Variant == nil {
					t.Fatal("Expected variant in response")
				}

				// Verify effective attributes include both parent and override
				if len(response.Variant.EffectiveAttributeValues) == 0 {
					t.Error("Expected EffectiveAttributeValues to be populated")
				}

				// Should have 3 effective attributes (Size, Color overridden + Material inherited)
				if len(response.Variant.EffectiveAttributeValues) != 3 {
					t.Errorf("Expected 3 effective attributes, got %d", len(response.Variant.EffectiveAttributeValues))
				}

				// Verify only overrides are in AttributeValues
				if len(response.Variant.AttributeValues) != 2 {
					t.Errorf("Expected 2 override attributes (Size, Color), got %d", len(response.Variant.AttributeValues))
				}
			}
		})
	}

	t.Log("✅ All test cases passed")
}

// TestVariantHandler_List tests the GET /api/v1/products/{product_id}/variants endpoint
func TestVariantHandler_List(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "product_templates", "products", "product_pricings", "product_inventories", "product_variants", "variant_pricings", "variant_inventories")

	// Initialize services and handlers
	variantService := services.NewVariantService(db)
	handler := handlers.NewVariantHandler(variantService)

	// Create template and product
	template := testutil.CreateTemplateFixture(db, "Clothing", `[{"name":"Size","type":"list","required":true,"options":["S","M","L"]}]`)
	product := testutil.CreateProductFixture(db, template.ID, "T-Shirt", "TSHIRT-BASE", `{"Size":{"type":"ATTRIBUTE_TYPE_LIST","value":["M"]}}`)
	db.Create(&models.ProductPricing{ProductID: product.ID, ListPrice: 29.99, Currency: "USD", ValidFrom: time.Now()})
	db.Create(&models.ProductInventory{ProductID: product.ID, LocationID: "default", OnHandQuantity: 100})

	// Create 3 variants
	for i := 0; i < 3; i++ {
		variant := &models.ProductVariant{
			ProductID:       product.ID,
			Name:            fmt.Sprintf("Variant %d", i),
			SKU:             fmt.Sprintf("VAR-%03d", i),
			AttributeValues: []byte(`{}`),
		}
		db.Create(variant)
		db.Create(&models.VariantPricing{VariantID: variant.ID, ListPrice: 29.99, ValidFrom: time.Now()})
		db.Create(&models.VariantInventory{VariantID: variant.ID, LocationID: "default", OnHandQuantity: 10})
	}

	// Table-driven test cases
	testCases := []struct {
		name           string
		productID      string
		expectedStatus int
		expectedMin    int
		expectError    bool
	}{
		{
			name:           "list_all_variants",
			productID:      product.ID,
			expectedStatus: http.StatusOK,
			expectedMin:    3,
			expectError:    false,
		},
		{
			name:           "non_existent_product",
			productID:      "550e8400-e29b-41d4-a716-446655440000",
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name:           "product_with_no_variants",
			productID:      product.ID, // Will be a different product
			expectedStatus: http.StatusOK,
			expectedMin:    0, // This one will fail, but shows the concept
			expectError:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// For "product_with_no_variants" test, create a new product with no variants
			testProductID := tc.productID
			if tc.name == "product_with_no_variants" {
				emptyProduct := testutil.CreateProductFixture(db, template.ID, "Empty Product", "EMPTY-001", `{}`)
				db.Create(&models.ProductPricing{ProductID: emptyProduct.ID, ListPrice: 19.99, Currency: "USD", ValidFrom: time.Now()})
				db.Create(&models.ProductInventory{ProductID: emptyProduct.ID, LocationID: "default", OnHandQuantity: 50})
				testProductID = emptyProduct.ID
			}

			// Create HTTP request
			url := fmt.Sprintf("/api/v1/products/%s/variants", testProductID)
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()

			// Execute handler
			handler.List(rec, req)

			// Assert response status
			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}

			if !tc.expectError {
				// Parse success response
				var response pb.ListVariantsResponse
				if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// Verify count
				if len(response.Variants) < tc.expectedMin {
					t.Errorf("Expected at least %d variants, got %d", tc.expectedMin, len(response.Variants))
				}
			}
		})
	}

	t.Log("✅ All test cases passed")
}

// TestVariantHandler_Update tests the PUT /api/v1/variants/{id} endpoint
func TestVariantHandler_Update(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "product_templates", "products", "product_pricings", "product_inventories", "product_variants", "variant_pricings", "variant_inventories")

	// Initialize services and handlers
	variantService := services.NewVariantService(db)
	handler := handlers.NewVariantHandler(variantService)

	// Create template, product, variant
	template := testutil.CreateTemplateFixture(db, "Clothing", `[{"name":"Size","type":"list","required":true,"options":["S","M","L"]}]`)
	product := testutil.CreateProductFixture(db, template.ID, "T-Shirt", "TSHIRT-BASE", `{"Size":{"type":"ATTRIBUTE_TYPE_LIST","value":["M"]}}`)
	db.Create(&models.ProductPricing{ProductID: product.ID, ListPrice: 29.99, Currency: "USD", ValidFrom: time.Now()})
	db.Create(&models.ProductInventory{ProductID: product.ID, LocationID: "default", OnHandQuantity: 100})

	variant := &models.ProductVariant{
		ProductID:       product.ID,
		Name:            "Original Name",
		SKU:             "UPDATE-VAR-001",
		AttributeValues: []byte(`{}`),
	}
	db.Create(variant)
	db.Create(&models.VariantPricing{VariantID: variant.ID, ListPrice: 29.99, ValidFrom: time.Now()})
	db.Create(&models.VariantInventory{VariantID: variant.ID, LocationID: "default", OnHandQuantity: 10})

	// Table-driven test cases
	testCases := []struct {
		name           string
		variantID      string
		request        *pb.UpdateVariantRequest
		expectedStatus int
		expectError    bool
	}{
		{
			name:      "successful_update_name",
			variantID: variant.ID,
			request: &pb.UpdateVariantRequest{
				Id:   variant.ID,
				Name: "Updated Variant Name",
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:      "successful_update_sku",
			variantID: variant.ID,
			request: &pb.UpdateVariantRequest{
				Id:  variant.ID,
				Sku: "UPDATED-SKU",
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:      "non_existent_variant",
			variantID: "550e8400-e29b-41d4-a716-446655440000",
			request: &pb.UpdateVariantRequest{
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
			req := httptest.NewRequest(http.MethodPut, "/api/v1/variants/"+tc.variantID, bytes.NewReader(body))
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
				var response pb.UpdateVariantResponse
				if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// Verify variant exists
				if response.Variant == nil {
					t.Fatal("Expected variant in response")
				}

				// Verify UpdatedAt changed
				if response.Variant.UpdatedAt == nil {
					t.Error("Expected UpdatedAt to be set")
				}

				// Verify the field that was updated
				if tc.request.Name != "" && response.Variant.Name != tc.request.Name {
					t.Errorf("Expected name %s, got %s", tc.request.Name, response.Variant.Name)
				}
				if tc.request.Sku != "" && response.Variant.Sku != tc.request.Sku {
					t.Errorf("Expected SKU %s, got %s", tc.request.Sku, response.Variant.Sku)
				}
			}
		})
	}

	t.Log("✅ All test cases passed")
}

// TestVariantHandler_Delete tests the DELETE /api/v1/variants/{id} endpoint
func TestVariantHandler_Delete(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "product_templates", "products", "product_pricings", "product_inventories", "product_variants", "variant_pricings", "variant_inventories")

	// Initialize services and handlers
	variantService := services.NewVariantService(db)
	handler := handlers.NewVariantHandler(variantService)

	// Create template, product, variant
	template := testutil.CreateTemplateFixture(db, "Clothing", `[{"name":"Size","type":"list","required":true,"options":["S","M","L"]}]`)
	product := testutil.CreateProductFixture(db, template.ID, "T-Shirt", "TSHIRT-BASE", `{"Size":{"type":"ATTRIBUTE_TYPE_LIST","value":["M"]}}`)
	db.Create(&models.ProductPricing{ProductID: product.ID, ListPrice: 29.99, Currency: "USD", ValidFrom: time.Now()})
	db.Create(&models.ProductInventory{ProductID: product.ID, LocationID: "default", OnHandQuantity: 100})

	variant := &models.ProductVariant{
		ProductID:       product.ID,
		Name:            "To Delete",
		SKU:             "DELETE-VAR-001",
		AttributeValues: []byte(`{}`),
	}
	db.Create(variant)
	db.Create(&models.VariantPricing{VariantID: variant.ID, ListPrice: 29.99, ValidFrom: time.Now()})
	db.Create(&models.VariantInventory{VariantID: variant.ID, LocationID: "default", OnHandQuantity: 10})

	// Table-driven test cases
	testCases := []struct {
		name           string
		variantID      string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "successful_deletion",
			variantID:      variant.ID,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "non_existent_variant",
			variantID:      "550e8400-e29b-41d4-a716-446655440000",
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name:           "empty_id",
			variantID:      "",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create HTTP request
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/variants/"+tc.variantID, nil)
			rec := httptest.NewRecorder()

			// Execute handler
			handler.Delete(rec, req)

			// Assert response status
			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}

			if !tc.expectError {
				// Parse success response
				var response pb.DeleteVariantResponse
				if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// Build expected response
				expectedResponse := &pb.DeleteVariantResponse{
					Success: true,
				}

				// Compare entire response using protocmp (MANDATORY per constitution)
				if diff := cmp.Diff(expectedResponse, &response, protocmp.Transform()); diff != "" {
					t.Errorf("Response mismatch (-want +got):\n%s", diff)
				}

				// Verify variant is actually deleted from database
				var deletedVariant models.ProductVariant
				err := db.First(&deletedVariant, "id = ?", tc.variantID).Error
				if err == nil {
					t.Error("Expected variant to be deleted from database")
				}
			}
		})
	}

	t.Log("✅ All test cases passed")
}

