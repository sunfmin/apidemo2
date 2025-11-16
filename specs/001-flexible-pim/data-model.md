# Data Model: Flexible PIM System

**Feature**: Flexible Product Information Management (PIM) System  
**Branch**: `001-flexible-pim`  
**Created**: November 16, 2025

## Overview

This document defines the database schema and data models for the PIM system. The design uses PostgreSQL with JSONB for flexible attribute storage, following GORM conventions and constitution principles.

## Entity Relationship Diagram

```
ProductTemplate
    ├── 1:N → Product (template_id)
    │         ├── 1:N → ProductVariant (product_id)
    │         │         └── 1:N → MediaFile (entity_type='variant')
    │         └── 1:N → MediaFile (entity_type='product')
    └── attributes (JSONB array of AttributeDefinition)

AttributeDefinition (embedded in ProductTemplate.Attributes)
    └── Options (for list type attributes)

AttributeValue (embedded in Product.AttributeValues and ProductVariant.AttributeValues)
    └── Can contain: string, number, boolean, date, list, map, or media reference
```

## Database Tables

### 1. product_templates

Stores product template definitions with flexible attribute schemas.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Unique template identifier |
| name | VARCHAR(255) | NOT NULL, UNIQUE | Template name (e.g., "Clothing", "Electronics") |
| attributes | JSONB | NOT NULL, DEFAULT '[]' | Array of attribute definitions |
| created_at | TIMESTAMP | NOT NULL | Creation timestamp |
| updated_at | TIMESTAMP | NOT NULL | Last modification timestamp |

**JSONB Structure for `attributes`**:
```json
[
  {
    "name": "Size",
    "type": "list",
    "required": true,
    "options": ["S", "M", "L", "XL"],
    "validation": {
      "min_selections": 1,
      "max_selections": 1
    }
  },
  {
    "name": "Material",
    "type": "text",
    "required": true,
    "validation": {
      "max_length": 500
    }
  },
  {
    "name": "Price",
    "type": "number",
    "required": true,
    "validation": {
      "min": 0,
      "max": 999999.99
    }
  },
  {
    "name": "Technical Specs",
    "type": "map",
    "required": false
  },
  {
    "name": "Product Images",
    "type": "image",
    "required": true,
    "validation": {
      "min_files": 1,
      "max_files": 20
    }
  }
]
```

**Attribute Types**:
- `text`: String value
- `number`: Numeric value (integer or decimal)
- `boolean`: True/false value
- `date`: ISO 8601 date string
- `list`: Array of predefined values
- `map`: Key-value pairs
- `image`: Reference to image media files
- `video`: Reference to video media files

**Indexes**:
- PRIMARY KEY on `id`
- UNIQUE INDEX on `name`
- GIN INDEX on `attributes` (for JSONB queries)

**GORM Model**:
```go
type ProductTemplate struct {
    ID         string                 `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    Name       string                 `gorm:"type:varchar(255);not null;uniqueIndex"`
    Attributes datatypes.JSON         `gorm:"type:jsonb;not null;default:'[]'"`
    CreatedAt  time.Time              `gorm:"not null"`
    UpdatedAt  time.Time              `gorm:"not null"`
    
    // Relations
    Products   []Product              `gorm:"foreignKey:TemplateID"`
}

