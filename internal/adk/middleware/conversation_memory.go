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
	"github.com/cloudwego/eino/components/tool"
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
	cycleMessages  []*schema.Message // all output + tool messages in execution order
	lastUpdate     time.Time
	responseID     string
	ctx            context.Context
}

// conversationMemoryMiddleware handles conversation persistence for Eino ADK agents.
// It loads conversation history before model invocation and saves responses asynchronously.
// Uses buffering to prevent multiple saves for the same request.
type conversationMemoryMiddleware struct {
	conversationRepo repositories.ConversationRepository
	logger           *logger.Logger
	saveTimeout      time.Duration
	mu               sync.Mutex
	pendingSaves     map[string]*pendingConversation // key: request identifier
}

// NewConversationMemory creates a new conversation memory middleware.
// Default save timeout is 10 seconds.
func NewConversationMemory(conversationRepo repositories.ConversationRepository, log *logger.Logger) adk.ChatModelAgentMiddleware {
	return &conversationMemoryMiddleware{
		conversationRepo: conversationRepo,
		logger:           log,
		saveTimeout:      10 * time.Second,
	}
}

// SetSaveTimeout sets the timeout for saving conversations.
func (m *conversationMemoryMiddleware) SetSaveTimeout(timeout time.Duration) {
	m.saveTimeout = timeout
}

// BeforeAgent is called before each agent run.
// Injects a stable run ID for grouping model calls.
func (m *conversationMemoryMiddleware) BeforeAgent(ctx context.Context, runCtx *adk.ChatModelAgentContext) (context.Context, *adk.ChatModelAgentContext, error) {
	runID := appuuid.New().String()
	ctx = context.WithValue(ctx, runIDKey, runID)
	return ctx, runCtx, nil
}

// BeforeModelRewriteState is called before each model invocation.
// Loads conversation history and injects it into the model input.
func (m *conversationMemoryMiddleware) BeforeModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState, mc *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
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

	// Load messages for the conversation
	storedMessages, err := m.conversationRepo.GetMessages(ctx, conversation.ID, 100, nil, nil)
	if err != nil {
		m.logger.Warn("Failed to load messages for conversation",
			"conversation_id", conversation.ID,
			"error", err)
		return ctx, state, nil
	}

	// Inject messages into state (prepend loaded messages for proper conversation threading)
	var messages []*schema.Message
	for i := len(storedMessages) - 1; i >= 0; i-- { // Reverse if returned desc, but GetMessages has ORDER BY sequence_num DESC
		for _, sm := range storedMessages[i].Messages {
			if sm != nil {
				msg := sm.ToEinoMessage()
				if msg.Extra == nil {
					msg.Extra = make(map[string]any)
				}
				msg.Extra["_is_history"] = true
				messages = append(messages, msg)
			}
		}
	}
	state.Messages = append(messages, state.Messages...)

	// Store history message count for position-based filtering during save
	// This is more reliable than relying on Extra field which gets lost during message cloning
	ctx = context.WithValue(ctx, "history_message_count", len(messages))

	m.logger.Debug("Loaded conversation history",
		"previous_response_id", previousResponseID,
		"message_count", len(messages))

	return ctx, state, nil
}

// AfterModelRewriteState is called after each model invocation.
// No-op for conversation memory (we save after agent completion).
func (m *conversationMemoryMiddleware) AfterModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState, mc *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
	return ctx, state, nil
}

// WrapInvokableToolCall wraps tool calls to capture tool results for conversation persistence.
func (m *conversationMemoryMiddleware) WrapInvokableToolCall(ctx context.Context, endpoint adk.InvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.InvokableToolCallEndpoint, error) {
	return func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
		result, err := endpoint(ctx, argumentsInJSON, opts...)
		if err != nil {
			return "", err
		}

		toolMsg := schema.ToolMessage(result, tCtx.CallID, schema.WithToolName(tCtx.Name))

		reqID := m.getRequestID(ctx, nil)
		m.mu.Lock()
		if pending, exists := m.pendingSaves[reqID]; exists {
			pending.cycleMessages = append(pending.cycleMessages, toolMsg)
		}
		m.mu.Unlock()

		return result, nil
	}, nil
}

