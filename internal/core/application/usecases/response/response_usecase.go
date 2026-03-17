package response

import (
	"context"
	"encoding/json"
	errors2 "errors"
	"fmt"
	"io"
	"strings"
	"time"

	stduuid "github.com/google/uuid"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/chat"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/errors"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/retriever/pgvector"
	appuuid "github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// TokenEstimationRatio is the approximate number of characters per token.
// This is a rough estimation: 1 token ≈ 4 characters for English text.
// TODO: Use tiktoken or similar library for accurate token counting.
const TokenEstimationRatio = 4

type responseUseCase struct {
	notebookRepo repositories.NotebookRepository
	retriever    retriever.Retriever
	embedder     embedding.Embedder
	chatModel    model.BaseChatModel
	defaultModel string
}

// message represents a simple chat message for internal use
type message struct {
	Role    string
	Content string
}

func NewResponseUseCase(
	notebookRepo repositories.NotebookRepository,
	retriever retriever.Retriever,
	embedder embedding.Embedder,
	chatModel model.BaseChatModel,
	defaultModel string,
) chat.ResponseUseCase {
	return &responseUseCase{
		notebookRepo: notebookRepo,
		retriever:    retriever,
		embedder:     embedder,
		chatModel:    chatModel,
		defaultModel: defaultModel,
	}
}

// buildRAGChain creates a chain that: Input -> Prompt Template -> Chat Model -> Output
func (uc *responseUseCase) buildRAGChain(ctx context.Context) (compose.Runnable[map[string]any, *schema.Message], error) {
	// Create a prompt template with system prompt and context support
	systemTemplate := prompt.FromMessages(
		schema.FString,
		&schema.Message{
			Role:    schema.System,
			Content: "{system_prompt}",
		},
		&schema.Message{
			Role:    schema.User,
			Content: "{user_input}",
		},
	)

	// Build the chain: Template -> ChatModel
	chain := compose.NewChain[map[string]any, *schema.Message]()
	chain.AppendChatTemplate(systemTemplate)
	chain.AppendChatModel(uc.chatModel)

	// Compile the chain with request context
	return chain.Compile(ctx)
}

func (uc *responseUseCase) CreateResponse(ctx context.Context, req *dtos.ResponseRequest) (*dtos.ResponseResource, error) {
	// Validate notebook if provided
	_, err := uc.validateNotebook(ctx, req)
	if err != nil {
		return nil, err
	}

	// Get input text and messages
	messages, err := uc.convertInputToMessages(req.Input)
	if err != nil {
		return nil, fmt.Errorf("failed to convert input: %w", err)
	}

	// Validate last message is from user
	if len(messages) > 0 && messages[len(messages)-1].Role != "user" {
		return nil, fmt.Errorf("last message must be from user")
	}

	// Retrieve relevant context using conversation history
	contextText, err := uc.retrieveContextWithConversation(ctx, req, messages)
	if err != nil {
		return nil, err
	}

	// Generate response using chain
	result, err := uc.generateWithChain(ctx, messages, contextText)
	if err != nil {
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	// Build Responses API format response
	return uc.buildResponseResource(uc.defaultModel, result), nil
}

func (uc *responseUseCase) CreateResponseStream(ctx context.Context, req *dtos.ResponseRequest) (io.ReadCloser, error) {
	// Validate notebook if provided
	_, err := uc.validateNotebook(ctx, req)
	if err != nil {
		return nil, err
	}

	// Get input text and messages
	messages, err := uc.convertInputToMessages(req.Input)
	if err != nil {
		return nil, fmt.Errorf("failed to convert input: %w", err)
	}

	// Validate last message is from user
	if len(messages) > 0 && messages[len(messages)-1].Role != "user" {
		return nil, fmt.Errorf("last message must be from user")
	}

	// Retrieve relevant context using conversation history
	contextText, err := uc.retrieveContextWithConversation(ctx, req, messages)
	if err != nil {
		return nil, err
	}

	// Build messages for streaming
	einoMessages := uc.buildEinoMessages(messages, contextText)

	// Create streaming pipe
	pr, pw := io.Pipe()

	// Stream response in goroutine
	go func() {
		defer pw.Close()

		seqNum := 0
		responseID := fmt.Sprintf("resp_%s", stduuid.New().String())
		messageID := fmt.Sprintf("msg_%s", stduuid.New().String())
		createdAt := time.Now().Unix()
		modelName := uc.defaultModel

		// Send response.created event
		uc.sendStreamingEvent(pw, &dtos.ResponseCreatedEvent{
			Type:           "response.created",
			SequenceNumber: seqNum,
			Response: &dtos.ResponseResource{
				ID:                responseID,
				Object:            "response",
				CreatedAt:         createdAt,
				Status:            "in_progress",
				Model:             modelName,
				Output:            []dtos.ItemField{},
				Tools:             []dtos.Tool{},
				ToolChoice:        dtos.ToolChoiceAuto,
				Truncation:        "disabled",
				ParallelToolCalls: true,
				Text:              &dtos.TextField{Format: &dtos.TextFormatParam{Type: "text"}},
			},
		})
		seqNum++

		// Send response.in_progress event
		uc.sendStreamingEvent(pw, &dtos.ResponseInProgressEvent{
			Type:           "response.in_progress",
			SequenceNumber: seqNum,
			Response: &dtos.ResponseResource{
				ID:     responseID,
				Status: "in_progress",
			},
		})
		seqNum++

		// Send output_item.added event
		uc.sendStreamingEvent(pw, &dtos.ResponseOutputItemAddedEvent{
			Type:           "response.output_item.added",
			SequenceNumber: seqNum,
			OutputIndex:    0,
			Item: &dtos.Message{
				ID:      messageID,
				Type:    "message",
				Status:  "in_progress",
				Role:    "assistant",
				Content: []dtos.ContentPart{},
			},
		})
		seqNum++

		// Stream via chat model
		stream, err := uc.chatModel.Stream(ctx, einoMessages)
		if err != nil {
			uc.sendStreamingEvent(pw, &dtos.ResponseFailedEvent{
				Type:           "response.failed",
				SequenceNumber: seqNum,
				Response: &dtos.ResponseResource{
					ID:     responseID,
					Status: "failed",
					Error:  &dtos.Error{Code: "internal_error", Message: err.Error()},
				},
			})
			return
		}

		accumulatedText := ""

		// Send content_part.added event
		uc.sendStreamingEvent(pw, &dtos.ResponseContentPartAddedEvent{
			Type:           "response.content_part.added",
			SequenceNumber: seqNum,
			ItemID:         messageID,
			OutputIndex:    0,
			ContentIndex:   0,
			Part: &dtos.OutputTextContent{
				Type:        "output_text",
				Text:        "",
				Annotations: []dtos.Annotation{},
			},
		})
		seqNum++

		// Send chunks
		for {
			chunk, err := stream.Recv()
			if err != nil {
				break
			}

			accumulatedText += chunk.Content

			// Send delta event
			uc.sendStreamingEvent(pw, &dtos.ResponseOutputTextDeltaEvent{
				Type:           "response.output_text.delta",
				SequenceNumber: seqNum,
				ItemID:         messageID,
				OutputIndex:    0,
				ContentIndex:   0,
				Delta:          chunk.Content,
			})
			seqNum++
		}

		// Send output_text.done event
		uc.sendStreamingEvent(pw, &dtos.ResponseOutputTextDoneEvent{
			Type:           "response.output_text.done",
			SequenceNumber: seqNum,
			ItemID:         messageID,
			OutputIndex:    0,
			ContentIndex:   0,
			Text:           accumulatedText,
		})
		seqNum++

		// Send content_part.done event
		uc.sendStreamingEvent(pw, &dtos.ResponseContentPartDoneEvent{
			Type:           "response.content_part.done",
			SequenceNumber: seqNum,
			ItemID:         messageID,
			OutputIndex:    0,
			ContentIndex:   0,
			Part: &dtos.OutputTextContent{
				Type:        "output_text",
				Text:        accumulatedText,
				Annotations: []dtos.Annotation{},
			},
		})
		seqNum++

		// Send output_item.done event
		uc.sendStreamingEvent(pw, &dtos.ResponseOutputItemDoneEvent{
			Type:           "response.output_item.done",
			SequenceNumber: seqNum,
			OutputIndex:    0,
			Item: &dtos.Message{
				ID:     messageID,
				Type:   "message",
				Status: "completed",
				Role:   "assistant",
				Content: []dtos.ContentPart{
					&dtos.OutputTextContent{
						Type:        "output_text",
						Text:        accumulatedText,
						Annotations: []dtos.Annotation{},
					},
				},
			},
		})
		seqNum++

		// Send response.completed event
		completedAt := time.Now().Unix()
		uc.sendStreamingEvent(pw, &dtos.ResponseCompletedEvent{
			Type:           "response.completed",
			SequenceNumber: seqNum,
			Response: &dtos.ResponseResource{
				ID:          responseID,
				Object:      "response",
				CreatedAt:   createdAt,
				CompletedAt: &completedAt,
				Status:      "completed",
				Model:       modelName,
				Output: []dtos.ItemField{
					&dtos.Message{
						ID:     messageID,
						Type:   "message",
						Status: "completed",
						Role:   "assistant",
						Content: []dtos.ContentPart{
							&dtos.OutputTextContent{
								Type:        "output_text",
								Text:        accumulatedText,
								Annotations: []dtos.Annotation{},
							},
						},
					},
				},
				Tools:             []dtos.Tool{},
				ToolChoice:        dtos.ToolChoiceAuto,
				Truncation:        "disabled",
				ParallelToolCalls: true,
				Text:              &dtos.TextField{Format: &dtos.TextFormatParam{Type: "text"}},
			},
		})
	}()

	return pr, nil
}

// generateWithChain builds and uses a chain to generate responses
func (uc *responseUseCase) generateWithChain(ctx context.Context, messages []message, contextText string) (*schema.Message, error) {
	// Build the chain on the fly for this request
	chain, err := uc.buildRAGChain(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to build chain: %w", err)
	}

	// Prepare the input for the chain as map[string]any
	userInput := uc.buildUserInput(messages, contextText)
	vars := map[string]any{
		"system_prompt": uc.getSystemPrompt(contextText),
		"user_input":    userInput,
	}

	// Invoke the chain with the variables map
	return chain.Invoke(ctx, vars)
}

// buildUserInput creates the user input string with context
func (uc *responseUseCase) buildUserInput(messages []message, contextText string) string {
	if len(messages) == 0 {
		return ""
	}

	lastMsg := messages[len(messages)-1]
	if contextText != "" && lastMsg.Role == "user" {
		return contextText + "\n\nQuestion: " + lastMsg.Content
	}
	return lastMsg.Content
}

func (uc *responseUseCase) sendStreamingEvent(w io.Writer, event dtos.StreamingEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "data: %s\n\n", string(data))
	return nil
}

func (uc *responseUseCase) validateNotebook(ctx context.Context, req *dtos.ResponseRequest) (*appuuid.UUID, error) {
	if req.NotebookID == nil || *req.NotebookID == "" {
		return nil, errors2.New("notebook id is required")
	}

	notebookID, err := appuuid.Parse(*req.NotebookID)
	if err != nil {
		return nil, fmt.Errorf("invalid notebook_id: %w", err)
	}

	exists, err := uc.notebookRepo.Exists(ctx, notebookID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate notebook: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("notebook not found: %s", *req.NotebookID)
	}

	return &notebookID, nil
}

func (uc *responseUseCase) convertInputToMessages(input interface{}) ([]message, error) {
	var messages []message

	switch v := input.(type) {
	case string:
		if v == "" {
			return nil, errors.NewValidationError("input string cannot be empty")
		}
		messages = []message{
			{Role: "user", Content: v},
		}
	case []interface{}:
		for i, item := range v {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				return nil, errors.NewValidationError(fmt.Sprintf("input item at index %d is not a valid object", i))
			}

			itemType, ok := itemMap["type"].(string)
			if !ok {
				return nil, errors.NewValidationError(fmt.Sprintf("input item at index %d missing 'type' field", i))
			}

			role, ok := itemMap["role"].(string)
			if !ok {
				return nil, errors.NewValidationError(fmt.Sprintf("input item at index %d missing 'role' field", i))
			}

			if itemType == "message" {
				content, ok := itemMap["content"].(string)
				if !ok {
					return nil, errors.NewValidationError(fmt.Sprintf("input item at index %d has invalid 'content' field", i))
				}

				messages = append(messages, message{
					Role:    role,
					Content: content,
				})
			} else {
				return nil, errors.NewValidationError(fmt.Sprintf("unsupported item type '%s' at index %d", itemType, i))
			}
		}
	default:
		return nil, errors.NewDomainError("INVALID_INPUT_TYPE", fmt.Sprintf("unsupported input type: %T", input), errors.ErrInvalidInputType)
	}

	return messages, nil
}

