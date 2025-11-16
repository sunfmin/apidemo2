package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/opentracing/opentracing-go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/sunfmin/apidemo2/backend/internal/handlers"
	"github.com/sunfmin/apidemo2/backend/internal/middleware"
	"github.com/sunfmin/apidemo2/backend/internal/models"
	"github.com/sunfmin/apidemo2/backend/internal/services"
)

func main() {
	log.Println("🚀 Starting PIM API Server...")

	// Check prerequisites
	if err := checkFFmpeg(); err != nil {
		log.Fatalf("❌ Prerequisite check failed: %v", err)
	}

	log.Println("✅ All prerequisites satisfied")

	// Initialize OpenTracing (NoopTracer for now)
	opentracing.SetGlobalTracer(opentracing.NoopTracer{})
	log.Println("✅ OpenTracing initialized (NoopTracer)")

	// Initialize database connection
	// Load from environment variable or use default for development
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=postgres dbname=pim_dev port=5432 sslmode=disable"
		log.Println("⚠️  Using default database connection (set DATABASE_URL env variable for production)")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	log.Println("✅ Database connected")

	// Run auto-migration
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
		log.Fatalf("❌ Failed to run migrations: %v", err)
	}
	log.Println("✅ Database migrations complete")

	// Initialize services
	templateService := services.NewTemplateService(db)
	productService := services.NewProductService(db)
	variantService := services.NewVariantService(db)
	log.Println("✅ Services initialized")

	// Initialize handlers
	templateHandler := handlers.NewTemplateHandler(templateService)
	productHandler := handlers.NewProductHandler(productService)
	variantHandler := handlers.NewVariantHandler(variantService)
	log.Println("✅ Handlers initialized")

	// Create HTTP router
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Template routes - Collection endpoints
	mux.HandleFunc("/api/v1/templates", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			templateHandler.Create(w, r)
		case http.MethodGet:
			templateHandler.List(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Template routes - Item endpoints (with trailing slash for ID)
	mux.HandleFunc("/api/v1/templates/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			templateHandler.Get(w, r)
		case http.MethodPut:
			templateHandler.Update(w, r)
		case http.MethodDelete:
			templateHandler.Delete(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Product routes - Collection endpoints
	mux.HandleFunc("/api/v1/products", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			productHandler.Create(w, r)
		case http.MethodGet:
			productHandler.List(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Product routes - Item endpoints (with trailing slash for ID)
	mux.HandleFunc("/api/v1/products/", func(w http.ResponseWriter, r *http.Request) {
		// Check for bulk endpoints first
		if strings.HasPrefix(r.URL.Path, "/api/v1/products/bulk/status") {
			if r.Method == http.MethodPost {
				productHandler.BulkUpdateStatus(w, r)
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Individual product endpoints
		switch r.Method {
		case http.MethodGet:
			productHandler.Get(w, r)
		case http.MethodPut:
			productHandler.Update(w, r)
		case http.MethodDelete:
			productHandler.Delete(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Variant routes - Item endpoints
	mux.HandleFunc("/api/v1/variants/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			variantHandler.Get(w, r)
		case http.MethodPut:
			variantHandler.Update(w, r)
		case http.MethodDelete:
			variantHandler.Delete(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Variant routes - Product-scoped endpoints
	// Note: This needs careful routing to distinguish from /api/v1/products/{id}
	// Pattern: /api/v1/products/{product_id}/variants
	mux.HandleFunc("/api/v1/products/", func(w http.ResponseWriter, r *http.Request) {
		// Check for bulk variant creation first
		if strings.HasSuffix(r.URL.Path, "/variants/bulk") {
			if r.Method == http.MethodPost {
				variantHandler.BulkCreate(w, r)
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check if this is a variant-related request
		if strings.HasSuffix(r.URL.Path, "/variants") {
			if r.Method == http.MethodGet {
				variantHandler.List(w, r)
				return
			} else if r.Method == http.MethodPost {
				variantHandler.Create(w, r)
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check for bulk endpoints first
		if strings.HasPrefix(r.URL.Path, "/api/v1/products/bulk/status") {
			if r.Method == http.MethodPost {
				productHandler.BulkUpdateStatus(w, r)
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Individual product endpoints
		switch r.Method {
		case http.MethodGet:
			productHandler.Get(w, r)
		case http.MethodPut:
			productHandler.Update(w, r)
		case http.MethodDelete:
			productHandler.Delete(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	log.Println("✅ Routes registered")

	// TODO: Register media routes

	// Apply middleware chain
	handler := middleware.Recovery(
		middleware.Logging(
			middleware.Tracing(
				middleware.CORS(mux),
			),
		),
	)

	// Start server
	addr := ":8080"
	log.Printf("📡 Server listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}

// checkFFmpeg verifies that ffmpeg is installed and available
func checkFFmpeg() error {
	cmd := exec.Command("ffmpeg", "-version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg is not installed or not in PATH. Install with: brew install ffmpeg (Mac) or apt-get install ffmpeg (Linux)")
	}
	log.Println("✅ ffmpeg found")
	return nil
}

