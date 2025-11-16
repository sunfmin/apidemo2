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

// ProductService defines the interface for product management operations
type ProductService interface {
	Create(ctx context.Context, req *pb.CreateProductRequest) (*pb.Product, error)
	Get(ctx context.Context, req *pb.GetProductRequest) (*pb.GetProductResponse, error)
	List(ctx context.Context, req *pb.ListProductsRequest) ([]*pb.Product, *pb.PaginationResponse, error)
	Update(ctx context.Context, req *pb.UpdateProductRequest) (*pb.Product, error)
	Delete(ctx context.Context, req *pb.DeleteProductRequest) (*pb.DeleteProductResponse, error)
	BulkUpdateStatus(ctx context.Context, req *pb.BulkUpdateStatusRequest) (*pb.BulkUpdateStatusResponse, error)
}

// productService implements the ProductService interface
type productService struct {
	db *gorm.DB
}

// NewProductService creates a new ProductService instance
func NewProductService(db *gorm.DB) ProductService {
	return &productService{db: db}
}

// Create creates a new product
func (s *productService) Create(ctx context.Context, req *pb.CreateProductRequest) (*pb.Product, error) {
	// Verify template exists and load it FIRST (before validation)
	var template models.ProductTemplate
	if err := s.db.WithContext(ctx).First(&template, "id = ?", req.TemplateId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("template not found: %s", req.TemplateId)
		}
		return nil, err
	}

	// Validate request (after template is confirmed to exist)
	if err := s.validateCreateRequest(req); err != nil {
		return nil, err
	}

	// Validate attribute values against template
	if err := s.validateAttributeValues(req.AttributeValues, template.Attributes); err != nil {
		return nil, err
	}

	// Convert attribute values to JSONB
	attrJSON, err := s.protoAttributesToJSON(req.AttributeValues)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize attributes: %w", err)
	}

	// Create product model (master data only, no price/stock)
	product := &models.Product{
		TemplateID:      req.TemplateId,
		Name:            req.Name,
		SKU:             req.Sku,
		Description:     req.Description,
		AttributeValues: attrJSON,
		Status:          s.protoStatusToString(req.Status),
	}

	// Use transaction to create product, pricing, and inventory atomically
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	// Save product (master data)
	if err := tx.Create(product).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") && strings.Contains(err.Error(), "sku") {
			return nil, fmt.Errorf("SKU already exists: %s", req.Sku)
		}
		return nil, err
	}

	// Create pricing record (Marketing domain)
	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}
	pricing := &models.ProductPricing{
		ProductID: product.ID,
		ListPrice: req.InitialListPrice,
		SalePrice: req.InitialSalePrice,
		Cost:      req.InitialCost,
		Currency:  currency,
		ValidFrom: time.Now(),
	}
	if err := tx.Create(pricing).Error; err != nil {
		return nil, fmt.Errorf("failed to create pricing: %w", err)
	}

	// Create inventory record (Operations domain)
	location := req.InitialLocation
	if location == "" {
		location = "default"
	}
	inventory := &models.ProductInventory{
		ProductID:      product.ID,
		LocationID:     location,
		OnHandQuantity: req.InitialStock,
	}
	if err := tx.Create(inventory).Error; err != nil {
		return nil, fmt.Errorf("failed to create inventory: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// Load product with pricing and inventory for response
	product.Pricing = pricing
	product.Inventories = []models.ProductInventory{*inventory}

	// Convert to protobuf response
	return s.modelToProto(product, template.Name)
}

// Get retrieves a product by ID
func (s *productService) Get(ctx context.Context, req *pb.GetProductRequest) (*pb.GetProductResponse, error) {
	if req.Id == "" {
		return nil, errors.New("product ID is required")
	}

	var product models.Product
	query := s.db.WithContext(ctx).Preload("Template").Preload("Pricing").Preload("Inventories")

	if err := query.First(&product, "id = ?", req.Id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("product not found: %s", req.Id)
		}
		return nil, err
	}

	response := &pb.GetProductResponse{
		Product: &pb.Product{},
	}

	var err error
	response.Product, err = s.modelToProto(&product, product.Template.Name)
	if err != nil {
		return nil, err
	}

	// TODO: Load variants if include_variants = true
	// TODO: Load media if include_media = true

	return response, nil
}

