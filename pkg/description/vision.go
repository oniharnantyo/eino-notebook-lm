package description

import "context"

// VisionDescriber defines the interface for generating text descriptions of images.
type VisionDescriber interface {
	// Describe generates a text description of the image.
	// ocrText is provided as grounding context from the OCR process.
	Describe(ctx context.Context, image []byte, mimeType string, ocrText string) (string, error)
}