// WrapStreamableToolCall wraps streaming tool calls.
// No-op for conversation memory.
func (m *conversationMemoryMiddleware) WrapStreamableToolCall(ctx context.Context, endpoint adk.StreamableToolCallEndpoint, tCtx *adk.ToolContext) (adk.StreamableToolCallEndpoint, error) {
	return endpoint, nil
}

// WrapEnhancedInvokableToolCall wraps enhanced tool calls.
// No-op for conversation memory.
func (m *conversationMemoryMiddleware) WrapEnhancedInvokableToolCall(ctx context.Context, endpoint adk.EnhancedInvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.EnhancedInvokableToolCallEndpoint, error) {
	return endpoint, nil
}

// WrapEnhancedStreamableToolCall wraps enhanced streaming tool calls.
// No-op for conversation memory.
func (m *conversationMemoryMiddleware) WrapEnhancedStreamableToolCall(ctx context.Context, endpoint adk.EnhancedStreamableToolCallEndpoint, tCtx *adk.ToolContext) (adk.EnhancedStreamableToolCallEndpoint, error) {
	return endpoint, nil
}

// WrapModel wraps model calls with a custom model that saves conversations.
func (m *conversationMemoryMiddleware) WrapModel(ctx context.Context, baseModel model.BaseChatModel, mc *adk.ModelContext) (model.BaseChatModel, error) {
	return &conversationSavingModel{
		base:         baseModel,
		middleware:   m,
		modelContext: mc,
	}, nil
}

// conversationSavingModel wraps a BaseChatModel to save conversations after generation.
type conversationSavingModel struct {
	base         model.BaseChatModel
	middleware   *conversationMemoryMiddleware
	modelContext *adk.ModelContext
}

// Generate delegates to the base model and saves the conversation asynchronously.
func (w *conversationSavingModel) Generate(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	resp, err := w.base.Generate(ctx, messages, opts...)
	if err != nil {
		return resp, err
	}


	// Extract reasoning content from extra field before saving
	extractReasoningContent(resp)

	// Save conversation asynchronously
	go w.middleware.saveAsync(ctx, messages, []*schema.Message{resp})

	return resp, nil
}

