package model

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/embedding"
	visionembedding "github.com/oniharnantyo/eino-notebook/pkg/embedding"
	"google.golang.org/genai"
)

// geminiVisionEmbedder implements visionembedding.VisionEmbedder using Google's genai SDK.
// It supports multimodal embeddings (text + image) for models like gemini-embedding-2.
type geminiVisionEmbedder struct {
	embedding.Embedder
	client    *genai.Client
	modelName string
}

var _ visionembedding.VisionEmbedder = (*geminiVisionEmbedder)(nil)

func (g *geminiVisionEmbedder) EmbedVision(ctx context.Context, text string, imageData []byte, mimeType string) ([]float64, error) {
	contents := []*genai.Content{
		{
			Parts: []*genai.Part{
				{Text: text},
				{
					InlineData: &genai.Blob{
						MIMEType: mimeType,
						Data:     imageData,
					},
				},
			},
		},
	}

	// Note: EmbedContent returns float32 values in result.Embeddings[0].Values
	result, err := g.client.Models.EmbedContent(ctx, ExtractModelName(g.modelName), contents, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Gemini multimodal embedding: %w", err)
	}

	if len(result.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned from Gemini")
	}

	values := result.Embeddings[0].Values
	res := make([]float64, len(values))
	for i, v := range values {
		res[i] = float64(v)
	}

	return res, nil
}
