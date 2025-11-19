package services

import (
	"gorm.io/gorm"

	"github.com/sunfmin/apidemo2/backend/internal/models"
)

// AutoMigrate runs all necessary database migrations for services
// External applications MUST call this function before using services
// to ensure the database schema is properly initialized
//
// Example usage:
//
//	db, _ := gorm.Open(...)
//	if err := services.AutoMigrate(db); err != nil {
//	    log.Fatalf("Migration failed: %v", err)
//	}
//	productService := services.NewProductService(db)
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.ProductTemplate{},
		&models.Product{},
		&models.ProductPricing{},
		&models.ProductInventory{},
		&models.ProductVariant{},
		&models.VariantPricing{},
		&models.VariantInventory{},
		&models.MediaFile{},
	)
}

