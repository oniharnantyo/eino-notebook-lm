package middleware

import (
	"context"
	"fmt"
	"hash/fnv"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/logger"
	appuuid "github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

type contextKey string

const runIDKey contextKey = "conversation_run_id"

// pendingConversation holds buffered conversation data for a request.
type pendingConversation struct {
	inputMessages  []*schema.Message
	outputMessages []*schema.Message
	lastUpdate     time.Time
	responseID     string
}

// ConversationMemoryMiddleware handles conversation persistence for Eino ADK agents.
// It loads conversation history before model invocation and saves responses asynchronously.
// Uses buffering to prevent multiple saves for the same request.
type ConversationMemoryMiddleware struct {
	conversationRepo repositories.ConversationRepository
	logger           *logger.Logger
	saveTimeout      time.Duration
	mu               sync.Mutex
	pendingSaves     map[string]*pendingConversation // key: request identifier
}

// NewConversationMemory creates a new conversation memory middleware.
// Default save timeout is 10 seconds.
func NewConversationMemory(conversationRepo repositories.ConversationRepository, log *logger.Logger) adk.ChatModelAgentMiddleware {
	return &ConversationMemoryMiddleware{
		conversationRepo: conversationRepo,
		logger:           log,
		saveTimeout:      10 * time.Second,
	}
}

// SetSaveTimeout sets the timeout for saving conversations.
func (m *ConversationMemoryMiddleware) SetSaveTimeout(timeout time.Duration) {
	m.saveTimeout = timeout
}

// BeforeAgent is called before each agent run.
// Injects a stable run ID for grouping model calls.
func (m *ConversationMemoryMiddleware) BeforeAgent(ctx context.Context, runCtx *adk.ChatModelAgentContext) (context.Context, *adk.ChatModelAgentContext, error) {
	runID := appuuid.New().String()
	ctx = context.WithValue(ctx, runIDKey, runID)
	return ctx, runCtx, nil
}

// BeforeModelRewriteState is called before each model invocation.
// Loads conversation history and injects it into the model input.
func (m *ConversationMemoryMiddleware) BeforeModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState, mc *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
	// Extract previous_response_id from context
	previousResponseID, ok := ctx.Value("previous_response_id").(string)
	if !ok || previousResponseID == "" {
		// No previous conversation to load
		return ctx, state, nil
	}

	// Load conversation from repository
	conversation, err := m.conversationRepo.FindByResponseID(ctx, previousResponseID)
	if err != nil {
		m.logger.Warn("Failed to load conversation history",
			"previous_response_id", previousResponseID,
			"error", err)
		// Graceful degradation: continue without history
		return ctx, state, nil
	}

	if conversation == nil {
		m.logger.Warn("Conversation not found",
			"previous_response_id", previousResponseID)
		return ctx, state, nil
	}

	// Inject messages into state (prepend loaded messages for proper conversation threading)
	messages := conversation.GetEinoMessages()
	state.Messages = append(messages, state.Messages...)

	m.logger.Debug("Loaded conversation history",
		"previous_response_id", previousResponseID,
		"message_count", len(messages))

	return ctx, state, nil
}

// AfterModelRewriteState is called after each model invocation.
// No-op for conversation memory (we save after agent completion).
func (m *ConversationMemoryMiddleware) AfterModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState, mc *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
	return ctx, state, nil
}

// WrapInvokableToolCall wraps tool calls.
// No-op for conversation memory.
func (m *ConversationMemoryMiddleware) WrapInvokableToolCall(ctx context.Context, endpoint adk.InvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.InvokableToolCallEndpoint, error) {
	return endpoint, nil
}

// WrapStreamableToolCall wraps streaming tool calls.
// No-op for conversation memory.
func (m *ConversationMemoryMiddleware) WrapStreamableToolCall(ctx context.Context, endpoint adk.StreamableToolCallEndpoint, tCtx *adk.ToolContext) (adk.StreamableToolCallEndpoint, error) {
	return endpoint, nil
}

// WrapEnhancedInvokableToolCall wraps enhanced tool calls.
// No-op for conversation memory.
func (m *ConversationMemoryMiddleware) WrapEnhancedInvokableToolCall(ctx context.Context, endpoint adk.EnhancedInvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.EnhancedInvokableToolCallEndpoint, error) {
	return endpoint, nil
}

// WrapEnhancedStreamableToolCall wraps enhanced streaming tool calls.
// No-op for conversation memory.
func (m *ConversationMemoryMiddleware) WrapEnhancedStreamableToolCall(ctx context.Context, endpoint adk.EnhancedStreamableToolCallEndpoint, tCtx *adk.ToolContext) (adk.EnhancedStreamableToolCallEndpoint, error) {
	return endpoint, nil
}

