package model

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	geminimodel "github.com/cloudwego/eino-ext/components/model/gemini"
	openaimodel "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/oniharnantyo/eino-notebook/internal/infrastructure/config"
	"google.golang.org/genai"
)

// CreateChatModel creates a chat model based on the configuration
func CreateChatModel(ctx context.Context, cfg *config.ChatConfig) (model.BaseChatModel, error) {
	if cfg == nil {
		return nil, fmt.Errorf("chat configuration is nil")
	}

	switch Provider(cfg.Provider) {
	case ProviderGemini:
		return createGeminiChatModel(ctx, cfg)
	case ProviderOpenAI:
		return createOpenAIChatModel(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported chat provider: %s", cfg.Provider)
	}
}

func createGeminiChatModel(ctx context.Context, cfg *config.ChatConfig) (model.BaseChatModel, error) {
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

func createOpenAIChatModel(ctx context.Context, cfg *config.ChatConfig) (model.BaseChatModel, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required for OpenAI provider")
	}

	return openaimodel.NewChatModel(ctx, &openaimodel.ChatModelConfig{
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
		BaseURL: cfg.BaseURL,
	})
}
