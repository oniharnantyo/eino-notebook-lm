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
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
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
	notebookRepo     repositories.NotebookRepository
	conversationRepo  repositories.ConversationRepository
	retriever         retriever.Retriever
	embedder          embedding.Embedder
	chatModel         model.BaseChatModel
	defaultModel      string
	historyManager    *HistoryManager
}

// message represents a simple chat message for internal use
type message struct {
	Role    string
	Content string
}

func NewResponseUseCase(
	notebookRepo repositories.NotebookRepository,
	conversationRepo repositories.ConversationRepository,
	retriever retriever.Retriever,
	embedder embedding.Embedder,
	chatModel model.BaseChatModel,
	defaultModel string,
	historyConfig *HistoryConfig,
) chat.ResponseUseCase {
	return &responseUseCase{
		notebookRepo:     notebookRepo,
		conversationRepo:  conversationRepo,
		retriever:        retriever,
		embedder:         embedder,
		chatModel:        chatModel,
		defaultModel:     defaultModel,
		historyManager:    NewHistoryManager(historyConfig),
	}
}

// buildRAGChain creates a chain with conversation history support using Eino's MessagesPlaceholder
// Template structure: System -> [History] -> Current User Input -> ChatModel
func (uc *responseUseCase) buildRAGChain(ctx context.Context) (compose.Runnable[map[string]any, *schema.Message], error) {
	// Create a prompt template with conversation history support
	// The key is schema.MessagesPlaceholder which injects []*schema.Message at that position
	systemTemplate := prompt.FromMessages(
		schema.FString,
		schema.SystemMessage("{system_prompt}"),
		// MessagesPlaceholder injects conversation history here
		// This is Eino's built-in feature for handling multi-turn conversations
		schema.MessagesPlaceholder("history", false),
		schema.UserMessage("{user_input}"),
	)

	// Build the chain: Template -> ChatModel
	chain := compose.NewChain[map[string]any, *schema.Message]()
	chain.AppendChatTemplate(systemTemplate)
	chain.AppendChatModel(uc.chatModel)

	// Compile the chain with request context
	return chain.Compile(ctx)
}

