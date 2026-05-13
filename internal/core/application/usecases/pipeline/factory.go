package pipeline

import (
	"context"
	"github.com/cloudwego/eino/components/embedding"
	documentusecase "github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/document"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/extractor"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/internal/infrastructure/storage"
	"github.com/oniharnantyo/eino-notebook/pkg/description"
	"github.com/oniharnantyo/eino-notebook/pkg/logger"
	usecases "github.com/oniharnantyo/eino-notebook/internal/core/application/usecases"
)

// PipelineFactory defines the interface for creating ingestion pipelines.
type PipelineFactory interface {
	Create(contentExtractor extractor.ContentExtractor, contentType usecases.ContentType) *IngestionPipeline
}

type pipelineFactory struct {
	sourceRepo            repositories.SourceRepository
	knowledgeRepo         repositories.KnowledgeRepository
	sentenceRepo          repositories.SentenceRepository
	imageRepo             repositories.ImageRepository
	documentParserFactory *documentusecase.DocumentParserFactory
	embedder              embedding.Embedder
	s3Storage             *storage.S3Storage
	visionDescriber       description.VisionDescriber
	logger                *logger.Logger
}

// NewPipelineFactory creates a new pipeline factory.
func NewPipelineFactory(
	sourceRepo repositories.SourceRepository,
	knowledgeRepo repositories.KnowledgeRepository,
	sentenceRepo repositories.SentenceRepository,
	imageRepo repositories.ImageRepository,
	documentParserFactory *documentusecase.DocumentParserFactory,
	embedder embedding.Embedder,
	s3Storage *storage.S3Storage,
	visionDescriber description.VisionDescriber,
	logger *logger.Logger,
) PipelineFactory {
	return &pipelineFactory{
		sourceRepo:            sourceRepo,
		knowledgeRepo:         knowledgeRepo,
		sentenceRepo:          sentenceRepo,
		imageRepo:             imageRepo,
		documentParserFactory: documentParserFactory,
		embedder:              embedder,
		s3Storage:             s3Storage,
		visionDescriber:       visionDescriber,
		logger:                logger,
	}
}

// Create creates a new IngestionPipeline with stages based on content type.
// For ContentTypeFile (Kreuzberg sources): Extraction → KnowledgeMapping → SentenceSplitting → Embedding → ImageProcessing → Storage → StatusUpdate
// For ContentTypeURL/ContentTypeText (non-Kreuzberg): Extraction → Parsing → Chunking → Embedding → Storage → StatusUpdate
func (f *pipelineFactory) Create(contentExtractor extractor.ContentExtractor, contentType usecases.ContentType) *IngestionPipeline {
	switch contentType {
	case usecases.ContentTypeFile:
		return f.createKreuzbergPipeline(contentExtractor)
	case usecases.ContentTypeURL, usecases.ContentTypeText:
		return f.createStandardPipeline(contentExtractor)
	default:
		// Default to standard pipeline for unknown types
		return f.createStandardPipeline(contentExtractor)
	}
}

// createKreuzbergPipeline creates a pipeline for Kreuzberg-based file extraction.
// Stages: Extraction → KnowledgeMapping → SentenceSplitting → Embedding → ImageProcessing → Storage → StatusUpdate
func (f *pipelineFactory) createKreuzbergPipeline(contentExtractor extractor.ContentExtractor) *IngestionPipeline {
	stages := []Stage{
		NewExtractionStage(contentExtractor),
		NewKnowledgeMappingStage(f.logger),
		NewSentenceSplittingStage(),
		NewEmbeddingStage(f.embedder),
		NewImageProcessingStage(f.s3Storage, f.visionDescriber, f.embedder, f.logger),
		NewStorageStage(f.knowledgeRepo, f.sentenceRepo, f.imageRepo, f.sourceRepo),
		NewStatusUpdateStage(f.sourceRepo),
	}

	return NewIngestionPipeline(stages, 1)
}

// createStandardPipeline creates a pipeline for URL/text content extraction.
// Stages: Extraction → Parsing → Chunking → Embedding → KnowledgeMapping → Storage → StatusUpdate
func (f *pipelineFactory) createStandardPipeline(contentExtractor extractor.ContentExtractor) *IngestionPipeline {
	parser := f.documentParserFactory.GetParser(context.Background())

	stages := []Stage{
		NewExtractionStage(contentExtractor),
		NewParsingStage(parser),
		NewChunkingStage(1000), // Default token limit, can be made configurable
		NewDocumentEmbeddingStage(f.embedder),
		NewDocumentToKnowledgeStage(),
		NewStorageStage(f.knowledgeRepo, f.sentenceRepo, nil, f.sourceRepo),
		NewStatusUpdateStage(f.sourceRepo),
	}

	return NewIngestionPipeline(stages, 1)
}
