package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	pb "github.com/sunfmin/apidemo2/backend/api/gen/pim/v1"
	"github.com/sunfmin/apidemo2/backend/internal/models"
)

// VariantService defines the interface for variant management operations
type VariantService interface {
	Create(ctx context.Context, req *pb.CreateVariantRequest) (*pb.ProductVariant, error)
	Get(ctx context.Context, id string) (*pb.ProductVariant, error)
	List(ctx context.Context, productID string) ([]*pb.ProductVariant, error)
	Update(ctx context.Context, req *pb.UpdateVariantRequest) (*pb.ProductVariant, error)
	Delete(ctx context.Context, id string) error
	BulkCreate(ctx context.Context, req *pb.BulkCreateVariantsRequest) (*pb.BulkCreateVariantsResponse, error)
}

type variantService struct {
	db *gorm.DB
}

// NewVariantService creates a new VariantService
func NewVariantService(db *gorm.DB) VariantService {
	return &variantService{db: db}
}

// Create creates a new product variant
func (s *variantService) Create(ctx context.Context, req *pb.CreateVariantRequest) (*pb.ProductVariant, error) {
	// Check context cancellation before starting (Principle X)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Verify product exists and load it with template
	var product models.Product
	if err := s.db.WithContext(ctx).Preload("Template").First(&product, "id = ?", req.ProductId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("get product %s: %w", req.ProductId, ErrProductNotFound)
		}
		return nil, fmt.Errorf("query product %s with template: %w", req.ProductId, err)
	}

	// Validate request
	if err := s.validateCreateRequest(req); err != nil {
		return nil, fmt.Errorf("validate create variant request: %w", err)
	}

	// Validate attribute overrides against template
	if err := s.validateAttributeOverrides(req.AttributeValues, product.Template.Attributes); err != nil {
		return nil, fmt.Errorf("validate attribute overrides for variant: %w", err)
	}

	// Convert attribute overrides to JSON
	attrJSON, err := s.protoAttributesToJSON(req.AttributeValues)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize attributes: %w", err)
	}

	// Create variant model
	variant := &models.ProductVariant{
		ProductID:       req.ProductId,
		Name:            req.Name,
		SKU:             req.Sku,
		AttributeValues: attrJSON,
	}

	// Use transaction to create variant, pricing, and inventory atomically
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	// Save variant
	if err := tx.Create(variant).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") && strings.Contains(err.Error(), "sku") {
			return nil, fmt.Errorf("create variant with SKU %s: %w", req.Sku, ErrDuplicateSKU)
		}
		return nil, fmt.Errorf("create variant in database (name=%s, sku=%s): %w", req.Name, req.Sku, err)
	}

	// Create variant pricing
	pricing := &models.VariantPricing{
		VariantID: variant.ID,
		ListPrice: req.InitialPrice,
		ValidFrom: time.Now(),
	}
	if err := tx.Create(pricing).Error; err != nil {
		return nil, fmt.Errorf("failed to create variant pricing: %w", err)
	}

	// Create variant inventory
	inventory := &models.VariantInventory{
		VariantID:      variant.ID,
		LocationID:     "default",
		OnHandQuantity: req.InitialStock,
	}
	if err := tx.Create(inventory).Error; err != nil {
		return nil, fmt.Errorf("failed to create variant inventory: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("commit variant creation transaction (sku=%s): %w", req.Sku, err)
	}

	// Load variant with pricing and inventory
	variant.Pricing = pricing
	variant.Inventories = []models.VariantInventory{*inventory}

	// Convert to protobuf with attribute merging
	return s.modelToProto(variant, &product)
}

// Get retrieves a variant by ID
func (s *variantService) Get(ctx context.Context, id string) (*pb.ProductVariant, error) {
	var variant models.ProductVariant
	if err := s.db.WithContext(ctx).
		Preload("Pricing").
		Preload("Inventories").
		First(&variant, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("get variant %s: %w", id, ErrVariantNotFound)
		}
		return nil, fmt.Errorf("query variant %s: %w", id, err)
	}

	// Load parent product with template for attribute merging
	var product models.Product
	if err := s.db.WithContext(ctx).Preload("Template").First(&product, "id = ?", variant.ProductID).Error; err != nil {
		return nil, fmt.Errorf("query parent product %s: %w", variant.ProductID, err)
	}

	return s.modelToProto(&variant, &product)
}

