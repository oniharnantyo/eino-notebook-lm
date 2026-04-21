package imageutil

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"math"

	"golang.org/x/image/draw"
)

const (
	MaxEmbeddingSize = 325 * 1024 // 325 KB
)

// ResizeToFit scales an image down proportionally if its byte size exceeds the threshold.
// It returns the potentially resized image data.
func ResizeToFit(data []byte, mimeType string, maxSize int) ([]byte, error) {
	if len(data) <= maxSize {
		return data, nil
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Heuristic: scale based on area to get closer to target size in one pass.
	// We use the ratio of current size to max size as a rough area factor.
	sizeRatio := float64(len(data)) / float64(maxSize)
	scaleFactor := 1.0 / math.Sqrt(sizeRatio)
	
	// Safety margin to ensure we hit the target under 325KB
	scaleFactor *= 0.9 

	newWidth := int(float64(width) * scaleFactor)
	newHeight := int(float64(height) * scaleFactor)

	newImg := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	draw.BiLinear.Scale(newImg, newImg.Bounds(), img, img.Bounds(), draw.Over, nil)

	var buf bytes.Buffer
	switch mimeType {
	case "image/jpeg", "image/jpg":
		// Use a slightly lower quality to further reduce size if needed
		if err := jpeg.Encode(&buf, newImg, &jpeg.Options{Quality: 85}); err != nil {
			return nil, fmt.Errorf("failed to encode jpeg: %w", err)
		}
	case "image/png":
		if err := png.Encode(&buf, newImg); err != nil {
			return nil, fmt.Errorf("failed to encode png: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported mime type for resizing: %s", mimeType)
	}

	// If it's still too large (rare with 0.9 margin), recurse once with aggressive scaling
	if buf.Len() > maxSize {
		return ResizeToFit(buf.Bytes(), mimeType, maxSize)
	}

	return buf.Bytes(), nil
}