// List retrieves products with pagination, filtering, and sorting
func (s *productService) List(ctx context.Context, req *pb.ListProductsRequest) ([]*pb.Product, *pb.PaginationResponse, error) {
	query := s.db.WithContext(ctx).Model(&models.Product{}).Preload("Template")

	// Apply filters
	if req.TemplateId != "" {
		query = query.Where("template_id = ?", req.TemplateId)
	}

	if req.Status != pb.ProductStatus_PRODUCT_STATUS_UNSPECIFIED {
		query = query.Where("status = ?", s.protoStatusToString(req.Status))
	}

	if req.Search != "" {
		searchPattern := "%" + req.Search + "%"
		query = query.Where("name ILIKE ? OR sku ILIKE ?", searchPattern, searchPattern)
	}

	// TODO: Implement attribute_filters for JSONB queries

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, nil, err
	}

	// Apply sorting
	sortBy := req.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortOrder := req.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}
	query = query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))

	// Apply pagination
	page := req.Pagination.GetPage()
	if page < 1 {
		page = 1
	}
	pageSize := req.Pagination.GetPageSize()
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize
	query = query.Limit(int(pageSize)).Offset(int(offset))

	// Execute query
	var products []models.Product
	if err := query.Find(&products).Error; err != nil {
		return nil, nil, err
	}

	// Convert to protobuf
	pbProducts := make([]*pb.Product, 0, len(products))
	for i := range products {
		pbProduct, err := s.modelToProto(&products[i], products[i].Template.Name)
		if err != nil {
			return nil, nil, err
		}
		pbProducts = append(pbProducts, pbProduct)
	}

	// Build pagination response
	totalPages := (total + int64(pageSize) - 1) / int64(pageSize)
	pagination := &pb.PaginationResponse{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: int32(total),
		TotalPages: int32(totalPages),
	}

	return pbProducts, pagination, nil
}

// Update updates an existing product
func (s *productService) Update(ctx context.Context, req *pb.UpdateProductRequest) (*pb.Product, error) {
	if req.Id == "" {
		return nil, errors.New("product ID is required")
	}

	// Load existing product
	var product models.Product
	if err := s.db.WithContext(ctx).Preload("Template").First(&product, "id = ?", req.Id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("product not found: %s", req.Id)
		}
		return nil, err
	}

	// Update fields
	updates := make(map[string]interface{})

	if req.Name != "" {
		updates["name"] = req.Name
	}

	if req.Sku != "" {
		updates["sku"] = req.Sku
	}

	if req.Description != "" {
		updates["description"] = req.Description
	}

	// Note: Price and stock are now in ProductInventory table
	// Use separate inventory update API to modify price/stock

	if req.Status != pb.ProductStatus_PRODUCT_STATUS_UNSPECIFIED {
		updates["status"] = s.protoStatusToString(req.Status)
	}

	if len(req.AttributeValues) > 0 {
		// Validate attribute values against template
		if err := s.validateAttributeValues(req.AttributeValues, product.Template.Attributes); err != nil {
			return nil, err
		}

		attrJSON, err := s.protoAttributesToJSON(req.AttributeValues)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize attributes: %w", err)
		}
		updates["attribute_values"] = attrJSON
	}

	// Apply updates
	if len(updates) > 0 {
		if err := s.db.WithContext(ctx).Model(&product).Updates(updates).Error; err != nil {
			if strings.Contains(err.Error(), "duplicate key") && strings.Contains(err.Error(), "sku") {
				return nil, fmt.Errorf("SKU already exists: %s", req.Sku)
			}
			return nil, err
		}
	}

	// Reload product
	if err := s.db.WithContext(ctx).Preload("Template").First(&product, "id = ?", req.Id).Error; err != nil {
		return nil, err
	}

	return s.modelToProto(&product, product.Template.Name)
}

