package middleware

import (
	"log"
	"net/http"
	"time"
)

// Logging middleware logs HTTP requests with method, path, duration, and status
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create response writer wrapper to capture status
		rw := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}

		// Call next handler
		next.ServeHTTP(rw, r)

		// Log request details
		duration := time.Since(start)
		log.Printf(
			"[%s] %s %s - %d (%v)",
			r.Method,
			r.URL.Path,
			r.RemoteAddr,
			rw.statusCode,
			duration,
		)
	})
}

