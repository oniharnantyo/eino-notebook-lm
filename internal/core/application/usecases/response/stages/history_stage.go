package stages

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/response/history"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
)

type HistoryStage struct {
	historyManager   *history.HistoryManager
	conversationRepo repositories.ConversationRepository
}

func NewHistoryStage(hm *history.HistoryManager, repo repositories.ConversationRepository) *HistoryStage {
	return &HistoryStage{
		historyManager:   hm,
		conversationRepo: repo,
	}
}

func (s *HistoryStage) Execute(ctx context.Context, input HistoryInput) (HistoryOutput, error) {
	return s.Load(ctx, input)
}

func (s *HistoryStage) Load(ctx context.Context, input HistoryInput) (HistoryOutput, error) {
	if input.PreviousResponseID == nil || *input.PreviousResponseID == "" {
		return HistoryOutput{Messages: []*schema.Message{}}, nil // No history
	}

	conv, err := s.conversationRepo.FindByResponseID(ctx, *input.PreviousResponseID)
	if err != nil {
		return HistoryOutput{}, fmt.Errorf("failed to find previous conversation: %w", err)
	}

	fullHistory := conv.GetEinoMessages()
	trimmedHistory := s.historyManager.TrimHistory(fullHistory)

	// Apply additional token limit if specified
	if input.MaxTokens > 0 {
		trimmedHistory = s.applyTokenLimit(trimmedHistory, input.MaxTokens)
	}

	return HistoryOutput{Messages: trimmedHistory}, nil
}

func (s *HistoryStage) applyTokenLimit(messages []*schema.Message, maxTokens int) []*schema.Message {
	totalTokens := 0
	for i := len(messages) - 1; i >= 0; i-- {
		// Simple token estimation: 1 token per 4 characters
		msgTokens := len(messages[i].Content) / 4
		if len(messages[i].Content) > 0 && msgTokens == 0 {
			msgTokens = 1
		}

		if totalTokens+msgTokens > maxTokens {
			return messages[i+1:]
		}
		totalTokens += msgTokens
	}
	return messages
}

func (s *HistoryStage) Save(ctx context.Context, input HistorySaveInput) error {
	// Build the complete message history: previous history + current user input + assistant response
	storedMessages := make([]*entities.StoredMessage, 0)

	// 1. Add previous history
	for _, msg := range input.History {
		storedMessages = append(storedMessages, &entities.StoredMessage{
			Role:      string(msg.Role),
			Content:   msg.Content,
			Extra:     msg.Extra,
			Timestamp: time.Now().Unix(),
		})
	}

	// 2. Add current user input
	storedMessages = append(storedMessages, &entities.StoredMessage{
		Role:      string(schema.User),
		Content:   input.UserInput,
		Timestamp: time.Now().Unix(),
	})

	// 3. Add assistant response
	storedMessages = append(storedMessages, &entities.StoredMessage{
		Role:      string(input.ResponseMessage.Role),
		Content:   input.ResponseMessage.Content,
		Extra:     input.ResponseMessage.Extra,
		Timestamp: time.Now().Unix(),
	})

	// Normalize request_input to always be an array of message items
	normalizedInput := s.normalizeInputForStorage(input.RawInput)

	// Create conversation entity
	conversation := entities.NewConversation(
		&input.NotebookID,
		input.PreviousResponseID,
		input.ResponseID,
		storedMessages,
		normalizedInput,
		input.ResponseMessage.Content,
		input.ResponseMessage,
		input.Model,
		input.Metadata,
	)

	return s.conversationRepo.Save(ctx, conversation)
}

func (s *HistoryStage) normalizeInputForStorage(input interface{}) interface{} {
	switch v := input.(type) {
	case string:
		return []map[string]interface{}{
			{
				"role":    "user",
				"content": v,
			},
		}
	case []interface{}:
		items := make([]map[string]interface{}, 0, len(v))
		for _, item := range v {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
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
		return input
	}
}
