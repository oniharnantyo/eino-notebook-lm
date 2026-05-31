package dtos

// ============================================================
// REQUEST STRUCTURES - Responses API Format
// ============================================================

// ResponseRequest represents an OpenAI Responses API request
type ResponseRequest struct {
	Model              *string             `json:"model,omitempty" validate:"omitempty,min=1"`
	Input              interface{}         `json:"input" validate:"required"`
	PreviousResponseID *string             `json:"previous_response_id,omitempty" validate:"omitempty,min=1"`
	Tools              []ResponsesTool     `json:"tools,omitempty"`
	ToolChoice         ToolChoiceParam     `json:"tool_choice,omitempty"`
	Temperature        *float64            `json:"temperature,omitempty" validate:"omitempty,min=0,max=2"`
	MaxOutputTokens    *int                `json:"max_output_tokens,omitempty" validate:"omitempty,min=1,max=8192"`
	MaxToolCalls       *int                `json:"max_tool_calls,omitempty" validate:"omitempty,min=1"`
	Instructions       *string             `json:"instructions,omitempty"`
	Stream             bool                `json:"stream,omitempty"`
	StreamOptions      *StreamOptionsParam `json:"stream_options,omitempty"`
	Metadata           map[string]string   `json:"metadata,omitempty"`
	// Preserve RAG-specific fields
	NotebookID    *string  `json:"notebook_id,omitempty" validate:"omitempty,uuid"`
	SourceIDs     []string `json:"source_ids,omitempty"`
	SourceTypes   []string `json:"source_types,omitempty"`
	RetrievalMode string   `json:"retrieval_mode,omitempty"`
}

// ItemParam types (discriminated by "type" field)
type ItemParam interface {
	GetItemType() string
}

type UserMessageItemParam struct {
	ID      *string     `json:"id,omitempty"`
	Type    string      `json:"type"`    // "message"
	Role    string      `json:"role"`    // "user"
	Content interface{} `json:"content"` // string or []ContentPart
	Status  *string     `json:"status,omitempty"`
}

func (u *UserMessageItemParam) GetItemType() string { return "user_message" }

type AssistantMessageItemParam struct {
	ID      *string     `json:"id,omitempty"`
	Type    string      `json:"type"` // "message"
	Role    string      `json:"role"` // "assistant"
	Content interface{} `json:"content"`
	Status  *string     `json:"status,omitempty"`
}

func (a *AssistantMessageItemParam) GetItemType() string { return "assistant_message" }

type SystemMessageItemParam struct {
	ID      *string     `json:"id,omitempty"`
	Type    string      `json:"type"` // "message"
	Role    string      `json:"role"` // "system"
	Content interface{} `json:"content"`
	Status  *string     `json:"status,omitempty"`
}

func (s *SystemMessageItemParam) GetItemType() string { return "system_message" }

// Content parts
type InputTextContentParam struct {
	Type string `json:"type"` // "input_text"
	Text string `json:"text"`
}

type InputImageContentParam struct {
	Type     string  `json:"type"` // "input_image"
	ImageURL *string `json:"image_url,omitempty"`
	Detail   *string `json:"detail,omitempty"` // "low", "high", "auto"
}

type InputFileContentParam struct {
	Type     string  `json:"type"` // "input_file"
	Filename *string `json:"filename,omitempty"`
	FileURL  *string `json:"file_url,omitempty"`
	FileData *string `json:"file_data,omitempty"`
}

// ============================================================
// RESPONSE STRUCTURES - Responses API Format
// ============================================================