func (uc *responseUseCase) buildFilterOptions(req *dtos.ResponseRequest) []retriever.Option {
	var opts []retriever.Option

	if len(req.SourceIDs) > 0 {
		opts = append(opts, pgvector.WithFilterReferenceIDs(req.SourceIDs))
	}

	// Use safe parameterized filtering for source_types
	if len(req.SourceTypes) > 0 {
		opts = append(opts, pgvector.WithFilterSourceTypes(req.SourceTypes))
	}

	return opts
}

// retrieveContextWithConversation uses conversation history to build a better retrieval query
func (uc *responseUseCase) retrieveContextWithConversation(ctx context.Context, req *dtos.ResponseRequest, messages []message) (string, error) {
	if len(messages) == 0 {
		return "", errors2.New("context id is required")
	}

	// Build conversation-aware query
	query := uc.buildConversationQuery(messages)
	if query == "" {
		return "", nil
	}

	opts := []retriever.Option{
		retriever.WithEmbedding(uc.embedder),
	}

	// Filter by source IDs and/or source types
	if len(req.SourceIDs) > 0 || len(req.SourceTypes) > 0 {
		opts = append(opts, uc.buildFilterOptions(req)...)
	}

	docs, err := uc.retriever.Retrieve(ctx, query, opts...)
	if err != nil {
		return "", err
	}

	if len(docs) == 0 {
		return "", nil
	}

	return uc.buildContextPrompt(docs), nil
}

