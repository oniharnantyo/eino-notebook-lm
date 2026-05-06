package model

import (
	"context"
	"fmt"

	geminimodel "github.com/cloudwego/eino-ext/components/model/gemini"
	ollamamodel "github.com/cloudwego/eino-ext/components/model/ollama"
	openaimodel "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/oniharnantyo/eino-notebook/internal/infrastructure/config"
	"google.golang.org/genai"
)

// CreateToolCallingChatModel creates a chat model that supports tool calling
func CreateToolCallingChatModel(ctx context.Context, cfg *config.ChatConfig) (model.ToolCallingChatModel, error) {
	if cfg == nil {
		return nil, fmt.Errorf("chat configuration is nil")
	}

	// Assuming the same providers, but returning model.ChatModel which supports tool calling
	switch Provider(cfg.Provider) {
	case ProviderGemini:
		return createGeminiChatModel(ctx, cfg)
	case ProviderOpenAI:
		return createOpenAIChatModel(ctx, cfg)
	case ProviderOllama:
		return createOllamaChatModel(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported chat provider: %s", cfg.Provider)
	}
}

func createGeminiChatModel(ctx context.Context, cfg *config.ChatConfig) (model.ToolCallingChatModel, error) {
	clientConfig := &genai.ClientConfig{
		APIKey: cfg.APIKey,
	}

	client, err := genai.NewClient(ctx, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return geminimodel.NewChatModel(ctx, &geminimodel.Config{
		Client: client,
		Model:  cfg.Model,
	})
}

func createOpenAIChatModel(ctx context.Context, cfg *config.ChatConfig) (model.ToolCallingChatModel, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required for OpenAI provider")
	}

	return openaimodel.NewChatModel(ctx, &openaimodel.ChatModelConfig{
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
		BaseURL: cfg.BaseURL,
	})
}

func createOllamaChatModel(ctx context.Context, cfg *config.ChatConfig) (model.ToolCallingChatModel, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	config := &ollamamodel.ChatModelConfig{
		BaseURL: baseURL,
		Model:   cfg.Model,
		Timeout: cfg.Timeout,
	}

	if cfg.KeepAlive > 0 {
		config.KeepAlive = &cfg.KeepAlive
	}

	return ollamamodel.NewChatModel(ctx, config)
}
