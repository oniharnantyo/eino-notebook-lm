package pipeline

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// mockEmbedder is a mock implementation of embedding.Embedder
type mockEmbedder struct {
	mock.Mock
}

func (m *mockEmbedder) EmbedStrings(ctx context.Context, texts []string, opts ...embedding.Option) ([][]float64, error) {
	args := m.Called(ctx, texts, opts)
	return args.Get(0).([][]float64), args.Error(1)
}

func TestEmbeddingStage_Execute(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully embeds sentences", func(t *testing.T) {
		mockEmb := new(mockEmbedder)
		stage := NewEmbeddingStage(mockEmb)

		sourceID := uuid.New()
		knowledgeID := uuid.New()

		sentences := []Sentence{
			{
				ID:          uuid.New(),
				KnowledgeID: knowledgeID,
				Content:     "This is a test sentence.",
				Position:    0,
				Embedding:   nil,
			},
			{
				ID:          uuid.New(),
				KnowledgeID: knowledgeID,
				Content:     "This is another test sentence.",
				Position:    1,
				Embedding:   nil,
			},
		}

		mockEmb.On("EmbedStrings", ctx, []string{"This is a test sentence.", "This is another test sentence."}, mock.Anything).
			Return([][]float64{{0.1, 0.2, 0.3, 0.4}, {0.5, 0.6, 0.7, 0.8}}, nil)

		input := StageInput{
			SourceID: sourceID,
			Data:     &PipelineData{Sentences: sentences},
		}

		output, err := stage.Execute(ctx, input)

		assert.NoError(t, err)
		assert.NotNil(t, output.Data)

		data, ok := output.Data.(*PipelineData)
		assert.True(t, ok)
		assert.Len(t, data.Sentences, 2)

		// Verify embeddings were attached
		assert.Equal(t, []float32{0.1, 0.2, 0.3, 0.4}, data.Sentences[0].Embedding)
		assert.Equal(t, []float32{0.5, 0.6, 0.7, 0.8}, data.Sentences[1].Embedding)

		mockEmb.AssertExpectations(t)
	})

	t.Run("handles empty sentence slice", func(t *testing.T) {
		mockEmb := new(mockEmbedder)
		stage := NewEmbeddingStage(mockEmb)

		input := StageInput{
			Data: &PipelineData{Sentences: []Sentence{}},
		}

		output, err := stage.Execute(ctx, input)

		assert.NoError(t, err)
		assert.NotNil(t, output.Data)

		data, ok := output.Data.(*PipelineData)
		assert.True(t, ok)
		assert.Len(t, data.Sentences, 0)
	})

	t.Run("returns error for invalid input type", func(t *testing.T) {
		mockEmb := new(mockEmbedder)
		stage := NewEmbeddingStage(mockEmb)

		input := StageInput{
			Data: "invalid",
		}

		_, err := stage.Execute(ctx, input)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid input type for EmbeddingStage: expected *PipelineData")
	})

	t.Run("returns error when embedding fails", func(t *testing.T) {
		mockEmb := new(mockEmbedder)
		stage := NewEmbeddingStage(mockEmb)

		sentences := []Sentence{
			{
				ID:          uuid.New(),
				KnowledgeID: uuid.New(),
				Content:     "Test sentence",
				Position:    0,
				Embedding:   nil,
			},
		}

		mockEmb.On("EmbedStrings", ctx, []string{"Test sentence"}, mock.Anything).
			Return([][]float64{}, assert.AnError)

		input := StageInput{
			Data: &PipelineData{Sentences: sentences},
		}

		_, err := stage.Execute(ctx, input)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to generate embeddings")

		mockEmb.AssertExpectations(t)
	})
}

func TestConvertFloat64ToFloat32(t *testing.T) {
	t.Run("converts float64 slice to float32", func(t *testing.T) {
		input := []float64{0.1, 0.2, 0.3, 0.4, 0.5}
		expected := []float32{0.1, 0.2, 0.3, 0.4, 0.5}

		result := ConvertFloat64ToFloat32(input)

		assert.Equal(t, expected, result)
	})

	t.Run("handles empty slice", func(t *testing.T) {
		input := []float64{}
		expected := []float32{}

		result := ConvertFloat64ToFloat32(input)

		assert.Equal(t, expected, result)
	})
}
