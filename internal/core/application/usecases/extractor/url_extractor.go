package extractor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases"
)

// URLContentExtractor extracts content from URLs
// Single Responsibility: Only handles URL content extraction
type URLContentExtractor struct {
	client     *http.Client
	maxTimeout time.Duration
}

// NewURLContentExtractor creates a new URL content extractor
func NewURLContentExtractor(maxTimeout time.Duration) *URLContentExtractor {
	return &URLContentExtractor{
		client: &http.Client{
			Timeout: maxTimeout,
			// Follow redirects
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return nil
			},
		},
		maxTimeout: maxTimeout,
	}
}

// Extract extracts content from a URL
func (e *URLContentExtractor) Extract(ctx context.Context, source usecases.ContentSource) (string, map[string]interface{}, error) {
	if source.URL == "" {
		return "", nil, fmt.Errorf("no URL provided for URL extraction")
	}

	// Validate URL
	parsedURL, err := url.Parse(source.URL)
	if err != nil {
		return "", nil, fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", nil, fmt.Errorf("unsupported URL scheme: %s (only http/https supported)", parsedURL.Scheme)
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.URL, nil)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent to avoid being blocked
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; EinoNotebook/1.0)")

	// Execute request
	resp, err := e.client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("URL returned status %d: %s", resp.StatusCode, resp.Status)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !e.isTextContent(contentType) {
		return "", nil, fmt.Errorf("unsupported content type: %s (only text content supported)", contentType)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Create metadata
	metadata := make(map[string]interface{})
	metadata["url"] = source.URL
	metadata["status_code"] = resp.StatusCode
	metadata["content_type"] = contentType
	metadata["content_length"] = len(body)

	// Return the content
	return string(body), metadata, nil
}

// isTextContent checks if the content type is text-based
func (e *URLContentExtractor) isTextContent(contentType string) bool {
	textTypes := []string{
		"text/html",
		"text/plain",
		"text/xml",
		"application/json",
		"application/xml",
		"application/xhtml+xml",
	}

	for _, tt := range textTypes {
		if contains(contentType, tt) {
			return true
		}
	}
	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || indexOf(s, substr) >= 0)
}

// indexOf finds the index of a substring
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
