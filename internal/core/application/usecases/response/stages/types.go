package stages

import (
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ToolPreparation
type ToolPreparationInput struct {
	SourceIDs   []string
	SourceTypes []string
}

type ToolPreparationOutput struct {
	Tools []tool.BaseTool
}

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