func (uc *responseUseCase) CreateResponse(ctx context.Context, req *dtos.ResponseRequest) (*dtos.ResponseResource, error) {
	// DIAGNOSTIC: Log request start
	fmt.Printf("[DEBUG] CreateResponse called: notebook_id=%v, model=%s\n", req.NotebookID, uc.defaultModel)

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
	fmt.Printf("[DEBUG] Converted to %d messages\n", len(messages))

	// Validate last message is from user
	if len(messages) > 0 && messages[len(messages)-1].Role != "user" {
		return nil, fmt.Errorf("last message must be from user")
	}

	// Load conversation history if PreviousResponseID is provided
	history, err := uc.loadConversationHistory(ctx, req.PreviousResponseID)
	if err != nil {
		return nil, fmt.Errorf("failed to load conversation history: %w", err)
	}
	fmt.Printf("[DEBUG] Loaded history: %d messages\n", len(history))

	// Retrieve relevant context using conversation history
	contextText, err := uc.retrieveContextWithConversation(ctx, req, messages)
	if err != nil {
		return nil, err
	}
	fmt.Printf("[DEBUG] Retrieved context: %d chars\n", len(contextText))

	// Generate response using chain with history
	result, err := uc.generateWithChain(ctx, messages, history, contextText)
	if err != nil {
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	// DIAGNOSTIC: Verify result before building response
	if result == nil {
		fmt.Printf("[DEBUG] ERROR: result is nil after generateWithChain!\n")
		return nil, fmt.Errorf("generateWithChain returned nil result")
	}
	if result.Content == "" {
		fmt.Printf("[DEBUG] WARNING: result.Content is empty!\n")
	}

	// Build Responses API format response
	response := uc.buildResponseResource(uc.defaultModel, result)
	fmt.Printf("[DEBUG] Built response resource: id=%s, status=%s\n", response.ID, response.Status)

	// Save conversation for future use
	if err := uc.saveConversation(ctx, req, response, history, messages, result); err != nil {
		return nil, fmt.Errorf("failed to save conversation: %w", err)
	}

	return response, nil
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

	// Load conversation history if PreviousResponseID is provided
	history, err := uc.loadConversationHistory(ctx, req.PreviousResponseID)
	if err != nil {
		return nil, fmt.Errorf("failed to load conversation history: %w", err)
	}

	// Retrieve relevant context using conversation history
	contextText, err := uc.retrieveContextWithConversation(ctx, req, messages)
	if err != nil {
		return nil, err
	}

	// Build messages for streaming with history
	einoMessages := uc.buildEinoMessagesWithHistory(messages, history, contextText)

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

		// Save conversation after streaming completes
		// If this fails, we need to notify the client via the stream
		streamResponseMessage := &schema.Message{
			Role:    schema.Assistant,
			Content: accumulatedText,
		}
		if err := uc.saveConversation(ctx, req, uc.buildResponseResourceFromStream(responseID, messageID, createdAt, modelName, accumulatedText), history, messages, streamResponseMessage); err != nil {
			// Send response.failed event since we couldn't save the conversation
			uc.sendStreamingEvent(pw, &dtos.ResponseFailedEvent{
				Type:           "response.failed",
				SequenceNumber: seqNum,
				Response: &dtos.ResponseResource{
					ID:     responseID,
					Status: "failed",
					Error:  &dtos.Error{Code: "conversation_save_failed", Message: fmt.Sprintf("Failed to save conversation: %v", err)},
				},
			})
			return
		}

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

// loadConversationHistory loads conversation history from previous response
// applies history management strategy to limit the size
func (uc *responseUseCase) loadConversationHistory(ctx context.Context, previousResponseID *string) ([]*schema.Message, error) {
	if previousResponseID == nil || *previousResponseID == "" {
		return nil, nil // No history
	}

	conv, err := uc.conversationRepo.FindByResponseID(ctx, *previousResponseID)
	if err != nil {
		return nil, fmt.Errorf("failed to find previous conversation: %w", err)
	}

	// Get the full stored message history
	fullHistory := conv.GetEinoMessages()

	// Apply history management strategy (sliding window, token limit, etc.)
	trimmedHistory := uc.historyManager.TrimHistory(fullHistory)

	// Log history stats for monitoring (optional)
	stats := uc.historyManager.GetHistoryStats(trimmedHistory)
	fmt.Printf("History stats: messages=%d, tokens≈%d, strategy=%s\n",
		stats["message_count"], stats["estimated_tokens"], stats["strategy"])

	return trimmedHistory, nil
}

// saveConversation saves the conversation for future retrieval
func (uc *responseUseCase) saveConversation(
	ctx context.Context,
	req *dtos.ResponseRequest,
	response *dtos.ResponseResource,
	history []*schema.Message,
	messages []message,
	responseMessage *schema.Message,
) error {
	// Build the complete message history: previous history + current messages + assistant response
	storedMessages := make([]*entities.StoredMessage, 0)

	// Add previous history
	for _, msg := range history {
		storedMessages = append(storedMessages, &entities.StoredMessage{
			Role:      string(msg.Role),
			Content:   msg.Content,
			Extra:     msg.Extra,
			Timestamp: time.Now().Unix(),
		})
	}

	// Add current user messages
	for _, msg := range messages {
		storedMessages = append(storedMessages, &entities.StoredMessage{
			Role:      msg.Role,
			Content:   msg.Content,
			Timestamp: time.Now().Unix(),
		})
	}

	// Add assistant response
	storedMessages = append(storedMessages, &entities.StoredMessage{
		Role:      string(responseMessage.Role),
		Content:   responseMessage.Content,
		Extra:     responseMessage.Extra,
		Timestamp: time.Now().Unix(),
	})

	// Normalize request_input to always be an array of message items
	// This ensures consistent structure in the database
	normalizedInput := uc.normalizeInputForStorage(req.Input)

	// Create conversation entity with full response message
	conversation := entities.NewConversation(
		req.NotebookID,
		req.PreviousResponseID,
		response.ID,
		storedMessages,
		normalizedInput,
		responseMessage.Content, // ResponseText for quick access
		responseMessage,          // Full schema.Message for complete data
		response.Model,
		response.Metadata,
	)

	return uc.conversationRepo.Save(ctx, conversation)
}

// normalizeInputForStorage converts input to a consistent message items array format
func (uc *responseUseCase) normalizeInputForStorage(input interface{}) interface{} {
	switch v := input.(type) {
	case string:
		// Convert simple string to message item format (without type field)
		return []map[string]interface{}{
			{
				"role":    "user",
				"content": v,
			},
		}
	case []interface{}:
		// Remove "type" field from each item if present
		items := make([]map[string]interface{}, 0, len(v))
		for _, item := range v {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			// Create new map without "type" field
			cleaned := make(map[string]interface{})
			for key, val := range itemMap {
				if key != "type" {
					cleaned[key] = val
				}
			}
			items = append(items, cleaned)
		}
		return items
	default:
		// For any other type, return as-is
		return v
	}
}

// generateWithChain builds and uses a chain to generate responses with conversation history
func (uc *responseUseCase) generateWithChain(ctx context.Context, messages []message, history []*schema.Message, contextText string) (*schema.Message, error) {
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
		"history":       history, // This is passed to MessagesPlaceholder
	}

	// DIAGNOSTIC: Log before invoking chain
	fmt.Printf("[DEBUG] Invoking chain with: system_prompt_len=%d, user_input_len=%d, history_count=%d\n",
		len(vars["system_prompt"].(string)), len(userInput), len(history))

	// Invoke the chain with the variables map
	result, err := chain.Invoke(ctx, vars)
	if err != nil {
		fmt.Printf("[DEBUG] Chain invocation failed: %v\n", err)
		return nil, fmt.Errorf("chain invocation failed: %w", err)
	}

	// DIAGNOSTIC: Log result from chain
	if result == nil {
		fmt.Printf("[DEBUG] Chain returned nil result!\n")
		return nil, fmt.Errorf("chain returned nil result")
	}
	fmt.Printf("[DEBUG] Chain returned: role=%s, content_len=%d, content_preview=%q\n",
		result.Role, len(result.Content), result.Content)

	return result, nil
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

// buildEinoMessagesWithHistory builds Eino messages with conversation history for streaming
func (uc *responseUseCase) buildEinoMessagesWithHistory(messages []message, history []*schema.Message, context string) []*schema.Message {
	// Start with system prompt
	einoMsgs := []*schema.Message{
		{Role: schema.System, Content: uc.getSystemPrompt(context)},
	}

	// Add conversation history
	einoMsgs = append(einoMsgs, history...)

	// Add current messages
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

func (uc *responseUseCase) buildResponseResourceFromStream(responseID, messageID string, createdAt int64, model, text string) *dtos.ResponseResource {
	return &dtos.ResponseResource{
		ID:        responseID,
		Object:    "response",
		CreatedAt: createdAt,
		Status:    "completed",
		Model:     model,
		Output: []dtos.ItemField{
			&dtos.Message{
				ID:     messageID,
				Type:   "message",
				Status: "completed",
				Role:   "assistant",
				Content: []dtos.ContentPart{
					&dtos.OutputTextContent{
						Type:        "output_text",
						Text:        text,
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
		Metadata:          make(map[string]string),
	}
}