package testutil

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	pgdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/sunfmin/apidemo2/backend/internal/models"
)

// SetupTestDB creates a PostgreSQL test container and returns a GORM connection
// The container is automatically cleaned up when the returned cleanup function is called
func SetupTestDB(t *testing.T) (*gorm.DB, func()) {
	ctx := context.Background()

	// Create PostgreSQL container
	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("Failed to start PostgreSQL container: %v", err)
	}

	// Cleanup function
	cleanup := func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		cleanup()
		t.Fatalf("Failed to get connection string: %v", err)
	}

	// Connect using GORM
	db, err := gorm.Open(pgdriver.Open(connStr), &gorm.Config{})
	if err != nil {
		cleanup()
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Run AutoMigrate for all models
	// Note: Add models as they are implemented
	if err := db.AutoMigrate(
		&models.ProductTemplate{},
		&models.Product{},
		&models.ProductPricing{},
		&models.ProductInventory{},
		&models.ProductVariant{},
		&models.VariantPricing{},
		&models.VariantInventory{},
		&models.MediaFile{},
	); err != nil {
		cleanup()
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return db, cleanup
}

// TruncateTables truncates the specified tables in reverse order with CASCADE
// This ensures foreign key constraints are handled properly
func TruncateTables(db *gorm.DB, tables ...string) {
	// Truncate in reverse order (children before parents)
	for i := len(tables) - 1; i >= 0; i-- {
		db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", tables[i]))
	}
}

// CreateTemplateFixture creates a test product template
func CreateTemplateFixture(db *gorm.DB, name string, attributes string) *models.ProductTemplate {
	if attributes == "" {
		// Default attributes
		attributes = `[{"name":"Size","type":"ATTRIBUTE_TYPE_TEXT","required":true}]`
	}

	template := &models.ProductTemplate{
		Name:       name,
		Attributes: []byte(attributes),
	}

	db.Create(template)
	return template
}

// CreateProductFixture creates a test product
func CreateProductFixture(db *gorm.DB, templateID string, name string, sku string, attributeValues string) *models.Product {
	if attributeValues == "" {
		// Default attribute values
		attributeValues = `{}`
	}

	product := &models.Product{
		TemplateID:      templateID,
		Name:            name,
		SKU:             sku,
		Description:     "Test product description",
		AttributeValues: []byte(attributeValues),
		Status:          "active",
	}

	db.Create(product)
	return product
}

// CreateVariantFixture creates a test product variant with pricing and inventory
func CreateVariantFixture(db *gorm.DB, productID string, name string, sku string, attributeValues string) *models.ProductVariant {
	if attributeValues == "" {
		// Default attribute values (empty overrides)
		attributeValues = `{}`
	}

	variant := &models.ProductVariant{
		ProductID:       productID,
		Name:            name,
		SKU:             sku,
		AttributeValues: []byte(attributeValues),
	}

	db.Create(variant)

	// Create variant pricing
	pricing := &models.VariantPricing{
		VariantID: variant.ID,
		ListPrice: 99.99,
		SalePrice: 0,
		ValidFrom: time.Now(),
	}
	db.Create(pricing)

	// Create variant inventory
	inventory := &models.VariantInventory{
		VariantID:      variant.ID,
		LocationID:     "default",
		OnHandQuantity: 100,
	}
	db.Create(inventory)

	return variant
}

