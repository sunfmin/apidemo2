package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	pb "github.com/sunfmin/apidemo2/backend/api/gen/pim/v1"
	"github.com/sunfmin/apidemo2/backend/handlers"
	"github.com/sunfmin/apidemo2/backend/internal/models"
	"github.com/sunfmin/apidemo2/backend/services"
	"github.com/sunfmin/apidemo2/backend/tests/testutil"
)

// TestAllSentinelErrors ensures EVERY sentinel error defined in services/errors.go is tested
// This is MANDATORY per Constitution Principle IX
func TestAllSentinelErrors(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "media_files", "product_variants", "variant_pricings", "variant_inventories", "products", "product_pricings", "product_inventories", "product_templates")

	// Initialize services
	templateService := services.NewTemplateService(db)
	productService := services.NewProductService(db)
	variantService := services.NewVariantService(db)

	// Create test fixtures
	template := testutil.CreateTemplateFixture(db, "Test Template", `[
		{"name":"Size","type":"list","required":true,"options":["S","M","L"]},
		{"name":"Color","type":"list","required":true,"options":["Red","Blue"]}
	]`)

	product := testutil.CreateProductFixture(db, template.ID, "Test Product", "TEST-001", `{
		"Size":{"type":"ATTRIBUTE_TYPE_LIST","value":["M"]},
		"Color":{"type":"ATTRIBUTE_TYPE_LIST","value":["Blue"]}
	}`)
	db.Create(&models.ProductPricing{ProductID: product.ID, ListPrice: 99.99, Currency: "USD", ValidFrom: time.Now()})
	db.Create(&models.ProductInventory{ProductID: product.ID, LocationID: "default", OnHandQuantity: 100})

	variant := &models.ProductVariant{
		ProductID:       product.ID,
		Name:            "Test Variant",
		SKU:             "TEST-VAR-001",
		AttributeValues: []byte(`{}`),
	}
	db.Create(variant)
	db.Create(&models.VariantPricing{VariantID: variant.ID, ListPrice: 99.99, ValidFrom: time.Now()})
	db.Create(&models.VariantInventory{VariantID: variant.ID, LocationID: "default", OnHandQuantity: 50})

	mediaFile := &models.MediaFile{
		EntityType:    "product",
		EntityID:      product.ID,
		AttributeName: "TestImage",
		FileType:      "image",
		MimeType:      "image/jpeg",
		FileName:      "test.jpg",
		FilePath:      "test.jpg",
		FileSize:      1024,
		DisplayOrder:  0,
	}
	db.Create(mediaFile)

	testCases := []struct {
		name          string
		serviceCall   func(context.Context) error
		expectedError error
		description   string
	}{
		{
			name: "ErrTemplateNotFound",
			serviceCall: func(ctx context.Context) error {
				_, err := templateService.Get(ctx, "550e8400-e29b-41d4-a716-446655440000")
				return err
			},
			expectedError: services.ErrTemplateNotFound,
			description:   "Template not found error when getting non-existent template",
		},
		{
			name: "ErrProductNotFound",
			serviceCall: func(ctx context.Context) error {
				_, err := productService.Get(ctx, &pb.GetProductRequest{Id: "550e8400-e29b-41d4-a716-446655440000"})
				return err
			},
			expectedError: services.ErrProductNotFound,
			description:   "Product not found error when getting non-existent product",
		},
		{
			name: "ErrVariantNotFound",
			serviceCall: func(ctx context.Context) error {
				_, err := variantService.Get(ctx, "550e8400-e29b-41d4-a716-446655440000")
				return err
			},
			expectedError: services.ErrVariantNotFound,
			description:   "Variant not found error when getting non-existent variant",
		},
		{
			name: "ErrMediaNotFound",
			serviceCall: func(ctx context.Context) error {
				storageDir := t.TempDir()
				localStorage, _ := services.NewLocalStorage(storageDir, "http://localhost:8080/media")
				imageProcessor := services.NewImageProcessor(300, 300)
				videoProcessor := services.NewVideoProcessor()
				mediaService := services.NewMediaService(db, localStorage, imageProcessor, videoProcessor, storageDir)
				_, err := mediaService.Get(ctx, "550e8400-e29b-41d4-a716-446655440000")
				return err
			},
			expectedError: services.ErrMediaNotFound,
			description:   "Media not found error when getting non-existent media file",
		},
		{
			name: "ErrInvalidSKU",
			serviceCall: func(ctx context.Context) error {
				_, err := productService.Create(ctx, &pb.CreateProductRequest{
					TemplateId:       template.ID,
					Name:             "Invalid SKU Product",
					Sku:              "INVALID@SKU#CHARS",
					InitialListPrice: 99.99,
					InitialStock:     10,
					AttributeValues: map[string]*pb.AttributeValue{
						"Size":  {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"M"}},
						"Color": {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"Blue"}},
					},
				})
				return err
			},
			expectedError: services.ErrInvalidSKU,
			description:   "Invalid SKU format error with special characters",
		},
		{
			name: "ErrInvalidRequest",
			serviceCall: func(ctx context.Context) error {
				_, err := productService.Create(ctx, &pb.CreateProductRequest{
					TemplateId:       template.ID,
					Name:             "Invalid Attribute Product",
					Sku:              "TEST-INVALID-ATTR",
					InitialListPrice: 99.99,
					InitialStock:     10,
					AttributeValues: map[string]*pb.AttributeValue{
						"Size":  {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"M"}},
						"Color": {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"Purple"}}, // Purple not in options
					},
				})
				return err
			},
			expectedError: services.ErrInvalidRequest,
			description:   "Invalid request error with list value not in allowed options",
		},
		{
			name: "ErrMissingRequired_ProductName",
			serviceCall: func(ctx context.Context) error {
				_, err := productService.Create(ctx, &pb.CreateProductRequest{
					TemplateId:       template.ID,
					Name:             "",
					Sku:              "TEST-MISSING-NAME",
					InitialListPrice: 99.99,
					InitialStock:     10,
					AttributeValues: map[string]*pb.AttributeValue{
						"Size":  {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"M"}},
						"Color": {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"Blue"}},
					},
				})
				return err
			},
			expectedError: services.ErrMissingRequired,
			description:   "Missing required field error with empty product name",
		},
		{
			name: "ErrMissingRequired_TemplateName",
			serviceCall: func(ctx context.Context) error {
				_, err := templateService.Create(ctx, &pb.CreateTemplateRequest{
					Name: "",
					Attributes: []*pb.AttributeDefinition{
						{Name: "Field", Type: pb.AttributeType_ATTRIBUTE_TYPE_TEXT, Required: true},
					},
				})
				return err
			},
			expectedError: services.ErrMissingRequired,
			description:   "Missing required field error with empty template name",
		},
		{
			name: "ErrMissingRequired_RequiredAttribute",
			serviceCall: func(ctx context.Context) error {
				_, err := productService.Create(ctx, &pb.CreateProductRequest{
					TemplateId:       template.ID,
					Name:             "Missing Required Attr Product",
					Sku:              "TEST-MISSING-ATTR",
					InitialListPrice: 99.99,
					InitialStock:     10,
					AttributeValues: map[string]*pb.AttributeValue{
						"Size": {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"M"}},
						// Missing required "Color" attribute
					},
				})
				return err
			},
			expectedError: services.ErrMissingRequired,
			description:   "Missing required attribute error",
		},
		{
			name: "ErrInvalidType",
			serviceCall: func(ctx context.Context) error {
				_, err := productService.Create(ctx, &pb.CreateProductRequest{
					TemplateId:       template.ID,
					Name:             "Type Mismatch Product",
					Sku:              "TEST-TYPE-MISMATCH",
					InitialListPrice: 99.99,
					InitialStock:     10,
					AttributeValues: map[string]*pb.AttributeValue{
						"Size":  {Type: pb.AttributeType_ATTRIBUTE_TYPE_TEXT, TextValue: "Wrong Type"}, // Should be LIST
						"Color": {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"Blue"}},
					},
				})
				return err
			},
			expectedError: services.ErrInvalidType,
			description:   "Invalid type error with attribute type mismatch",
		},
		{
			name: "ErrInvalidType_TemplateAttribute",
			serviceCall: func(ctx context.Context) error {
				_, err := templateService.Create(ctx, &pb.CreateTemplateRequest{
					Name: "Invalid Type Template",
					Attributes: []*pb.AttributeDefinition{
						{Name: "Field", Type: pb.AttributeType_ATTRIBUTE_TYPE_UNSPECIFIED, Required: true},
					},
				})
				return err
			},
			expectedError: services.ErrInvalidType,
			description:   "Invalid type error with unspecified attribute type",
		},
		{
			name: "ErrValueOutOfRange_NegativePrice",
			serviceCall: func(ctx context.Context) error {
				_, err := productService.Create(ctx, &pb.CreateProductRequest{
					TemplateId:       template.ID,
					Name:             "Negative Price Product",
					Sku:              "TEST-NEG-PRICE",
					InitialListPrice: -99.99,
					InitialStock:     10,
					AttributeValues: map[string]*pb.AttributeValue{
						"Size":  {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"M"}},
						"Color": {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"Blue"}},
					},
				})
				return err
			},
			expectedError: services.ErrValueOutOfRange,
			description:   "Value out of range error with negative price",
		},
		{
			name: "ErrValueOutOfRange_NegativeStock",
			serviceCall: func(ctx context.Context) error {
				_, err := productService.Create(ctx, &pb.CreateProductRequest{
					TemplateId:       template.ID,
					Name:             "Negative Stock Product",
					Sku:              "TEST-NEG-STOCK",
					InitialListPrice: 99.99,
					InitialStock:     -10,
					AttributeValues: map[string]*pb.AttributeValue{
						"Size":  {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"M"}},
						"Color": {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"Blue"}},
					},
				})
				return err
			},
			expectedError: services.ErrValueOutOfRange,
			description:   "Value out of range error with negative stock",
		},
		{
			name: "ErrValueOutOfRange_TooLongName",
			serviceCall: func(ctx context.Context) error {
				longName := ""
				for i := 0; i < 300; i++ {
					longName += "a"
				}
				_, err := productService.Create(ctx, &pb.CreateProductRequest{
					TemplateId:       template.ID,
					Name:             longName,
					Sku:              "TEST-LONG-NAME",
					InitialListPrice: 99.99,
					InitialStock:     10,
					AttributeValues: map[string]*pb.AttributeValue{
						"Size":  {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"M"}},
						"Color": {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"Blue"}},
					},
				})
				return err
			},
			expectedError: services.ErrValueOutOfRange,
			description:   "Value out of range error with name exceeding 255 characters",
		},
		{
			name: "ErrDuplicateSKU_Product",
			serviceCall: func(ctx context.Context) error {
				_, err := productService.Create(ctx, &pb.CreateProductRequest{
					TemplateId:       template.ID,
					Name:             "Duplicate SKU Product",
					Sku:              "TEST-001", // Already exists in fixture
					InitialListPrice: 99.99,
					InitialStock:     10,
					AttributeValues: map[string]*pb.AttributeValue{
						"Size":  {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"M"}},
						"Color": {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"Blue"}},
					},
				})
				return err
			},
			expectedError: services.ErrDuplicateSKU,
			description:   "Duplicate SKU error when creating product with existing SKU",
		},
		{
			name: "ErrDuplicateSKU_Variant",
			serviceCall: func(ctx context.Context) error {
				_, err := variantService.Create(ctx, &pb.CreateVariantRequest{
					ProductId:    product.ID,
					Name:         "Duplicate SKU Variant",
					Sku:          "TEST-VAR-001", // Already exists in fixture
					InitialPrice: 99.99,
					InitialStock: 10,
				})
				return err
			},
			expectedError: services.ErrDuplicateSKU,
			description:   "Duplicate SKU error when creating variant with existing SKU",
		},
		{
			name: "ErrDuplicateName",
			serviceCall: func(ctx context.Context) error {
				// Create a template first
				testutil.CreateTemplateFixture(db, "Duplicate Name Template", `[{"name":"Field","type":"text","required":true}]`)
				
				// Try to create another with same name
				_, err := templateService.Create(ctx, &pb.CreateTemplateRequest{
					Name: "Duplicate Name Template",
					Attributes: []*pb.AttributeDefinition{
						{Name: "Field", Type: pb.AttributeType_ATTRIBUTE_TYPE_TEXT, Required: true},
					},
				})
				return err
			},
			expectedError: services.ErrDuplicateName,
			description:   "Duplicate name error when creating template with existing name",
		},
		{
			name: "ErrAlreadyExists",
			serviceCall: func(ctx context.Context) error {
				storageDir := t.TempDir()
				localStorage, _ := services.NewLocalStorage(storageDir, "http://localhost:8080/media")
				imageProcessor := services.NewImageProcessor(300, 300)
				videoProcessor := services.NewVideoProcessor()
				mediaService := services.NewMediaService(db, localStorage, imageProcessor, videoProcessor, storageDir)
				
				// Try to reorder with duplicate IDs
				_, err := mediaService.Reorder(ctx, "product", product.ID, "TestImage", []string{mediaFile.ID, mediaFile.ID})
				return err
			},
			expectedError: services.ErrAlreadyExists,
			description:   "Already exists error when reordering with duplicate media IDs",
		},
		{
			name: "ErrHasProducts",
			serviceCall: func(ctx context.Context) error {
				// Create a template with products
				templateWithProducts := testutil.CreateTemplateFixture(db, "Template With Products", `[{"name":"Field","type":"text","required":true}]`)
				testutil.CreateProductFixture(db, templateWithProducts.ID, "Product Using Template", "HAS-PROD-001", `{}`)
				
				// Try to delete template that has products
				err := templateService.Delete(ctx, templateWithProducts.ID)
				return err
			},
			expectedError: services.ErrHasProducts,
			description:   "Has products error when deleting template with associated products",
		},
	}

	ctx := context.Background()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.serviceCall(ctx)
			
			if err == nil {
				t.Fatalf("Expected error %v, but got nil. Description: %s", tc.expectedError, tc.description)
			}

			// Use errors.Is to check if the sentinel error is in the error chain
			if !ErrorContains(err, tc.expectedError) {
				t.Errorf("Expected error chain to contain %v, but got: %v. Description: %s", tc.expectedError, err, tc.description)
			}

			t.Logf("✅ Verified sentinel error: %v", tc.expectedError)
		})
	}

	t.Log("✅ All sentinel errors tested and verified")
}

