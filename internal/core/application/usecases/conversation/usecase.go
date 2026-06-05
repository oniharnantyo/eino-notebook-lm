package conversation

import (
	"context"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/errors"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
)

// ConversationUseCase defines the interface for conversation business logic
type ConversationUseCase interface {
	List(ctx context.Context, req *dtos.ListConversationsRequest) (*dtos.ListConversationsResponse, error)
	GetMessages(ctx context.Context, req *dtos.GetMessagesRequest) (*dtos.GetMessagesResponse, error)
}

// conversationUseCase implements ConversationUseCase
type conversationUseCase struct {
	conversationRepo repositories.ConversationRepository
}

// NewConversationUseCase creates a new conversation use case
func NewConversationUseCase(conversationRepo repositories.ConversationRepository) ConversationUseCase {
	return &conversationUseCase{
		conversationRepo: conversationRepo,
	}
}

// List retrieves a paginated list of conversations
func (uc *conversationUseCase) List(ctx context.Context, req *dtos.ListConversationsRequest) (*dtos.ListConversationsResponse, error) {
	// Set defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	offset := (req.Page - 1) * req.Limit

	// Build filter
	filter := repositories.ConversationFilter{
		Limit:   req.Limit,
		Offset:  offset,
		OrderBy: "created_at DESC",
	}

	if req.NotebookID != "" {
		filter.NotebookID = &req.NotebookID
	}

	if req.Model != "" {
		filter.Model = &req.Model
	}

	// Get conversations
	conversations, total, err := uc.conversationRepo.List(ctx, filter)
	if err != nil {
		return nil, errors.NewInternalError("failed to list conversations", err)
	}

	totalPages := int(total) / req.Limit
	if int(total)%req.Limit > 0 {
		totalPages++
	}

	return &dtos.ListConversationsResponse{
		Conversations: dtos.ToConversationResponses(conversations),
		Total:         int64(total),
		Page:          req.Page,
		Limit:         req.Limit,
		TotalPages:    totalPages,
	}, nil
}

// GetMessages retrieves paginated messages for a conversation
func (uc *conversationUseCase) GetMessages(ctx context.Context, req *dtos.GetMessagesRequest) (*dtos.GetMessagesResponse, error) {
	if req.NotebookID == "" {
		return nil, errors.NewValidationError("notebook_id is required")
	}

	if req.Limit < 1 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	convID := req.ConversationID
	if convID == "" {
		// Get latest conversation for notebook
		var err error
		convID, err = uc.conversationRepo.GetLatestConversationID(ctx, req.NotebookID)
		if err != nil {
			return nil, errors.NewInternalError("failed to get latest conversation", err)
		}
		if convID == "" {
			// No conversations yet
			return &dtos.GetMessagesResponse{
				Messages: []dtos.MessageResponse{},
				HasMore:  false,
			}, nil
		}
	}

	// The repository GetMessages orders by sequence_num DESC
	// This means we fetch limit+1 to determine if there are more
	fetchLimit := req.Limit + 1
	isHistory := true
	messages, err := uc.conversationRepo.GetMessages(ctx, convID, fetchLimit, req.BeforeSequence, &isHistory)
	if err != nil {
		return nil, errors.NewInternalError("failed to get messages", err)
	}

	hasMore := len(messages) > req.Limit
	if hasMore {
		// Since we sorted ascending in the query, the extra element (which represents
		// the older message) is at index 0. We discard it.
		messages = messages[1:]
	}

	var oldestSequence *int
	if len(messages) > 0 {
		// In ascending order, the first element (index 0) is the oldest message of the current page
		seq := messages[0].SequenceNum
		oldestSequence = &seq
	}

	return &dtos.GetMessagesResponse{
		Messages:       dtos.ToMessageResponses(messages),
		ConversationID: convID,
		HasMore:        hasMore,
		OldestSequence: oldestSequence,
	}, nil
}

func boolPtr(b bool) *bool {
	return &b
}