// List retrieves all variants for a product
func (s *variantService) List(ctx context.Context, productID string) ([]*pb.ProductVariant, error) {
	// Verify product exists
	var product models.Product
	if err := s.db.WithContext(ctx).Preload("Template").First(&product, "id = ?", productID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("get product %s: %w", productID, ErrProductNotFound)
		}
		return nil, fmt.Errorf("query product %s with template: %w", productID, err)
	}

	// Load variants
	var variants []models.ProductVariant
	if err := s.db.WithContext(ctx).
		Preload("Pricing").
		Preload("Inventories").
		Where("product_id = ?", productID).
		Order("created_at DESC").
		Find(&variants).Error; err != nil {
		return nil, fmt.Errorf("query variants for product %s: %w", productID, err)
	}

	// Convert to protobuf
	pbVariants := make([]*pb.ProductVariant, 0, len(variants))
	for _, variant := range variants {
		pbVariant, err := s.modelToProto(&variant, &product)
		if err != nil {
			return nil, fmt.Errorf("convert variant %s to proto: %w", variant.ID, err)
		}
		pbVariants = append(pbVariants, pbVariant)
	}

	return pbVariants, nil
}

// Update updates a variant
func (s *variantService) Update(ctx context.Context, req *pb.UpdateVariantRequest) (*pb.ProductVariant, error) {
	// Load variant
	var variant models.ProductVariant
	if err := s.db.WithContext(ctx).First(&variant, "id = ?", req.Id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("get variant %s: %w", req.Id, ErrVariantNotFound)
		}
		return nil, fmt.Errorf("query variant %s: %w", req.Id, err)
	}

	// Load parent product with template for validation
	var product models.Product
	if err := s.db.WithContext(ctx).Preload("Template").First(&product, "id = ?", variant.ProductID).Error; err != nil {
		return nil, fmt.Errorf("query parent product %s: %w", variant.ProductID, err)
	}

	// Update fields
	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Sku != "" {
		updates["sku"] = req.Sku
	}
	if len(req.AttributeValues) > 0 {
		// Validate attribute overrides against template
		if err := s.validateAttributeOverrides(req.AttributeValues, product.Template.Attributes); err != nil {
			return nil, fmt.Errorf("validate attribute overrides: %w", err)
		}

		attrJSON, err := s.protoAttributesToJSON(req.AttributeValues)
		if err != nil {
			return nil, fmt.Errorf("serialize attribute overrides: %w", err)
		}
		updates["attribute_values"] = attrJSON
	}

	// Apply updates
	if len(updates) > 0 {
		if err := s.db.WithContext(ctx).Model(&variant).Updates(updates).Error; err != nil {
			if strings.Contains(err.Error(), "duplicate key") {
				return nil, fmt.Errorf("update variant %s with SKU %s: %w", req.Id, req.Sku, ErrDuplicateSKU)
			}
			return nil, fmt.Errorf("update variant %s: %w", req.Id, err)
		}
	}

	// Reload with relationships
	if err := s.db.WithContext(ctx).
		Preload("Pricing").
		Preload("Inventories").
		First(&variant, "id = ?", req.Id).Error; err != nil {
		return nil, fmt.Errorf("reload variant %s after update: %w", req.Id, err)
	}

	// Product is already loaded with Template from earlier, use it for conversion
	return s.modelToProto(&variant, &product)
}

// Delete deletes a variant
func (s *variantService) Delete(ctx context.Context, id string) error {
	// Check if variant exists
	var variant models.ProductVariant
	if err := s.db.WithContext(ctx).First(&variant, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("get variant %s: %w", id, ErrVariantNotFound)
		}
		return fmt.Errorf("query variant %s: %w", id, err)
	}

	// Use transaction to delete variant, pricing, and inventory
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	// Delete variant pricing
	if err := tx.Where("variant_id = ?", id).Delete(&models.VariantPricing{}).Error; err != nil {
		return fmt.Errorf("failed to delete variant pricing: %w", err)
	}

	// Delete variant inventory
	if err := tx.Where("variant_id = ?", id).Delete(&models.VariantInventory{}).Error; err != nil {
		return fmt.Errorf("failed to delete variant inventory: %w", err)
	}

	// Delete variant
	if err := tx.Delete(&variant).Error; err != nil {
		return fmt.Errorf("delete variant %s: %w", id, err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("commit variant deletion (id=%s): %w", id, err)
	}

	return nil
}

