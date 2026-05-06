package stages

import (
	"context"
	"testing"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/agent/tools"
	"github.com/oniharnantyo/eino-notebook/pkg/retriever/pgvector"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

func TestToolPreparationStage_Execute(t *testing.T) {
	t.Run("success_empty_source_types", func(t *testing.T) {
		factory := tools.NewToolFactory(nil, nil, nil, nil, nil, nil)
		stage := NewToolPreparationStage(factory)

		input := ToolPreparationInput{
			SourceIDs:   []string{},
			SourceTypes: []string{},
		}

		output, err := stage.Execute(context.Background(), input)

		assert.NoError(t, err)
		assert.NotNil(t, output)
	})

	t.Run("success_with_valid_source_types", func(t *testing.T) {
		// Mock retrievers to satisfy IsSourceTypeSupported
		mockRetriever := &pgvector.SentencesRetriever{}
		mockImageRetriever := &pgvector.ImagesRetriever{}
		
		factory := tools.NewToolFactory(mockRetriever, mockImageRetriever, nil, nil, nil, nil)
		stage := NewToolPreparationStage(factory)

		input := ToolPreparationInput{
			SourceTypes: []string{"sentence", "image"},
		}

		output, err := stage.Execute(context.Background(), input)

		assert.NoError(t, err)
		assert.NotNil(t, output)
		assert.NotEmpty(t, output.Tools)
	})

	t.Run("failure_unsupported_source_type", func(t *testing.T) {
		factory := tools.NewToolFactory(nil, nil, nil, nil, nil, nil)
		stage := NewToolPreparationStage(factory)

		input := ToolPreparationInput{
			SourceTypes: []string{"unsupported_type"},
		}

		_, err := stage.Execute(context.Background(), input)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported source type")
	})

	t.Run("success_with_valid_uuid_source_ids", func(t *testing.T) {
		factory := tools.NewToolFactory(nil, nil, nil, nil, nil, nil)
		stage := NewToolPreparationStage(factory)

		validID := uuid.New().String()
		input := ToolPreparationInput{
			SourceIDs: []string{validID},
		}

		output, err := stage.Execute(context.Background(), input)

		assert.NoError(t, err)
		assert.NotNil(t, output)
	})
}
