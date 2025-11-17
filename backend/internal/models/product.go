package models

import (
	"time"

	"gorm.io/datatypes"
)

// Product represents a product master data (relatively static information)
// Price and stock are managed in ProductInventory table for frequent updates
type Product struct {
	ID              string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TemplateID      string         `gorm:"type:uuid;not null;index"`
	Name            string         `gorm:"type:varchar(255);not null;index"`
	SKU             string         `gorm:"type:varchar(100);not null;uniqueIndex"`
	Description     string         `gorm:"type:text"`
	AttributeValues datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"` // Template-specific flexible attributes
	Status          string         `gorm:"type:varchar(50);not null;default:'active';index"`
	CreatedAt       time.Time      `gorm:"not null"`
	UpdatedAt       time.Time      `gorm:"not null"`

	// Relations
	Template    ProductTemplate    `gorm:"foreignKey:TemplateID"`
	Pricing     *ProductPricing    `gorm:"foreignKey:ProductID"`
	Inventories []ProductInventory `gorm:"foreignKey:ProductID"` // Multiple locations
	Variants    []ProductVariant   `gorm:"foreignKey:ProductID"`
	MediaFiles  []MediaFile        `gorm:"polymorphic:Entity;"`
}

// TableName specifies the table name for Product
func (Product) TableName() string {
	return "products"
}

// ProductPricing represents pricing information for a product
// Owned by Marketing/Business team, updated occasionally (daily/weekly)
type ProductPricing struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ProductID string    `gorm:"type:uuid;not null;uniqueIndex"` // One pricing record per product
	ListPrice float64   `gorm:"type:decimal(10,2);not null;index"`
	SalePrice float64   `gorm:"type:decimal(10,2);not null;default:0;index"` // 0 if not on sale
	Cost      float64   `gorm:"type:decimal(10,2);not null;default:0"`       // Wholesale cost
	Currency  string    `gorm:"type:varchar(3);not null;default:'USD'"`
	ValidFrom time.Time `gorm:"not null"`
	ValidTo   *time.Time
	UpdatedAt time.Time `gorm:"not null"`
}

// TableName specifies the table name for ProductPricing
func (ProductPricing) TableName() string {
	return "product_pricings"
}

// ProductInventory represents stock information for a product
// Owned by Warehouse/Operations team, updated constantly (real-time)
type ProductInventory struct {
	ID               string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ProductID        string `gorm:"type:uuid;not null;index:idx_product_location"`
	LocationID       string `gorm:"type:varchar(50);not null;default:'default';index:idx_product_location"`
	OnHandQuantity   int32  `gorm:"type:integer;not null;default:0;index"`
	ReservedQuantity int32  `gorm:"type:integer;not null;default:0"`
	OnOrderQuantity  int32  `gorm:"type:integer;not null;default:0"`
	LastCountedAt    *time.Time
	UpdatedAt        time.Time `gorm:"not null"`
}

// TableName specifies the table name for ProductInventory
func (ProductInventory) TableName() string {
	return "product_inventories"
}

// ProductVariant represents a product variant (stub for now)
type ProductVariant struct {
	ID              string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ProductID       string         `gorm:"type:uuid;not null;index"`
	Name            string         `gorm:"type:varchar(255);not null"`
	SKU             string         `gorm:"type:varchar(100);not null;uniqueIndex"`
	AttributeValues datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"` // Template-specific overrides only
	CreatedAt       time.Time      `gorm:"not null"`
	UpdatedAt       time.Time      `gorm:"not null"`

	// Relations
	Pricing     *VariantPricing    `gorm:"foreignKey:VariantID"`
	Inventories []VariantInventory `gorm:"foreignKey:VariantID"` // Multiple locations
}

// VariantPricing represents pricing information for a variant
type VariantPricing struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	VariantID string    `gorm:"type:uuid;not null;uniqueIndex"`
	ListPrice float64   `gorm:"type:decimal(10,2);not null;index"`
	SalePrice float64   `gorm:"type:decimal(10,2);not null;default:0;index"`
	Cost      float64   `gorm:"type:decimal(10,2);not null;default:0"`
	ValidFrom time.Time `gorm:"not null"`
	ValidTo   *time.Time
	UpdatedAt time.Time `gorm:"not null"`
}

// TableName specifies the table name for VariantPricing
func (VariantPricing) TableName() string {
	return "variant_pricings"
}

// VariantInventory represents stock information for a variant
type VariantInventory struct {
	ID               string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	VariantID        string `gorm:"type:uuid;not null;index:idx_variant_location"`
	LocationID       string `gorm:"type:varchar(50);not null;default:'default';index:idx_variant_location"`
	OnHandQuantity   int32  `gorm:"type:integer;not null;default:0;index"`
	ReservedQuantity int32  `gorm:"type:integer;not null;default:0"`
	OnOrderQuantity  int32  `gorm:"type:integer;not null;default:0"`
	LastCountedAt    *time.Time
	UpdatedAt        time.Time `gorm:"not null"`
}

// TableName specifies the table name for VariantInventory
func (VariantInventory) TableName() string {
	return "variant_inventories"
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
