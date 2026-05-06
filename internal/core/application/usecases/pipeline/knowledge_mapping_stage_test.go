package pipeline

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/extractor"
	"github.com/oniharnantyo/eino-notebook/pkg/logger"
	"github.com/oniharnantyo/eino-notebook/pkg/parser/kreuzberg"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

func TestKnowledgeMappingStage_Execute(t *testing.T) {
	ctx := context.Background()
	stage := NewKnowledgeMappingStage(logger.New(logger.LevelInfo, "text"))
	sourceID := uuid.New()

	t.Run("success with metadata", func(t *testing.T) {
		extractionResult := &extractor.ExtractionResult{
			Content: "full content",
			Metadata: map[string]any{
				"title":      "Test Document",
				"authors":    "Jane Doe",
				"page_count": 10,
			},
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
		assert.Len(t, data.Knowledges, 1)

		// Verify metadata transfer
		k1 := data.Knowledges[0]
		assert.Equal(t, "text", k1.Metadata["chunk_type"])
		assert.Equal(t, "Test Document", k1.Metadata["title"])
		assert.Equal(t, "Jane Doe", k1.Metadata["authors"])
		assert.Equal(t, 10, k1.Metadata["page_count"])
	})

	t.Run("success without document metadata", func(t *testing.T) {
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
		data := output.Data.(*PipelineData)
		assert.Len(t, data.Knowledges, 1)
		assert.Equal(t, "text", data.Knowledges[0].Metadata["chunk_type"])
		assert.Len(t, data.Knowledges[0].Metadata, 1) // Only chunk_type
	})

	t.Run("metadata conflict - document wins", func(t *testing.T) {
		extractionResult := &extractor.ExtractionResult{
			Metadata: map[string]any{
				"chunk_type": "OVERWRITTEN",
			},
			Chunks: []kreuzberg.KreuzbergChunk{
				{
					Content:   "chunk 1",
					ChunkType: "text",
				},
			},
		}

		input := StageInput{
			SourceID: sourceID,
			Data:     &PipelineData{ExtractionResult: extractionResult},
		}

		output, err := stage.Execute(ctx, input)

		assert.NoError(t, err)
		data := output.Data.(*PipelineData)
		assert.Equal(t, "OVERWRITTEN", data.Knowledges[0].Metadata["chunk_type"])
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