// buildConversationQuery creates a search query from conversation history
func (uc *responseUseCase) buildConversationQuery(messages []message) string {
	if len(messages) == 0 {
		return ""
	}

	// Get the last user message (primary query)
	lastMsg := messages[len(messages)-1]
	if lastMsg.Role != "user" {
		return ""
	}

	// For single message conversations, use it directly
	if len(messages) == 1 {
		return lastMsg.Content
	}

	// For multi-turn conversations, build a context-aware query
	// Collect recent user messages to capture the full context
	var userQueries []string
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role == "user" {
			userQueries = append([]string{msg.Content}, userQueries...)
			// Limit to last 3 user messages to keep query focused
			if len(userQueries) >= 3 {
				break
			}
		}
	}

	// Build query: current question with context from previous exchanges
	if len(userQueries) == 1 {
		return userQueries[0]
	}

	// Join with context separator for better semantic search
	// Format: "previous topic 1; previous topic 2; current question"
	return strings.Join(userQueries, "; ")
}

func (uc *responseUseCase) buildContextPrompt(docs []*schema.Document) string {
	contextStr := "Context from relevant documents:\n\n"
	for i, doc := range docs {
		contextStr += fmt.Sprintf("[Document %d]\n%s\n\n", i+1, doc.Content)
	}
	return contextStr
}