// BulkCreate creates multiple variants at once with transaction support
func (s *variantService) BulkCreate(ctx context.Context, req *pb.BulkCreateVariantsRequest) (*pb.BulkCreateVariantsResponse, error) {
	// Validate request
	if req.ProductId == "" {
		return nil, fmt.Errorf("product_id: %w", ErrMissingRequired)
	}
	if len(req.Variants) == 0 {
		return nil, fmt.Errorf("variants array: %w", ErrMissingRequired)
	}

	// Verify product exists and load template
	var product models.Product
	if err := s.db.WithContext(ctx).Preload("Template").First(&product, "id = ?", req.ProductId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("get product %s: %w", req.ProductId, ErrProductNotFound)
		}
		return nil, fmt.Errorf("query product %s with template: %w", req.ProductId, err)
	}

	response := &pb.BulkCreateVariantsResponse{
		Variants: make([]*pb.ProductVariant, 0),
		Errors:   make([]*pb.BulkCreateError, 0),
	}

	// Use transaction for atomic creation
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	// Track SKUs to detect duplicates within the batch
	skusSeen := make(map[string]int)

	for i, variantInput := range req.Variants {
		// Validate individual variant
		if err := s.validateVariantInput(variantInput, i, skusSeen); err != nil {
			response.Errors = append(response.Errors, &pb.BulkCreateError{
				Index:        int32(i),
				Sku:          variantInput.Sku,
				ErrorMessage: err.Error(),
			})
			continue
		}

		// Validate attribute overrides against template
		if err := s.validateAttributeOverrides(variantInput.AttributeValues, product.Template.Attributes); err != nil {
			response.Errors = append(response.Errors, &pb.BulkCreateError{
				Index:        int32(i),
				Sku:          variantInput.Sku,
				ErrorMessage: err.Error(),
			})
			continue
		}

		// Convert attributes to JSON
		attrJSON, err := s.protoAttributesToJSON(variantInput.AttributeValues)
		if err != nil {
			response.Errors = append(response.Errors, &pb.BulkCreateError{
				Index:        int32(i),
				Sku:          variantInput.Sku,
				ErrorMessage: fmt.Sprintf("failed to serialize attributes: %v", err),
			})
			continue
		}

		// Create variant model
		variant := &models.ProductVariant{
			ProductID:       req.ProductId,
			Name:            variantInput.Name,
			SKU:             variantInput.Sku,
			AttributeValues: attrJSON,
		}

		// Save variant
		if err := tx.Create(variant).Error; err != nil {
			if strings.Contains(err.Error(), "duplicate key") && strings.Contains(err.Error(), "sku") {
				response.Errors = append(response.Errors, &pb.BulkCreateError{
					Index:        int32(i),
					Sku:          variantInput.Sku,
					ErrorMessage: fmt.Sprintf("SKU already exists: %s", variantInput.Sku),
				})
			} else {
				response.Errors = append(response.Errors, &pb.BulkCreateError{
					Index:        int32(i),
					Sku:          variantInput.Sku,
					ErrorMessage: fmt.Sprintf("database error: %v", err),
				})
			}
			continue
		}

		// Create variant pricing with default values
		pricing := &models.VariantPricing{
			VariantID: variant.ID,
			ListPrice: 0.0,
			ValidFrom: time.Now(),
		}
		if err := tx.Create(pricing).Error; err != nil {
			response.Errors = append(response.Errors, &pb.BulkCreateError{
				Index:        int32(i),
				Sku:          variantInput.Sku,
				ErrorMessage: fmt.Sprintf("failed to create pricing: %v", err),
			})
			// Delete the variant to maintain consistency
			tx.Delete(variant)
			continue
		}

		// Create variant inventory with default values
		inventory := &models.VariantInventory{
			VariantID:      variant.ID,
			LocationID:     "default",
			OnHandQuantity: 0,
		}
		if err := tx.Create(inventory).Error; err != nil {
			response.Errors = append(response.Errors, &pb.BulkCreateError{
				Index:        int32(i),
				Sku:          variantInput.Sku,
				ErrorMessage: fmt.Sprintf("failed to create inventory: %v", err),
			})
			// Delete the variant and pricing to maintain consistency
			tx.Delete(pricing)
			tx.Delete(variant)
			continue
		}

		// Load relationships
		variant.Pricing = pricing
		variant.Inventories = []models.VariantInventory{*inventory}

		// Convert to protobuf
		pbVariant, err := s.modelToProto(variant, &product)
		if err != nil {
			response.Errors = append(response.Errors, &pb.BulkCreateError{
				Index:        int32(i),
				Sku:          variantInput.Sku,
				ErrorMessage: fmt.Sprintf("failed to convert to protobuf: %v", err),
			})
			continue
		}

		response.Variants = append(response.Variants, pbVariant)
	}

	// Commit transaction if at least one variant was created successfully
	if len(response.Variants) > 0 {
		if err := tx.Commit().Error; err != nil {
			return nil, fmt.Errorf("failed to commit transaction: %w", err)
		}
	} else {
		// All variants failed, rollback
		tx.Rollback()
	}

	return response, nil
}

