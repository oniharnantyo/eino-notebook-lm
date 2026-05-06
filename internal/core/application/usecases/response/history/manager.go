package history

import (
	"fmt"

	"github.com/cloudwego/eino/schema"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
)

// HistoryStrategy defines how to manage conversation history
type HistoryStrategy string

const (
	// HistoryStrategyAll keeps all messages (default)
	HistoryStrategyAll HistoryStrategy = "all"
	// HistoryStrategySlidingWindow keeps the last N messages
	HistoryStrategySlidingWindow HistoryStrategy = "sliding_window"
	// HistoryStrategyTokenLimit keeps messages until a token threshold
	HistoryStrategyTokenLimit HistoryStrategy = "token_limit"
	// HistoryStrategySummarization summarizes old messages
	HistoryStrategySummarization HistoryStrategy = "summarization"
	// HistoryStrategyHybrid combines multiple strategies
	HistoryStrategyHybrid HistoryStrategy = "hybrid"
)

// HistoryConfig configures how conversation history is managed
type HistoryConfig struct {
	// Strategy determines which history management approach to use
	Strategy HistoryStrategy

	// MaxMessages is used for SlidingWindow strategy (default: 10)
	MaxMessages int

	// MaxTokens is used for TokenLimit strategy (default: 4000)
	MaxTokens int

	// TokenEstimationRatio is characters per token (default: 4)
	TokenEstimationRatio int

	// SummarizeThreshold is used for Summarization strategy
	// Messages older than this many turns get summarized (default: 5)
	SummarizeThreshold int

	// Hybrid config combines multiple strategies
	// - Summarize messages older than SummarizeThreshold
	// - Then apply sliding window to remaining
	// - Then apply token limit
}

// DefaultHistoryConfig returns sensible defaults
func DefaultHistoryConfig() *HistoryConfig {
	return &HistoryConfig{
		Strategy:             HistoryStrategySlidingWindow,
		MaxMessages:          10,
		MaxTokens:            4000,
		TokenEstimationRatio: 4,
		SummarizeThreshold:   5,
	}
}

// HistoryManager manages conversation history based on the configured strategy
type HistoryManager struct {
	config *HistoryConfig
}

// NewHistoryManager creates a new history manager
func NewHistoryManager(config *HistoryConfig) *HistoryManager {
	if config == nil {
		config = DefaultHistoryConfig()
	}
	return &HistoryManager{config: config}
}

// TrimHistory applies the configured strategy to limit conversation history
func (hm *HistoryManager) TrimHistory(history []*schema.Message) []*schema.Message {
	if len(history) == 0 {
		return history
	}

	switch hm.config.Strategy {
	case HistoryStrategyAll:
		return history
	case HistoryStrategySlidingWindow:
		return hm.applySlidingWindow(history)
	case HistoryStrategyTokenLimit:
		return hm.applyTokenLimit(history)
	case HistoryStrategySummarization:
		return hm.applySummarization(history)
	case HistoryStrategyHybrid:
		return hm.applyHybrid(history)
	default:
		return hm.applySlidingWindow(history) // Default to sliding window
	}
}

// applySlidingWindow keeps only the last N messages
func (hm *HistoryManager) applySlidingWindow(history []*schema.Message) []*schema.Message {
	maxMsgs := hm.config.MaxMessages
	if maxMsgs <= 0 {
		maxMsgs = 10 // Default
	}

	if len(history) <= maxMsgs {
		return history
	}

	// Keep the last maxMsgs messages
	return history[len(history)-maxMsgs:]
}

// applyTokenLimit keeps messages until reaching the token threshold
func (hm *HistoryManager) applyTokenLimit(history []*schema.Message) []*schema.Message {
	maxTokens := hm.config.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4000 // Default
	}

	ratio := hm.config.TokenEstimationRatio
	if ratio <= 0 {
		ratio = 4
	}

	totalTokens := 0
	// Iterate from newest to oldest
	for i := len(history) - 1; i >= 0; i-- {
		// Estimate tokens for this message
		msgTokens := hm.estimateTokens(history[i].Content)
		totalTokens += msgTokens

		if totalTokens > maxTokens {
			// Return messages from i+1 to end
			return history[i+1:]
		}
	}

	return history
}

