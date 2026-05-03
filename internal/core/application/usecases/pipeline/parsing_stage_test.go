package pipeline

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/cloudwego/eino/components/document/parser"
	"github.com/cloudwego/eino/schema"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/extractor"
	"github.com/oniharnantyo/eino-notebook/pkg/parser/kreuzberg"
)

type mockParser struct {
	docs []*schema.Document
	err  error
}

func (m *mockParser) Parse(ctx context.Context, reader io.Reader, opts ...parser.Option) ([]*schema.Document, error) {
	return m.docs, m.err
}

func (m *mockParser) ParseFull(ctx context.Context, reader io.Reader) ([]kreuzberg.KreuzbergExtractResponse, error) {
	return nil, nil
}

func (m *mockParser) IsAvailable(ctx context.Context) bool {
	return true
}

func TestParsingStage_Execute(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		expectedDocs := []*schema.Document{{Content: "parsed content"}}
		mock := &mockParser{docs: expectedDocs}
		stage := &ParsingStage{parser: mock}

		input := StageInput{Data: &extractor.ExtractionResult{Content: "raw content"}}
		output, err := stage.Execute(ctx, input)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		docs, ok := output.Data.([]*schema.Document)
		if !ok {
			t.Fatalf("expected output data to be []*schema.Document, got %T", output.Data)
		}
		if len(docs) != 1 {
			t.Errorf("expected 1 document, got %v", len(docs))
		}
		if docs[0].Content != "parsed content" {
			t.Errorf("expected content 'parsed content', got '%s'", docs[0].Content)
		}
	})

	t.Run("failure", func(t *testing.T) {
		expectedErr := errors.New("parsing failed")
		mock := &mockParser{err: expectedErr}
		stage := &ParsingStage{parser: mock}

		input := StageInput{Data: &extractor.ExtractionResult{Content: "raw content"}}
		_, err := stage.Execute(ctx, input)

		if err != expectedErr {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("invalid input", func(t *testing.T) {
		stage := &ParsingStage{parser: &mockParser{}}
		input := StageInput{Data: "invalid input"}
		_, err := stage.Execute(ctx, input)

		if err == nil {
			t.Fatal("expected error for invalid input type")
		}
	})
}
