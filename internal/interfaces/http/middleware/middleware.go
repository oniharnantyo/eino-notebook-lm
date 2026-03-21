package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

// Logger is a middleware that logs HTTP requests with comprehensive error details
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap the response writer to capture status code and body
		wrapped := &responseWrapper{
			ResponseWriter: w,
			status:         http.StatusOK,
			bodyBuffer:     &bytes.Buffer{},
		}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Milliseconds()

		// Log based on status code level
		if wrapped.status >= 500 {
			// Server errors - log with error level and include response body
			slog.Error("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.status,
				"duration", duration,
				"error", wrapped.bodyBuffer.String(),
			)
		} else if wrapped.status >= 400 {
			// Client errors - log with warn level and include response body
			slog.Warn("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.status,
				"duration", duration,
				"error", wrapped.bodyBuffer.String(),
			)
		} else {
			// Successful responses - info level without body
			slog.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.status,
				"duration", duration,
			)
		}
	})
}

// CORS is a middleware that handles CORS
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Recovery is a middleware that recovers from panics
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered",
					"error", err,
					"path", r.URL.Path,
					"stack", string(debug.Stack()),
				)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// ContentType is a middleware that sets the content type
func ContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// responseWrapper wraps http.ResponseWriter to capture status code and body
type responseWrapper struct {
	http.ResponseWriter
	status     int
	bodyBuffer *bytes.Buffer
	written    bool
}

func (rw *responseWrapper) WriteHeader(status int) {
	rw.status = status
	// Don't call underlying WriteHeader yet - we'll do it in Write or flushBody
}

func (rw *responseWrapper) Write(b []byte) (int, error) {
	// Buffer the body for logging (limit to 4KB to prevent memory issues)
	if rw.bodyBuffer.Len() < 4096 {
		remaining := 4096 - rw.bodyBuffer.Len()
		if len(b) > remaining {
			rw.bodyBuffer.Write(b[:remaining])
		} else {
			rw.bodyBuffer.Write(b)
		}
	}

	// Write to underlying response
	if !rw.written {
		rw.ResponseWriter.WriteHeader(rw.status)
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}

// flushBody ensures the header is written if Write was never called
func (rw *responseWrapper) flushBody() {
	if !rw.written {
		rw.ResponseWriter.WriteHeader(rw.status)
		rw.written = true
	}
}

// Flush implements http.Flusher
func (rw *responseWrapper) Flush() {
	rw.flushBody()
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Hijack implements http.Hijacker
func (rw *responseWrapper) Hijack() (c interface{}, rw2 interface{}, err error) {
	if hijacker, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// Push implements http.Pusher
func (rw *responseWrapper) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := rw.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}