// applySummarization summarizes old messages (simplified version)
// In production, you'd call an LLM to generate summaries
func (hm *HistoryManager) applySummarization(history []*schema.Message) []*schema.Message {
	threshold := hm.config.SummarizeThreshold
	if threshold <= 0 {
		threshold = 5
	}

	if len(history) <= threshold {
		return history
	}

	// Messages to summarize (everything except last threshold)
	toSummarize := history[:len(history)-threshold]
	keep := history[len(history)-threshold:]

	// In production: Call LLM to summarize toSummarize
	// For now: create a simple summary placeholder
	summaryText := hm.createSimpleSummary(toSummarize)

	// Build result: summary + recent messages
	result := []*schema.Message{
		{
			Role:    schema.System,
			Content: fmt.Sprintf("[Previous conversation summary: %s]", summaryText),
		},
	}
	result = append(result, keep...)

	return result
}

// applyHybrid combines multiple strategies
func (hm *HistoryManager) applyHybrid(history []*schema.Message) []*schema.Message {
	// Step 1: Apply summarization if needed
	result := history
	if hm.config.SummarizeThreshold > 0 && len(history) > hm.config.SummarizeThreshold*2 {
		result = hm.applySummarization(history)
	}

	// Step 2: Apply sliding window
	if hm.config.MaxMessages > 0 && len(result) > hm.config.MaxMessages {
		result = hm.applySlidingWindow(result)
	}

	// Step 3: Apply token limit as final safeguard
	if hm.config.MaxTokens > 0 {
		result = hm.applyTokenLimit(result)
	}

	return result
}

// estimateTokens estimates token count from text (rough approximation)
func (hm *HistoryManager) estimateTokens(content string) int {
	if content == "" {
		return 0
	}

	ratio := hm.config.TokenEstimationRatio
	if ratio <= 0 {
		ratio = 4
	}

	// Rough estimate: 1 token ≈ ratio characters
	tokenCount := len(content) / ratio

	// Minimum of 1 token for non-empty content
	if tokenCount < 1 {
		tokenCount = 1
	}

	return tokenCount
}

// createSimpleSummary creates a basic summary (placeholder for LLM-based summarization)
func (hm *HistoryManager) createSimpleSummary(messages []*schema.Message) string {
	userMsgCount := 0
	assistantMsgCount := 0

	for _, msg := range messages {
		if msg.Role == schema.User {
			userMsgCount++
		} else if msg.Role == schema.Assistant {
			assistantMsgCount++
		}
	}

	return fmt.Sprintf("The user and assistant had %d exchanges covering various topics. ", userMsgCount+assistantMsgCount)
}

// GetHistoryStats returns statistics about the history for monitoring
func (hm *HistoryManager) GetHistoryStats(history []*schema.Message) map[string]interface{} {
	totalTokens := 0
	for _, msg := range history {
		totalTokens += hm.estimateTokens(msg.Content)
	}

	return map[string]interface{}{
		"message_count":    len(history),
		"estimated_tokens": totalTokens,
		"strategy":         hm.config.Strategy,
		"max_messages":     hm.config.MaxMessages,
		"max_tokens":       hm.config.MaxTokens,
	}
}

// ConvertStoredMessages converts stored messages to Eino messages with history trimming
func (hm *HistoryManager) ConvertStoredMessages(stored []*entities.StoredMessage) []*schema.Message {
	messages := make([]*schema.Message, len(stored))
	for i, msg := range stored {
		messages[i] = msg.ToEinoMessage()
	}

	// Apply history management strategy
	return hm.TrimHistory(messages)
}