type AttributeDefinition struct {
    Name       string                 `json:"name"`
    Type       string                 `json:"type"` // text, number, boolean, date, list, map, image, video
    Required   bool                   `json:"required"`
    Options    []string               `json:"options,omitempty"` // For list type
    Validation map[string]interface{} `json:"validation,omitempty"`
}
```

---

### 2. products

Stores product instances based on templates with actual attribute values.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Unique product identifier |
| template_id | UUID | NOT NULL, FOREIGN KEY → product_templates(id) | Reference to template |
| name | VARCHAR(255) | NOT NULL | Product name |
| sku | VARCHAR(100) | NOT NULL, UNIQUE | Stock keeping unit |
| description | TEXT | NULL | Product description |
| attribute_values | JSONB | NOT NULL, DEFAULT '{}' | Map of attribute name → value |
| status | VARCHAR(50) | NOT NULL, DEFAULT 'active' | Product status (active/inactive/draft) |
| created_at | TIMESTAMP | NOT NULL | Creation timestamp |
| updated_at | TIMESTAMP | NOT NULL | Last modification timestamp |

**JSONB Structure for `attribute_values`**:
```json
{
  "Size": {
    "type": "list",
    "value": ["M", "L"]
  },
  "Material": {
    "type": "text",
    "value": "100% Cotton"
  },
  "Price": {
    "type": "number",
    "value": 29.99
  },
  "In Stock": {
    "type": "boolean",
    "value": true
  },
  "Release Date": {
    "type": "date",
    "value": "2025-12-01"
  },
  "Technical Specs": {
    "type": "map",
    "value": {
      "Weight": "200g",
      "Dimensions": "30x40cm",
      "Warranty": "1 year"
    }
  },
  "Product Images": {
    "type": "image",
    "value": ["uuid-1", "uuid-2", "uuid-3"]
  },
  "Demo Video": {
    "type": "video",
    "value": ["uuid-4"]
  }
}
```

**Indexes**:
- PRIMARY KEY on `id`
- UNIQUE INDEX on `sku`
- INDEX on `template_id`
- INDEX on `status`
- INDEX on `name` (for text search)
- GIN INDEX on `attribute_values` (for JSONB queries)

**GORM Model**:
```go
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
    Template        ProductTemplate `gorm:"foreignKey:TemplateID"`
    Variants        []ProductVariant `gorm:"foreignKey:ProductID"`
    MediaFiles      []MediaFile     `gorm:"polymorphic:Entity;"`
}

