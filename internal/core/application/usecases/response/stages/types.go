package stages

import (
	"github.com/cloudwego/eino/schema"
)

// Generation
type GenerationOutput struct {
	Stream *schema.StreamReader[*schema.Message]
}

// History
type HistoryInput struct {
	NotebookID         string
	PreviousResponseID *string
	MaxTokens          int
}

type HistoryOutput struct {
	Messages []*schema.Message
}
