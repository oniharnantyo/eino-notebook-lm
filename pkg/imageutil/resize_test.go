package imageutil

import (
	"image"
	"image/color"
	"image/jpeg"
	"testing"
	"bytes"
)

func TestResizeToFit(t *testing.T) {
	// Create a large dummy image (1000x1000)
	img := image.NewRGBA(image.Rect(0, 0, 1000, 1000))
	for x := 0; x < 1000; x++ {
		for y := 0; y < 1000; y++ {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 255, 255})
		}
	}

	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 100})
	if err != nil {
		t.Fatal(err)
	}

	largeData := buf.Bytes()
	limit := 50 * 1024 // Set a small limit for testing (50KB)

	if len(largeData) <= limit {
		t.Fatalf("Test setup failed: dummy image is too small (%d bytes)", len(largeData))
	}

	t.Logf("Original size: %d bytes", len(largeData))

	resized, err := ResizeToFit(largeData, "image/jpeg", limit)
	if err != nil {
		t.Fatalf("ResizeToFit failed: %v", err)
	}

	t.Logf("Resized size: %d bytes", len(resized))

	if len(resized) > limit {
		t.Errorf("Resized image still too large: %d > %d", len(resized), limit)
	}

	if len(resized) == len(largeData) {
		t.Errorf("Image was not resized")
	}
}
