package model

import (
	"context"
	"fmt"

	"github.com/oniharnantyo/eino-notebook/internal/infrastructure/config"
	"github.com/oniharnantyo/eino-notebook/pkg/description"
)

// CreateVisionDescriber creates a vision describer based on the configuration
func CreateVisionDescriber(ctx context.Context, cfg *config.ChatConfig) (description.VisionDescriber, error) {
	if cfg == nil {
		return nil, fmt.Errorf("vision description configuration is nil")
	}

	switch Provider(cfg.Provider) {
	case ProviderGemini:
		return createGeminiVisionDescriber(ctx, cfg)
	case ProviderOpenAI:
		return createOpenAIVisionDescriber(ctx, cfg)
	case ProviderLlamaCpp:
		return createLlamaCPPVisionDescriber(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported vision description provider: %s", cfg.Provider)
	}
}
