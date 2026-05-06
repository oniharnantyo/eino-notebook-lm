package model

import (
	"context"
	"testing"

	"github.com/oniharnantyo/eino-notebook/internal/infrastructure/config"
)

func TestCreateToolCallingChatModel(t *testing.T) {
	ctx := context.Background()

	t.Run("UnsupportedProvider", func(t *testing.T) {
		cfg := &config.ChatConfig{
			Provider: "unsupported",
			Model:    "some-model",
		}
		_, err := CreateToolCallingChatModel(ctx, cfg)
		if err == nil {
			t.Errorf("expected error for unsupported provider, got nil")
		}
	})

	t.Run("NilConfig", func(t *testing.T) {
		_, err := CreateToolCallingChatModel(ctx, nil)
		if err == nil {
			t.Errorf("expected error for nil config, got nil")
		}
	})

	t.Run("OllamaProvider", func(t *testing.T) {
		cfg := &config.ChatConfig{
			Provider: "ollama",
			Model:    "qwen2.5",
			BaseURL:  "http://localhost:11434",
		}
		model, err := CreateToolCallingChatModel(ctx, cfg)
		if err != nil {
			t.Errorf("failed to create Ollama chat model: %v", err)
		}
		if model == nil {
			t.Errorf("expected model, got nil")
		}
	})

	// Note: Testing actual Gemini creation requires a valid API key or mock
}
