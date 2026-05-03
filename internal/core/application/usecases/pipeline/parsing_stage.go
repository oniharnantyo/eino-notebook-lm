package pipeline

import (
	"context"
	"fmt"
	"strings"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/document"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/extractor"
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
	result, ok := input.Data.(*extractor.ExtractionResult)
	if !ok {
		return StageOutput{}, fmt.Errorf("invalid input type for ParsingStage: expected *extractor.ExtractionResult, got %T", input.Data)
	}

	reader := strings.NewReader(result.Content)
	docs, err := s.parser.Parse(ctx, reader)
	if err != nil {
		return StageOutput{}, err
	}

	// Carry over metadata from extraction result to documents
	for _, doc := range docs {
		if doc.MetaData == nil {
			doc.MetaData = make(map[string]any)
		}
		for k, v := range result.Metadata {
			doc.MetaData[k] = v
		}
	}

	return StageOutput{Data: docs}, nil
}