// WrapModel wraps model calls with a custom model that saves conversations.
func (m *ConversationMemoryMiddleware) WrapModel(ctx context.Context, baseModel model.BaseChatModel, mc *adk.ModelContext) (model.BaseChatModel, error) {
	return &conversationSavingModel{
		base:         baseModel,
		middleware:   m,
		modelContext: mc,
	}, nil
}

// conversationSavingModel wraps a BaseChatModel to save conversations after generation.
type conversationSavingModel struct {
	base         model.BaseChatModel
	middleware   *ConversationMemoryMiddleware
	modelContext *adk.ModelContext
}

// Generate delegates to the base model and saves the conversation asynchronously.
func (w *conversationSavingModel) Generate(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	resp, err := w.base.Generate(ctx, messages, opts...)
	if err != nil {
		return resp, err
	}

	// Save conversation asynchronously
	go w.middleware.saveAsync(ctx, messages, []*schema.Message{resp})

	return resp, nil
}

// Stream delegates to the base model and returns a wrapped stream that saves on close.
// This prevents race conditions by ensuring only the consumer consumes the stream,
// while the middleware handles saving after streaming completes.
func (w *conversationSavingModel) Stream(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	baseStream, err := w.base.Stream(ctx, messages, opts...)
	if err != nil {
		return baseStream, err
	}

	// Wrap the stream to collect chunks and save on close
	reader, writer := schema.Pipe[*schema.Message](10)

	go func() {
		defer writer.Close()
		defer baseStream.Close()

		collectedChunks := make([]*schema.Message, 0)

		// Forward chunks to consumer and collect for saving
		for {
			chunk, err := baseStream.Recv()
			if err != nil {
				// Stream ended - save collected chunks
				if len(collectedChunks) > 0 {
					merged := mergeStreamChunks(collectedChunks)
					if merged != nil {
						w.middleware.saveAsync(ctx, messages, []*schema.Message{merged})
					}
				}
				return
			}

			// Send to consumer
			if chunk != nil {
				collectedChunks = append(collectedChunks, chunk)
				writer.Send(chunk, nil)
			}
		}
	}()

	return reader, nil
}

// mergeStreamChunks merges streaming response chunks into a single message.
func mergeStreamChunks(chunks []*schema.Message) *schema.Message {
	if len(chunks) == 0 {
		return nil
	}

	if len(chunks) == 1 {
		return chunks[0]
	}

	// Merge content from all chunks
	merged := &schema.Message{
		Role:      chunks[0].Role,
		Extra:     chunks[0].Extra,
		ToolCalls: chunks[0].ToolCalls,
	}

	var textParts []string
	var reasoningParts []string
	var multimodalParts []schema.MessageOutputPart

	for _, chunk := range chunks {
		// Merge text content
		if chunk.Content != "" {
			textParts = append(textParts, chunk.Content)
		}

		// Merge reasoning content
		if chunk.ReasoningContent != "" {
			reasoningParts = append(reasoningParts, chunk.ReasoningContent)
		}

		// Merge multimodal content
		if len(chunk.AssistantGenMultiContent) > 0 {
			multimodalParts = append(multimodalParts, chunk.AssistantGenMultiContent...)
		}

		// Take the last ResponseMeta (most complete)
		if chunk.ResponseMeta != nil {
			merged.ResponseMeta = chunk.ResponseMeta
		}
	}

	merged.Content = strings.Join(textParts, "")
	merged.ReasoningContent = strings.Join(reasoningParts, "")
	merged.AssistantGenMultiContent = multimodalParts

	return merged
}

// getRequestID creates a unique identifier for a request based on context or input messages.
func (m *ConversationMemoryMiddleware) getRequestID(ctx context.Context, messages []*schema.Message) string {
	if runID, ok := ctx.Value(runIDKey).(string); ok && runID != "" {
		return runID
	}
	h := fnv.New32a()
	for _, msg := range messages {
		h.Write([]byte(string(msg.Role) + msg.Content))
	}
	return fmt.Sprintf("req-%x", h.Sum32())
}

// isFinalResponse checks if the output represents a final response (not intermediate tool calls).
func (m *ConversationMemoryMiddleware) isFinalResponse(messages []*schema.Message) bool {
	if len(messages) == 0 {
		return false
	}

	lastMsg := messages[len(messages)-1]

	// Final response should have actual content (not just tool calls)
	if lastMsg.Content == "" {
		return false
	}

	// Check if it has substantial content (not just thinking)
	return len(lastMsg.Content) > 50 // Has meaningful content
}

