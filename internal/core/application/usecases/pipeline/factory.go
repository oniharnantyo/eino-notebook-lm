package pipeline

import (
	"context"
	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/components/embedding"
	documentusecase "github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/document"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/extractor"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
)

// PipelineFactory defines the interface for creating ingestion pipelines.
type PipelineFactory interface {
	Create(contentExtractor extractor.ContentExtractor, mimeType string) *IngestionPipeline
}

type pipelineFactory struct {
	sourceRepo            repositories.SourceRepository
	knowledgeRepo         repositories.KnowledgeRepository
	sentenceRepo          repositories.SentenceRepository
	documentParserFactory *documentusecase.DocumentParserFactory
	embedder              embedding.Embedder
	transformer           document.Transformer
}

// NewPipelineFactory creates a new pipeline factory.
func NewPipelineFactory(
	sourceRepo repositories.SourceRepository,
	knowledgeRepo repositories.KnowledgeRepository,
	sentenceRepo repositories.SentenceRepository,
	documentParserFactory *documentusecase.DocumentParserFactory,
	embedder embedding.Embedder,
	transformer document.Transformer,
) PipelineFactory {
	return &pipelineFactory{
		sourceRepo:            sourceRepo,
		knowledgeRepo:         knowledgeRepo,
		sentenceRepo:          sentenceRepo,
		documentParserFactory: documentParserFactory,
		embedder:              embedder,
		transformer:           transformer,
	}
}

// Create creates a new IngestionPipeline with all necessary stages.
func (f *pipelineFactory) Create(contentExtractor extractor.ContentExtractor, mimeType string) *IngestionPipeline {
	parser := f.documentParserFactory.GetParser(context.Background())

	stages := []Stage{
		NewExtractionStage(contentExtractor),
		NewParsingStage(parser),
		NewChunkingStage(1000), // Default token limit, can be made configurable
		NewEmbeddingStage(f.embedder),
		NewStorageStage(f.knowledgeRepo, f.sentenceRepo, f.sourceRepo, f.transformer, f.embedder),
		NewStatusUpdateStage(f.sourceRepo),
	}

	return NewIngestionPipeline(stages, 1)
}
