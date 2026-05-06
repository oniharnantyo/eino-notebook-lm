package pipeline

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/recursive"
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
// Input: *PipelineData with Documents populated
// Output: *PipelineData with chunked Documents
func (s *ChunkingStage) Execute(ctx context.Context, input StageInput) (StageOutput, error) {
	data, ok := input.Data.(*PipelineData)
	if !ok {
		return StageOutput{}, fmt.Errorf("invalid input type for ChunkingStage: expected *PipelineData, got %T", input.Data)
	}

	if len(data.Documents) == 0 {
		return StageOutput{Data: data}, nil
	}

	splitter, err := recursive.NewSplitter(ctx, &recursive.Config{
		ChunkSize:   s.tokenLimit * 4,
		OverlapSize: (s.tokenLimit * 4) / 5,
	})
	if err != nil {
		return StageOutput{}, fmt.Errorf("failed to create recursive splitter: %w", err)
	}

	chunks, err := splitter.Transform(ctx, data.Documents)
	if err != nil {
		return StageOutput{}, fmt.Errorf("failed to chunk documents: %w", err)
	}

	// Update PipelineData with chunked documents
	data.Documents = chunks
	return StageOutput{Data: data}, nil
}