// Delete deletes a product
func (s *productService) Delete(ctx context.Context, req *pb.DeleteProductRequest) (*pb.DeleteProductResponse, error) {
	if req.Id == "" {
		return nil, errors.New("product ID is required")
	}

	// Check if product exists
	var product models.Product
	if err := s.db.WithContext(ctx).First(&product, "id = ?", req.Id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("product not found: %s", req.Id)
		}
		return nil, err
	}

	// Count variants
	var variantCount int64
	s.db.Model(&models.ProductVariant{}).Where("product_id = ?", req.Id).Count(&variantCount)

	// Delete product (variants will cascade if delete_variants = true)
	if err := s.db.WithContext(ctx).Delete(&product).Error; err != nil {
		return nil, err
	}

	// TODO: Actually delete variants if req.DeleteVariants = true

	return &pb.DeleteProductResponse{
		Success:         true,
		VariantsDeleted: int32(variantCount),
	}, nil
}

// BulkUpdateStatus updates status for multiple products
func (s *productService) BulkUpdateStatus(ctx context.Context, req *pb.BulkUpdateStatusRequest) (*pb.BulkUpdateStatusResponse, error) {
	if len(req.ProductIds) == 0 {
		return nil, errors.New("product_ids cannot be empty")
	}

	if req.Status == pb.ProductStatus_PRODUCT_STATUS_UNSPECIFIED {
		return nil, errors.New("status is required")
	}

	status := s.protoStatusToString(req.Status)

	// Update all products
	result := s.db.WithContext(ctx).
		Model(&models.Product{}).
		Where("id IN ?", req.ProductIds).
		Update("status", status)

	if result.Error != nil {
		return nil, result.Error
	}

	// Find failed IDs (products that don't exist)
	var existingIDs []string
	s.db.WithContext(ctx).Model(&models.Product{}).
		Where("id IN ?", req.ProductIds).
		Pluck("id", &existingIDs)

	existingSet := make(map[string]bool)
	for _, id := range existingIDs {
		existingSet[id] = true
	}

	failedIDs := []string{}
	for _, id := range req.ProductIds {
		if !existingSet[id] {
			failedIDs = append(failedIDs, id)
		}
	}

	return &pb.BulkUpdateStatusResponse{
		UpdatedCount: int32(result.RowsAffected),
		FailedIds:    failedIDs,
	}, nil
}

// validateCreateRequest validates product creation request
func (s *productService) validateCreateRequest(req *pb.CreateProductRequest) error {
	if req.TemplateId == "" {
		return errors.New("template_id is required")
	}

	if req.Name == "" {
		return errors.New("product name is required")
	}

	if len(req.Name) > 255 {
		return errors.New("product name must be 255 characters or less")
	}

	if req.Sku == "" {
		return errors.New("SKU is required")
	}

	if len(req.Sku) > 100 {
		return errors.New("SKU must be 100 characters or less")
	}

	// Validate SKU format (alphanumeric, dash, underscore)
	for _, char := range req.Sku {
		if !(char >= 'a' && char <= 'z') && !(char >= 'A' && char <= 'Z') &&
			!(char >= '0' && char <= '9') && char != '-' && char != '_' {
			return fmt.Errorf("SKU contains invalid character: %c", char)
		}
	}

	if req.InitialListPrice < 0 {
		return errors.New("initial_list_price must be >= 0")
	}

	if req.InitialSalePrice < 0 {
		return errors.New("initial_sale_price must be >= 0")
	}

	if req.InitialStock < 0 {
		return errors.New("initial_stock must be >= 0")
	}

	return nil
}

// validateAttributeValues validates attribute values against template definition
func (s *productService) validateAttributeValues(values map[string]*pb.AttributeValue, templateAttrsJSON []byte) error {
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

	// Check required attributes
	for _, attr := range templateAttrs {
		if attr.Required {
			if _, ok := values[attr.Name]; !ok {
				return fmt.Errorf("required attribute missing: %s", attr.Name)
			}
		}
	}

	// Validate each provided attribute
	for name, value := range values {
		attrDef, ok := attrDefs[name]
		if !ok {
			return fmt.Errorf("attribute not defined in template: %s", name)
		}

		// Validate type matches
		if value.Type != s.stringToAttributeType(attrDef.Type) {
			return fmt.Errorf("attribute %s: expected type %s, got %s", name, attrDef.Type, value.Type.String())
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
						return fmt.Errorf("attribute %s: value '%s' not in allowed options", name, val)
					}
				}
			}
		}
	}

	return nil
}