// ResponseResource represents the complete Responses API response
type ResponseResource struct {
	ID                string            `json:"id"`
	Object            string            `json:"object"` // "response"
	CreatedAt         int64             `json:"created_at"`
	CompletedAt       *int64            `json:"completed_at,omitempty"`
	Status            string            `json:"status"` // "in_progress", "completed", "failed"
	IncompleteDetails interface{}       `json:"incomplete_details,omitempty"`
	Model             string            `json:"model"`
	Output            []ItemField       `json:"output"`
	Error             *Error            `json:"error,omitempty"`
	Tools             []Tool            `json:"tools"`
	ToolChoice        interface{}       `json:"tool_choice,omitempty"`
	Truncation        string            `json:"truncation"` // "auto", "disabled"
	ParallelToolCalls bool              `json:"parallel_tool_calls"`
	Text              *TextField        `json:"text,omitempty"`
	Temperature       *float64          `json:"temperature,omitempty"`
	TopP              *float64          `json:"top_p,omitempty"`
	Usage             *Usage            `json:"usage,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	// New spec fields
	Background         bool               `json:"background"`
	FrequencyPenalty   *float64           `json:"frequency_penalty,omitempty"`
	PresencePenalty    *float64           `json:"presence_penalty,omitempty"`
	Instructions       *string            `json:"instructions,omitempty"`
	MaxOutputTokens    *int               `json:"max_output_tokens,omitempty"`
	MaxToolCalls       *int               `json:"max_tool_calls,omitempty"`
	Store              *bool              `json:"store,omitempty"`
	ServiceTier        *string            `json:"service_tier,omitempty"`
	TopLogprobs        *int               `json:"top_logprobs,omitempty"`
	PreviousResponseID *string            `json:"previous_response_id,omitempty"`
	PromptCacheKey     *string            `json:"prompt_cache_key,omitempty"`
	Reasoning          *ReasoningResource `json:"reasoning,omitempty"`
	SafetyIdentifier   *string            `json:"safety_identifier,omitempty"`
}

// ItemField represents output items (discriminated by "type" field)
type ItemField interface {
	GetItemType() string
}

type Message struct {
	ID      string        `json:"id"`
	Type    string        `json:"type"`   // "message"
	Status  string        `json:"status"` // "in_progress", "completed"
	Role    string        `json:"role"`   // "assistant"
	Content []ContentPart `json:"content"`
}

func (m *Message) GetItemType() string { return "message" }

// ReasoningItem represents a reasoning output item
type ReasoningItem struct {
	ID                string             `json:"id"`
	EncryptedContent  *string            `json:"encrypted_content,omitempty"`
	Type              string             `json:"type"`   // "reasoning"
	Status            string             `json:"status"` // "in_progress", "completed"
	Summary           []ReasoningSummary `json:"summary,omitempty"`
	Content           []ReasoningContent `json:"content,omitempty"`
}

func (r *ReasoningItem) GetItemType() string { return "reasoning" }

// ReasoningSummary represents a summary text in reasoning
type ReasoningSummary struct {
	Type string `json:"type"` // "summary_text"
	Text string `json:"text"`
}

type SummaryTextContent struct {
	Type string `json:"type"` // "summary_text"
	Text string `json:"text"`
}

type ReasoningResource struct {
	Effort  string             `json:"effort"`
	Summary []ReasoningSummary `json:"summary"`
}

// ReasoningContent represents content in reasoning
type ReasoningContent struct {
	Type string `json:"type"` // "output_text"
	Text string `json:"text"`
}

func (r *ReasoningContent) GetContentType() string { return "output_text" }

// FunctionCallItem represents a function call output item
type FunctionCallItem struct {
	ID        string `json:"id"`
	CallID    string `json:"call_id"`
	Type      string `json:"type"`   // "function_call"
	Status    string `json:"status"` // "in_progress", "completed"
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

func (f *FunctionCallItem) GetItemType() string { return "function_call" }

type ContentPart interface {
	GetContentType() string
}

type OutputTextContent struct {
	Type        string       `json:"type"` // "output_text"
	Text        string       `json:"text"`
	Annotations []Annotation `json:"annotations"`
}

func (o *OutputTextContent) GetContentType() string { return "output_text" }

type Annotation struct {
	Type       string `json:"type"` // "url_citation"
	URL        string `json:"url"`
	StartIndex int    `json:"start_index"`
	EndIndex   int    `json:"end_index"`
	Title      string `json:"title"`
}

type RefusalContent struct {
	Type    string `json:"type"` // "refusal"
	Refusal string `json:"refusal"`
}

func (r *RefusalContent) GetContentType() string { return "refusal" }

// ToolCallContent represents a tool call in the response
type ToolCallContent struct {
	Type       string              `json:"type"` // "tool_call"
	ID         string              `json:"id"`
	ToolCallID string              `json:"tool_call_id,omitempty"`
	Name       string              `json:"name"`
	Arguments  string              `json:"arguments,omitempty"`
	Status     string              `json:"status,omitempty"` // "in_progress", "completed"
	Error      *ToolCallError      `json:"error,omitempty"`
}

func (t *ToolCallContent) GetContentType() string { return "tool_call" }

// ToolCallError represents an error in tool call execution
type ToolCallError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Usage statistics
type Usage struct {
	InputTokens         int                  `json:"input_tokens"`
	OutputTokens        int                  `json:"output_tokens"`
	TotalTokens         int                  `json:"total_tokens"`
	InputTokensDetails  *InputTokensDetails  `json:"input_tokens_details,omitempty"`
	OutputTokensDetails *OutputTokensDetails `json:"output_tokens_details,omitempty"`
}

type InputTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

type OutputTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens"`
}

type TextField struct {
	Format    *TextFormatParam `json:"format,omitempty"`
	Verbosity *string          `json:"verbosity,omitempty"` // "low", "medium", "high"
}

type TextFormatParam struct {
	Type string `json:"type"` // "text", "json_object"
}

