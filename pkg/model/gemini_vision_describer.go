package model

import (
	"context"
	"fmt"

	"github.com/oniharnantyo/eino-notebook/internal/infrastructure/config"
	"github.com/oniharnantyo/eino-notebook/pkg/description"
	"google.golang.org/genai"
)

type geminiVisionDescriber struct {
	client    *genai.Client
	modelName string
}

func newGeminiVisionDescriber(client *genai.Client, modelName string) description.VisionDescriber {
	return &geminiVisionDescriber{
		client:    client,
		modelName: modelName,
	}
}

func (g *geminiVisionDescriber) Describe(ctx context.Context, image []byte, mimeType string, ocrText string) (string, error) {
	prompt := "Provide a detailed description of this image. "
	if ocrText != "" {
		prompt += fmt.Sprintf("The following text was extracted from the image via OCR and should be used as grounding context to ensure accuracy of names, technical terms, and data: \n\n%s\n\n", ocrText)
	}
	prompt += "Focus on factual observation, identifying key elements, and explaining the contextual meaning of the image within a technical document. If it's a diagram or chart, explain the relationships and data points shown."

	parts := []*genai.Part{
		{Text: prompt},
		{
			InlineData: &genai.Blob{
				MIMEType: mimeType,
				Data:     image,
			},
		},
	}

	result, err := g.client.Models.GenerateContent(ctx, ExtractModelName(g.modelName), []*genai.Content{
		{Parts: parts},
	}, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate image description with Gemini: %w", err)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini returned an empty description")
	}

	return result.Candidates[0].Content.Parts[0].Text, nil
}

func createGeminiVisionDescriber(ctx context.Context, cfg *config.ChatConfig) (description.VisionDescriber, error) {
	client, err := NewGeminiClient(ctx, cfg.APIKey)
	if err != nil {
		return nil, err
	}

	return newGeminiVisionDescriber(client, cfg.Model), nil
}
