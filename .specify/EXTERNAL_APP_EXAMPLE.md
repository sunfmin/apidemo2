# External Application Usage Example

**Constitution Version**: 1.9.1  
**Date**: 2025-11-19

---

## Complete Example: Using apidemo2 Services in External Go Application

This example shows how another Go application can import and use the apidemo2 services.

---

## Step 1: Import the Module

```go
// external-app/go.mod
module github.com/yourorg/external-app

go 1.21

require (
    github.com/sunfmin/apidemo2/backend v0.0.0  // Import the services
    gorm.io/driver/postgres v1.5.0
    gorm.io/gorm v1.25.0
)
```

---

## Step 2: External Application Code

```go
// external-app/main.go
package main

import (
    "context"
    "log"
    
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    
    // Import public packages from apidemo2
    "github.com/sunfmin/apidemo2/backend/services"
    pb "github.com/sunfmin/apidemo2/backend/api/gen/pim/v1"
)

func main() {
    log.Println("Starting external app using apidemo2 services...")
    
    // Step 1: Connect to YOUR database
    dsn := "host=localhost user=myapp password=myapp dbname=myapp_db port=5432"
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    log.Println("✅ Connected to database")
    
    // Step 2: Run migrations (REQUIRED)
    // This creates the necessary tables in YOUR database
    if err := services.AutoMigrate(db); err != nil {
        log.Fatalf("❌ Migration failed: %v", err)
    }
    log.Println("✅ Database schema migrated")
    
    // Step 3: Use services directly!
    productService := services.NewProductService(db)
    templateService := services.NewTemplateService(db)
    
    ctx := context.Background()
    
    // Create a template
    template, err := templateService.Create(ctx, &pb.CreateTemplateRequest{
        Name: "Electronics",
        Attributes: []*pb.AttributeDefinition{
            {
                Name:     "Brand",
                Type:     pb.AttributeType_ATTRIBUTE_TYPE_TEXT,
                Required: true,
            },
            {
                Name:     "Color",
                Type:     pb.AttributeType_ATTRIBUTE_TYPE_LIST,
                Required: true,
                Options:  []string{"Black", "White", "Silver"},
            },
        },
    })
    if err != nil {
        log.Fatalf("Failed to create template: %v", err)
    }
    log.Printf("✅ Created template: %s (ID: %s)", template.Name, template.Id)
    
    // Create a product
    product, err := productService.Create(ctx, &pb.CreateProductRequest{
        TemplateId:       template.Id,
        Name:             "Laptop Pro",
        Sku:              "LAPTOP-001",
        Description:      "High-performance laptop",
        InitialListPrice: 1299.99,
        InitialStock:     50,
        AttributeValues: map[string]*pb.AttributeValue{
            "Brand": {
                Type:      pb.AttributeType_ATTRIBUTE_TYPE_TEXT,
                TextValue: "TechCorp",
            },
            "Color": {
                Type:      pb.AttributeType_ATTRIBUTE_TYPE_LIST,
                ListValue: []string{"Silver"},
            },
        },
        Status: pb.ProductStatus_PRODUCT_STATUS_ACTIVE,
    })
    if err != nil {
        log.Fatalf("Failed to create product: %v", err)
    }
    log.Printf("✅ Created product: %s (SKU: %s, ID: %s)", product.Name, product.Sku, product.Id)
    
    // List products
    products, pagination, err := productService.List(ctx, &pb.ListProductsRequest{
        Pagination: &pb.PaginationRequest{
            Page:     1,
            PageSize: 10,
        },
    })
    if err != nil {
        log.Fatalf("Failed to list products: %v", err)
    }
    log.Printf("✅ Found %d products (total: %d)", len(products), pagination.TotalItems)
    
    // Get a specific product
    getResp, err := productService.Get(ctx, &pb.GetProductRequest{
        Id: product.Id,
    })
    if err != nil {
        log.Fatalf("Failed to get product: %v", err)
    }
    log.Printf("✅ Retrieved product: %s", getResp.Product.Name)
    
    log.Println("🎉 External app successfully using apidemo2 services!")
}
```

---

## Key Points

### ✅ What External Apps Can Import
```go
import (
    "github.com/sunfmin/apidemo2/backend/services"  // ✅ Public services
    "github.com/sunfmin/apidemo2/backend/handlers"  // ✅ Public handlers (optional)
    pb "github.com/sunfmin/apidemo2/backend/api/gen/pim/v1"  // ✅ Protobuf types
)
```

