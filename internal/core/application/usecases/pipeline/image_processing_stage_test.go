package pipeline

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/extractor"
	"github.com/oniharnantyo/eino-notebook/pkg/logger"
	"github.com/oniharnantyo/eino-notebook/pkg/parser/kreuzberg"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// mockImageUploader is a mock implementation of ImageUploader
type mockImageUploader struct {
	uploadFunc func(ctx context.Context, key string, data []byte) error
}

func (m *mockImageUploader) Upload(ctx context.Context, key string, data []byte) error {
	if m.uploadFunc != nil {
		return m.uploadFunc(ctx, key, data)
	}
	return nil
}

// mockVisionDescriber is a mock implementation of description.VisionDescriber
type mockVisionDescriber struct {
	describeFunc func(ctx context.Context, image []byte, mimeType string, ocrText string) (string, error)
}

func (m *mockVisionDescriber) Describe(ctx context.Context, image []byte, mimeType string, ocrText string) (string, error) {
	if m.describeFunc != nil {
		return m.describeFunc(ctx, image, mimeType, ocrText)
	}
	return "test description", nil
}

// mockTextEmbedder is a mock implementation of embedding.Embedder
type mockTextEmbedder struct {
	embedStringsFunc func(ctx context.Context, texts []string, opts ...embedding.Option) ([][]float64, error)
}

func (m *mockTextEmbedder) EmbedStrings(ctx context.Context, texts []string, opts ...embedding.Option) ([][]float64, error) {
	if m.embedStringsFunc != nil {
		return m.embedStringsFunc(ctx, texts, opts...)
	}
	result := make([][]float64, len(texts))
	for i := range texts {
		result[i] = []float64{0.1, 0.2, 0.3}
	}
	return result, nil
}

