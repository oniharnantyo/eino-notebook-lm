package stages

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/schema"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/response/history"
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
	if conv == nil {
		return HistoryOutput{Messages: []*schema.Message{}}, nil
	}

	storedMessages, err := s.conversationRepo.GetMessages(ctx, conv.ID, 100, nil, nil)
	if err != nil {
		return HistoryOutput{}, fmt.Errorf("failed to get conversation messages: %w", err)
	}

	var fullHistory []*schema.Message
	for i := len(storedMessages) - 1; i >= 0; i-- {
		for _, m := range storedMessages[i].Messages {
			if m != nil {
				fullHistory = append(fullHistory, m.ToEinoMessage())
			}
		}
	}

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