type AttributeValue struct {
    Type  string      `json:"type"`
    Value interface{} `json:"value"`
}
```

---

### 3. product_variants

Stores product variants with attribute overrides.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Unique variant identifier |
| product_id | UUID | NOT NULL, FOREIGN KEY → products(id) ON DELETE CASCADE | Reference to parent product |
| name | VARCHAR(255) | NOT NULL | Variant name (e.g., "Blue - Large") |
| sku | VARCHAR(100) | NOT NULL, UNIQUE | Variant-specific SKU |
| attribute_values | JSONB | NOT NULL, DEFAULT '{}' | Override attribute values |
| created_at | TIMESTAMP | NOT NULL | Creation timestamp |
| updated_at | TIMESTAMP | NOT NULL | Last modification timestamp |

**JSONB Structure for `attribute_values`**:
Variants only store overridden attributes. Attributes not in this map are inherited from parent product.

```json
{
  "Size": {
    "type": "list",
    "value": ["L"]
  },
  "Color": {
    "type": "text",
    "value": "Blue"
  },
  "Product Images": {
    "type": "image",
    "value": ["variant-uuid-1", "variant-uuid-2"]
  }
}
```

**Indexes**:
- PRIMARY KEY on `id`
- UNIQUE INDEX on `sku`
- INDEX on `product_id`
- GIN INDEX on `attribute_values`

**GORM Model**:
```go
type ProductVariant struct {
    ID              string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    ProductID       string         `gorm:"type:uuid;not null;index"`
    Name            string         `gorm:"type:varchar(255);not null"`
    SKU             string         `gorm:"type:varchar(100);not null;uniqueIndex"`
    AttributeValues datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
    CreatedAt       time.Time      `gorm:"not null"`
    UpdatedAt       time.Time      `gorm:"not null"`
    
    // Relations
    Product         Product         `gorm:"foreignKey:ProductID"`
    MediaFiles      []MediaFile     `gorm:"polymorphic:Entity;"`
}
```

---

### 4. media_files

Stores metadata for uploaded media files (images and videos).

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Unique media file identifier |
| entity_type | VARCHAR(50) | NOT NULL | Polymorphic type (product/variant) |
| entity_id | UUID | NOT NULL | Polymorphic foreign key |
| attribute_name | VARCHAR(255) | NOT NULL | Attribute this media belongs to |
| file_type | VARCHAR(50) | NOT NULL | image or video |
| mime_type | VARCHAR(100) | NOT NULL | Mime type (image/jpeg, video/mp4, etc.) |
| file_name | VARCHAR(255) | NOT NULL | Original filename |
| storage_path | VARCHAR(500) | NOT NULL | Path to original file in storage |
| thumbnail_path | VARCHAR(500) | NULL | Path to thumbnail/preview |
| file_size | BIGINT | NOT NULL | File size in bytes |
| width | INTEGER | NULL | Image/video width in pixels |
| height | INTEGER | NULL | Image/video height in pixels |
| duration | INTEGER | NULL | Video duration in seconds |
| display_order | INTEGER | NOT NULL, DEFAULT 0 | Order for display (0 = primary) |
| created_at | TIMESTAMP | NOT NULL | Upload timestamp |

**Indexes**:
- PRIMARY KEY on `id`
- INDEX on `(entity_type, entity_id)` (polymorphic relation)
- INDEX on `attribute_name`
- INDEX on `display_order`

**GORM Model**:
```go
type MediaFile struct {
    ID            string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    EntityType    string    `gorm:"type:varchar(50);not null;index:idx_entity"`
    EntityID      string    `gorm:"type:uuid;not null;index:idx_entity"`
    AttributeName string    `gorm:"type:varchar(255);not null;index"`
    FileType      string    `gorm:"type:varchar(50);not null"` // image, video
    MimeType      string    `gorm:"type:varchar(100);not null"`
    FileName      string    `gorm:"type:varchar(255);not null"`
    StoragePath   string    `gorm:"type:varchar(500);not null"`
    ThumbnailPath string    `gorm:"type:varchar(500)"`
    FileSize      int64     `gorm:"not null"`
    Width         *int      `gorm:"type:integer"`
    Height        *int      `gorm:"type:integer"`
    Duration      *int      `gorm:"type:integer"` // Video duration in seconds
    DisplayOrder  int       `gorm:"not null;default:0;index"`
    CreatedAt     time.Time `gorm:"not null"`
}
```

---

## Validation Rules

### Product Templates

1. **Template Name**:
   - Required, 1-255 characters
   - Must be unique across all templates
   - No leading/trailing whitespace

2. **Attributes**:
   - Must be valid JSON array
   - Each attribute must have: name, type
   - Attribute names must be unique within template
   - Type must be one of: text, number, boolean, date, list, map, image, video

3. **List Type Attributes**:
   - Must have at least one option
   - Options must be non-empty strings
   - Options should be unique within the list

### Products

1. **Product Name**:
   - Required, 1-255 characters
   - No uniqueness constraint (different products can have same name)

2. **SKU**:
   - Required, 1-100 characters
   - Must be unique across all products and variants
   - Alphanumeric and dashes/underscores only

3. **Template Reference**:
   - Must reference existing template
   - Cannot change template after creation (immutable)

4. **Attribute Values**:
   - Must satisfy all required attributes from template
   - Values must match attribute type
   - List values must be from predefined options
   - Number values must be within min/max if specified

5. **Status**:
   - Must be one of: active, inactive, draft

### Product Variants

1. **Product Reference**:
   - Must reference existing product
   - Deleted when parent product is deleted (CASCADE)

2. **Variant SKU**:
   - Required, 1-100 characters
   - Must be unique across all products and variants
   - Different from parent product SKU

3. **Attribute Values**:
   - Can override any parent attribute
   - Must satisfy same type constraints as product
   - Attributes not specified inherit from parent

### Media Files

1. **File Type Validation**:
   - Image: jpeg, png, gif, webp
   - Video: mp4, webm, mov
   - Validate by content (magic bytes), not just extension

2. **File Size Limits**:
   - Images: Maximum 10MB
   - Videos: Maximum 100MB

3. **Entity Reference**:
   - Must reference existing product or variant
   - Deleted when parent entity is deleted (handled in application layer)

4. **Attribute Reference**:
   - AttributeName must match an attribute in the product's template
   - Attribute type must be image or video

## State Transitions

### Product Status

```
draft → active → inactive
  ↓       ↓         ↓
  └───────┴─────────┘
       (any direction allowed)
