package image

import (
	"context"
	"fmt"
	"time"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/internal/infrastructure/storage"
	"github.com/oniharnantyo/eino-notebook/pkg/description"
	visionembedding "github.com/oniharnantyo/eino-notebook/pkg/embedding"
	"github.com/oniharnantyo/eino-notebook/pkg/imageutil"
	"github.com/oniharnantyo/eino-notebook/pkg/logger"
	"github.com/oniharnantyo/eino-notebook/pkg/parser/kreuzberg"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// ImageUseCase defines the interface for image business logic
type ImageUseCase interface {
	// ProcessImages processes images from Kreuzberg into image entities with embeddings
	ProcessImages(ctx context.Context, sourceID uuid.UUID, images []kreuzberg.KreuzbergImage) error
}

type imageUseCase struct {
	imageRepo       repositories.ImageRepository
	s3Storage       *storage.S3Storage
	visionEmbedder  visionembedding.VisionEmbedder
	visionDescriber description.VisionDescriber
	logger          *logger.Logger
}

// NewImageUseCase creates a new image use case
func NewImageUseCase(
	imageRepo repositories.ImageRepository,
	s3Storage *storage.S3Storage,
	visionEmbedder visionembedding.VisionEmbedder,
	visionDescriber description.VisionDescriber,
	log *logger.Logger,
) ImageUseCase {
	return &imageUseCase{
		imageRepo:       imageRepo,
		s3Storage:       s3Storage,
		visionEmbedder:  visionEmbedder,
		visionDescriber: visionDescriber,
		logger:          log,
	}
}

// ProcessImages uploads images to S3, generates LLM descriptions, generates vision embeddings, and saves to repository.
// ProcessImages processes images from Kreuzberg into image entities with embeddings.
// It logs and skips individual image failures instead of aborting the entire process.
func (uc *imageUseCase) ProcessImages(ctx context.Context, sourceID uuid.UUID, images []kreuzberg.KreuzbergImage) error {
	if len(images) == 0 {
		return nil
	}

	for i, img := range images {
		if err := uc.processSingleImage(ctx, sourceID, img); err != nil {
			uc.logger.Error("failed to process image",
				"source_id", sourceID,
				"index", i,
				"page", img.PageNumber,
				"error", err,
			)
			continue
		}
	}

	return nil
}

func (uc *imageUseCase) processSingleImage(ctx context.Context, sourceID uuid.UUID, img kreuzberg.KreuzbergImage) error {
	// 1. Generate image ID
	imageID := uuid.New()

	format := img.Format
	if format == "jpg" {
		format = "jpeg"
	}
	mimeType := "image/" + format

	// 2. Upload image binary to S3
	s3Key := fmt.Sprintf("%s/%s.%s", sourceID.String(), imageID.String(), img.Format)
	if err := uc.s3Storage.Upload(ctx, s3Key, img.Data); err != nil {
		return fmt.Errorf("failed to upload image to S3: %w", err)
	}

	// 3. Generate LLM description using OCR as grounding
	if uc.visionDescriber == nil {
		return fmt.Errorf("vision describer is not initialized, cannot process images")
	}

	desc, err := uc.visionDescriber.Describe(ctx, img.Data, mimeType, img.OCRResult.Content)
	if err != nil {
		return fmt.Errorf("failed to generate vision description for image %s: %w", imageID, err)
	}

	// 4. Generate multimodal vision embedding using description as text prompt
	if uc.visionEmbedder == nil {
		return fmt.Errorf("vision embedder is not initialized, cannot process images")
	}

	// Resize image for embedding if it exceeds 325 KB
	embeddingData, err := imageutil.ResizeToFit(img.Data, mimeType, imageutil.MaxEmbeddingSize)
	if err != nil {
		return fmt.Errorf("failed to resize image for embedding %s: %w", imageID, err)
	}

	vec, err := uc.visionEmbedder.EmbedVision(ctx, desc, embeddingData, mimeType)
	if err != nil {
		return fmt.Errorf("failed to generate vision embedding for image %s: %w", imageID, err)
	}
	emb := convertToFloat32(vec)

	// 5. Create image entity
	imageEntity := &entities.Image{
		ID:          imageID,
		SourceID:    sourceID,
		S3Key:       s3Key,
		Format:      img.Format,
		Width:       img.Width,
		Height:      img.Height,
		Description: desc,
		PageNumber:  img.PageNumber,
		Embedding:   emb,
		Metadata: map[string]any{
			"ocr_elements": len(img.OCRResult.OCRElements),
		},
		CreatedAt: time.Now(),
	}

	// 6. Save to image repository
	if err := uc.imageRepo.Save(ctx, imageEntity); err != nil {
		return fmt.Errorf("failed to save image entity %s: %w", imageID, err)
	}

	return nil
}

func convertToFloat32(v []float64) []float32 {
	res := make([]float32, len(v))
	for i, f := range v {
		res[i] = float32(f)
	}
	return res
}
