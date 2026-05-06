package pipeline

import (
	"context"
	"fmt"
	"strings"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/document"
)

type ParsingStage struct {
	parser document.DocumentParser
}

func NewParsingStage(parser document.DocumentParser) *ParsingStage {
	return &ParsingStage{
		parser: parser,
	}
}

func (s *ParsingStage) Name() string { return "ParsingStage" }

func (s *ParsingStage) Execute(ctx context.Context, input StageInput) (StageOutput, error) {
	data, ok := input.Data.(*PipelineData)
	if !ok {
		return StageOutput{}, fmt.Errorf("invalid input type for ParsingStage: expected *PipelineData, got %T", input.Data)
	}

	if data.ExtractionResult == nil {
		return StageOutput{}, fmt.Errorf("ExtractionResult is nil in PipelineData")
	}

	reader := strings.NewReader(data.ExtractionResult.Content)
	docs, err := s.parser.Parse(ctx, reader)
	if err != nil {
		return StageOutput{}, err
	}

	// Carry over metadata from extraction result to documents
	for _, doc := range docs {
		if doc.MetaData == nil {
			doc.MetaData = make(map[string]any)
		}
		for k, v := range data.ExtractionResult.Metadata {
			doc.MetaData[k] = v
		}
	}

	// Store documents in PipelineData for next stages
	data.Documents = docs
	return StageOutput{Data: data}, nil
}
