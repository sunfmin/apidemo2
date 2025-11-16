package testutil

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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
	db, err := gorm.Open(postgres.Open(connStr), &gorm.Config{})
	if err != nil {
		cleanup()
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Run AutoMigrate for all models
	// Note: Models will be added here as they are implemented
	// if err := db.AutoMigrate(&models.ProductTemplate{}, &models.Product{}, &models.ProductVariant{}, &models.MediaFile{}); err != nil {
	// 	cleanup()
	// 	t.Fatalf("Failed to run migrations: %v", err)
	// }

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