### ❌ What External Apps CANNOT Import
```go
import (
    "github.com/sunfmin/apidemo2/backend/internal/models"      // ❌ Internal - won't compile
    "github.com/sunfmin/apidemo2/backend/internal/middleware"  // ❌ Internal - won't compile
    "github.com/sunfmin/apidemo2/backend/internal/database"    // ❌ Internal - won't compile
)
```

### ✅ What External Apps Interact With
- **Protobuf types** (`*pb.Product`, `*pb.ProductTemplate`) - All service inputs and outputs
- **Service interfaces** - All business logic
- **Handlers** (optional) - Can reuse HTTP handlers with their own middleware
- **AutoMigrate function** - Database schema setup

### ❌ What Stays Hidden
- **GORM models** - Internal database mapping (never exposed)
- **Middleware** - Application-specific (each app has their own)
- **Internal utilities** - Implementation details

---

## Database Setup

External apps have two options:

### Option 1: Use services.AutoMigrate() (Recommended)
```go
db, _ := gorm.Open(...)
services.AutoMigrate(db)  // ✅ Handles all migrations
```

**Pros**:
- ✅ Simple one-line call
- ✅ Services control their schema
- ✅ Guaranteed compatibility
- ✅ No need to know internal models

**Cons**:
- ⚠️  Uses GORM AutoMigrate (not for production migrations)
- ⚠️  External app doesn't control migration timing

### Option 2: Custom Migration Tool
External app can create their own migration tool if they need more control:

```go
// But they need the schema structure
// Solution: services provide schema via AutoMigrate or documentation
```

---

## Complete Working Example

```bash
# Create new external app
mkdir external-app
cd external-app
go mod init github.com/myorg/external-app

# Add dependency
go get github.com/sunfmin/apidemo2/backend
go get gorm.io/driver/postgres
go get gorm.io/gorm

# Create main.go with example above
# Run it!
go run main.go
```

**Output**:
```
Starting external app using apidemo2 services...
✅ Connected to database
✅ Database schema migrated
✅ Created template: Electronics (ID: xxx)
✅ Created product: Laptop Pro (SKU: LAPTOP-001, ID: xxx)
✅ Found 1 products (total: 1)
✅ Retrieved product: Laptop Pro
🎉 External app successfully using apidemo2 services!
```

---

## Use Cases

### 1. Background Workers
```go
import "github.com/sunfmin/apidemo2/backend/services"

func worker() {
    services.AutoMigrate(db)
    svc := services.NewProductService(db)
    
    products, _, _ := svc.List(ctx, &pb.ListProductsRequest{...})
    for _, p := range products {
        // Process products
    }
}
```

### 2. CLI Tools
```go
import "github.com/sunfmin/apidemo2/backend/services"

func main() {
    services.AutoMigrate(db)
    svc := services.NewProductService(db)
    
    product, _ := svc.Get(ctx, &pb.GetProductRequest{Id: "123"})
    fmt.Printf("Product: %s - $%.2f\n", product.Name, product.EffectivePrice)
}
```

### 3. gRPC Services
```go
import "github.com/sunfmin/apidemo2/backend/services"

type ProductGRPCServer struct {
    svc services.ProductService
}

func main() {
    services.AutoMigrate(db)
    svc := services.NewProductService(db)
    
    grpcServer := &ProductGRPCServer{svc: svc}
    // Serve gRPC...
}
```

### 4. GraphQL Resolvers
```go
import "github.com/sunfmin/apidemo2/backend/services"

type Resolver struct {
    productSvc services.ProductService
}

func NewResolver(db *gorm.DB) *Resolver {
    services.AutoMigrate(db)
    return &Resolver{
        productSvc: services.NewProductService(db),
    }
}
```

---

## Summary

**External apps get**:
- ✅ Full service business logic
- ✅ Protobuf type contracts
- ✅ Database schema (via AutoMigrate)
- ✅ Type-safe APIs

**External apps DON'T see**:
- ✅ Internal models (encapsulated)
- ✅ Internal middleware (app-specific)
- ✅ Internal utilities (hidden)

**Result**: Clean, reusable services with proper encapsulation!