// extractReasoningContent copies reasoning-content from Extra to ReasoningContent
// if ReasoningContent is not already set. Some models put reasoning in Extra.
func extractReasoningContent(msg *schema.Message) {
	if msg == nil || msg.ReasoningContent != "" {
		return
	}
	if msg.Extra != nil {
		if rc, ok := msg.Extra["reasoning-content"]; ok {
			if s, ok := rc.(string); ok && s != "" {
				msg.ReasoningContent = s
			}
		}
	}
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
					extractReasoningContent(merged)
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
// Uses schema.ConcatMessages which properly merges ToolCalls (grouped by Index,
// with Arguments concatenated), Content, ReasoningContent, and ResponseMeta.
func mergeStreamChunks(chunks []*schema.Message) *schema.Message {
	if len(chunks) == 0 {
		return nil
	}

	if len(chunks) == 1 {
		return chunks[0]
	}

	merged, err := schema.ConcatMessages(chunks)
	if err != nil {
		return chunks[len(chunks)-1]
	}

	return merged
}

// getRequestID creates a unique identifier for a request based on response_id.
// The Eino ADK framework calls BeforeAgent/AfterAgent per model-call cycle within
// a multi-step agent run, so runID changes each cycle. response_id is stable across
// all cycles in one user request, making it the correct grouping key.
func (m *conversationMemoryMiddleware) getRequestID(ctx context.Context, messages []*schema.Message) string {
	if respID, ok := ctx.Value("response_id").(string); ok && respID != "" {
		return "resp-" + respID
	}
	if runID, ok := ctx.Value(runIDKey).(string); ok && runID != "" {
		return runID
	}
	h := fnv.New32a()
	for _, msg := range messages {
		h.Write([]byte(string(msg.Role) + msg.Content))
	}
	return fmt.Sprintf("req-%x", h.Sum32())
}

// isFinalResponse checks if the output represents a final response (finish_reason != "tool_calls").
// Intermediate tool-calling responses are NOT final — the agent will make more calls.
// Only flush when the turn is truly complete (stop, length, content_filter, etc.).
func (m *conversationMemoryMiddleware) isFinalResponse(messages []*schema.Message) bool {
	if len(messages) == 0 {
		return false
	}

	lastMsg := messages[len(messages)-1]
	if lastMsg.ResponseMeta == nil || lastMsg.ResponseMeta.FinishReason == "" {
		return false
	}

	return lastMsg.ResponseMeta.FinishReason != "tool_calls"
}

// saveAsync buffers the conversation and saves only for final responses or debounces rapid saves.
func (m *conversationMemoryMiddleware) saveAsync(ctx context.Context, inputMessages []*schema.Message, outputMessages []*schema.Message) {
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
			cycleMessages:  outputMessages,
			lastUpdate:     time.Now(),
			responseID:     respID,
			ctx:            Detach(ctx),
		}
		m.pendingSaves[reqID] = pending
		m.mu.Unlock()

		return
	}

	// Update existing pending conversation — append in execution order
	pending.cycleMessages = append(pending.cycleMessages, outputMessages...)
	pending.lastUpdate = time.Now()
	m.mu.Unlock()

	// If this is a final response (finish_reason != "tool_calls"), save immediately.
	// For intermediate tool-calling responses, just accumulate — AfterAgent will
	// flush the complete turn when the agent run finishes.
	if m.isFinalResponse(outputMessages) {
		m.flushSave(reqID)
	}
}

// flushSave immediately saves a buffered conversation.
func (m *conversationMemoryMiddleware) flushSave(reqID string) {
	m.mu.Lock()
	pending, exists := m.pendingSaves[reqID]
	if !exists {
		m.mu.Unlock()
		return
	}
	delete(m.pendingSaves, reqID)
	m.mu.Unlock()

	m.doSave(pending.ctx, pending)
}

// delayedFlush waits before saving to debounce rapid updates.
func (m *conversationMemoryMiddleware) delayedFlush(reqID string) {
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

	m.doSave(pending.ctx, pending)
}

// detachedContext is a context that is never cancelled but carries values from a parent.
type detachedContext struct {
	parent context.Context
}