// Tool definitions
type ResponsesTool struct {
	Type        string                 `json:"type"` // "function"
	Name        string                 `json:"name"`
	Description *string                `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Strict      *bool                  `json:"strict,omitempty"`
}

// Tool represents a tool in the response
type Tool struct {
	Type        string                 `json:"type"` // "function"
	Name        string                 `json:"name"`
	Description *string                `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Strict      *bool                  `json:"strict,omitempty"`
}

type ToolChoiceParam interface{}

type SpecificFunctionParam struct {
	Type string `json:"type"` // "function"
	Name string `json:"name"`
}

type ToolChoiceValueEnum string

const (
	ToolChoiceNone     ToolChoiceValueEnum = "none"
	ToolChoiceAuto     ToolChoiceValueEnum = "auto"
	ToolChoiceRequired ToolChoiceValueEnum = "required"
)

type StreamOptionsParam struct {
	IncludeUsage *bool `json:"include_usage,omitempty"`
}

// Error types
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ============================================================
// STREAMING EVENT STRUCTURES - Responses API Format
// ============================================================

// Streaming event types (discriminated by "type" field)
type StreamingEvent interface {
	GetEventType() string
}

type ResponseCreatedEvent struct {
	Type           string            `json:"type"` // "response.created"
	SequenceNumber int               `json:"sequence_number"`
	Response       *ResponseResource `json:"response"`
}

func (r *ResponseCreatedEvent) GetEventType() string { return "response.created" }

type ResponseInProgressEvent struct {
	Type           string            `json:"type"` // "response.in_progress"
	SequenceNumber int               `json:"sequence_number"`
	Response       *ResponseResource `json:"response"`
}

func (r *ResponseInProgressEvent) GetEventType() string { return "response.in_progress" }

type ResponseCompletedEvent struct {
	Type           string            `json:"type"` // "response.completed"
	SequenceNumber int               `json:"sequence_number"`
	Response       *ResponseResource `json:"response"`
}

func (r *ResponseCompletedEvent) GetEventType() string { return "response.completed" }

type ResponseFailedEvent struct {
	Type           string            `json:"type"` // "response.failed"
	SequenceNumber int               `json:"sequence_number"`
	Response       *ResponseResource `json:"response"`
}

func (r *ResponseFailedEvent) GetEventType() string { return "response.failed" }

type ResponseOutputItemAddedEvent struct {
	Type           string    `json:"type"` // "response.output_item.added"
	SequenceNumber int       `json:"sequence_number"`
	OutputIndex    int       `json:"output_index"`
	Item           ItemField `json:"item"`
}

func (r *ResponseOutputItemAddedEvent) GetEventType() string { return "response.output_item.added" }

type ResponseOutputItemDoneEvent struct {
	Type           string    `json:"type"` // "response.output_item.done"
	SequenceNumber int       `json:"sequence_number"`
	OutputIndex    int       `json:"output_index"`
	Item           ItemField `json:"item"`
}

func (r *ResponseOutputItemDoneEvent) GetEventType() string { return "response.output_item.done" }

type ResponseContentPartAddedEvent struct {
	Type           string      `json:"type"` // "response.content_part.added"
	SequenceNumber int         `json:"sequence_number"`
	ItemID         string      `json:"item_id"`
	OutputIndex    int         `json:"output_index"`
	ContentIndex   int         `json:"content_index"`
	Part           ContentPart `json:"part"`
}

func (r *ResponseContentPartAddedEvent) GetEventType() string { return "response.content_part.added" }

type ResponseContentPartDoneEvent struct {
	Type           string      `json:"type"` // "response.content_part.done"
	SequenceNumber int         `json:"sequence_number"`
	ItemID         string      `json:"item_id"`
	OutputIndex    int         `json:"output_index"`
	ContentIndex   int         `json:"content_index"`
	Part           ContentPart `json:"part"`
}

func (r *ResponseContentPartDoneEvent) GetEventType() string { return "response.content_part.done" }

type ResponseOutputTextDeltaEvent struct {
	Type           string    `json:"type"` // "response.output_text.delta"
	SequenceNumber int       `json:"sequence_number"`
	ItemID         string    `json:"item_id"`
	OutputIndex    int       `json:"output_index"`
	ContentIndex   int       `json:"content_index"`
	Delta          string    `json:"delta"`
	Logprobs       []LogProb `json:"logprobs,omitempty"`
}

func (r *ResponseOutputTextDeltaEvent) GetEventType() string { return "response.output_text.delta" }

type ResponseOutputTextDoneEvent struct {
	Type           string    `json:"type"` // "response.output_text.done"
	SequenceNumber int       `json:"sequence_number"`
	ItemID         string    `json:"item_id"`
	OutputIndex    int       `json:"output_index"`
	ContentIndex   int       `json:"content_index"`
	Text           string    `json:"text"`
	Logprobs       []LogProb `json:"logprobs,omitempty"`
}

