package pipeline

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/extractor"
	"github.com/oniharnantyo/eino-notebook/pkg/parser/kreuzberg"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

func TestKnowledgeMappingStage_Execute(t *testing.T) {
	ctx := context.Background()
	stage := NewKnowledgeMappingStage()
	sourceID := uuid.New()

	t.Run("success", func(t *testing.T) {
		extractionResult := &extractor.ExtractionResult{
			Content: "full content",
			Chunks: []kreuzberg.KreuzbergChunk{
				{
					Content:   "chunk 1",
					ChunkType: "text",
					Metadata: kreuzberg.KreuzbergChunkMeta{
						ChunkIndex: 0,
						FirstPage:  1,
						LastPage:   1,
						HeadingContext: map[string]any{
							"h1": "Header 1",
						},
					},
				},
				{
					Content:   "chunk 2",
					ChunkType: "table",
					Metadata: kreuzberg.KreuzbergChunkMeta{
						ChunkIndex: 1,
						FirstPage:  2,
						LastPage:   2,
						HeadingContext: map[string]any{
							"h1": "Header 1",
							"h2": "Subheader",
						},
					},
				},
			},
		}

		input := StageInput{
			SourceID: sourceID,
			Data:     &PipelineData{ExtractionResult: extractionResult},
		}

		output, err := stage.Execute(ctx, input)

		assert.NoError(t, err)
		assert.NotNil(t, output.Data)

		data, ok := output.Data.(*PipelineData)
		assert.True(t, ok)
		assert.Equal(t, extractionResult, data.ExtractionResult)
		assert.Len(t, data.Knowledges, 2)

		// Verify first knowledge entity
		k1 := data.Knowledges[0]
		assert.Equal(t, sourceID, k1.SourceID)
		assert.Equal(t, "chunk 1", k1.Content)
		assert.Equal(t, 0, k1.ChunkIndex)
		assert.Equal(t, 1, k1.FirstPage)
		assert.Equal(t, 1, k1.LastPage)
		assert.Equal(t, "Header 1", k1.HeadingContext["h1"])
		assert.Equal(t, "text", k1.Metadata["chunk_type"])

		// Verify second knowledge entity
		k2 := data.Knowledges[1]
		assert.Equal(t, sourceID, k2.SourceID)
		assert.Equal(t, "chunk 2", k2.Content)
		assert.Equal(t, 1, k2.ChunkIndex)
		assert.Equal(t, 2, k2.FirstPage)
		assert.Equal(t, 2, k2.LastPage)
		assert.Equal(t, "Header 1", k2.HeadingContext["h1"])
		assert.Equal(t, "Subheader", k2.HeadingContext["h2"])
		assert.Equal(t, "table", k2.Metadata["chunk_type"])
	})

	t.Run("invalid input", func(t *testing.T) {
		input := StageInput{
			SourceID: sourceID,
			Data:     "not a PipelineData",
		}

		_, err := stage.Execute(ctx, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid input type")
	})

	t.Run("nil ExtractionResult", func(t *testing.T) {
		input := StageInput{
			SourceID: sourceID,
			Data:     &PipelineData{ExtractionResult: nil},
		}

		_, err := stage.Execute(ctx, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ExtractionResult is nil")
	})

	t.Run("empty chunks", func(t *testing.T) {
		extractionResult := &extractor.ExtractionResult{
			Chunks: []kreuzberg.KreuzbergChunk{},
		}

		input := StageInput{
			SourceID: sourceID,
			Data:     &PipelineData{ExtractionResult: extractionResult},
		}

		output, err := stage.Execute(ctx, input)

		assert.NoError(t, err)
		data := output.Data.(*PipelineData)
		assert.Empty(t, data.Knowledges)
	})
}