// validateVariantInput validates a single variant input for bulk creation
func (s *variantService) validateVariantInput(input *pb.VariantInput, index int, skusSeen map[string]int) error {
	if input.Name == "" {
		return fmt.Errorf("variant name: %w", ErrMissingRequired)
	}
	if len(input.Name) > 255 {
		return fmt.Errorf("variant name length %d: %w", len(input.Name), ErrValueOutOfRange)
	}
	if input.Sku == "" {
		return fmt.Errorf("SKU: %w", ErrMissingRequired)
	}
	if len(input.Sku) > 100 {
		return fmt.Errorf("SKU length %d: %w", len(input.Sku), ErrValueOutOfRange)
	}

	// Validate SKU format (alphanumeric, dash, underscore)
	for _, char := range input.Sku {
		if !(char >= 'a' && char <= 'z') && !(char >= 'A' && char <= 'Z') &&
			!(char >= '0' && char <= '9') && char != '-' && char != '_' {
			return fmt.Errorf("SKU contains invalid character: %c", char)
		}
	}

	// Check for duplicate SKU within the batch
	if prevIndex, exists := skusSeen[input.Sku]; exists {
		return fmt.Errorf("duplicate SKU in batch (also at index %d): %s", prevIndex, input.Sku)
	}
	skusSeen[input.Sku] = index

	return nil
}

// validateCreateRequest validates variant creation request
func (s *variantService) validateCreateRequest(req *pb.CreateVariantRequest) error {
	if req.ProductId == "" {
		return fmt.Errorf("product_id: %w", ErrMissingRequired)
	}
	if req.Name == "" {
		return fmt.Errorf("variant name: %w", ErrMissingRequired)
	}
	if len(req.Name) > 255 {
		return fmt.Errorf("variant name length %d: %w", len(req.Name), ErrValueOutOfRange)
	}
	if req.Sku == "" {
		return fmt.Errorf("SKU: %w", ErrMissingRequired)
	}
	if len(req.Sku) > 100 {
		return fmt.Errorf("SKU length %d: %w", len(req.Sku), ErrValueOutOfRange)
	}

	// Validate SKU format (alphanumeric, dash, underscore)
	for _, char := range req.Sku {
		if !(char >= 'a' && char <= 'z') && !(char >= 'A' && char <= 'Z') &&
			!(char >= '0' && char <= '9') && char != '-' && char != '_' {
			return fmt.Errorf("SKU contains invalid character '%c': %w", char, ErrInvalidSKU)
		}
	}

	if req.InitialPrice < 0 {
		return fmt.Errorf("initial_price %.2f: %w", req.InitialPrice, ErrValueOutOfRange)
	}

	if req.InitialStock < 0 {
		return fmt.Errorf("initial_stock %d: %w", req.InitialStock, ErrValueOutOfRange)
	}

	return nil
}

