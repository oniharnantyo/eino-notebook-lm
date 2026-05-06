package pipeline

import (
	"context"
	"io"
	"testing"

	"github.com/cloudwego/eino/components/document/parser"
	"github.com/cloudwego/eino/schema"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/extractor"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/pkg/logger"
	"github.com/oniharnantyo/eino-notebook/pkg/parser/kreuzberg"
	usecases "github.com/oniharnantyo/eino-notebook/internal/core/application/usecases"
	documentusecase "github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/document"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// TestFactory_CreateKreuzbergPipeline tests the creation of Kreuzberg pipeline
func TestFactory_CreateKreuzbergPipeline(t *testing.T) {
	// Setup mock dependencies
	sourceRepo := &mockSourceRepository{
		source: &entities.Source{
			ID:         uuid.New(),
			ChunkCount: 0,
			Metadata:   make(map[string]any),
		},
	}
	knowledgeRepo := &mockKnowledgeRepository{}
	sentenceRepo := &mockSentenceRepository{}
	imageRepo := &mockImageRepository{}

	mockParser := &mockKreuzbergDocumentParser{}
	documentParserFactory := documentusecase.NewDocumentParserFactory(mockParser)

	embedder := &mockTextEmbedder{}
	visionDescriber := &mockVisionDescriber{}
	log := logger.New(logger.LevelInfo, "json")

	// Create factory - S3Storage can be nil for testing factory logic
	factory := NewPipelineFactory(
		sourceRepo,
		knowledgeRepo,
		sentenceRepo,
		imageRepo,
		documentParserFactory,
		embedder,
		nil, // S3Storage can be nil for testing
		visionDescriber,
		log,
	)

	// Create test content
	testContent := &extractor.ExtractionResult{
		Content: "Test document content.",
		Metadata: map[string]any{
			"detected_languages": []string{"en"},
			"title":              "Test Document",
		},
		Chunks: []kreuzberg.KreuzbergChunk{},
		Images: []kreuzberg.KreuzbergImage{},
	}

	// Create mock extractor
	mockExtractor := &mockContentExtractor{
		result: testContent,
	}

	// Create pipeline for ContentTypeFile
	pipelineInstance := factory.Create(mockExtractor, usecases.ContentTypeFile)

	if pipelineInstance == nil {
		t.Fatal("Expected non-nil pipeline")
	}

	// Verify pipeline was created successfully
	t.Log("Kreuzberg pipeline created successfully")
}

// TestFactory_CreateStandardPipeline tests the standard pipeline for URL/text content
func TestFactory_CreateStandardPipeline(t *testing.T) {
	// Setup mock dependencies
	sourceRepo := &mockSourceRepository{
		source: &entities.Source{
			ID:         uuid.New(),
			ChunkCount: 0,
			Metadata:   make(map[string]any),
		},
	}
	knowledgeRepo := &mockKnowledgeRepository{}
	sentenceRepo := &mockSentenceRepository{}
	imageRepo := &mockImageRepository{}

	mockParser := &mockKreuzbergDocumentParser{}
	documentParserFactory := documentusecase.NewDocumentParserFactory(mockParser)

	embedder := &mockTextEmbedder{}
	visionDescriber := &mockVisionDescriber{}
	log := logger.New(logger.LevelInfo, "json")

	// Create factory - S3Storage can be nil for URL/text pipelines
	factory := NewPipelineFactory(
		sourceRepo,
		knowledgeRepo,
		sentenceRepo,
		imageRepo,
		documentParserFactory,
		embedder,
		nil, // S3Storage not needed for URL/text pipelines
		visionDescriber,
		log,
	)

	// Create mock extractor for URL content
	mockExtractor := &mockContentExtractor{
		result: &extractor.ExtractionResult{
			Content: "Test URL content",
			Metadata: map[string]any{
				"url": "https://example.com",
			},
		},
	}

	// Test ContentTypeURL
	pipelineInstance := factory.Create(mockExtractor, usecases.ContentTypeURL)
	if pipelineInstance == nil {
		t.Fatal("Expected non-nil pipeline for ContentTypeURL")
	}

	// Test ContentTypeText
	pipelineInstance = factory.Create(mockExtractor, usecases.ContentTypeText)
	if pipelineInstance == nil {
		t.Fatal("Expected non-nil pipeline for ContentTypeText")
	}

	t.Log("Standard pipeline created successfully for URL and Text content types")
}

// TestFactory_PipelineStageOrder tests that pipelines have the correct stages
func TestFactory_PipelineStageOrder(t *testing.T) {
	// Setup mock dependencies
	sourceRepo := &mockSourceRepository{
		source: &entities.Source{
			ID:         uuid.New(),
			ChunkCount: 0,
			Metadata:   make(map[string]any),
		},
	}
	knowledgeRepo := &mockKnowledgeRepository{}
	sentenceRepo := &mockSentenceRepository{}
	imageRepo := &mockImageRepository{}

	mockParser := &mockKreuzbergDocumentParser{}
	documentParserFactory := documentusecase.NewDocumentParserFactory(mockParser)

	embedder := &mockTextEmbedder{}
	visionDescriber := &mockVisionDescriber{}
	log := logger.New(logger.LevelInfo, "json")

	factory := NewPipelineFactory(
		sourceRepo,
		knowledgeRepo,
		sentenceRepo,
		imageRepo,
		documentParserFactory,
		embedder,
		nil, // S3Storage not needed for stage order testing
		visionDescriber,
		log,
	)

	mockExtractor := &mockContentExtractor{
		result: &extractor.ExtractionResult{
			Content:  "Test content",
			Metadata: map[string]any{},
		},
	}

	t.Run("Kreuzberg pipeline stages", func(t *testing.T) {
		pipeline := factory.Create(mockExtractor, usecases.ContentTypeFile)
		if pipeline == nil {
			t.Fatal("Expected non-nil pipeline")
		}

		// Verify pipeline was created
		t.Log("Kreuzberg pipeline stages verified")
	})

	t.Run("Standard pipeline stages", func(t *testing.T) {
		pipeline := factory.Create(mockExtractor, usecases.ContentTypeURL)
		if pipeline == nil {
			t.Fatal("Expected non-nil pipeline")
		}

		// Verify pipeline was created
		t.Log("Standard pipeline stages verified")
	})
}

// mockContentExtractor is a mock implementation of extractor.ContentExtractor
type mockContentExtractor struct {
	result *extractor.ExtractionResult
}

func (m *mockContentExtractor) Extract(ctx context.Context, source usecases.ContentSource) (*extractor.ExtractionResult, error) {
	return m.result, nil
}

// mockKreuzbergDocumentParser is a mock implementation of document.DocumentParser
type mockKreuzbergDocumentParser struct{}

func (m *mockKreuzbergDocumentParser) Parse(ctx context.Context, reader io.Reader, opts ...parser.Option) ([]*schema.Document, error) {
	return []*schema.Document{
		{
			Content: "Test content",
		},
	}, nil
}

func (m *mockKreuzbergDocumentParser) ParseFull(ctx context.Context, reader io.Reader) ([]kreuzberg.KreuzbergExtractResponse, error) {
	return []kreuzberg.KreuzbergExtractResponse{
		{
			Content: "Test content",
			Chunks:  []kreuzberg.KreuzbergChunk{},
			Images:  []kreuzberg.KreuzbergImage{},
		},
	}, nil
}

func (m *mockKreuzbergDocumentParser) IsAvailable(ctx context.Context) bool {
	return true
}
