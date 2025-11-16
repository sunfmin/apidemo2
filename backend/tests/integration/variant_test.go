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

