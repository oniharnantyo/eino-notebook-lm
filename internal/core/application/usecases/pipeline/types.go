package pipeline

import (
	"github.com/cloudwego/eino/schema"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/extractor"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// Sentence represents an intermediate sentence during pipeline processing.
// This is distinct from entities.Sentence - it's used for in-pipeline
// transformations before persistence.
type Sentence struct {
	ID          uuid.UUID
	KnowledgeID uuid.UUID
	Content     string
	Position    int
	Embedding   []float32
}

// PipelineData carries data through pipeline stages.
// It accumulates results as the pipeline progresses.
type PipelineData struct {
	ExtractionResult *extractor.ExtractionResult
	Documents        []*schema.Document // Parsed documents (passed through parsing/chunking/embedding stages)
	Knowledges       []*entities.Knowledge
	Sentences        []Sentence
	Images           []*entities.Image
}

// ConvertFloat64ToFloat32 converts a float64 slice to float32.
func ConvertFloat64ToFloat32(v []float64) []float32 {
	res := make([]float32, len(v))
	for i, f := range v {
		res[i] = float32(f)
	}
	return res
}