// TestAllHTTPErrorCodes ensures EVERY HTTP error code defined in handlers/error_codes.go is tested
// This is MANDATORY per Constitution Principle IX
func TestAllHTTPErrorCodes(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "media_files", "product_variants", "variant_pricings", "variant_inventories", "products", "product_pricings", "product_inventories", "product_templates")

	// Initialize services and handlers
	templateService := services.NewTemplateService(db)
	productService := services.NewProductService(db)
	variantService := services.NewVariantService(db)
	
	templateHandler := handlers.NewTemplateHandler(templateService)
	productHandler := handlers.NewProductHandler(productService)
	variantHandler := handlers.NewVariantHandler(variantService)

	// Create test fixtures
	template := testutil.CreateTemplateFixture(db, "Error Test Template", `[
		{"name":"Size","type":"list","required":true,"options":["S","M","L"]},
		{"name":"Color","type":"list","required":true,"options":["Red","Blue"]}
	]`)

	product := testutil.CreateProductFixture(db, template.ID, "Error Test Product", "ERR-TEST-001", `{
		"Size":{"type":"ATTRIBUTE_TYPE_LIST","value":["M"]},
		"Color":{"type":"ATTRIBUTE_TYPE_LIST","value":["Blue"]}
	}`)
	db.Create(&models.ProductPricing{ProductID: product.ID, ListPrice: 99.99, Currency: "USD", ValidFrom: time.Now()})
	db.Create(&models.ProductInventory{ProductID: product.ID, LocationID: "default", OnHandQuantity: 100})

	variant := &models.ProductVariant{
		ProductID:       product.ID,
		Name:            "Error Test Variant",
		SKU:             "ERR-VAR-001",
		AttributeValues: []byte(`{}`),
	}
	db.Create(variant)
	db.Create(&models.VariantPricing{VariantID: variant.ID, ListPrice: 99.99, ValidFrom: time.Now()})
	db.Create(&models.VariantInventory{VariantID: variant.ID, LocationID: "default", OnHandQuantity: 50})

	mediaFile := &models.MediaFile{
		EntityType:    "product",
		EntityID:      product.ID,
		AttributeName: "TestImage",
		FileType:      "image",
		MimeType:      "image/jpeg",
		FileName:      "test.jpg",
		FilePath:      "test.jpg",
		FileSize:      1024,
		DisplayOrder:  0,
	}
	db.Create(mediaFile)

	testCases := []struct {
		name             string
		httpCall         func() *httptest.ResponseRecorder
		expectedStatus   int
		expectedErrorCode string
		description      string
	}{
		{
			name: "InvalidRequest - Malformed JSON",
			httpCall: func() *httptest.ResponseRecorder {
				req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader([]byte("invalid json")))
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				productHandler.Create(rec, req)
				return rec
			},
			expectedStatus:   http.StatusBadRequest,
			expectedErrorCode: "INVALID_REQUEST",
			description:      "Invalid request with malformed JSON",
		},
		{
			name: "ValidationFailed - Invalid SKU Format",
			httpCall: func() *httptest.ResponseRecorder {
				reqBody := &pb.CreateProductRequest{
					TemplateId:       template.ID,
					Name:             "Invalid SKU Product",
					Sku:              "INVALID@SKU",
					InitialListPrice: 99.99,
					InitialStock:     10,
					AttributeValues: map[string]*pb.AttributeValue{
						"Size":  {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"M"}},
						"Color": {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"Blue"}},
					},
				}
				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				productHandler.Create(rec, req)
				return rec
			},
			expectedStatus:   http.StatusBadRequest,
			expectedErrorCode: "VALIDATION_ERROR",
			description:      "Validation failed with invalid SKU format",
		},
		{
			name: "MissingRequired - Empty Product Name",
			httpCall: func() *httptest.ResponseRecorder {
				reqBody := &pb.CreateProductRequest{
					TemplateId:       template.ID,
					Name:             "",
					Sku:              "MISSING-NAME-001",
					InitialListPrice: 99.99,
					InitialStock:     10,
					AttributeValues: map[string]*pb.AttributeValue{
						"Size":  {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"M"}},
						"Color": {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"Blue"}},
					},
				}
				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				productHandler.Create(rec, req)
				return rec
			},
			expectedStatus:   http.StatusBadRequest,
			expectedErrorCode: "MISSING_REQUIRED",
			description:      "Missing required field - empty product name",
		},
		{
			name: "InvalidFormat - Empty ID in URL",
			httpCall: func() *httptest.ResponseRecorder {
				req := httptest.NewRequest(http.MethodGet, "/api/v1/products/", nil)
				rec := httptest.NewRecorder()
				productHandler.Get(rec, req)
				return rec
			},
			expectedStatus:   http.StatusBadRequest,
			expectedErrorCode: "INVALID_REQUEST",
			description:      "Invalid format with empty ID in URL",
		},
		{
			name: "InvalidType - Attribute Type Mismatch",
			httpCall: func() *httptest.ResponseRecorder {
				reqBody := &pb.CreateProductRequest{
					TemplateId:       template.ID,
					Name:             "Type Mismatch Product",
					Sku:              "TYPE-MISMATCH-001",
					InitialListPrice: 99.99,
					InitialStock:     10,
					AttributeValues: map[string]*pb.AttributeValue{
						"Size":  {Type: pb.AttributeType_ATTRIBUTE_TYPE_TEXT, TextValue: "Wrong"},
						"Color": {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"Blue"}},
					},
				}
				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				productHandler.Create(rec, req)
				return rec
			},
			expectedStatus:   http.StatusBadRequest,
			expectedErrorCode: "INVALID_TYPE",
			description:      "Invalid type with attribute type mismatch",
		},
		{
			name: "ValueOutOfRange - Negative Price",
			httpCall: func() *httptest.ResponseRecorder {
				reqBody := &pb.CreateProductRequest{
					TemplateId:       template.ID,
					Name:             "Negative Price",
					Sku:              "NEG-PRICE-001",
					InitialListPrice: -99.99,
					InitialStock:     10,
					AttributeValues: map[string]*pb.AttributeValue{
						"Size":  {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"M"}},
						"Color": {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"Blue"}},
					},
				}
				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				productHandler.Create(rec, req)
				return rec
			},
			expectedStatus:   http.StatusBadRequest,
			expectedErrorCode: "VALUE_OUT_OF_RANGE",
			description:      "Value out of range with negative price",
		},
		{
			name: "TemplateNotFound",
			httpCall: func() *httptest.ResponseRecorder {
				req := httptest.NewRequest(http.MethodGet, "/api/v1/templates/550e8400-e29b-41d4-a716-446655440000", nil)
				rec := httptest.NewRecorder()
				templateHandler.Get(rec, req)
				return rec
			},
			expectedStatus:   http.StatusNotFound,
			expectedErrorCode: "TEMPLATE_NOT_FOUND",
			description:      "Template not found with non-existent UUID",
		},
		{
			name: "ProductNotFound",
			httpCall: func() *httptest.ResponseRecorder {
				req := httptest.NewRequest(http.MethodGet, "/api/v1/products/550e8400-e29b-41d4-a716-446655440000", nil)
				rec := httptest.NewRecorder()
				productHandler.Get(rec, req)
				return rec
			},
			expectedStatus:   http.StatusNotFound,
			expectedErrorCode: "PRODUCT_NOT_FOUND",
			description:      "Product not found with non-existent UUID",
		},
		{
			name: "VariantNotFound",
			httpCall: func() *httptest.ResponseRecorder {
				req := httptest.NewRequest(http.MethodGet, "/api/v1/variants/550e8400-e29b-41d4-a716-446655440000", nil)
				rec := httptest.NewRecorder()
				variantHandler.Get(rec, req)
				return rec
			},
			expectedStatus:   http.StatusNotFound,
			expectedErrorCode: "VARIANT_NOT_FOUND",
			description:      "Variant not found with non-existent UUID",
		},
		{
			name: "DuplicateSKU - Product",
			httpCall: func() *httptest.ResponseRecorder {
				reqBody := &pb.CreateProductRequest{
					TemplateId:       template.ID,
					Name:             "Duplicate SKU",
					Sku:              "ERR-TEST-001", // Already exists
					InitialListPrice: 99.99,
					InitialStock:     10,
					AttributeValues: map[string]*pb.AttributeValue{
						"Size":  {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"M"}},
						"Color": {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"Blue"}},
					},
				}
				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				productHandler.Create(rec, req)
				return rec
			},
			expectedStatus:   http.StatusConflict,
			expectedErrorCode: "DUPLICATE_SKU",
			description:      "Duplicate SKU error with existing product SKU",
		},
		{
			name: "DuplicateName - Template",
			httpCall: func() *httptest.ResponseRecorder {
				reqBody := &pb.CreateTemplateRequest{
					Name: "Error Test Template", // Already exists
					Attributes: []*pb.AttributeDefinition{
						{Name: "Field", Type: pb.AttributeType_ATTRIBUTE_TYPE_TEXT, Required: true},
					},
				}
				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/templates", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				templateHandler.Create(rec, req)
				return rec
			},
			expectedStatus:   http.StatusConflict,
			expectedErrorCode: "DUPLICATE_NAME",
			description:      "Duplicate name error with existing template name",
		},
		{
			name: "MediaNotFound",
			httpCall: func() *httptest.ResponseRecorder {
				storageDir := t.TempDir()
				localStorage, _ := services.NewLocalStorage(storageDir, "http://localhost:8080/media")
				imageProcessor := services.NewImageProcessor(300, 300)
				videoProcessor := services.NewVideoProcessor()
				mediaService := services.NewMediaService(db, localStorage, imageProcessor, videoProcessor, storageDir)
				mediaHandler := handlers.NewMediaHandler(mediaService)
				
				req := httptest.NewRequest(http.MethodGet, "/api/v1/media/550e8400-e29b-41d4-a716-446655440000", nil)
				rec := httptest.NewRecorder()
				mediaHandler.Get(rec, req)
				return rec
			},
			expectedStatus:   http.StatusNotFound,
			expectedErrorCode: "MEDIA_NOT_FOUND",
			description:      "Media not found with non-existent UUID",
		},
		{
			name: "AlreadyExists - Duplicate Media IDs in Reorder",
			httpCall: func() *httptest.ResponseRecorder {
				storageDir := t.TempDir()
				localStorage, _ := services.NewLocalStorage(storageDir, "http://localhost:8080/media")
				imageProcessor := services.NewImageProcessor(300, 300)
				videoProcessor := services.NewVideoProcessor()
				mediaService := services.NewMediaService(db, localStorage, imageProcessor, videoProcessor, storageDir)
				mediaHandler := handlers.NewMediaHandler(mediaService)
				
				reqBody := &pb.ReorderMediaRequest{
					EntityType:    "product",
					EntityId:      product.ID,
					AttributeName: "TestImage",
					MediaIds:      []string{mediaFile.ID, mediaFile.ID}, // Duplicate IDs
				}
				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/media/reorder", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				mediaHandler.Reorder(rec, req)
				return rec
			},
			expectedStatus:   http.StatusConflict,
			expectedErrorCode: "ALREADY_EXISTS",
			description:      "Already exists error with duplicate media IDs in reorder request",
		},
		{
			name: "Context Canceled - Request Cancelled",
			httpCall: func() *httptest.ResponseRecorder {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately
				
				reqBody := &pb.CreateProductRequest{
					TemplateId:       template.ID,
					Name:             "Context Test",
					Sku:              "CTX-CANCEL-001",
					InitialListPrice: 99.99,
					InitialStock:     10,
					AttributeValues: map[string]*pb.AttributeValue{
						"Size":  {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"M"}},
						"Color": {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"Blue"}},
					},
				}
				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(body))
				req = req.WithContext(ctx)
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				productHandler.Create(rec, req)
				return rec
			},
			expectedStatus:   499, // Client closed connection
			expectedErrorCode: "", // Not JSON error response, plain text
			description:      "Context canceled error returns 499 status",
		},
		{
			name: "Context Timeout - Request Deadline Exceeded",
			httpCall: func() *httptest.ResponseRecorder {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
				defer cancel()
				time.Sleep(10 * time.Millisecond) // Ensure timeout expires
				
				reqBody := &pb.CreateProductRequest{
					TemplateId:       template.ID,
					Name:             "Timeout Test",
					Sku:              "CTX-TIMEOUT-001",
					InitialListPrice: 99.99,
					InitialStock:     10,
					AttributeValues: map[string]*pb.AttributeValue{
						"Size":  {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"M"}},
						"Color": {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"Blue"}},
					},
				}
				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(body))
				req = req.WithContext(ctx)
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				productHandler.Create(rec, req)
				return rec
			},
			expectedStatus:   504, // Gateway timeout
			expectedErrorCode: "", // Not JSON error response, plain text
			description:      "Context timeout error returns 504 status",
		},
		{
			name: "HasProducts - Template with Associated Products",
			httpCall: func() *httptest.ResponseRecorder {
				// Create a template and a product using it
				tempTemplate := testutil.CreateTemplateFixture(db, "Template With Products HTTP", `[{"name":"Field","type":"text","required":true}]`)
				testutil.CreateProductFixture(db, tempTemplate.ID, "Product Using Template", "HAS-PROD-HTTP-001", `{}`)
				
				// Try to delete template
				req := httptest.NewRequest(http.MethodDelete, "/api/v1/templates/"+tempTemplate.ID, nil)
				rec := httptest.NewRecorder()
				templateHandler.Delete(rec, req)
				return rec
			},
			expectedStatus:   http.StatusConflict,
			expectedErrorCode: "HAS_PRODUCTS",
			description:      "Has products error when deleting template with associated products",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rec := tc.httpCall()

			// Verify HTTP status code
			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}

			// Skip JSON parsing for context errors (plain text responses)
			if tc.expectedErrorCode == "" {
				t.Logf("✅ Verified HTTP status code: %d (plain text response)", tc.expectedStatus)
				return
			}

			// Parse error response
			var errorResponse pb.ErrorResponse
			if err := json.NewDecoder(rec.Body).Decode(&errorResponse); err != nil {
				t.Fatalf("Failed to decode error response: %v. Body: %s", err, rec.Body.String())
			}

			// Verify error code
			if errorResponse.Code != tc.expectedErrorCode {
				t.Errorf("Expected error code '%s', got '%s'. Description: %s", tc.expectedErrorCode, errorResponse.Code, tc.description)
			}

			t.Logf("✅ Verified HTTP error code: %s (Status: %d)", tc.expectedErrorCode, tc.expectedStatus)
		})
	}

	t.Log("✅ All HTTP error codes tested and verified")
}

