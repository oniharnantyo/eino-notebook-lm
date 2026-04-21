package model

import (
	"context"
	"testing"

	"github.com/oniharnantyo/eino-notebook/internal/infrastructure/config"
)

func TestCreateEmbedder(t *testing.T) {
	ctx := context.Background()

	t.Run("UnsupportedProvider", func(t *testing.T) {
		cfg := &config.EmbeddingConfig{
			Provider: "unsupported",
			Model:    "some-model",
		}
		_, err := CreateEmbedder(ctx, cfg)
		if err == nil {
			t.Errorf("expected error for unsupported provider, got nil")
		}
	})

	t.Run("NilConfig", func(t *testing.T) {
		_, err := CreateEmbedder(ctx, nil)
		if err == nil {
			t.Errorf("expected error for nil config, got nil")
		}
	})

	t.Run("LlamaCppMissingBaseURL", func(t *testing.T) {
		cfg := &config.EmbeddingConfig{
			Provider: "llamacpp",
			Model:    "nomic",
		}
		_, err := CreateEmbedder(ctx, cfg)
		if err == nil {
			t.Errorf("expected error for missing base_url in llamacpp, got nil")
		}
	})
}
