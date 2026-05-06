package model

import (
	"context"
	"fmt"
	"strings"

	geminiembedder "github.com/cloudwego/eino-ext/components/embedding/gemini"
	geminimodel "github.com/cloudwego/eino-ext/components/model/gemini"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/model"
	"google.golang.org/genai"
)

type Provider string

const (
	ProviderGemini   Provider = "gemini"
	ProviderOpenAI   Provider = "openai"
	ProviderLlamaCpp Provider = "llamacpp"
	ProviderOllama   Provider = "ollama"
)

// ExtractModelName extracts the actual model name from the provider-prefixed format
// e.g., "gemini/gemini-2.0-flash-exp" -> "gemini-2.0-flash-exp"
//
//	"openai/gpt-4o-mini" -> "gpt-4o-mini"
//	"gemini-2.0-flash-exp" -> "gemini-2.0-flash-exp" (no prefix)
func ExtractModelName(modelName string) string {
	if idx := strings.Index(modelName, "/"); idx != -1 {
		return modelName[idx+1:]
	}
	return modelName
}

// CreateGeminiEmbedder creates a Gemini embedder
func CreateGeminiEmbedder(ctx context.Context, client *genai.Client, modelName string, dimension int) (embedding.Embedder, error) {
	if client == nil {
		return nil, fmt.Errorf("Gemini client not initialized")
	}

	var outputDim *int32
	if dimension > 0 {
		dim := int32(dimension)
		outputDim = &dim
	}

	return geminiembedder.NewEmbedder(ctx, &geminiembedder.EmbeddingConfig{
		Client:               client,
		Model:                modelName,
		OutputDimensionality: outputDim,
	})
}

// CreateGeminiChatModel creates a Gemini chat model
func CreateGeminiChatModel(ctx context.Context, client *genai.Client, modelName string) (model.BaseChatModel, error) {
	if client == nil {
		return nil, fmt.Errorf("Gemini client not initialized")
	}

	return geminimodel.NewChatModel(ctx, &geminimodel.Config{
		Client: client,
		Model:  modelName,
	})
}

// NewGeminiClient creates a new Gemini client
func NewGeminiClient(ctx context.Context, apiKey string) (*genai.Client, error) {
	return genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
}