func (d *detachedContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (d *detachedContext) Done() <-chan struct{}       { return nil }
func (d *detachedContext) Err() error                  { return nil }
func (d *detachedContext) Value(key any) any           { return d.parent.Value(key) }

// Detach returns a context that carries the values of the parent but is not cancelled when the parent is.
func Detach(ctx context.Context) context.Context {
	return &detachedContext{parent: ctx}
}

// doSave performs the actual save operation.
func (m *conversationMemoryMiddleware) doSave(ctx context.Context, pending *pendingConversation) {
	// Use a detached context so the save survives request completion
	// while still carrying necessary metadata values (notebook_id, etc.)
	detached := Detach(ctx)
	saveCtx, cancel := context.WithTimeout(detached, m.saveTimeout)
	defer cancel()

	// Build conversation and messages
	conversation, messages, err := m.buildConversation(saveCtx, pending)
	if err != nil {
		m.logger.Error("Failed to build conversation and messages", "error", err)
		return
	}

	// Save to repository
	if err := m.conversationRepo.Save(saveCtx, conversation, messages); err != nil {
		m.logger.Error("Failed to save conversation and messages",
			"conversation_id", conversation.ID,
			"error", err)
		return
	}

	m.logger.Debug("Saved conversation and messages",
		"conversation_id", conversation.ID,
		"message_count", len(messages))
}

// buildConversation constructs conversation and message entities from pending data.
func (m *conversationMemoryMiddleware) buildConversation(ctx context.Context, pending *pendingConversation) (*entities.Conversation, []*entities.Message, error) {
	// Extract metadata from context
	var notebookID *string
	if nbID, ok := ctx.Value("notebook_id").(string); ok && nbID != "" {
		notebookID = &nbID
	}

	var previousResponseID *string
	if prevID, ok := ctx.Value("previous_response_id").(string); ok && prevID != "" {
		previousResponseID = &prevID
	}

	modelName := ""
	if mdl, ok := ctx.Value("model").(string); ok && mdl != "" {
		modelName = mdl
	}

	var conversation *entities.Conversation
	sequenceNum := 1

	// Try conversation_id first (explicit session tracking)
	if conversationID, ok := ctx.Value("conversation_id").(string); ok && conversationID != "" {
		existingConv, err := m.conversationRepo.FindByID(ctx, conversationID)
		if err != nil {
			m.logger.Warn("Failed to find conversation by ID, creating new one", "error", err, "conversation_id", conversationID)
		} else if existingConv != nil {
			conversation = existingConv
			existingMsgs, err := m.conversationRepo.GetMessages(ctx, conversation.ID, 1, nil, nil)
			if err == nil && len(existingMsgs) > 0 {
				sequenceNum = existingMsgs[0].SequenceNum + 1
			}
		}
	}

	// Fallback to previous_response_id (backward compat)
	if conversation == nil && previousResponseID != nil {
		var err error
		conversation, err = m.conversationRepo.FindByResponseID(ctx, *previousResponseID)
		if err != nil {
			m.logger.Warn("Failed to find previous conversation, creating new one", "error", err, "previous_response_id", *previousResponseID)
			conversation = nil
		} else if conversation != nil {
			existingMsgs, err := m.conversationRepo.GetMessages(ctx, conversation.ID, 1, nil, nil)
			if err == nil && len(existingMsgs) > 0 {
				sequenceNum = existingMsgs[0].SequenceNum + 1
			}
		}
	}

	// Create conversation entity if not found
	if conversation == nil {
		conversation = entities.NewConversation(
			notebookID,
			make(map[string]string),
		)
		// Use the conversation_id from context if available (matches what was sent to client)
		if convID, ok := ctx.Value("conversation_id").(string); ok && convID != "" {
			conversation.ID = convID
		}
	}

	// Get history message count to filter out already saved historical messages from inputMessages
	historyCount := 0
	if hc, ok := ctx.Value("history_message_count").(int); ok {
		historyCount = hc
	}

	var turnMessages []*schema.Message

	// 1. Add new input messages (skipping loaded history)
	if historyCount < len(pending.inputMessages) {
		turnMessages = appendUnique(turnMessages, pending.inputMessages[historyCount:])
	}

	// 2. Add all cycle messages (assistant + tool interleaved in execution order)
	turnMessages = appendUnique(turnMessages, pending.cycleMessages)

	// Convert Eino messages to StoredMessage structs, skipping system messages
	storedMessages := make([]*entities.StoredMessage, 0, len(turnMessages))
	for _, msg := range turnMessages {
		if msg == nil || msg.Role == schema.System {
			continue
		}
		storedMessages = append(storedMessages, entities.MessageToStoredContent(msg))
	}

	// If there are no messages, return empty slice
	if len(storedMessages) == 0 {
		return conversation, nil, nil
	}

	// Extract token usage and finish reason from the last cycle message
	var lastOutputMsg *schema.Message
	if len(pending.cycleMessages) > 0 {
		lastOutputMsg = pending.cycleMessages[len(pending.cycleMessages)-1]
	}

	var finishReason string
	var promptTokens, completionTokens, totalTokens int
	if lastOutputMsg != nil {
		finishReason = m.ExtractFinishReason(lastOutputMsg)
		promptTokens = m.ExtractPromptTokens(lastOutputMsg)
		completionTokens = m.ExtractCompletionTokens(lastOutputMsg)
		totalTokens = m.ExtractTotalTokens(lastOutputMsg)
	}

	// Group all turn messages into a single entities.Message
	message := entities.NewMessage(
		conversation.ID,
		sequenceNum,
		pending.responseID,
		previousResponseID,
		storedMessages,
		modelName,
		finishReason,
		promptTokens,
		completionTokens,
		totalTokens,
	)

	return conversation, []*entities.Message{message}, nil
}

// ExtractResponseText extracts the text content from a message.
func (m *conversationMemoryMiddleware) ExtractResponseText(msg *schema.Message) string {
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

// ExtractFinishReason extracts the finish reason from a message's ResponseMeta.
func (m *conversationMemoryMiddleware) ExtractFinishReason(msg *schema.Message) string {
	if msg == nil || msg.ResponseMeta == nil {
		return ""
	}
	return msg.ResponseMeta.FinishReason
}

// ExtractPromptTokens extracts the prompt token count from a message's ResponseMeta.
func (m *conversationMemoryMiddleware) ExtractPromptTokens(msg *schema.Message) int {
	if msg == nil || msg.ResponseMeta == nil || msg.ResponseMeta.Usage == nil {
		return 0
	}
	return msg.ResponseMeta.Usage.PromptTokens
}

// ExtractCompletionTokens extracts the completion token count from a message's ResponseMeta.
func (m *conversationMemoryMiddleware) ExtractCompletionTokens(msg *schema.Message) int {
	if msg == nil || msg.ResponseMeta == nil || msg.ResponseMeta.Usage == nil {
		return 0
	}
	return msg.ResponseMeta.Usage.CompletionTokens
}

// ExtractTotalTokens extracts the total token count from a message's ResponseMeta.
func (m *conversationMemoryMiddleware) ExtractTotalTokens(msg *schema.Message) int {
	if msg == nil || msg.ResponseMeta == nil || msg.ResponseMeta.Usage == nil {
		return 0
	}
	return msg.ResponseMeta.Usage.TotalTokens
}

// AfterAgent is called after each model-call cycle within an agent run.
// Only flushes pending saves when the turn is complete (finish_reason != "tool_calls").
// Intermediate cycles just accumulate — the final cycle with "stop" will trigger the flush.
func (m *conversationMemoryMiddleware) AfterAgent(ctx context.Context, state *adk.TypedChatModelAgentState[*schema.Message]) (context.Context, error) {
	m.mu.Lock()

	var toFlush []string
	for reqID, pending := range m.pendingSaves {
		if m.isFinalResponse(pending.cycleMessages) {
			toFlush = append(toFlush, reqID)
		}
	}
	m.mu.Unlock()

	for _, reqID := range toFlush {
		go m.flushSave(reqID)
	}

	return ctx, nil
}

func appendUnique(target []*schema.Message, source []*schema.Message) []*schema.Message {
	for _, srcMsg := range source {
		if srcMsg == nil {
			continue
		}
		duplicate := false
		for _, tgtMsg := range target {
			if messagesAreEqual(tgtMsg, srcMsg) {
				duplicate = true
				break
			}
		}
		if !duplicate {
			target = append(target, srcMsg)
		}
	}
	return target
}

func messagesAreEqual(a, b *schema.Message) bool {
	if a.Role != b.Role {
		return false
	}
	if a.Content != b.Content {
		return false
	}
	if len(a.ToolCalls) != len(b.ToolCalls) {
		return false
	}
	for i := range a.ToolCalls {
		if a.ToolCalls[i].ID != b.ToolCalls[i].ID ||
			a.ToolCalls[i].Function.Name != b.ToolCalls[i].Function.Name ||
			a.ToolCalls[i].Function.Arguments != b.ToolCalls[i].Function.Arguments {
			return false
		}
	}
	return true
}