// validateAttributeOverrides validates variant attribute overrides against template
func (s *variantService) validateAttributeOverrides(values map[string]*pb.AttributeValue, templateAttrsJSON []byte) error {
	// If no overrides, nothing to validate
	if len(values) == 0 {
		return nil
	}

	// Parse template attributes
	var templateAttrs []models.AttributeDefinition
	if err := json.Unmarshal(templateAttrsJSON, &templateAttrs); err != nil {
		return fmt.Errorf("failed to parse template attributes: %w", err)
	}

	// Build attribute definition map
	attrDefs := make(map[string]models.AttributeDefinition)
	for _, attr := range templateAttrs {
		attrDefs[attr.Name] = attr
	}

	// Validate each override
	for name, value := range values {
		// Check if attribute exists in template
		attrDef, ok := attrDefs[name]
		if !ok {
			return fmt.Errorf("attribute '%s': %w", name, ErrInvalidRequest)
		}

		// Validate type matches
		expectedType := s.stringToAttributeType(attrDef.Type)
		if value.Type != expectedType {
			return fmt.Errorf("attribute %s expected type %s got %s: %w", name, attrDef.Type, value.Type.String(), ErrInvalidType)
		}

		// For list type, validate values are in options
		if attrDef.Type == "list" && len(attrDef.Options) > 0 {
			if len(value.ListValue) > 0 {
				optionsSet := make(map[string]bool)
				for _, opt := range attrDef.Options {
					optionsSet[opt] = true
				}

				for _, val := range value.ListValue {
					if !optionsSet[val] {
						return fmt.Errorf("attribute %s value '%s' not in allowed options: %w", name, val, ErrInvalidRequest)
					}
				}
			}
		}
	}

	return nil
}

// protoAttributesToJSON converts protobuf AttributeValue map to JSONB
func (s *variantService) protoAttributesToJSON(attrs map[string]*pb.AttributeValue) ([]byte, error) {
	jsonAttrs := make(map[string]models.AttributeValue)

	for name, value := range attrs {
		attrValue := models.AttributeValue{
			Type: value.Type.String(),
		}

		switch value.Type {
		case pb.AttributeType_ATTRIBUTE_TYPE_TEXT:
			attrValue.Value = value.TextValue
		case pb.AttributeType_ATTRIBUTE_TYPE_NUMBER:
			attrValue.Value = value.NumberValue
		case pb.AttributeType_ATTRIBUTE_TYPE_BOOLEAN:
			attrValue.Value = value.BooleanValue
		case pb.AttributeType_ATTRIBUTE_TYPE_DATE:
			attrValue.Value = value.DateValue
		case pb.AttributeType_ATTRIBUTE_TYPE_LIST:
			if len(value.ListValue) > 0 {
				attrValue.Value = value.ListValue
			}
		case pb.AttributeType_ATTRIBUTE_TYPE_MAP:
			if len(value.MapValue) > 0 {
				attrValue.Value = value.MapValue
			}
		case pb.AttributeType_ATTRIBUTE_TYPE_IMAGE, pb.AttributeType_ATTRIBUTE_TYPE_VIDEO:
			if len(value.MediaIds) > 0 {
				attrValue.Value = value.MediaIds
			}
		}

		jsonAttrs[name] = attrValue
	}

	return json.Marshal(jsonAttrs)
}

// modelToProto converts variant model to protobuf with attribute merging
func (s *variantService) modelToProto(model *models.ProductVariant, product *models.Product) (*pb.ProductVariant, error) {
	// Parse variant's override attributes
	var variantAttrs map[string]models.AttributeValue
	if err := json.Unmarshal(model.AttributeValues, &variantAttrs); err != nil {
		return nil, fmt.Errorf("failed to parse variant attributes: %w", err)
	}

	// Parse product's attributes
	var productAttrs map[string]models.AttributeValue
	if err := json.Unmarshal(product.AttributeValues, &productAttrs); err != nil {
		return nil, fmt.Errorf("failed to parse product attributes: %w", err)
	}

	// Convert variant overrides to protobuf
	pbOverrides := make(map[string]*pb.AttributeValue)
	for name, value := range variantAttrs {
		pbOverrides[name] = s.attributeValueToProto(value)
	}

	// Merge product attributes with variant overrides for effective attributes
	pbEffective := make(map[string]*pb.AttributeValue)

	// Start with product attributes
	for name, value := range productAttrs {
		pbEffective[name] = s.attributeValueToProto(value)
	}

	// Override with variant-specific attributes
	for name, value := range variantAttrs {
		pbEffective[name] = s.attributeValueToProto(value)
	}

	// Get pricing and inventory
	var price, listPrice, salePrice float64
	var stockQuantity, availableQuantity int32

	if model.Pricing != nil {
		listPrice = model.Pricing.ListPrice
		salePrice = model.Pricing.SalePrice
		if salePrice > 0 {
			price = salePrice
		} else {
			price = listPrice
		}
	}

	for _, inv := range model.Inventories {
		stockQuantity += inv.OnHandQuantity
		availableQuantity += (inv.OnHandQuantity - inv.ReservedQuantity)
	}

	return &pb.ProductVariant{
		Id:                       model.ID,
		ProductId:                model.ProductID,
		ProductName:              product.Name,
		Name:                     model.Name,
		Sku:                      model.SKU,
		AttributeValues:          pbOverrides, // Only overrides
		EffectiveAttributeValues: pbEffective, // Merged parent + overrides
		Price:                    price,
		StockQuantity:            stockQuantity,
		AvailableQuantity:        availableQuantity,
		CreatedAt:                timestamppb.New(model.CreatedAt),
		UpdatedAt:                timestamppb.New(model.UpdatedAt),
		PrimaryImageUrls:         []string{}, // TODO: Load from media
	}, nil
}