func (r *ResponseOutputTextDoneEvent) GetEventType() string { return "response.output_text.done" }

type LogProb struct {
	Token       string       `json:"token"`
	Logprob     float64      `json:"logprob"`
	Bytes       []int        `json:"bytes"`
	TopLogprobs []TopLogProb `json:"top_logprobs,omitempty"`
}

type TopLogProb struct {
	Token   string  `json:"token"`
	Logprob float64 `json:"logprob"`
	Bytes   []int   `json:"bytes"`
}

type ReasoningSummaryPartAddedEvent struct {
	Type           string           `json:"type"` // "response.reasoning_summary_part.added"
	SequenceNumber int              `json:"sequence_number"`
	ItemID         string           `json:"item_id"`
	OutputIndex    int              `json:"output_index"`
	SummaryIndex   int              `json:"summary_index"`
	Part           ReasoningSummary `json:"part"`
}

func (r *ReasoningSummaryPartAddedEvent) GetEventType() string {
	return "response.reasoning_summary_part.added"
}

type ReasoningSummaryPartDoneEvent struct {
	Type           string           `json:"type"` // "response.reasoning_summary_part.done"
	SequenceNumber int              `json:"sequence_number"`
	ItemID         string           `json:"item_id"`
	OutputIndex    int              `json:"output_index"`
	SummaryIndex   int              `json:"summary_index"`
	Part           ReasoningSummary `json:"part"`
}

func (r *ReasoningSummaryPartDoneEvent) GetEventType() string {
	return "response.reasoning_summary_part.done"
}

type ReasoningSummaryTextDeltaEvent struct {
	Type           string `json:"type"` // "response.reasoning_summary_text.delta"
	SequenceNumber int    `json:"sequence_number"`
	ItemID         string `json:"item_id"`
	OutputIndex    int    `json:"output_index"`
	SummaryIndex   int    `json:"summary_index"`
	Delta          string `json:"delta"`
}

func (r *ReasoningSummaryTextDeltaEvent) GetEventType() string {
	return "response.reasoning_summary_text.delta"
}

type ReasoningSummaryTextDoneEvent struct {
	Type           string `json:"type"` // "response.reasoning_summary_text.done"
	SequenceNumber int    `json:"sequence_number"`
	ItemID         string `json:"item_id"`
	OutputIndex    int    `json:"output_index"`
	SummaryIndex   int    `json:"summary_index"`
	Text           string `json:"text"`
}

func (r *ReasoningSummaryTextDoneEvent) GetEventType() string {
	return "response.reasoning_summary_text.done"
}

type ResponseReasoningDeltaEvent struct {
	Type           string `json:"type"` // "response.reasoning.delta"
	SequenceNumber int    `json:"sequence_number"`
	ItemID         string `json:"item_id"`
	OutputIndex    int    `json:"output_index"`
	ContentIndex   int    `json:"content_index"`
	Delta          string `json:"delta"`
}

func (r *ResponseReasoningDeltaEvent) GetEventType() string { return "response.reasoning.delta" }

type ResponseReasoningDoneEvent struct {
	Type           string `json:"type"` // "response.reasoning.done"
	SequenceNumber int    `json:"sequence_number"`
	ItemID         string `json:"item_id"`
	OutputIndex    int    `json:"output_index"`
	ContentIndex   int    `json:"content_index"`
	Text           string `json:"text"`
}

func (r *ResponseReasoningDoneEvent) GetEventType() string { return "response.reasoning.done" }

type ResponseFunctionCallArgumentsDeltaEvent struct {
	Type           string `json:"type"` // "response.function_call_arguments.delta"
	SequenceNumber int    `json:"sequence_number"`
	ItemID         string `json:"item_id"`
	CallID         string `json:"call_id"`
	OutputIndex    int    `json:"output_index"`
	ContentIndex   int    `json:"content_index"`
	Delta          string `json:"delta"`
}

func (r *ResponseFunctionCallArgumentsDeltaEvent) GetEventType() string {
	return "response.function_call_arguments.delta"
}

type ResponseFunctionCallArgumentsDoneEvent struct {
	Type           string `json:"type"` // "response.function_call_arguments.done"
	SequenceNumber int    `json:"sequence_number"`
	ItemID         string `json:"item_id"`
	CallID         string `json:"call_id"`
	OutputIndex    int    `json:"output_index"`
	ContentIndex   int    `json:"content_index"`
	Arguments      string `json:"arguments"`
}

func (r *ResponseFunctionCallArgumentsDoneEvent) GetEventType() string {
	return "response.function_call_arguments.done"
}
