package pipeline

import (
	"context"
	"errors"
	"testing"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/extractor"
)

type mockExtractor struct {
	result *extractor.ExtractionResult
	err    error
}

func (m *mockExtractor) Extract(ctx context.Context, source usecases.ContentSource) (*extractor.ExtractionResult, error) {
	return m.result, m.err
}

func TestExtractionStage_Execute(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		expectedResult := &extractor.ExtractionResult{Content: "test content"}
		mock := &mockExtractor{result: expectedResult}
		stage := &ExtractionStage{extractor: mock}

		input := StageInput{Data: usecases.ContentSource{URL: "http://example.com"}}
		output, err := stage.Execute(ctx, input)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, ok := output.Data.(*PipelineData)
		if !ok {
			t.Fatalf("expected *PipelineData, got %T", output.Data)
		}
		if data.ExtractionResult != expectedResult {
			t.Errorf("expected result %v, got %v", expectedResult, data.ExtractionResult)
		}
	})

	t.Run("failure", func(t *testing.T) {
		expectedErr := errors.New("extraction failed")
		mock := &mockExtractor{err: expectedErr}
		stage := &ExtractionStage{extractor: mock}

		input := StageInput{Data: usecases.ContentSource{URL: "http://example.com"}}
		_, err := stage.Execute(ctx, input)

		if err != expectedErr {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("invalid input", func(t *testing.T) {
		stage := &ExtractionStage{extractor: &mockExtractor{}}
		input := StageInput{Data: "invalid input"}
		_, err := stage.Execute(ctx, input)

		if err == nil {
			t.Fatal("expected error for invalid input type")
		}
	})
}
