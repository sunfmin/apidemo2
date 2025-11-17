package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	pb "github.com/sunfmin/apidemo2/backend/api/gen/pim/v1"
	"github.com/sunfmin/apidemo2/backend/internal/models"
)

// TemplateService defines the interface for template business logic
type TemplateService interface {
	Create(ctx context.Context, req *pb.CreateTemplateRequest) (*pb.ProductTemplate, error)
	Get(ctx context.Context, id string) (*pb.ProductTemplate, error)
	List(ctx context.Context, req *pb.ListTemplatesRequest) ([]*pb.ProductTemplate, *pb.PaginationResponse, error)
	Update(ctx context.Context, req *pb.UpdateTemplateRequest) (*pb.ProductTemplate, error)
	Delete(ctx context.Context, id string) error
}

// templateService implements TemplateService
type templateService struct {
	db *gorm.DB
}

// NewTemplateService creates a new TemplateService
func NewTemplateService(db *gorm.DB) TemplateService {
	return &templateService{db: db}
}

// Create creates a new product template
func (s *templateService) Create(ctx context.Context, req *pb.CreateTemplateRequest) (*pb.ProductTemplate, error) {
	// Validate request
	if err := s.validateCreateRequest(req); err != nil {
		return nil, err
	}

	// Convert attributes to JSON
	attrs, err := s.protoAttributesToJSON(req.Attributes)
	if err != nil {
		return nil, fmt.Errorf("failed to convert attributes: %w", err)
	}

	// Create model
	template := &models.ProductTemplate{
		Name:       strings.TrimSpace(req.Name),
		Attributes: attrs,
	}

	// Save to database
	if err := s.db.WithContext(ctx).Create(template).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			return nil, fmt.Errorf("template with name '%s' already exists", req.Name)
		}
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	// Convert to protobuf response
	return s.modelToProto(template), nil
}

// Get retrieves a template by ID
func (s *templateService) Get(ctx context.Context, id string) (*pb.ProductTemplate, error) {
	if id == "" {
		return nil, errors.New("template ID is required")
	}

	var template models.ProductTemplate
	if err := s.db.WithContext(ctx).First(&template, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("template not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	return s.modelToProto(&template), nil
}

// List retrieves templates with pagination and filtering
func (s *templateService) List(ctx context.Context, req *pb.ListTemplatesRequest) ([]*pb.ProductTemplate, *pb.PaginationResponse, error) {
	// Default pagination values
	page := int32(1)
	pageSize := int32(20)

	if req.Pagination != nil {
		if req.Pagination.Page > 0 {
			page = req.Pagination.Page
		}
		if req.Pagination.PageSize > 0 {
			pageSize = req.Pagination.PageSize
		}
		// Enforce max page size
		if pageSize > 100 {
			pageSize = 100
		}
	}

	// Build query
	query := s.db.WithContext(ctx).Model(&models.ProductTemplate{})

	// Apply search filter
	if req.Search != "" {
		query = query.Where("name ILIKE ?", "%"+req.Search+"%")
	}

	// Get total count
	var totalCount int64
	if err := query.Count(&totalCount).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to count templates: %w", err)
	}

	// Apply sorting
	sortBy := "created_at"
	sortOrder := "asc"
	if req.SortBy != "" {
		sortBy = req.SortBy
	}
	if req.SortOrder != "" {
		sortOrder = strings.ToLower(req.SortOrder)
	}
	query = query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))

	// Apply pagination
	offset := (page - 1) * pageSize
	query = query.Offset(int(offset)).Limit(int(pageSize))

	// Execute query
	var templates []models.ProductTemplate
	if err := query.Find(&templates).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to list templates: %w", err)
	}

	// Convert to protobuf
	protoTemplates := make([]*pb.ProductTemplate, len(templates))
	for i, t := range templates {
		protoTemplates[i] = s.modelToProto(&t)
	}

	// Build pagination response
	totalPages := (totalCount + int64(pageSize) - 1) / int64(pageSize)
	pagination := &pb.PaginationResponse{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: int32(totalCount),
		TotalPages: int32(totalPages),
	}

	return protoTemplates, pagination, nil
}

// Update updates an existing template
func (s *templateService) Update(ctx context.Context, req *pb.UpdateTemplateRequest) (*pb.ProductTemplate, error) {
	if req.Id == "" {
		return nil, errors.New("template ID is required")
	}

	// Get existing template
	var template models.ProductTemplate
	if err := s.db.WithContext(ctx).First(&template, "id = ?", req.Id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("template not found: %s", req.Id)
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// Update fields
	if req.Name != "" {
		template.Name = strings.TrimSpace(req.Name)
	}

	if req.Attributes != nil {
		attrs, err := s.protoAttributesToJSON(req.Attributes)
		if err != nil {
			return nil, fmt.Errorf("failed to convert attributes: %w", err)
		}
		template.Attributes = attrs
	}

	// Save changes
	if err := s.db.WithContext(ctx).Save(&template).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			return nil, fmt.Errorf("template with name '%s' already exists", template.Name)
		}
		return nil, fmt.Errorf("failed to update template: %w", err)
	}

	return s.modelToProto(&template), nil
}

