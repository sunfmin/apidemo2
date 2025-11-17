package testutil

import (
	"testing"
)

// TestSetupTestDB verifies that the testcontainers setup works correctly
func TestSetupTestDB(t *testing.T) {
	// Setup test database
	db, cleanup := SetupTestDB(t)
	defer cleanup()

	// Verify connection is working
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying database: %v", err)
	}

	// Ping database
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	// Verify we can execute a simple query
	var result int
	if err := db.Raw("SELECT 1").Scan(&result).Error; err != nil {
		t.Fatalf("Failed to execute simple query: %v", err)
	}

	if result != 1 {
		t.Errorf("Expected result 1, got %d", result)
	}

	t.Log("✅ Testcontainers setup working correctly")
}

// TestTruncateTables verifies that the truncation helper works correctly
func TestTruncateTables(t *testing.T) {
	// Setup test database
	db, cleanup := SetupTestDB(t)
	defer cleanup()

	// Create a simple test table
	db.Exec(`
		CREATE TABLE IF NOT EXISTS test_items (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL
		)
	`)

	// Insert test data
	db.Exec("INSERT INTO test_items (name) VALUES ('Item 1'), ('Item 2'), ('Item 3')")

	// Verify data exists
	var count int64
	db.Raw("SELECT COUNT(*) FROM test_items").Scan(&count)
	if count != 3 {
		t.Fatalf("Expected 3 items, got %d", count)
	}

	// Truncate table
	TruncateTables(db, "test_items")

	// Verify table is empty
	db.Raw("SELECT COUNT(*) FROM test_items").Scan(&count)
	if count != 0 {
		t.Errorf("Expected 0 items after truncation, got %d", count)
	}

	t.Log("✅ Table truncation working correctly")
}

// TestTruncateTablesWithForeignKeys verifies CASCADE behavior
func TestTruncateTablesWithForeignKeys(t *testing.T) {
	// Setup test database
	db, cleanup := SetupTestDB(t)
	defer cleanup()

	// Create parent and child tables
	db.Exec(`
		CREATE TABLE IF NOT EXISTS test_parents (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL
		)
	`)

	db.Exec(`
		CREATE TABLE IF NOT EXISTS test_children (
			id SERIAL PRIMARY KEY,
			parent_id INTEGER REFERENCES test_parents(id),
			name VARCHAR(255) NOT NULL
		)
	`)

	// Insert test data
	db.Exec("INSERT INTO test_parents (name) VALUES ('Parent 1'), ('Parent 2')")
	db.Exec("INSERT INTO test_children (parent_id, name) VALUES (1, 'Child 1'), (1, 'Child 2'), (2, 'Child 3')")

	// Verify data exists
	var parentCount, childCount int64
	db.Raw("SELECT COUNT(*) FROM test_parents").Scan(&parentCount)
	db.Raw("SELECT COUNT(*) FROM test_children").Scan(&childCount)

	if parentCount != 2 || childCount != 3 {
		t.Fatalf("Expected 2 parents and 3 children, got %d parents and %d children", parentCount, childCount)
	}

	// Truncate both tables (children before parents with CASCADE)
	TruncateTables(db, "test_parents", "test_children")

	// Verify both tables are empty
	db.Raw("SELECT COUNT(*) FROM test_parents").Scan(&parentCount)
	db.Raw("SELECT COUNT(*) FROM test_children").Scan(&childCount)

	if parentCount != 0 || childCount != 0 {
		t.Errorf("Expected empty tables, got %d parents and %d children", parentCount, childCount)
	}

	t.Log("✅ CASCADE truncation working correctly")
}