// attributeValueToProto converts internal AttributeValue to protobuf
func (s *variantService) attributeValueToProto(value models.AttributeValue) *pb.AttributeValue {
	pbAttr := &pb.AttributeValue{
		Type: s.stringToAttributeType(value.Type),
	}

	switch value.Type {
	case "ATTRIBUTE_TYPE_TEXT", "text":
		if str, ok := value.Value.(string); ok {
			pbAttr.TextValue = str
		}
	case "ATTRIBUTE_TYPE_NUMBER", "number":
		if num, ok := value.Value.(float64); ok {
			pbAttr.NumberValue = num
		}
	case "ATTRIBUTE_TYPE_BOOLEAN", "boolean":
		if b, ok := value.Value.(bool); ok {
			pbAttr.BooleanValue = b
		}
	case "ATTRIBUTE_TYPE_DATE", "date":
		if str, ok := value.Value.(string); ok {
			pbAttr.DateValue = str
		}
	case "ATTRIBUTE_TYPE_LIST", "list":
		if list, ok := value.Value.([]interface{}); ok {
			strList := make([]string, 0, len(list))
			for _, v := range list {
				if str, ok := v.(string); ok {
					strList = append(strList, str)
				}
			}
			pbAttr.ListValue = strList
		}
	case "ATTRIBUTE_TYPE_MAP", "map":
		if mapVal, ok := value.Value.(map[string]interface{}); ok {
			strMap := make(map[string]string)
			for k, v := range mapVal {
				if str, ok := v.(string); ok {
					strMap[k] = str
				}
			}
			pbAttr.MapValue = strMap
		}
	case "ATTRIBUTE_TYPE_IMAGE", "ATTRIBUTE_TYPE_VIDEO", "image", "video":
		if list, ok := value.Value.([]interface{}); ok {
			strList := make([]string, 0, len(list))
			for _, v := range list {
				if str, ok := v.(string); ok {
					strList = append(strList, str)
				}
			}
			pbAttr.MediaIds = strList
		}
	}

	return pbAttr
}

// stringToAttributeType converts string type to protobuf AttributeType
func (s *variantService) stringToAttributeType(typeStr string) pb.AttributeType {
	upperType := strings.ToUpper(typeStr)
	switch {
	case strings.Contains(upperType, "TEXT"):
		return pb.AttributeType_ATTRIBUTE_TYPE_TEXT
	case strings.Contains(upperType, "NUMBER"):
		return pb.AttributeType_ATTRIBUTE_TYPE_NUMBER
	case strings.Contains(upperType, "BOOLEAN"):
		return pb.AttributeType_ATTRIBUTE_TYPE_BOOLEAN
	case strings.Contains(upperType, "DATE"):
		return pb.AttributeType_ATTRIBUTE_TYPE_DATE
	case strings.Contains(upperType, "LIST"):
		return pb.AttributeType_ATTRIBUTE_TYPE_LIST
	case strings.Contains(upperType, "MAP"):
		return pb.AttributeType_ATTRIBUTE_TYPE_MAP
	case strings.Contains(upperType, "IMAGE"):
		return pb.AttributeType_ATTRIBUTE_TYPE_IMAGE
	case strings.Contains(upperType, "VIDEO"):
		return pb.AttributeType_ATTRIBUTE_TYPE_VIDEO
	default:
		return pb.AttributeType_ATTRIBUTE_TYPE_UNSPECIFIED
	}
}