```

- **draft**: Product being created, not visible to end users
- **active**: Product is live and visible
- **inactive**: Product is hidden but data retained

## Database Migration Strategy

Using GORM AutoMigrate for schema management:

```go
func RunMigrations(db *gorm.DB) error {
    return db.AutoMigrate(
        &ProductTemplate{},
        &Product{},
        &ProductVariant{},
        &MediaFile{},
    )
}
```

**Migration Considerations**:
- UUID generation: Use PostgreSQL's `gen_random_uuid()` function
- JSONB default values: Ensure proper defaults ('[]' for arrays, '{}' for objects)
- Indexes: Create indexes after initial migration for better performance
- Foreign keys: GORM handles CASCADE constraints automatically

## Query Patterns

### Common Queries

1. **Get Product with Template**:
```sql
SELECT p.*, t.name as template_name, t.attributes
FROM products p
JOIN product_templates t ON p.template_id = t.id
WHERE p.id = $1;
```

2. **Get Product with Variants**:
```sql
SELECT p.*, v.*
FROM products p
LEFT JOIN product_variants v ON p.id = v.product_id
WHERE p.id = $1
ORDER BY v.display_order;
```

3. **Get Product with All Media**:
```sql
SELECT p.*, m.*
FROM products p
LEFT JOIN media_files m ON m.entity_type = 'product' AND m.entity_id = p.id
WHERE p.id = $1
ORDER BY m.display_order;
```

4. **Search Products by Attribute**:
```sql
SELECT *
FROM products
WHERE attribute_values @> '{"Size": {"value": ["L"]}}';
```

5. **Find Templates Using Specific Attribute**:
```sql
SELECT *
FROM product_templates
WHERE attributes @> '[{"name": "Size"}]';
```

## Performance Considerations

1. **JSONB Indexing**:
   - GIN indexes on JSONB columns enable fast containment queries
   - Trade-off: Slower writes for faster reads (acceptable for PIM use case)

2. **Media File Loading**:
   - Lazy load media files (don't eager load unless needed)
   - Use display_order for efficient primary image selection

3. **Variant Queries**:
   - Index on product_id for efficient variant lookups
   - Consider limit if products can have many variants (1000 max per spec)

4. **Attribute Queries**:
   - JSONB queries can be slower than relational queries
   - For high-frequency queries, consider denormalizing to dedicated columns
   - Current design prioritizes flexibility over query speed (appropriate for admin tool)

## Data Integrity

### Referential Integrity

1. **Product → Template**:
   - Products reference templates via foreign key
   - Template deletion blocked if products exist (handled in service layer)

2. **Variant → Product**:
   - ON DELETE CASCADE: Variants deleted when product deleted
   - Ensures orphaned variants don't exist

3. **Media → Product/Variant**:
   - Application-level cascade (GORM handles cleanup)
   - Physical files cleaned up after database records removed

### Consistency Rules

1. **SKU Uniqueness**: 
   - Enforced at database level via unique constraint
   - Spans both products and variants tables

2. **Template Immutability**:
   - Products cannot change template after creation
   - If template changes needed, create new product

3. **Attribute Value Validation**:
   - Enforced in service layer before save
   - Database stores any valid JSON (application validates schema)

## Testing Strategy

Per constitution, all tests use real PostgreSQL database:

1. **Schema Tests**:
   - Verify GORM models match expected schema
   - Test AutoMigrate creates correct tables and indexes

2. **Validation Tests**:
   - Test constraint violations (unique, foreign key, not null)
   - Test JSONB validation

3. **Query Tests**:
   - Test JSONB queries for attribute searches
   - Test relationship loading (eager/lazy)

4. **Transaction Tests**:
   - Test cascade deletes
   - Test constraint enforcement

5. **Data Isolation**:
   - Use table truncation after each test
   - Truncate order: media_files, product_variants, products, product_templates

Example test cleanup:
```go
defer truncateTables(db, "media_files", "product_variants", "products", "product_templates")
```

