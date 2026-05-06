package pipeline

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/extractor"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

func TestSentenceSplittingStage_Execute(t *testing.T) {
	ctx := context.Background()
	stage := NewSentenceSplittingStage()
	sourceID := uuid.New()

	t.Run("success with abbreviations and short sentence filtering", func(t *testing.T) {
		k1, _ := entities.NewKnowledge(sourceID, "Mr. Smith went home. He was tired. OK.", 0, nil, 1, 1, nil)

		input := StageInput{
			SourceID: sourceID,
			Data: &PipelineData{
				Knowledges: []*entities.Knowledge{k1},
				ExtractionResult: &extractor.ExtractionResult{
					DetectedLanguages: []string{"en"},
				},
			},
		}

		output, err := stage.Execute(ctx, input)

		assert.NoError(t, err)
		assert.NotNil(t, output.Data)

		data, ok := output.Data.(*PipelineData)
		assert.True(t, ok)

		assert.Len(t, data.Sentences, 2)
		assert.Equal(t, "Mr. Smith went home.", data.Sentences[0].Content)
		assert.Equal(t, 0, data.Sentences[0].Position)
		assert.Equal(t, k1.ID, data.Sentences[0].KnowledgeID)
		assert.Nil(t, data.Sentences[0].Embedding) // Embedding should be nil initially

		assert.Equal(t, "He was tired.", data.Sentences[1].Content)
		assert.Equal(t, 1, data.Sentences[1].Position)
		assert.Equal(t, k1.ID, data.Sentences[1].KnowledgeID)
		assert.Nil(t, data.Sentences[1].Embedding) // Embedding should be nil initially
	})

	t.Run("language fallback to en", func(t *testing.T) {
		k1, _ := entities.NewKnowledge(sourceID, "This is a sentence. This is another one.", 0, nil, 1, 1, nil)

		input := StageInput{
			SourceID: sourceID,
			Data: &PipelineData{
				Knowledges: []*entities.Knowledge{k1},
				// No language metadata
				ExtractionResult: &extractor.ExtractionResult{},
			},
		}

		output, err := stage.Execute(ctx, input)

		assert.NoError(t, err)
		data := output.Data.(*PipelineData)
		assert.Len(t, data.Sentences, 2)
	})

	t.Run("multiple knowledge chunks", func(t *testing.T) {
		k1, _ := entities.NewKnowledge(sourceID, "Sentence one. Sentence two.", 0, nil, 1, 1, nil)
		k2, _ := entities.NewKnowledge(sourceID, "Sentence three. Sentence four.", 1, nil, 1, 1, nil)

		input := StageInput{
			SourceID: sourceID,
			Data: &PipelineData{
				Knowledges: []*entities.Knowledge{k1, k2},
				ExtractionResult: &extractor.ExtractionResult{
					DetectedLanguages: []string{"en"},
				},
			},
		}

		output, err := stage.Execute(ctx, input)

		assert.NoError(t, err)
		data := output.Data.(*PipelineData)
		assert.Len(t, data.Sentences, 4)

		assert.Equal(t, 0, data.Sentences[0].Position)
		assert.Equal(t, k1.ID, data.Sentences[0].KnowledgeID)

		assert.Equal(t, 1, data.Sentences[1].Position)
		assert.Equal(t, k1.ID, data.Sentences[1].KnowledgeID)

		assert.Equal(t, 0, data.Sentences[2].Position)
		assert.Equal(t, k2.ID, data.Sentences[2].KnowledgeID)

		assert.Equal(t, 1, data.Sentences[3].Position)
		assert.Equal(t, k2.ID, data.Sentences[3].KnowledgeID)
	})

	t.Run("invalid input type", func(t *testing.T) {
		input := StageInput{
			SourceID: sourceID,
			Data:     "invalid",
		}

		_, err := stage.Execute(ctx, input)
		assert.Error(t, err)
	})
}
