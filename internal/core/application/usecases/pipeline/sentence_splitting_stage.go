package pipeline

import (
	"context"
	"fmt"
	"strings"

	"github.com/oniharnantyo/eino-notebook/pkg/sentencex"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// SentenceSplittingStage segments knowledge content into sentences.
type SentenceSplittingStage struct{}

// NewSentenceSplittingStage creates a new SentenceSplittingStage.
func NewSentenceSplittingStage() *SentenceSplittingStage {
	return &SentenceSplittingStage{}
}

// Name returns "SentenceSplittingStage".
func (s *SentenceSplittingStage) Name() string {
	return "SentenceSplittingStage"
}

// Execute segments knowledge content into sentences using sentencex.
// Input: *PipelineData
// Output: *PipelineData with Sentences populated
func (s *SentenceSplittingStage) Execute(ctx context.Context, input StageInput) (StageOutput, error) {
	data, ok := input.Data.(*PipelineData)
	if !ok {
		return StageOutput{}, fmt.Errorf("invalid input type for SentenceSplittingStage: expected *PipelineData, got %T", input.Data)
	}

	lang := "en"
	if data.ExtractionResult != nil && len(data.ExtractionResult.DetectedLanguages) > 0 {
		lang = data.ExtractionResult.DetectedLanguages[0]
	}

	var allSentences []Sentence

	for _, k := range data.Knowledges {
		rawSentences := sentencex.Segment(lang, k.Content)

		position := 0
		for _, content := range rawSentences {
			trimmed := strings.TrimSpace(content)
			if len(trimmed) <= 10 {
				continue
			}

			sentence := Sentence{
				ID:          uuid.New(),
				KnowledgeID: k.ID,
				Content:     trimmed,
				Position:    position,
				Embedding:   nil, // Will be populated by EmbeddingStage
			}

			allSentences = append(allSentences, sentence)
			position++
		}
	}

	data.Sentences = allSentences

	return StageOutput{
		Data: data,
	}, nil
}
