package models

import (
	"time"

	"gorm.io/datatypes"
)

// ProductTemplate defines the structure for product templates
type ProductTemplate struct {
	ID         string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name       string         `gorm:"type:varchar(255);not null;uniqueIndex"`
	Attributes datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'"`
	CreatedAt  time.Time      `gorm:"not null"`
	UpdatedAt  time.Time      `gorm:"not null"`
}

// TableName specifies the table name for ProductTemplate
func (ProductTemplate) TableName() string {
	return "product_templates"
}

// AttributeDefinition represents an attribute definition in a template
// This struct is used for JSON marshaling/unmarshaling with the Attributes field
type AttributeDefinition struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"` // text, number, boolean, date, list, map, image, video
	Required   bool                   `json:"required"`
	Options    []string               `json:"options,omitempty"` // For list type
	Validation map[string]interface{} `json:"validation,omitempty"`
}
