package pipeline

import (
	"context"
	"fmt"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
)

// DocumentToKnowledgeStage converts schema.Document chunks to Knowledge entities.
// This stage bridges the gap between document processing (parsing, chunking, embedding)
// and domain persistence. It extracts embeddings from document metadata and carries
// over all metadata for persistence.
type DocumentToKnowledgeStage struct{}

// NewDocumentToKnowledgeStage creates a new DocumentToKnowledgeStage.
func NewDocumentToKnowledgeStage() *DocumentToKnowledgeStage {
	return &DocumentToKnowledgeStage{}
}

func (s *DocumentToKnowledgeStage) Name() string { return "DocumentToKnowledgeStage" }

func (s *DocumentToKnowledgeStage) Execute(ctx context.Context, input StageInput) (StageOutput, error) {
	data, ok := input.Data.(*PipelineData)
	if !ok {
		return StageOutput{}, fmt.Errorf("invalid input type for DocumentToKnowledgeStage: expected *PipelineData, got %T", input.Data)
	}

	if len(data.Documents) == 0 {
		return StageOutput{Data: data}, nil
	}

	knowledges := make([]*entities.Knowledge, len(data.Documents))
	for i, doc := range data.Documents {
		// Create knowledge entity with document content and metadata
		k, err := entities.NewKnowledge(
			input.SourceID,
			doc.Content,
			i, // Use array index as chunk index
			nil, // Heading context - not available in standard pipeline
			0,   // First page - not available in standard pipeline
			0,   // Last page - not available in standard pipeline
			doc.MetaData, // Carry over all metadata including embeddings
		)
		if err != nil {
			return StageOutput{}, fmt.Errorf("failed to create knowledge entity: %w", err)
		}

		knowledges[i] = k
	}

	// Add knowledges to PipelineData and return
	data.Knowledges = knowledges
	return StageOutput{Data: data}, nil
}
