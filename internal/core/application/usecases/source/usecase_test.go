package source

import (
	"context"
	"io"
	"testing"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/logger"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// mockSourceRepository is a minimal mock for testing
type mockSourceRepository struct {
	repositories.SourceRepository
	createFunc  func(ctx context.Context, source *entities.Source) error
	getByIDFunc func(ctx context.Context, id uuid.UUID) (*entities.Source, error)
	updateFunc  func(ctx context.Context, source *entities.Source) error
}

func (m *mockSourceRepository) Create(ctx context.Context, source *entities.Source) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, source)
	}
	return nil
}

func (m *mockSourceRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Source, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockSourceRepository) Update(ctx context.Context, source *entities.Source) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, source)
	}
	return nil
}

func (m *mockSourceRepository) GetByNotebookID(ctx context.Context, notebookID uuid.UUID) ([]*entities.Source, error) {
	return nil, nil
}

func (m *mockSourceRepository) GetByURI(ctx context.Context, notebookID uuid.UUID, uri string) (*entities.Source, error) {
	return nil, nil
}

func (m *mockSourceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockSourceRepository) List(ctx context.Context, filter repositories.SourceFilter) ([]*entities.Source, int, error) {
	return nil, 0, nil
}

func (m *mockSourceRepository) IncrementChunkCount(ctx context.Context, id uuid.UUID) error {
	return nil
}

func TestSanitizeContent(t *testing.T) {
	// Create a minimal usecase for testing
	uc := &sourceUseCase{
		sourceRepo:              &mockSourceRepository{},
		notebookRepo:            nil,
		contentExtractorFactory: nil,
		documentParserFactory:   nil,
		knowledgeUseCase:        nil,
		logger:                  logger.NewWithWriter(io.Discard, logger.LevelInfo, "json"),
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean content unchanged",
			input:    "Hello, world!",
			expected: "Hello, world!",
		},
		{
			name:     "single null byte removed",
			input:    "Hello\x00world",
			expected: "Helloworld",
		},
		{
			name:     "multiple null bytes removed",
			input:    "\x00Hello\x00world\x00",
			expected: "Helloworld",
		},
		{
			name:     "null byte in middle removed",
			input:    "Lorem ipsum dolor sit amet",
			expected: "Lorem ipsum dolor sit amet",
		},
		{
			name:     "only null bytes",
			input:    "\x00\x00\x00",
			expected: "",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "multiline with null bytes",
			input:    "Line 1\x00\nLine 2\x00\nLine 3",
			expected: "Line 1\nLine 2\nLine 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uc.sanitizeContent(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeContent(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
