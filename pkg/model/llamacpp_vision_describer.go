package model

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/oniharnantyo/eino-notebook/internal/infrastructure/config"
	"github.com/oniharnantyo/eino-notebook/pkg/description"
)

type llamacppVisionDescriber struct {
	baseURL   string
	modelName string
	apiKey    string
	client    *http.Client
}

func newLlamaCPPVisionDescriber(baseURL, modelName, apiKey string) description.VisionDescriber {
	return &llamacppVisionDescriber{
		baseURL:   baseURL,
		modelName: modelName,
		apiKey:    apiKey,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (l *llamacppVisionDescriber) Describe(ctx context.Context, image []byte, mimeType string, ocrText string) (string, error) {
	prompt := "Provide a detailed description of this image. "
	if ocrText != "" {
		prompt += fmt.Sprintf("The following text was extracted from the image via OCR and should be used as grounding context to ensure accuracy of names, technical terms, and data: \n\n%s\n\n", ocrText)
	}
	prompt += "Focus on factual observation, identifying key elements, and explaining the contextual meaning of the image within a technical document. If it's a diagram or chart, explain the relationships and data points shown."

	encodedImage := base64.StdEncoding.EncodeToString(image)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, encodedImage)

	content := []chatCompletionContent{
		{
			Type: "text",
			Text: prompt,
		},
		{
			Type: "image_url",
			ImageURL: &chatCompletionImage{
				URL: dataURL,
			},
		},
	}

	request := chatCompletionRequest{
		Model: ExtractModelName(l.modelName),
		Messages: []chatCompletionMessage{
			{
				Role:    "user",
				Content: content,
			},
		},
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	url := fmt.Sprintf("%s/v1/chat/completions", l.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if l.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+l.apiKey)
	}

	resp, err := l.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request to llamacpp failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llamacpp returned error status %d: %s", resp.StatusCode, string(body))
	}

	var response chatCompletionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to decode llamacpp response: %w", err)
	}

	if response.Error != nil {
		return "", fmt.Errorf("llamacpp returned error: %s", response.Error.Message)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("llamacpp returned no choices")
	}

	return response.Choices[0].Message.Content, nil
}

func createLlamaCPPVisionDescriber(ctx context.Context, cfg *config.ChatConfig) (description.VisionDescriber, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("base_url is required for llamacpp vision describer")
	}

	return newLlamaCPPVisionDescriber(cfg.BaseURL, cfg.Model, cfg.APIKey), nil
}
