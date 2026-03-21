package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
)

// PostgresConversationRepository implements ConversationRepository using PostgreSQL
type PostgresConversationRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresConversationRepository creates a new PostgreSQL conversation repository
func NewPostgresConversationRepository(pool *pgxpool.Pool) repositories.ConversationRepository {
	return &PostgresConversationRepository{
		pool: pool,
	}
}

// Save saves a conversation (create or update)
func (r *PostgresConversationRepository) Save(ctx context.Context, conversation *entities.Conversation) error {
	query := `
		INSERT INTO conversations (id, notebook_id, response_id, previous_response_id, messages, request_input, response_text, response_message, model, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (response_id) DO UPDATE SET
			notebook_id = EXCLUDED.notebook_id,
			previous_response_id = EXCLUDED.previous_response_id,
			messages = EXCLUDED.messages,
			request_input = EXCLUDED.request_input,
			response_text = EXCLUDED.response_text,
			response_message = EXCLUDED.response_message,
			model = EXCLUDED.model,
			metadata = EXCLUDED.metadata
	`

	messagesJSON, err := json.Marshal(conversation.Messages)
	if err != nil {
		return fmt.Errorf("failed to marshal messages: %w", err)
	}

	requestInputJSON, err := json.Marshal(conversation.RequestInput)
	if err != nil {
		return fmt.Errorf("failed to marshal request input: %w", err)
	}

	responseMessageJSON, err := json.Marshal(conversation.ResponseMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal response message: %w", err)
	}

	metadataJSON, err := json.Marshal(conversation.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Convert Unix timestamp to time.Time for PostgreSQL timestamptz
	createdAt := time.Unix(conversation.CreatedAt, 0)

	_, err = r.pool.Exec(ctx, query,
		conversation.ID,
		conversation.NotebookID,
		conversation.ResponseID,
		conversation.PreviousResponseID,
		messagesJSON,
		requestInputJSON,
		conversation.ResponseText,
		responseMessageJSON,
		conversation.Model,
		metadataJSON,
		createdAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save conversation: %w", err)
	}

	return nil
}

// FindByResponseID finds a conversation by its response ID
func (r *PostgresConversationRepository) FindByResponseID(ctx context.Context, responseID string) (*entities.Conversation, error) {
	if responseID == "" {
		return nil, fmt.Errorf("response ID cannot be empty")
	}

	query := `
		SELECT id, notebook_id, response_id, previous_response_id, messages, request_input, response_text, response_message, model, metadata, created_at
		FROM conversations
		WHERE response_id = $1
	`

	var conversation entities.Conversation
	var messagesJSON, requestInputJSON, responseMessageJSON, metadataJSON []byte
	var createdAt time.Time

	err := r.pool.QueryRow(ctx, query, responseID).Scan(
		&conversation.ID,
		&conversation.NotebookID,
		&conversation.ResponseID,
		&conversation.PreviousResponseID,
		&messagesJSON,
		&requestInputJSON,
		&conversation.ResponseText,
		&responseMessageJSON,
		&conversation.Model,
		&metadataJSON,
		&createdAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to find conversation: %w", err)
	}

	// Convert time.Time to Unix timestamp
	conversation.CreatedAt = createdAt.Unix()

	// Parse JSON fields
	if err := json.Unmarshal(messagesJSON, &conversation.Messages); err != nil {
		return nil, fmt.Errorf("failed to unmarshal messages: %w", err)
	}

	if requestInputJSON != nil {
		if err := json.Unmarshal(requestInputJSON, &conversation.RequestInput); err != nil {
			return nil, fmt.Errorf("failed to unmarshal request input: %w", err)
		}
	}

	if responseMessageJSON != nil {
		if err := json.Unmarshal(responseMessageJSON, &conversation.ResponseMessage); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response message: %w", err)
		}
	}

	if metadataJSON != nil {
		if err := json.Unmarshal(metadataJSON, &conversation.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &conversation, nil
}

// Delete deletes a conversation by response ID
func (r *PostgresConversationRepository) Delete(ctx context.Context, responseID string) error {
	if responseID == "" {
		return fmt.Errorf("response ID cannot be empty")
	}

	query := `DELETE FROM conversations WHERE response_id = $1`

	_, err := r.pool.Exec(ctx, query, responseID)
	if err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}

	return nil
}

// Exists checks if a conversation exists for a response ID
func (r *PostgresConversationRepository) Exists(ctx context.Context, responseID string) (bool, error) {
	if responseID == "" {
		return false, fmt.Errorf("response ID cannot be empty")
	}

	query := `SELECT EXISTS(SELECT 1 FROM conversations WHERE response_id = $1)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, responseID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check conversation existence: %w", err)
	}

	return exists, nil
}

// List retrieves conversations with pagination and optional filters
func (r *PostgresConversationRepository) List(ctx context.Context, filter repositories.ConversationFilter) ([]*entities.Conversation, int, error) {
	// Build WHERE clause dynamically
	whereClause := ""
	args := []interface{}{}
	argCount := 1

	if filter.NotebookID != nil {
		whereClause += fmt.Sprintf(" WHERE notebook_id = $%d", argCount)
		args = append(args, *filter.NotebookID)
		argCount++
	}

	if filter.Model != nil {
		if whereClause == "" {
			whereClause += " WHERE"
		} else {
			whereClause += " AND"
		}
		whereClause += fmt.Sprintf(" model = $%d", argCount)
		args = append(args, *filter.Model)
		argCount++
	}

	if filter.PreviousResponseID != nil {
		if whereClause == "" {
			whereClause += " WHERE"
		} else {
			whereClause += " AND"
		}
		whereClause += fmt.Sprintf(" previous_response_id = $%d", argCount)
		args = append(args, *filter.PreviousResponseID)
		argCount++
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM conversations" + whereClause
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count conversations: %w", err)
	}

	// Build ORDER BY clause with validation
	orderBy := "created_at DESC"
	if filter.OrderBy != "" {
		// Validate orderBy to prevent SQL injection
		allowedOrders := map[string]bool{
			"created_at DESC":  true,
			"created_at ASC":   true,
			"model DESC":       true,
			"model ASC":        true,
			"response_id DESC": true,
			"response_id ASC":  true,
		}
		if allowedOrders[filter.OrderBy] {
			orderBy = filter.OrderBy
		}
	}

	// Get conversations
	query := `
		SELECT id, notebook_id, response_id, previous_response_id, messages, request_input, response_text, response_message, model, metadata, created_at
		FROM conversations
		` + whereClause + `
		ORDER BY ` + orderBy + `
		LIMIT $` + fmt.Sprintf("%d", argCount) + ` OFFSET $` + fmt.Sprintf("%d", argCount+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list conversations: %w", err)
	}
	defer rows.Close()

	var conversations []*entities.Conversation
	for rows.Next() {
		conversation, err := r.scanConversation(rows)
		if err != nil {
			return nil, 0, err
		}
		conversations = append(conversations, conversation)
	}

	return conversations, total, nil
}

// scanConversation scans a conversation from a database row
func (r *PostgresConversationRepository) scanConversation(rows pgx.Rows) (*entities.Conversation, error) {
	var conversation entities.Conversation
	var messagesJSON, requestInputJSON, responseMessageJSON, metadataJSON []byte
	var createdAt time.Time

	err := rows.Scan(
		&conversation.ID,
		&conversation.NotebookID,
		&conversation.ResponseID,
		&conversation.PreviousResponseID,
		&messagesJSON,
		&requestInputJSON,
		&conversation.ResponseText,
		&responseMessageJSON,
		&conversation.Model,
		&metadataJSON,
		&createdAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan conversation: %w", err)
	}

	// Convert time.Time to Unix timestamp
	conversation.CreatedAt = createdAt.Unix()

	// Parse JSON fields
	if err := json.Unmarshal(messagesJSON, &conversation.Messages); err != nil {
		return nil, fmt.Errorf("failed to unmarshal messages: %w", err)
	}

	if requestInputJSON != nil {
		if err := json.Unmarshal(requestInputJSON, &conversation.RequestInput); err != nil {
			return nil, fmt.Errorf("failed to unmarshal request input: %w", err)
		}
	}

	if responseMessageJSON != nil {
		if err := json.Unmarshal(responseMessageJSON, &conversation.ResponseMessage); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response message: %w", err)
		}
	}

	if metadataJSON != nil {
		if err := json.Unmarshal(metadataJSON, &conversation.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &conversation, nil
}