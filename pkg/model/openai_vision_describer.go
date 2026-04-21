package model

import (
	"context"
	"strings"

	"github.com/oniharnantyo/eino-notebook/internal/infrastructure/config"
	"github.com/oniharnantyo/eino-notebook/pkg/description"
)

func createOpenAIVisionDescriber(ctx context.Context, cfg *config.ChatConfig) (description.VisionDescriber, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}

	// Remove trailing /v1 if present, as the describer adds it
	baseURL = strings.TrimSuffix(baseURL, "/v1")
	baseURL = strings.TrimSuffix(baseURL, "/")

	return newLlamaCPPVisionDescriber(baseURL, cfg.Model, cfg.APIKey), nil
}