func TestImageProcessingStage_Execute(t *testing.T) {
	ctx := context.Background()
	sourceID := uuid.New()
	log := logger.New(logger.LevelInfo, "json")

	t.Run("success - processes images successfully", func(t *testing.T) {
		mockUploader := &mockImageUploader{
			uploadFunc: func(ctx context.Context, key string, data []byte) error {
				return nil
			},
		}
		mockDescriber := &mockVisionDescriber{
			describeFunc: func(ctx context.Context, image []byte, mimeType string, ocrText string) (string, error) {
				return "test image description", nil
			},
		}
		mockEmbedder := &mockTextEmbedder{
			embedStringsFunc: func(ctx context.Context, texts []string, opts ...embedding.Option) ([][]float64, error) {
				result := make([][]float64, len(texts))
				for i := range texts {
					result[i] = []float64{0.1, 0.2, 0.3, 0.4}
				}
				return result, nil
			},
		}

		stage := NewImageProcessingStage(mockUploader, mockDescriber, mockEmbedder, log)

		images := []kreuzberg.KreuzbergImage{
			{
				Data:       []byte("fake image data 1"),
				Format:     "png",
				Width:      100,
				Height:     100,
				PageNumber: 1,
				OCRResult: kreuzberg.KreuzbergOCRResult{
					Content: "ocr text 1",
				},
			},
			{
				Data:       []byte("fake image data 2"),
				Format:     "jpg",
				Width:      200,
				Height:     200,
				PageNumber: 2,
				OCRResult: kreuzberg.KreuzbergOCRResult{
					Content: "ocr text 2",
				},
			},
		}

		input := StageInput{
			SourceID: sourceID,
			Data: &PipelineData{
				ExtractionResult: &extractor.ExtractionResult{
					Images: images,
				},
			},
		}

		output, err := stage.Execute(ctx, input)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, ok := output.Data.(*PipelineData)
		if !ok {
			t.Fatalf("expected output type *PipelineData, got %T", output.Data)
		}

		resultImages := data.Images

		if len(resultImages) != 2 {
			t.Errorf("expected 2 images, got %d", len(resultImages))
		}

		// Verify first image
		img1 := resultImages[0]
		if img1.SourceID != sourceID {
			t.Errorf("expected source_id %s, got %s", sourceID, img1.SourceID)
		}
		if img1.Format != "png" {
			t.Errorf("expected format png, got %s", img1.Format)
		}
		if img1.Description != "test image description" {
			t.Errorf("expected description 'test image description', got '%s'", img1.Description)
		}
		if len(img1.Embedding) != 4 {
			t.Errorf("expected embedding length 4, got %d", len(img1.Embedding))
		}

		// Verify second image (jpg should be converted to jpeg for S3 key)
		img2 := resultImages[1]
		if img2.PageNumber != 2 {
			t.Errorf("expected page_number 2, got %d", img2.PageNumber)
		}
	})

	t.Run("image failure - skips failed image and continues", func(t *testing.T) {
		uploadCallCount := 0
		mockUploader := &mockImageUploader{
			uploadFunc: func(ctx context.Context, key string, data []byte) error {
				uploadCallCount++
				// Fail on second image
				if uploadCallCount == 2 {
					return errors.New("upload failed")
				}
				return nil
			},
		}
		mockDescriber := &mockVisionDescriber{
			describeFunc: func(ctx context.Context, image []byte, mimeType string, ocrText string) (string, error) {
				return "description", nil
			},
		}
		mockEmbedder := &mockTextEmbedder{
			embedStringsFunc: func(ctx context.Context, texts []string, opts ...embedding.Option) ([][]float64, error) {
				return [][]float64{{0.1, 0.2}}, nil
			},
		}

		stage := NewImageProcessingStage(mockUploader, mockDescriber, mockEmbedder, log)

		images := []kreuzberg.KreuzbergImage{
			{
				Data:       []byte("image 1"),
				Format:     "png",
				Width:      100,
				Height:     100,
				PageNumber: 1,
				OCRResult:  kreuzberg.KreuzbergOCRResult{Content: "ocr 1"},
			},
			{
				Data:       []byte("image 2"),
				Format:     "png",
				Width:      100,
				Height:     100,
				PageNumber: 2,
				OCRResult:  kreuzberg.KreuzbergOCRResult{Content: "ocr 2"},
			},
			{
				Data:       []byte("image 3"),
				Format:     "png",
				Width:      100,
				Height:     100,
				PageNumber: 3,
				OCRResult:  kreuzberg.KreuzbergOCRResult{Content: "ocr 3"},
			},
		}

		input := StageInput{
			SourceID: sourceID,
			Data: &PipelineData{
				ExtractionResult: &extractor.ExtractionResult{
					Images: images,
				},
			},
		}

		output, err := stage.Execute(ctx, input)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, ok := output.Data.(*PipelineData)
		if !ok {
			t.Fatalf("expected output type *PipelineData, got %T", output.Data)
		}

		resultImages := data.Images

		// Should have processed 2 images (1 and 3), skipping 2 due to upload failure
		if len(resultImages) != 2 {
			t.Errorf("expected 2 successfully processed images, got %d", len(resultImages))
		}

		if uploadCallCount != 3 {
			t.Errorf("expected 3 upload attempts, got %d", uploadCallCount)
		}
	})

	t.Run("no images - returns empty slice", func(t *testing.T) {
		mockUploader := &mockImageUploader{}
		mockDescriber := &mockVisionDescriber{}
		mockEmbedder := &mockTextEmbedder{}

		stage := NewImageProcessingStage(mockUploader, mockDescriber, mockEmbedder, log)

		input := StageInput{
			SourceID: sourceID,
			Data: &PipelineData{
				ExtractionResult: &extractor.ExtractionResult{
					Images: []kreuzberg.KreuzbergImage{},
				},
			},
		}

		output, err := stage.Execute(ctx, input)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, ok := output.Data.(*PipelineData)
		if !ok {
			t.Fatalf("expected output type *PipelineData, got %T", output.Data)
		}

		if len(data.Images) != 0 {
			t.Errorf("expected 0 images, got %d", len(data.Images))
		}
	})

	t.Run("invalid input type - returns error", func(t *testing.T) {
		mockUploader := &mockImageUploader{}
		mockDescriber := &mockVisionDescriber{}
		mockEmbedder := &mockTextEmbedder{}

		stage := NewImageProcessingStage(mockUploader, mockDescriber, mockEmbedder, log)

		input := StageInput{
			SourceID: sourceID,
			Data:     "invalid input",
		}

		_, err := stage.Execute(ctx, input)

		if err == nil {
			t.Fatal("expected error for invalid input type")
		}

		if err.Error()[:len("invalid input type")] != "invalid input type" {
			t.Errorf("expected error to start with 'invalid input type', got: %v", err)
		}
	})


	t.Run("description failure - skips image and continues", func(t *testing.T) {
		mockUploader := &mockImageUploader{}
		mockDescriber := &mockVisionDescriber{
			describeFunc: func(ctx context.Context, image []byte, mimeType string, ocrText string) (string, error) {
				return "", errors.New("description failed")
			},
		}
		mockEmbedder := &mockTextEmbedder{}

		stage := NewImageProcessingStage(mockUploader, mockDescriber, mockEmbedder, log)

		images := []kreuzberg.KreuzbergImage{
			{
				Data:       []byte("image"),
				Format:     "png",
				Width:      100,
				Height:     100,
				PageNumber: 1,
				OCRResult:  kreuzberg.KreuzbergOCRResult{Content: "ocr"},
			},
		}

		input := StageInput{
			SourceID: sourceID,
			Data: &PipelineData{
				ExtractionResult: &extractor.ExtractionResult{
					Images: images,
				},
			},
		}

		output, err := stage.Execute(ctx, input)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, ok := output.Data.(*PipelineData)
		if !ok {
			t.Fatalf("expected output type *PipelineData, got %T", output.Data)
		}

		resultImages := data.Images

		if len(resultImages) != 0 {
			t.Errorf("expected 0 images due to description failure, got %d", len(resultImages))
		}
	})

	t.Run("embedding failure - skips image and continues", func(t *testing.T) {
		mockUploader := &mockImageUploader{}
		mockDescriber := &mockVisionDescriber{
			describeFunc: func(ctx context.Context, image []byte, mimeType string, ocrText string) (string, error) {
				return "description", nil
			},
		}
		mockEmbedder := &mockTextEmbedder{
			embedStringsFunc: func(ctx context.Context, texts []string, opts ...embedding.Option) ([][]float64, error) {
				return nil, errors.New("embedding failed")
			},
		}

		stage := NewImageProcessingStage(mockUploader, mockDescriber, mockEmbedder, log)

		images := []kreuzberg.KreuzbergImage{
			{
				Data:       []byte("image"),
				Format:     "png",
				Width:      100,
				Height:     100,
				PageNumber: 1,
				OCRResult:  kreuzberg.KreuzbergOCRResult{Content: "ocr"},
			},
		}

		input := StageInput{
			SourceID: sourceID,
			Data: &PipelineData{
				ExtractionResult: &extractor.ExtractionResult{
					Images: images,
				},
			},
		}

		output, err := stage.Execute(ctx, input)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, ok := output.Data.(*PipelineData)
		if !ok {
			t.Fatalf("expected output type *PipelineData, got %T", output.Data)
		}

		resultImages := data.Images

		if len(resultImages) != 0 {
			t.Errorf("expected 0 images due to embedding failure, got %d", len(resultImages))
		}
	})
}