func (uc *responseUseCase) getSystemPrompt(context string) string {
	if context == "" {
		return "You are a helpful assistant."
	}

	return `You are a helpful assistant with access to relevant context from a knowledge base.

Instructions:
- Use the provided context to answer questions accurately
- If the context doesn't contain enough information to answer the question, say so
- Cite information from the context when relevant
- Do not make up information that isn't in the context`
}

func (uc *responseUseCase) buildEinoMessages(messages []message, context string) []*schema.Message {
	// Start with system prompt
	einoMsgs := []*schema.Message{
		{Role: schema.System, Content: uc.getSystemPrompt(context)},
	}

	for _, msg := range messages {
		content := msg.Content
		if msg.Role == "user" && context != "" {
			content = context + "\n\nQuestion: " + content
		}

		role := schema.RoleType(msg.Role)
		einoMsgs = append(einoMsgs, &schema.Message{Role: role, Content: content})
	}

	return einoMsgs
}

func (uc *responseUseCase) buildResponseResource(model string, result *schema.Message) *dtos.ResponseResource {
	now := time.Now().Unix()
	responseID := fmt.Sprintf("resp_%s", stduuid.New().String())
	messageID := fmt.Sprintf("msg_%s", stduuid.New().String())

	message := &dtos.Message{
		ID:     messageID,
		Type:   "message",
		Status: "completed",
		Role:   "assistant",
		Content: []dtos.ContentPart{
			&dtos.OutputTextContent{
				Type:        "output_text",
				Text:        result.Content,
				Annotations: []dtos.Annotation{},
			},
		},
	}

	// Estimate token count using character ratio (1 token ≈ TokenEstimationRatio characters)
	// TODO: Use tiktoken or similar library for accurate token counting
	contentLen := len(result.Content)
	usage := &dtos.Usage{
		InputTokens:         contentLen / TokenEstimationRatio,
		OutputTokens:        contentLen / TokenEstimationRatio,
		TotalTokens:         contentLen / (TokenEstimationRatio / 2),
		InputTokensDetails:  &dtos.InputTokensDetails{CachedTokens: 0},
		OutputTokensDetails: &dtos.OutputTokensDetails{ReasoningTokens: 0},
	}

	return &dtos.ResponseResource{
		ID:                responseID,
		Object:            "response",
		CreatedAt:         now,
		CompletedAt:       &now,
		Status:            "completed",
		Model:             model,
		Output:            []dtos.ItemField{message},
		Tools:             []dtos.Tool{},
		ToolChoice:        dtos.ToolChoiceAuto,
		Truncation:        "disabled",
		ParallelToolCalls: true,
		Text:              &dtos.TextField{Format: &dtos.TextFormatParam{Type: "text"}},
		Usage:             usage,
		Metadata:          make(map[string]string),
	}
}