// Delete deletes a template
func (s *templateService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("template ID is required")
	}

	// Check if template exists
	var template models.ProductTemplate
	if err := s.db.WithContext(ctx).First(&template, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("template not found: %s", id)
		}
		return fmt.Errorf("failed to get template: %w", err)
	}

	// TODO: Check if template has products using it
	// For now, allow deletion

	// Delete template
	if err := s.db.WithContext(ctx).Delete(&template).Error; err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	return nil
}

// Helper functions

func (s *templateService) validateCreateRequest(req *pb.CreateTemplateRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("template name is required")
	}

	if len(req.Name) > 255 {
		return errors.New("template name must be 255 characters or less")
	}

	if len(req.Attributes) == 0 {
		return errors.New("at least one attribute is required")
	}

	// Validate attributes
	attrNames := make(map[string]bool)
	for _, attr := range req.Attributes {
		// Check attribute name
		if attr.Name == "" {
			return errors.New("attribute name is required")
		}

		// Check for duplicate names
		if attrNames[attr.Name] {
			return fmt.Errorf("duplicate attribute name: %s", attr.Name)
		}
		attrNames[attr.Name] = true

		// Check attribute type
		if attr.Type == pb.AttributeType_ATTRIBUTE_TYPE_UNSPECIFIED {
			return fmt.Errorf("invalid attribute type for '%s'", attr.Name)
		}

		// Check list options
		if attr.Type == pb.AttributeType_ATTRIBUTE_TYPE_LIST && len(attr.Options) == 0 {
			return fmt.Errorf("list attribute '%s' must have at least one option", attr.Name)
		}
	}

	return nil
}

func (s *templateService) protoAttributesToJSON(attrs []*pb.AttributeDefinition) ([]byte, error) {
	// Convert protobuf attributes to JSON-serializable format
	jsonAttrs := make([]models.AttributeDefinition, len(attrs))
	for i, attr := range attrs {
		jsonAttrs[i] = models.AttributeDefinition{
			Name:     attr.Name,
			Type:     attr.Type.String(),
			Required: attr.Required,
			Options:  attr.Options,
		}
	}

	return json.Marshal(jsonAttrs)
}

func (s *templateService) modelToProto(model *models.ProductTemplate) *pb.ProductTemplate {
	// Parse attributes from JSON
	var attrs []models.AttributeDefinition
	json.Unmarshal(model.Attributes, &attrs)

	// Convert to protobuf
	protoAttrs := make([]*pb.AttributeDefinition, len(attrs))
	for i, attr := range attrs {
		attrType := s.stringToAttributeType(attr.Type)
		protoAttrs[i] = &pb.AttributeDefinition{
			Name:     attr.Name,
			Type:     attrType,
			Required: attr.Required,
			Options:  attr.Options,
		}
	}

	return &pb.ProductTemplate{
		Id:         model.ID,
		Name:       model.Name,
		Attributes: protoAttrs,
		CreatedAt:  timestamppb.New(model.CreatedAt),
		UpdatedAt:  timestamppb.New(model.UpdatedAt),
	}
}

func (s *templateService) stringToAttributeType(typeStr string) pb.AttributeType {
	switch typeStr {
	case "ATTRIBUTE_TYPE_TEXT":
		return pb.AttributeType_ATTRIBUTE_TYPE_TEXT
	case "ATTRIBUTE_TYPE_NUMBER":
		return pb.AttributeType_ATTRIBUTE_TYPE_NUMBER
	case "ATTRIBUTE_TYPE_BOOLEAN":
		return pb.AttributeType_ATTRIBUTE_TYPE_BOOLEAN
	case "ATTRIBUTE_TYPE_DATE":
		return pb.AttributeType_ATTRIBUTE_TYPE_DATE
	case "ATTRIBUTE_TYPE_LIST":
		return pb.AttributeType_ATTRIBUTE_TYPE_LIST
	case "ATTRIBUTE_TYPE_MAP":
		return pb.AttributeType_ATTRIBUTE_TYPE_MAP
	case "ATTRIBUTE_TYPE_IMAGE":
		return pb.AttributeType_ATTRIBUTE_TYPE_IMAGE
	case "ATTRIBUTE_TYPE_VIDEO":
		return pb.AttributeType_ATTRIBUTE_TYPE_VIDEO
	default:
		return pb.AttributeType_ATTRIBUTE_TYPE_UNSPECIFIED
	}
}
