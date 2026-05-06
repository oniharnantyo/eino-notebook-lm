package model

import (
	"context"
	"fmt"

	geminiembedder "github.com/cloudwego/eino-ext/components/embedding/gemini"
	ollamaembedder "github.com/cloudwego/eino-ext/components/embedding/ollama"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/oniharnantyo/eino-notebook/internal/infrastructure/config"
	visionembedding "github.com/oniharnantyo/eino-notebook/pkg/embedding"
	"github.com/oniharnantyo/eino-notebook/pkg/embedding/llamacpp"
	"google.golang.org/genai"
)

// CreateEmbedder creates an embedder based on the configuration
func CreateEmbedder(ctx context.Context, cfg *config.EmbeddingConfig) (embedding.Embedder, error) {
	if cfg == nil {
		return nil, fmt.Errorf("embedding configuration is nil")
	}

	switch Provider(cfg.Provider) {
	case ProviderGemini:
		return createGeminiEmbedder(ctx, cfg)
	case ProviderLlamaCpp:
		return createLlamaCppEmbedder(ctx, cfg)
	case ProviderOllama:
		return createOllamaEmbedder(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s", cfg.Provider)
	}
}

// CreateVisionEmbedder creates a vision-capable embedder based on the configuration
// This returns VisionEmbedder which supports both text and image embeddings
func CreateVisionEmbedder(ctx context.Context, cfg *config.EmbeddingConfig) (visionembedding.VisionEmbedder, error) {
	if cfg == nil {
		return nil, fmt.Errorf("embedding configuration is nil")
	}

	switch Provider(cfg.Provider) {
	case ProviderLlamaCpp:
		return createLlamaCppVisionEmbedder(ctx, cfg)
	case ProviderGemini:
		return nil, fmt.Errorf("vision embedding not supported for provider: %s", cfg.Provider)
	case ProviderOllama:
		return nil, fmt.Errorf("vision embedding not supported for provider: %s", cfg.Provider)
	default:
		return nil, fmt.Errorf("unsupported embedding provider for vision: %s", cfg.Provider)
	}
}

func createLlamaCppVisionEmbedder(ctx context.Context, cfg *config.EmbeddingConfig) (visionembedding.VisionEmbedder, error) {
	return llamacpp.NewEmbedder(ctx, &llamacpp.Config{
		BaseURL:        cfg.BaseURL,
		APIKey:         cfg.APIKey,
		Model:          cfg.Model,
		Dimension:      cfg.Dimension,
		PromptTemplate: cfg.PromptTemplate,
		Timeout:        cfg.Timeout,
	})
}

func createGeminiEmbedder(ctx context.Context, cfg *config.EmbeddingConfig) (embedding.Embedder, error) {
	clientConfig := &genai.ClientConfig{
		APIKey: cfg.APIKey,
	}

	client, err := genai.NewClient(ctx, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	var outputDim *int32
	if cfg.Dimension > 0 {
		dim := int32(cfg.Dimension)
		outputDim = &dim
	}

	return geminiembedder.NewEmbedder(ctx, &geminiembedder.EmbeddingConfig{
		Client:               client,
		Model:                cfg.Model,
		OutputDimensionality: outputDim,
	})
}

func createLlamaCppEmbedder(ctx context.Context, cfg *config.EmbeddingConfig) (embedding.Embedder, error) {
	return llamacpp.NewEmbedder(ctx, &llamacpp.Config{
		BaseURL:        cfg.BaseURL,
		APIKey:         cfg.APIKey,
		Model:          cfg.Model,
		Dimension:      cfg.Dimension,
		PromptTemplate: cfg.PromptTemplate,
		Timeout:        cfg.Timeout,
	})
}

func createOllamaEmbedder(ctx context.Context, cfg *config.EmbeddingConfig) (embedding.Embedder, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	return ollamaembedder.NewEmbedder(ctx, &ollamaembedder.EmbeddingConfig{
		BaseURL: baseURL,
		Model:   cfg.Model,
		Timeout: cfg.Timeout,
	})
}
