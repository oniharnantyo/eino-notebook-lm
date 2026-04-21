package embedding

import (
	"context"

	"github.com/cloudwego/eino/components/embedding"
)

// VisionEmbedder extends the base Embedder interface to support multimodal embeddings
// with image data. This is used for vision-language models like qwen3-vl.
type VisionEmbedder interface {
	embedding.Embedder

	// EmbedVision generates an embedding for text with accompanying image data.
	// The image data should be raw bytes (e.g., from an image file).
	// The mimeType should be the image MIME type (e.g., "image/png", "image/jpeg").
	EmbedVision(ctx context.Context, text string, imageData []byte, mimeType string) ([]float64, error)
}
