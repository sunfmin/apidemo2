package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/opentracing/opentracing-go"
	"github.com/sunfmin/apidemo2/backend/internal/middleware"
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

	// Initialize database connection (TODO: Load from config)
	// For now, skip database connection in main - tests will use testcontainers
	// Production deployment will need proper database configuration

	// Create HTTP router
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// TODO: Initialize services and handlers once database is connected
	// db, err := database.Connect(config)
	// templateService := services.NewTemplateService(db)
	// templateHandler := handlers.NewTemplateHandler(templateService)
	// 
	// // Template routes
	// mux.HandleFunc("/api/v1/templates", func(w http.ResponseWriter, r *http.Request) {
	// 	switch r.Method {
	// 	case http.MethodPost:
	// 		templateHandler.Create(w, r)
	// 	case http.MethodGet:
	// 		templateHandler.List(w, r)
	// 	default:
	// 		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	// 	}
	// })
	// 
	// mux.HandleFunc("/api/v1/templates/", func(w http.ResponseWriter, r *http.Request) {
	// 	switch r.Method {
	// 	case http.MethodGet:
	// 		templateHandler.Get(w, r)
	// 	case http.MethodPut:
	// 		templateHandler.Update(w, r)
	// 	case http.MethodDelete:
	// 		templateHandler.Delete(w, r)
	// 	default:
	// 		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	// 	}
	// })

	// TODO: Register product routes
	// TODO: Register variant routes
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

