package model

import (
	"context"
	"testing"

	"github.com/oniharnantyo/eino-notebook/internal/infrastructure/config"
)

func TestCreateChatModel(t *testing.T) {
	ctx := context.Background()

	t.Run("UnsupportedProvider", func(t *testing.T) {
		cfg := &config.ChatConfig{
			Provider: "unsupported",
			Model:    "some-model",
		}
		_, err := CreateChatModel(ctx, cfg)
		if err == nil {
			t.Errorf("expected error for unsupported provider, got nil")
		}
	})

	t.Run("NilConfig", func(t *testing.T) {
		_, err := CreateChatModel(ctx, nil)
		if err == nil {
			t.Errorf("expected error for nil config, got nil")
		}
	})

	// Note: Testing actual Gemini creation requires a valid API key or mock
}
