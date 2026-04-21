package llamacpp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"text/template"
	"time"

	"github.com/cloudwego/eino/components/embedding"

	"github.com/oniharnantyo/eino-notebook/pkg/embedding/templates"
)

// LlamaCppEmbedder implements the embedding.Embedder interface for llama.cpp
type LlamaCppEmbedder struct {
	config *Config
	tmpl   *template.Template
	client *http.Client
}

// NewEmbedder creates a new LlamaCppEmbedder
func NewEmbedder(ctx context.Context, cfg *Config) (*LlamaCppEmbedder, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("base_url is required for llama.cpp embedder")
	}

	// Load template
	t := templates.GetTemplate(cfg.PromptTemplate)
	parsedTmpl, err := template.New("prompt").Parse(t.Template)
	if err != nil {
		return nil, fmt.Errorf("failed to parse prompt template: %w", err)
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &LlamaCppEmbedder{
		config: cfg,
		tmpl:   parsedTmpl,
		client: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// LlamaCppResponseItem represents a single embedding item in the llama.cpp items
type LlamaCppResponseItem struct {
	Index     int         `json:"index"`
	Embedding [][]float64 `json:"embedding"` // 2D array: batch dimension then embedding values
}

// LlamaCppResponse represents the OpenAI-compatible items wrapper
type LlamaCppResponse struct {
	Object string                 `json:"object"`
	Data   []LlamaCppResponseItem `json:"data"`
	Model  string                 `json:"model"`
}

// EmbeddingRequestItem represents a single item in the embedding request
type EmbeddingRequestItem struct {
	PromptString  string   `json:"prompt_string"`
	MultimodalData []string `json:"multimodal_data"` // Array of base64 strings
}

// EmbeddingRequest represents the request body for llama.cpp embeddings
type EmbeddingRequest struct {
	Content []EmbeddingRequestItem `json:"content"`
}

// EmbedStrings generates embeddings for the given texts
func (e *LlamaCppEmbedder) EmbedStrings(ctx context.Context, texts []string, opts ...embedding.Option) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	// Build request items directly
	requestItems := make([]EmbeddingRequestItem, len(texts))
	for i, text := range texts {
		requestItems[i] = EmbeddingRequestItem{
			PromptString:  text,
			MultimodalData: []string{}, // No multimodal data for text-only
		}
	}

	request := EmbeddingRequest{Content: requestItems}
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", e.config.BaseURL+"/embeddings", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if e.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.config.APIKey)
	}

	// Debug: log request
	// fmt.Println("llama.cpp embedding request",
	// 	"url", req.URL.String(),
	// 	"body", string(requestBody),
	// 	"num_texts", len(texts))

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to llama.cpp failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("llama.cpp embedding error items",
			"status", resp.StatusCode,
			"body", string(body))
		return nil, fmt.Errorf("llama.cpp returned error status %d: %s", resp.StatusCode, string(body))
	}

	// Debug: read items body for logging
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read items body: %w", err)
	}
	// fmt.Println("llama.cpp embedding items",
	// 	"status", resp.StatusCode,
	// 	"body", string(body))

	var items []LlamaCppResponseItem
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, fmt.Errorf("failed to decode llama.cpp items: %w", err)
	}

	// Sort by index to ensure order matches input
	sort.Slice(items, func(i, j int) bool {
		return items[i].Index < items[j].Index
	})

	if len(items) != len(texts) {
		return nil, fmt.Errorf("received %d embeddings for %d input strings", len(items), len(texts))
	}

	embeddings := make([][]float64, len(items))
	for i, item := range items {
		// Extract first element from 2D array (batch dimension)
		if len(item.Embedding) == 0 {
			return nil, fmt.Errorf("empty embedding array for item %d", i)
		}
		vec := item.Embedding[0]
		// Validate dimension if configured
		if e.config.Dimension > 0 && len(vec) != e.config.Dimension {
			// Log warning or handle mismatch as per requirements
			// For now, we'll just use what we got
		}
		embeddings[i] = vec
	}

	return embeddings, nil
}

// EmbedVision generates embeddings for text with accompanying image data (multimodal)
// This is used for vision-language models like qwen3-vl that can process both text and images
func (e *LlamaCppEmbedder) EmbedVision(ctx context.Context, text string, imageData []byte, mimeType string) ([]float64, error) {
	// Encode image to base64
	encoded := base64.StdEncoding.EncodeToString(imageData)

	// Build request with multimodal data as array of base64 strings
	request := EmbeddingRequest{
		Content: []EmbeddingRequestItem{
			{
				PromptString:  text + " <__media__>",
				MultimodalData: []string{encoded},
			},
		},
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", e.config.BaseURL+"/embeddings", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if e.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.config.APIKey)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to llama.cpp failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("llama.cpp vision embedding error",
			"status", resp.StatusCode,
			"body", string(body))
		return nil, fmt.Errorf("llama.cpp returned error status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var items []LlamaCppResponseItem
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, fmt.Errorf("failed to decode llama.cpp response: %w", err)
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	// Extract first element from 2D array (batch dimension)
	vec := items[0].Embedding[0]
	if e.config.Dimension > 0 && len(vec) != e.config.Dimension {
		slog.Warn("embedding dimension mismatch",
			"expected", e.config.Dimension,
			"got", len(vec))
	}

	return vec, nil
}
