package pipeline

import (
	"context"
	"reflect"
	"testing"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/logger"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// Mock repositories for testing
type mockKnowledgeRepository struct {
	saved []*entities.Knowledge
}

func (m *mockKnowledgeRepository) Save(ctx context.Context, knowledge *entities.Knowledge) error {
	return nil
}

func (m *mockKnowledgeRepository) SaveBatch(ctx context.Context, knowledges []*entities.Knowledge) error {
	m.saved = knowledges
	return nil
}

func (m *mockKnowledgeRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.Knowledge, error) {
	return nil, nil
}

func (m *mockKnowledgeRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.Knowledge, error) {
	return nil, nil
}

func (m *mockKnowledgeRepository) GetBySourceID(ctx context.Context, sourceID uuid.UUID) ([]*entities.Knowledge, error) {
	return nil, nil
}

func (m *mockKnowledgeRepository) FindAll(ctx context.Context, limit, offset int) ([]*entities.Knowledge, error) {
	return nil, nil
}

func (m *mockKnowledgeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockKnowledgeRepository) DeleteBySourceID(ctx context.Context, sourceID uuid.UUID) error {
	return nil
}

func (m *mockKnowledgeRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	return false, nil
}

func (m *mockKnowledgeRepository) Count(ctx context.Context) (int64, error) {
	return 0, nil
}

func (m *mockKnowledgeRepository) CountBySourceID(ctx context.Context, sourceID uuid.UUID) (int, error) {
	return 0, nil
}

type mockSentenceRepository struct {
	saved []*entities.Sentence
}

func (m *mockSentenceRepository) Save(ctx context.Context, sentence *entities.Sentence) error {
	return nil
}

func (m *mockSentenceRepository) SaveBatch(ctx context.Context, sentences []*entities.Sentence) error {
	m.saved = sentences
	return nil
}

func (m *mockSentenceRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.Sentence, error) {
	return nil, nil
}

func (m *mockSentenceRepository) FindByKnowledgeID(ctx context.Context, knowledgeID uuid.UUID) ([]*entities.Sentence, error) {
	return nil, nil
}

func (m *mockSentenceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockSentenceRepository) DeleteByKnowledgeID(ctx context.Context, knowledgeID uuid.UUID) error {
	return nil
}

func (m *mockSentenceRepository) DeleteBySourceID(ctx context.Context, sourceID uuid.UUID) error {
	return nil
}

type mockImageRepository struct {
	saved []*entities.Image
}

func (m *mockImageRepository) Save(ctx context.Context, image *entities.Image) error {
	m.saved = append(m.saved, image)
	return nil
}

func (m *mockImageRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.Image, error) {
	return nil, nil
}

func (m *mockImageRepository) FindBySourceID(ctx context.Context, sourceID uuid.UUID) ([]*entities.Image, error) {
	return nil, nil
}

func (m *mockImageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockImageRepository) DeleteBySourceID(ctx context.Context, sourceID uuid.UUID) error {
	return nil
}

func (m *mockImageRepository) CountBySourceID(ctx context.Context, sourceID uuid.UUID) (int, error) {
	return 0, nil
}

type mockSourceRepository struct {
	source *entities.Source
}

func (m *mockSourceRepository) Create(ctx context.Context, source *entities.Source) error {
	m.source = source
	return nil
}

func (m *mockSourceRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Source, error) {
	return m.source, nil
}

func (m *mockSourceRepository) GetByNotebookID(ctx context.Context, notebookID uuid.UUID) ([]*entities.Source, error) {
	return nil, nil
}

func (m *mockSourceRepository) GetByURI(ctx context.Context, notebookID uuid.UUID, uri string) (*entities.Source, error) {
	return nil, nil
}

func (m *mockSourceRepository) Update(ctx context.Context, source *entities.Source) error {
	m.source = source
	return nil
}

func (m *mockSourceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockSourceRepository) List(ctx context.Context, filter repositories.SourceFilter) ([]*entities.Source, int, error) {
	return nil, 0, nil
}

func (m *mockSourceRepository) ListSourceSummariesByID(ctx context.Context, ids []uuid.UUID) ([]*entities.Source, error) {
	return nil, nil
}

func (m *mockSourceRepository) IncrementChunkCount(ctx context.Context, id uuid.UUID) error {
	return nil
}

func TestExtractDocumentMetadata(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected map[string]any
	}{
		{
			name: "returns only document-level keys",
			input: map[string]any{
				"title":           "Test Document",
				"authors":         []string{"Author 1"},
				"page_count":      10,
				"first_page":      1,      // chunk-level - should be excluded
				"last_page":       5,      // chunk-level - should be excluded
				"heading_context": map[string]any{}, // chunk-level - should be excluded
				"embedding":       []float32{},      // chunk-level - should be excluded
			},
			expected: map[string]any{
				"title":      "Test Document",
				"authors":    []string{"Author 1"},
				"page_count": 10,
			},
		},
		{
			name:     "returns empty map when no document-level keys present",
			input:    map[string]any{"first_page": 1, "last_page": 5},
			expected: map[string]any{},
		},
		{
			name:     "handles nil input",
			input:    nil,
			expected: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractDocumentMetadata(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("extractDocumentMetadata() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestStorageStage_Execute(t *testing.T) {
	ctx := context.Background()
	sourceID := uuid.New()

	mockKnowledgeRepo := &mockKnowledgeRepository{}
	mockSentenceRepo := &mockSentenceRepository{}
	mockImageRepo := &mockImageRepository{}
	mockSourceRepo := &mockSourceRepository{
		source: &entities.Source{
			ID:        sourceID,
			ChunkCount: 0,
			Metadata:  make(map[string]any),
		},
	}

	stage := NewStorageStage(mockKnowledgeRepo, mockSentenceRepo, mockImageRepo, mockSourceRepo, logger.New(logger.LevelInfo, "text"))

	// Create test knowledge
	knowledge, err := entities.NewKnowledge(
		sourceID,
		"Test content",
		0,
		map[string]any{"heading": "Introduction"},
		1,
		2,
		map[string]any{
			"title":      "Test Document",
			"page_count": 10,
			"first_page": 1, // chunk-level - should not propagate to source
		},
	)
	if err != nil {
		t.Fatalf("failed to create knowledge: %v", err)
	}

	// Create test sentence with embedding
	sentenceID := uuid.New()
	sent := Sentence{
		ID:          sentenceID,
		KnowledgeID: knowledge.ID,
		Content:     "Test sentence content",
		Position:    0,
		Embedding:   []float32{0.1, 0.2, 0.3},
	}

	// Create test image
	image, err := entities.NewImage(
		sourceID,
		"test-key",
		"png",
		800,
		600,
		"Test image description",
		1,
		nil,
	)
	if err != nil {
		t.Fatalf("failed to create image: %v", err)
	}

	data := &PipelineData{
		Knowledges: []*entities.Knowledge{knowledge},
		Sentences:  []Sentence{sent},
		Images:     []*entities.Image{image},
	}

	input := StageInput{
		SourceID: sourceID,
		Data:     data,
	}

	output, err := stage.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Verify knowledge was saved
	if len(mockKnowledgeRepo.saved) != 1 {
		t.Errorf("expected 1 knowledge saved, got %d", len(mockKnowledgeRepo.saved))
	}

	// Verify sentence was saved with embedding
	if len(mockSentenceRepo.saved) != 1 {
		t.Fatalf("expected 1 sentence saved, got %d", len(mockSentenceRepo.saved))
	}
	savedSentence := mockSentenceRepo.saved[0]
	if savedSentence.Content != "Test sentence content" {
		t.Errorf("expected sentence content 'Test sentence content', got '%s'", savedSentence.Content)
	}
	if !reflect.DeepEqual(savedSentence.Embedding, []float32{0.1, 0.2, 0.3}) {
		t.Errorf("expected embedding [0.1, 0.2, 0.3], got %v", savedSentence.Embedding)
	}

	// Verify image was saved
	if len(mockImageRepo.saved) != 1 {
		t.Errorf("expected 1 image saved, got %d", len(mockImageRepo.saved))
	}

	// Verify source was updated
	if mockSourceRepo.source.ChunkCount != 1 {
		t.Errorf("expected source chunk count 1, got %d", mockSourceRepo.source.ChunkCount)
	}

	// Verify document-level metadata was merged into source
	if mockSourceRepo.source.Metadata["title"] != "Test Document" {
		t.Errorf("expected source metadata title 'Test Document', got '%v'", mockSourceRepo.source.Metadata["title"])
	}
	if mockSourceRepo.source.Metadata["page_count"] != 10 {
		t.Errorf("expected source metadata page_count 10, got %v", mockSourceRepo.source.Metadata["page_count"])
	}
	// Verify chunk-level metadata was NOT merged
	if _, exists := mockSourceRepo.source.Metadata["first_page"]; exists {
		t.Error("expected first_page (chunk-level) to not be in source metadata")
	}

	// Verify output is PipelineData
	if _, ok := output.Data.(*PipelineData); !ok {
		t.Errorf("expected output.Data to be *PipelineData, got %T", output.Data)
	}
}

func TestStorageStage_Execute_EmptyData(t *testing.T) {
	ctx := context.Background()
	sourceID := uuid.New()

	mockKnowledgeRepo := &mockKnowledgeRepository{}
	mockSentenceRepo := &mockSentenceRepository{}
	mockImageRepo := &mockImageRepository{}
	mockSourceRepo := &mockSourceRepository{
		source: &entities.Source{
			ID:        sourceID,
			ChunkCount: 0,
			Metadata:  make(map[string]any),
		},
	}

	stage := NewStorageStage(mockKnowledgeRepo, mockSentenceRepo, mockImageRepo, mockSourceRepo, logger.New(logger.LevelInfo, "text"))

	data := &PipelineData{
		Knowledges: []*entities.Knowledge{},
		Sentences:  []Sentence{},
		Images:     []*entities.Image{},
	}

	input := StageInput{
		SourceID: sourceID,
		Data:     data,
	}

	_, err := stage.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Nothing should be saved
	if len(mockKnowledgeRepo.saved) != 0 {
		t.Errorf("expected 0 knowledge saved, got %d", len(mockKnowledgeRepo.saved))
	}
	if len(mockSentenceRepo.saved) != 0 {
		t.Errorf("expected 0 sentences saved, got %d", len(mockSentenceRepo.saved))
	}
	if len(mockImageRepo.saved) != 0 {
		t.Errorf("expected 0 images saved, got %d", len(mockImageRepo.saved))
	}
}

func TestStorageStage_Execute_InvalidInput(t *testing.T) {
	ctx := context.Background()
	sourceID := uuid.New()

	mockKnowledgeRepo := &mockKnowledgeRepository{}
	mockSentenceRepo := &mockSentenceRepository{}
	mockImageRepo := &mockImageRepository{}
	mockSourceRepo := &mockSourceRepository{
		source: &entities.Source{
			ID:        sourceID,
			ChunkCount: 0,
			Metadata:  make(map[string]any),
		},
	}

	stage := NewStorageStage(mockKnowledgeRepo, mockSentenceRepo, mockImageRepo, mockSourceRepo, logger.New(logger.LevelInfo, "text"))

	input := StageInput{
		SourceID: sourceID,
		Data:     "invalid",
	}

	_, err := stage.Execute(ctx, input)
	if err == nil {
		t.Error("expected error for invalid input type, got nil")
	}
}