// protoAttributesToJSON converts protobuf AttributeValue map to JSONB
func (s *productService) protoAttributesToJSON(attrs map[string]*pb.AttributeValue) ([]byte, error) {
	// Convert protobuf AttributeValue to internal AttributeValue structure
	jsonAttrs := make(map[string]models.AttributeValue)

	for name, value := range attrs {
		attrValue := models.AttributeValue{
			Type: value.Type.String(),
		}

		// Set value based on type
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

// modelToProto converts GORM model to protobuf Product
func (s *productService) modelToProto(model *models.Product, templateName string) (*pb.Product, error) {
	// Parse attribute values from JSONB
	var attrs map[string]models.AttributeValue
	if err := json.Unmarshal(model.AttributeValues, &attrs); err != nil {
		return nil, fmt.Errorf("failed to parse attribute values: %w", err)
	}

	// Convert to protobuf AttributeValue map
	pbAttrs := make(map[string]*pb.AttributeValue)
	for name, value := range attrs {
		pbAttr := &pb.AttributeValue{
			Type: s.stringToAttributeType(value.Type),
		}

		// Set value based on type
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

		pbAttrs[name] = pbAttr
	}

	// Get pricing information (denormalized for display)
	var listPrice, salePrice, effectivePrice float64
	var currency string = "USD"
	if model.Pricing != nil {
		listPrice = model.Pricing.ListPrice
		salePrice = model.Pricing.SalePrice
		currency = model.Pricing.Currency
		if salePrice > 0 {
			effectivePrice = salePrice
		} else {
			effectivePrice = listPrice
		}
	}

	// Get inventory information (denormalized, aggregated across locations)
	var totalStock, availableStock, reservedStock int32
	for _, inv := range model.Inventories {
		totalStock += inv.OnHandQuantity
		reservedStock += inv.ReservedQuantity
	}
	availableStock = totalStock - reservedStock

	return &pb.Product{
		Id:               model.ID,
		TemplateId:       model.TemplateID,
		TemplateName:     templateName,
		Name:             model.Name,
		Sku:              model.SKU,
		Description:      model.Description,
		AttributeValues:  pbAttrs,
		Status:           s.stringToProtoStatus(model.Status),
		CreatedAt:        timestamppb.New(model.CreatedAt),
		UpdatedAt:        timestamppb.New(model.UpdatedAt),
		VariantCount:     0, // TODO: Count variants
		PrimaryImageUrls: []string{}, // TODO: Get primary images
		ListPrice:        listPrice,
		SalePrice:        salePrice,
		EffectivePrice:   effectivePrice,
		Currency:         currency,
		TotalStock:       totalStock,
		AvailableStock:   availableStock,
		ReservedStock:    reservedStock,
	}, nil
}

// protoStatusToString converts protobuf status to string
func (s *productService) protoStatusToString(status pb.ProductStatus) string {
	switch status {
	case pb.ProductStatus_PRODUCT_STATUS_ACTIVE:
		return "active"
	case pb.ProductStatus_PRODUCT_STATUS_INACTIVE:
		return "inactive"
	case pb.ProductStatus_PRODUCT_STATUS_DRAFT:
		return "draft"
	default:
		return "draft"
	}
}

// stringToProtoStatus converts string status to protobuf
func (s *productService) stringToProtoStatus(status string) pb.ProductStatus {
	switch strings.ToLower(status) {
	case "active":
		return pb.ProductStatus_PRODUCT_STATUS_ACTIVE
	case "inactive":
		return pb.ProductStatus_PRODUCT_STATUS_INACTIVE
	case "draft":
		return pb.ProductStatus_PRODUCT_STATUS_DRAFT
	default:
		return pb.ProductStatus_PRODUCT_STATUS_UNSPECIFIED
	}
}

// stringToAttributeType converts string type to protobuf AttributeType
func (s *productService) stringToAttributeType(typeStr string) pb.AttributeType {
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

