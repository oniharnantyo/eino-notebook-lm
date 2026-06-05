package dtos

// ============================================================
// CHAT COMPLETIONS REQUEST STRUCTURES - OpenAI-Compatible Format
// ============================================================

// ChatCompletionRequest represents an OpenAI chat.completions request
// This models the standard OpenAI Chat Completion API request format
// with support for custom context via extra_body fields
type ChatCompletionRequest struct {
	Messages            []ChatCompletionMessage `json:"messages" validate:"required,min=1"`
	Model               *string                 `json:"model,omitempty" validate:"omitempty,min=1"`
	Temperature         *float64                `json:"temperature,omitempty" validate:"omitempty,min=0,max=2"`
	MaxTokens           *int                    `json:"max_tokens,omitempty" validate:"omitempty,min=1"`
	MaxCompletionTokens *int                    `json:"max_completion_tokens,omitempty" validate:"omitempty,min=1"`
	Stream              bool                    `json:"stream,omitempty"`
	Tools               []ChatCompletionTool    `json:"tools,omitempty"`
	ToolChoice          interface{}             `json:"tool_choice,omitempty"`
	// extra_body fields (injected at root level by OpenAI clients)
	// These are extracted from the request body and used for RAG context
	NotebookID     *string `json:"notebook_id,omitempty" validate:"omitempty,uuid"`
	ConversationID *string `json:"conversation_id,omitempty" validate:"omitempty,uuid"`
	SourceID       *string `json:"source_id,omitempty" validate:"omitempty,uuid"`
}

// ChatCompletionMessage represents a message in the messages array
// Follows OpenAI's standard message format with role and content
type ChatCompletionMessage struct {
	Role    string      `json:"role" validate:"required,oneof=user assistant system"` // user, assistant, system
	Content interface{} `json:"content" validate:"required"`                          // string or []ContentPart
	Name    *string     `json:"name,omitempty"`                                       // optional name for the message
}

// ChatCompletionTool represents a tool definition in the request
// Follows OpenAI's tool format (currently only "function" type is supported)
type ChatCompletionTool struct {
	Type     string                 `json:"type" validate:"required,oneof=function"` // currently only "function"
	Function ChatCompletionFunction `json:"function"`
}

// ChatCompletionFunction defines a function that can be called by the model
type ChatCompletionFunction struct {
	Name        string                 `json:"name" validate:"required"`
	Description *string                `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Strict      *bool                  `json:"strict,omitempty"`
}
