package pipeline

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/recursive"
	"github.com/cloudwego/eino/schema"
)

// ChunkingStage splits large documents into smaller chunks based on token limits.
type ChunkingStage struct {
	tokenLimit int
}

// NewChunkingStage creates a new ChunkingStage with the given token limit.
func NewChunkingStage(tokenLimit int) *ChunkingStage {
	return &ChunkingStage{tokenLimit: tokenLimit}
}

// Name returns "ChunkingStage".
func (s *ChunkingStage) Name() string { return "ChunkingStage" }

// Execute splits documents into chunks.
// Input: []*schema.Document
// Output: []*schema.Document (chunked)
func (s *ChunkingStage) Execute(ctx context.Context, input StageInput) (StageOutput, error) {
	docs, ok := input.Data.([]*schema.Document)
	if !ok {
		return StageOutput{}, fmt.Errorf("invalid input type for ChunkingStage: expected []*schema.Document, got %T", input.Data)
	}

	splitter, err := recursive.NewSplitter(ctx, &recursive.Config{
		ChunkSize:   s.tokenLimit * 4,
		OverlapSize: (s.tokenLimit * 4) / 5,
	})
	if err != nil {
		return StageOutput{}, fmt.Errorf("failed to create recursive splitter: %w", err)
	}

	chunks, err := splitter.Transform(ctx, docs)
	if err != nil {
		return StageOutput{}, fmt.Errorf("failed to chunk documents: %w", err)
	}

	return StageOutput{Data: chunks}, nil
}
