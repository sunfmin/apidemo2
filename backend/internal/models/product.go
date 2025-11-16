package models

import (
	"time"

	"gorm.io/datatypes"
)

// Product represents a product instance based on a template
type Product struct {
	ID              string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TemplateID      string         `gorm:"type:uuid;not null;index"`
	Name            string         `gorm:"type:varchar(255);not null;index"`
	SKU             string         `gorm:"type:varchar(100);not null;uniqueIndex"`
	Description     string         `gorm:"type:text"`
	AttributeValues datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	Status          string         `gorm:"type:varchar(50);not null;default:'active';index"`
	CreatedAt       time.Time      `gorm:"not null"`
	UpdatedAt       time.Time      `gorm:"not null"`

	// Relations
	Template   ProductTemplate   `gorm:"foreignKey:TemplateID"`
	Variants   []ProductVariant  `gorm:"foreignKey:ProductID"`
	MediaFiles []MediaFile       `gorm:"polymorphic:Entity;"`
}

// TableName specifies the table name for Product
func (Product) TableName() string {
	return "products"
}

// ProductVariant represents a product variant (stub for now)
type ProductVariant struct {
	ID              string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ProductID       string         `gorm:"type:uuid;not null;index"`
	Name            string         `gorm:"type:varchar(255);not null"`
	SKU             string         `gorm:"type:varchar(100);not null;uniqueIndex"`
	AttributeValues datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt       time.Time      `gorm:"not null"`
	UpdatedAt       time.Time      `gorm:"not null"`
}

// TableName specifies the table name for ProductVariant
func (ProductVariant) TableName() string {
	return "product_variants"
}

// MediaFile represents a media file (stub for now)
type MediaFile struct {
	ID            string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	EntityType    string    `gorm:"type:varchar(50);not null;index:idx_entity"`
	EntityID      string    `gorm:"type:uuid;not null;index:idx_entity"`
	AttributeName string    `gorm:"type:varchar(255);not null"`
	FileType      string    `gorm:"type:varchar(20);not null"` // image, video
	MimeType      string    `gorm:"type:varchar(100);not null"`
	FileName      string    `gorm:"type:varchar(255);not null"`
	FilePath      string    `gorm:"type:text;not null"`
	FileSize      int64     `gorm:"not null"`
	Width         int32     `gorm:"default:0"`
	Height        int32     `gorm:"default:0"`
	Duration      int32     `gorm:"default:0"` // For videos, in seconds
	DisplayOrder  int32     `gorm:"not null;default:0"`
	CreatedAt     time.Time `gorm:"not null"`
	UpdatedAt     time.Time `gorm:"not null"`
}

// TableName specifies the table name for MediaFile
func (MediaFile) TableName() string {
	return "media_files"
}

// AttributeValue represents a single attribute value in JSONB
// This is used for unmarshaling the JSONB structure
type AttributeValue struct {
	Type  string      `json:"type"`  // text, number, boolean, date, list, map, image, video
	Value interface{} `json:"value"` // Actual value - type depends on Type field
}