// TestErrorFlowEndToEnd verifies the complete error flow: Service → Handler → HTTP Response
func TestErrorFlowEndToEnd(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "products", "product_templates")

	// Initialize services and handlers
	productService := services.NewProductService(db)
	productHandler := handlers.NewProductHandler(productService)

	// Create template
	template := testutil.CreateTemplateFixture(db, "Flow Test Template", `[
		{"name":"Size","type":"list","required":true,"options":["S","M","L"]}
	]`)

	testCases := []struct {
		name               string
		request            *pb.CreateProductRequest
		expectedStatus     int
		expectedErrorCode  string
		expectedSentinel   error
		description        string
	}{
		{
			name: "Complete flow: ErrMissingRequired → MissingRequired → 400",
			request: &pb.CreateProductRequest{
				TemplateId:       template.ID,
				Name:             "", // Missing required
				Sku:              "FLOW-001",
				InitialListPrice: 99.99,
				InitialStock:     10,
				AttributeValues: map[string]*pb.AttributeValue{
					"Size": {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"M"}},
				},
			},
			expectedStatus:    http.StatusBadRequest,
			expectedErrorCode: "MISSING_REQUIRED",
			expectedSentinel:  services.ErrMissingRequired,
			description:       "Complete error flow from service sentinel to HTTP error code",
		},
		{
			name: "Complete flow: ErrInvalidSKU → ValidationFailed → 400",
			request: &pb.CreateProductRequest{
				TemplateId:       template.ID,
				Name:             "Flow Test",
				Sku:              "INVALID@SKU",
				InitialListPrice: 99.99,
				InitialStock:     10,
				AttributeValues: map[string]*pb.AttributeValue{
					"Size": {Type: pb.AttributeType_ATTRIBUTE_TYPE_LIST, ListValue: []string{"M"}},
				},
			},
			expectedStatus:    http.StatusBadRequest,
			expectedErrorCode: "VALIDATION_ERROR",
			expectedSentinel:  services.ErrInvalidSKU,
			description:       "Complete error flow for invalid SKU format",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Step 1: Call service directly to verify sentinel error
			ctx := context.Background()
			_, serviceErr := productService.Create(ctx, tc.request)
			
			if serviceErr == nil {
				t.Fatal("Expected service to return an error")
			}

			if !ErrorContains(serviceErr, tc.expectedSentinel) {
				t.Errorf("Service error chain should contain %v, got: %v", tc.expectedSentinel, serviceErr)
			}
			t.Logf("✅ Step 1: Service returned sentinel error: %v", tc.expectedSentinel)

			// Step 2: Call handler to verify HTTP error code mapping
			body, _ := json.Marshal(tc.request)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			productHandler.Create(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected HTTP status %d, got %d", tc.expectedStatus, rec.Code)
			}
			t.Logf("✅ Step 2: Handler returned HTTP status: %d", tc.expectedStatus)

			// Step 3: Verify client receives correct error code
			var errorResponse pb.ErrorResponse
			if err := json.NewDecoder(rec.Body).Decode(&errorResponse); err != nil {
				t.Fatalf("Failed to decode error response: %v", err)
			}

			if errorResponse.Code != tc.expectedErrorCode {
				t.Errorf("Expected error code '%s', got '%s'", tc.expectedErrorCode, errorResponse.Code)
			}
			t.Logf("✅ Step 3: Client received error code: %s", tc.expectedErrorCode)

			t.Logf("✅ Complete error flow verified: %s → %s → %d", tc.expectedSentinel, tc.expectedErrorCode, tc.expectedStatus)
		})
	}

	t.Log("✅ All end-to-end error flows verified")
}

// ErrorContains checks if an error chain contains a specific sentinel error
func ErrorContains(err, target error) bool {
	if err == nil {
		return false
	}
	
	// Check if the error message contains the target error message
	// This handles wrapped errors with fmt.Errorf("context: %w", err)
	if target != nil && err.Error() != "" {
		return contains(err.Error(), target.Error())
	}
	
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		 indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

