package middleware

import (
	"bytes"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"runtime/debug"
	"strings"
	"time"
)

// Logger is a middleware that logs HTTP requests with comprehensive error details
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Check if this is a multipart/form-data request (file upload)
		isMultipart := strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data")

		// Read and buffer request body (limit to 4KB)
		// Skip reading for multipart requests to avoid logging binary file content
		var requestBody string
		var uploadedFile string
		if r.Body != nil && r.Method != http.MethodGet && r.Method != http.MethodHead {
			if isMultipart {
				// For multipart requests, just extract metadata without buffering body
				uploadedFile = extractFileMetadata(r)
			} else {
				// For other requests, buffer body for logging
				bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 4096))
				if err == nil && len(bodyBytes) > 0 {
					requestBody = string(bodyBytes)
					// Restore body for downstream handlers
					r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				}
			}
		}

		// Wrap the response writer to capture status code and body
		wrapped := &responseWrapper{
			ResponseWriter: w,
			status:         http.StatusOK,
			bodyBuffer:     &bytes.Buffer{},
		}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Milliseconds()
		responseBody := wrapped.bodyBuffer.String()

		// Build common log attributes
		baseAttrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.status,
			"duration", duration,
		}

		// Add uploaded file info if present
		if uploadedFile != "" {
			baseAttrs = append(baseAttrs, "file", uploadedFile)
		}

		// Add request body if present (skip for multipart)
		if requestBody != "" {
			baseAttrs = append(baseAttrs, "request", requestBody)
		}

		// Add response body if present
		if responseBody != "" {
			baseAttrs = append(baseAttrs, "response", responseBody)
		}

		// Log based on status code level
		if wrapped.status >= 500 {
			slog.Error("request", baseAttrs...)
		} else if wrapped.status >= 400 {
			slog.Warn("request", baseAttrs...)
		} else {
			slog.Info("request", baseAttrs...)
		}
	})
}

// extractFileMetadata extracts file metadata from multipart request without buffering body
func extractFileMetadata(r *http.Request) string {
	// Parse multipart form data to get file info
	// We use a small limit to avoid reading large files
	err := r.ParseMultipartForm(32 << 20) // 32MB max memory
	if err != nil {
		return ""
	}

	// Look for file in form
	if r.MultipartForm != nil {
		for _, files := range r.MultipartForm.File {
			for _, file := range files {
				// Return first file metadata found
				return file.Filename
			}
		}
	}

	return ""
}

// cleanMultipartReader wraps multipart.Reader to skip file content
type cleanMultipartReader struct {
	*multipart.Reader
}

func (cr *cleanMultipartReader) ReadForm(maxMemory int64) (*multipart.Form, error) {
	form := &multipart.Form{
		Value: make(map[string][]string),
		File:  make(map[string][]*multipart.FileHeader),
	}

	for {
		part, err := cr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		name := part.FormName()
		if name == "" {
			continue
		}

		// Check if this part is a file upload
		filename := part.FileName()
		if filename != "" {
			// For file uploads, create header without reading content
			header := part.Header
			form.File[name] = append(form.File[name], &multipart.FileHeader{
				Filename: filename,
				Header:   header,
			})
			continue
		}

		// For regular form fields, read value
		var value bytes.Buffer
		io.Copy(&value, part)
		form.Value[name] = append(form.Value[name], value.String())
	}

	return form, nil
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
func (rw *responseWrapper) Hijack() (c any, rw2 any, err error) {
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
