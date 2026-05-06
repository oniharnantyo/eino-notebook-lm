package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/pkg/description"
	"github.com/oniharnantyo/eino-notebook/pkg/logger"
	"github.com/oniharnantyo/eino-notebook/pkg/parser/kreuzberg"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// ImageUploader defines the interface for uploading images to storage
type ImageUploader interface {
	Upload(ctx context.Context, key string, data []byte) error
}

// ImageProcessingStage processes images from ExtractionResult.
// It uploads images to storage, generates LLM descriptions using OCR as grounding,
// embeds the description text using a text embedder, and creates Image entities.
type ImageProcessingStage struct {
	imageUploader   ImageUploader
	visionDescriber description.VisionDescriber
	textEmbedder    embedding.Embedder
	logger          *logger.Logger
}

// NewImageProcessingStage creates a new ImageProcessingStage.
func NewImageProcessingStage(
	imageUploader ImageUploader,
	visionDescriber description.VisionDescriber,
	textEmbedder embedding.Embedder,
	logger *logger.Logger,
) *ImageProcessingStage {
	return &ImageProcessingStage{
		imageUploader:   imageUploader,
		visionDescriber: visionDescriber,
		textEmbedder:    textEmbedder,
		logger:          logger,
	}
}

// Name returns "ImageProcessingStage".
func (s *ImageProcessingStage) Name() string {
	return "ImageProcessingStage"
}

// Execute processes images from the ExtractionResult.
// Input: *PipelineData
// Output: *PipelineData with Images populated
func (s *ImageProcessingStage) Execute(ctx context.Context, input StageInput) (StageOutput, error) {
	data, ok := input.Data.(*PipelineData)
	if !ok {
		return StageOutput{}, fmt.Errorf("invalid input type for ImageProcessingStage: expected *PipelineData, got %T", input.Data)
	}

	if data.ExtractionResult == nil {
		return StageOutput{Data: data}, nil
	}

	images := data.ExtractionResult.Images
	if len(images) == 0 {
		return StageOutput{Data: data}, nil
	}

	processedImages := make([]*entities.Image, 0, len(images))

	for i, img := range images {
		imageEntity, err := s.processSingleImage(ctx, input.SourceID, img, i)
		if err != nil {
			s.logger.Error("failed to process image",
				"source_id", input.SourceID,
				"index", i,
				"page", img.PageNumber,
				"error", err,
			)
			continue
		}
		processedImages = append(processedImages, imageEntity)
	}

	data.Images = processedImages

	return StageOutput{Data: data}, nil
}

// processSingleImage processes a single image through the pipeline:
// 1. Upload to storage
// 2. Generate LLM description with OCR grounding
// 3. Embed description text using text embedder
// 4. Create Image entity
func (s *ImageProcessingStage) processSingleImage(ctx context.Context, sourceID uuid.UUID, img kreuzberg.KreuzbergImage, index int) (*entities.Image, error) {
	imageID := uuid.New()

	format := img.Format
	if format == "jpg" {
		format = "jpeg"
	}
	mimeType := "image/" + format

	// 1. Upload image to storage
	s3Key := fmt.Sprintf("%s/%s.%s", sourceID.String(), imageID.String(), img.Format)
	if err := s.imageUploader.Upload(ctx, s3Key, img.Data); err != nil {
		return nil, fmt.Errorf("failed to upload image to storage: %w", err)
	}

	// 2. Generate LLM description using OCR as grounding
	desc, err := s.visionDescriber.Describe(ctx, img.Data, mimeType, img.OCRResult.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to generate vision description: %w", err)
	}

	// 3. Embed description text using text embedder
	embeddings, err := s.textEmbedder.EmbedStrings(ctx, []string{desc})
	if err != nil {
		return nil, fmt.Errorf("failed to embed description text: %w", err)
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned for description text")
	}

	embeddingVec := ConvertFloat64ToFloat32(embeddings[0])

	// 4. Create image entity
	imageEntity := &entities.Image{
		ID:          imageID,
		SourceID:    sourceID,
		S3Key:       s3Key,
		Format:      img.Format,
		Width:       img.Width,
		Height:      img.Height,
		Description: desc,
		PageNumber:  img.PageNumber,
		Embedding:   embeddingVec,
		Metadata: map[string]any{
			"ocr_elements": len(img.OCRResult.OCRElements),
		},
		CreatedAt: time.Now(),
	}

	return imageEntity, nil
}