// saveAsync buffers the conversation and saves only for final responses or debounces rapid saves.
func (m *ConversationMemoryMiddleware) saveAsync(ctx context.Context, inputMessages []*schema.Message, outputMessages []*schema.Message) {
	reqID := m.getRequestID(ctx, inputMessages)

	m.mu.Lock()
	if m.pendingSaves == nil {
		m.pendingSaves = make(map[string]*pendingConversation)
	}

	// Check if this is a new request or an update to an existing one
	pending, exists := m.pendingSaves[reqID]
	if !exists {
		respID, ok := ctx.Value("response_id").(string)
		if !ok || respID == "" {
			respID = appuuid.New().String()
		}

		pending = &pendingConversation{
			inputMessages:  inputMessages,
			outputMessages: outputMessages,
			lastUpdate:     time.Now(),
			responseID:     respID,
		}
		m.pendingSaves[reqID] = pending
		m.mu.Unlock()

		m.doSave(ctx, pending)
		return
	}

	// Update existing pending conversation
	pending.outputMessages = outputMessages
	pending.lastUpdate = time.Now()
	m.mu.Unlock()

	// If this looks like a final response, save immediately
	if m.isFinalResponse(outputMessages) {
		m.flushSave(ctx, reqID)
	} else {
		// Schedule delayed save with debounce
		go m.delayedFlush(ctx, reqID)
	}
}

// flushSave immediately saves a buffered conversation.
func (m *ConversationMemoryMiddleware) flushSave(ctx context.Context, reqID string) {
	m.mu.Lock()
	pending, exists := m.pendingSaves[reqID]
	if !exists {
		m.mu.Unlock()
		return
	}
	delete(m.pendingSaves, reqID)
	m.mu.Unlock()

	m.doSave(ctx, pending)
}

// delayedFlush waits before saving to debounce rapid updates.
func (m *ConversationMemoryMiddleware) delayedFlush(ctx context.Context, reqID string) {
	time.Sleep(2 * time.Second) // Wait for potential final response

	m.mu.Lock()
	pending, exists := m.pendingSaves[reqID]
	if !exists {
		m.mu.Unlock()
		return
	}

	// Only flush if no recent updates (debounce)
	if time.Since(pending.lastUpdate) < 1*time.Second {
		m.mu.Unlock()
		return
	}
	delete(m.pendingSaves, reqID)
	m.mu.Unlock()

	m.doSave(ctx, pending)
}

// doSave performs the actual save operation.
func (m *ConversationMemoryMiddleware) doSave(ctx context.Context, pending *pendingConversation) {
	// Create context with timeout, preserving original context values
	saveCtx, cancel := context.WithTimeout(ctx, m.saveTimeout)
	defer cancel()

	// Build conversation entity
	conversation, err := m.buildConversation(saveCtx, pending)
	if err != nil {
		m.logger.Error("Failed to build conversation", "error", err)
		return
	}

	// Save to repository
	if err := m.conversationRepo.Save(saveCtx, conversation); err != nil {
		m.logger.Error("Failed to save conversation",
			"response_id", conversation.ResponseID,
			"error", err)
		return
	}

	m.logger.Debug("Saved conversation",
		"response_id", conversation.ResponseID,
		"message_count", len(conversation.Messages))
}

