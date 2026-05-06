package repositories

import (
	"context"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
)

// ConversationRepository defines the interface for conversation storage
// Follows the Repository pattern from Clean Architecture
type ConversationRepository interface {
	// Save saves a conversation (create or update)
	Save(ctx context.Context, conversation *entities.Conversation) error

	// FindByResponseID finds a conversation by its response ID
	FindByResponseID(ctx context.Context, responseID string) (*entities.Conversation, error)

	// Delete deletes a conversation by response ID
	Delete(ctx context.Context, responseID string) error

	// Exists checks if a conversation exists for a response ID
	Exists(ctx context.Context, responseID string) (bool, error)

	// List retrieves conversations with pagination and optional filters
	List(ctx context.Context, filter ConversationFilter) ([]*entities.Conversation, int, error)
}

// ConversationFilter defines filtering options for listing conversations
type ConversationFilter struct {
	NotebookID         *string
	Model              *string
	Limit              int
	Offset             int
	OrderBy            string
	PreviousResponseID *string
}
