package source

import (
	"io"
	"testing"

	"github.com/oniharnantyo/eino-notebook/internal/mocks/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/logger"
)

func TestSanitizeContent(t *testing.T) {
	// Create a minimal usecase for testing
	uc := &sourceUseCase{
		sourceRepo:              &repositories.MockSourceRepository{},
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