// buildConversation constructs a conversation entity from input and output messages.
func (m *ConversationMemoryMiddleware) buildConversation(ctx context.Context, pending *pendingConversation) (*entities.Conversation, error) {
	// Extract metadata from context
	var notebookID *string
	if nbID, ok := ctx.Value("notebook_id").(string); ok && nbID != "" {
		notebookID = &nbID
	}

	var previousResponseID *string
	if prevID, ok := ctx.Value("previous_response_id").(string); ok && prevID != "" {
		previousResponseID = &prevID
	}

	// Use stable response ID
	responseID := pending.responseID

	// Build stored messages
	storedMessages := make([]*entities.StoredMessage, 0, len(pending.inputMessages)+len(pending.outputMessages))

	// Add input messages
	for _, msg := range pending.inputMessages {
		storedMsg := &entities.StoredMessage{
			Role:      string(msg.Role),
			Content:   entities.MessageToStoredContent(msg),
			Extra:     msg.Extra,
			Timestamp: time.Now().Unix(),
		}
		storedMessages = append(storedMessages, storedMsg)
	}

	// Add output messages
	for _, msg := range pending.outputMessages {
		storedMsg := &entities.StoredMessage{
			Role:      string(msg.Role),
			Content:   entities.MessageToStoredContent(msg),
			Extra:     msg.Extra,
			Timestamp: time.Now().Unix(),
		}
		storedMessages = append(storedMessages, storedMsg)
	}

	// Extract metadata from the last output message
	metadata := make(map[string]string)
	var responseText string
	var responseMessage interface{}
	finishReason := ""
	promptTokens := 0
	completionTokens := 0
	totalTokens := 0

	if len(pending.outputMessages) > 0 {
		lastMsg := pending.outputMessages[len(pending.outputMessages)-1]

		// Extract response text
		responseText = m.extractResponseText(lastMsg)

		// Store full response message
		responseMessage = entities.MessageToStoredContent(lastMsg)

		// Extract metadata from ResponseMeta
		finishReason = m.extractFinishReason(lastMsg)
		promptTokens = m.extractPromptTokens(lastMsg)
		completionTokens = m.extractCompletionTokens(lastMsg)
		totalTokens = m.extractTotalTokens(lastMsg)
	}

	// Get model from context
	model := "unknown"
	if mdl, ok := ctx.Value("model").(string); ok && mdl != "" {
		model = mdl
	}

	// Create conversation entity with metadata fields
	conversation := entities.NewConversation(
		notebookID,
		previousResponseID,
		responseID,
		storedMessages,
		pending.inputMessages, // request_input
		responseText,
		responseMessage,
		model,
		metadata,
		finishReason,
		promptTokens,
		completionTokens,
		totalTokens,
	)

	return conversation, nil
}

// extractResponseText extracts the text content from a message.
func (m *ConversationMemoryMiddleware) extractResponseText(msg *schema.Message) string {
	if msg == nil {
		return ""
	}

	// Handle multimodal content
	if len(msg.AssistantGenMultiContent) > 0 {
		var textParts []string
		for _, part := range msg.AssistantGenMultiContent {
			if part.Type == schema.ChatMessagePartTypeText && part.Text != "" {
				textParts = append(textParts, part.Text)
			}
		}
		return strings.Join(textParts, "\n")
	}

	// Handle simple text content
	if msg.Content != "" {
		return msg.Content
	}

	// Handle reasoning content
	if msg.ReasoningContent != "" {
		return msg.ReasoningContent
	}

	return ""
}

// extractFinishReason extracts the finish reason from a message's ResponseMeta.
func (m *ConversationMemoryMiddleware) extractFinishReason(msg *schema.Message) string {
	if msg == nil || msg.ResponseMeta == nil {
		return ""
	}
	return msg.ResponseMeta.FinishReason
}

// extractPromptTokens extracts the prompt token count from a message's ResponseMeta.
func (m *ConversationMemoryMiddleware) extractPromptTokens(msg *schema.Message) int {
	if msg == nil || msg.ResponseMeta == nil || msg.ResponseMeta.Usage == nil {
		return 0
	}
	return msg.ResponseMeta.Usage.PromptTokens
}

// extractCompletionTokens extracts the completion token count from a message's ResponseMeta.
func (m *ConversationMemoryMiddleware) extractCompletionTokens(msg *schema.Message) int {
	if msg == nil || msg.ResponseMeta == nil || msg.ResponseMeta.Usage == nil {
		return 0
	}
	return msg.ResponseMeta.Usage.CompletionTokens
}

// extractTotalTokens extracts the total token count from a message's ResponseMeta.
func (m *ConversationMemoryMiddleware) extractTotalTokens(msg *schema.Message) int {
	if msg == nil || msg.ResponseMeta == nil || msg.ResponseMeta.Usage == nil {
		return 0
	}
	return msg.ResponseMeta.Usage.TotalTokens
}

// AfterAgent is called after the agent run completes successfully.
// Flushes any pending conversation saves to ensure all conversations are persisted.
func (m *ConversationMemoryMiddleware) AfterAgent(ctx context.Context, state *adk.TypedChatModelAgentState[*schema.Message]) (context.Context, error) {
	m.mu.Lock()

	// Collect all pending request IDs to flush
	reqIDs := make([]string, 0, len(m.pendingSaves))
	for reqID := range m.pendingSaves {
		reqIDs = append(reqIDs, reqID)
	}
	m.mu.Unlock()

	// Create a detached context for final saves to avoid context cancellation issues
	// The agent context may already be canceled when AfterAgent runs
	saveCtx := context.Background()

	// Flush all pending saves asynchronously
	// We don't block the agent pipeline on these final saves
	for _, reqID := range reqIDs {
		go m.flushSave(saveCtx, reqID)
	}

	return ctx, nil
}
