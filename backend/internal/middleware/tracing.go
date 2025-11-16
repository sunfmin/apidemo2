package middleware

import (
	"net/http"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

// Tracing middleware instruments HTTP requests with OpenTracing spans
func Tracing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract span context from incoming request headers (if any)
		spanCtx, _ := opentracing.GlobalTracer().Extract(
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(r.Header),
		)

		// Start a new span for this HTTP request
		span := opentracing.StartSpan(
			r.Method+" "+r.URL.Path,
			ext.RPCServerOption(spanCtx),
		)
		defer span.Finish()

		// Set standard HTTP tags
		ext.HTTPMethod.Set(span, r.Method)
		ext.HTTPUrl.Set(span, r.URL.String())
		ext.Component.Set(span, "http")

		// Create context with span for downstream handlers
		ctx := opentracing.ContextWithSpan(r.Context(), span)
		r = r.WithContext(ctx)

		// Create a response writer wrapper to capture status code
		rw := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}

		// Call next handler
		next.ServeHTTP(rw, r)

		// Set HTTP status code tag after response
		ext.HTTPStatusCode.Set(span, uint16(rw.statusCode))

		// Mark span as error if status code >= 400
		if rw.statusCode >= 400 {
			ext.Error.Set(span, true)
		}
	})
}

// statusRecorder wraps http.ResponseWriter to capture the status code
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	// WriteHeader is called by Write if not already called
	return r.ResponseWriter.Write(b)
}

