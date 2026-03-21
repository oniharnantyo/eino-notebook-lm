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
		Limit:  req.Limit,
		Offset: offset,
		OrderBy: "created_at DESC",
	}

	if req.NotebookID != "" {
		filter.NotebookID = &req.NotebookID
	}

	if req.Model != "" {
		filter.Model = &req.Model
	}

	if req.PreviousResponseID != "" {
		filter.PreviousResponseID = &req.PreviousResponseID
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
