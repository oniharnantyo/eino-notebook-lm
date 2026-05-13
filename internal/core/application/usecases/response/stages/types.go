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

type HistorySaveInput struct {
	NotebookID         string
	PreviousResponseID *string
	ResponseID         string
	Model              string
	History            []*schema.Message
	UserInput          string
	ResponseMessage    *schema.Message
	RawInput           interface{}
	Metadata           map[string]string
}
