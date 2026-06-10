package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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

// Save saves a conversation (create or update) with its messages
func (r *PostgresConversationRepository) Save(ctx context.Context, conversation *entities.Conversation, messages []*entities.Message) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert or update conversation
	convQuery := `
		INSERT INTO conversations (id, notebook_id, metadata, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET
			notebook_id = EXCLUDED.notebook_id,
			metadata = EXCLUDED.metadata
	`
	_, err = tx.Exec(ctx, convQuery,
		conversation.ID,
		conversation.NotebookID,
		conversation.Metadata,
		time.Unix(conversation.CreatedAt, 0),
	)
	if err != nil {
		return fmt.Errorf("failed to save conversation: %w", err)
	}

	// Insert messages
	msgQuery := `
		INSERT INTO messages (id, conversation_id, sequence_num, response_id, previous_response_id, messages, model, finish_reason, prompt_tokens, completion_tokens, total_tokens, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	for _, msg := range messages {
		msgContent, err := json.Marshal(msg.Messages)
		if err != nil {
			return fmt.Errorf("failed to marshal message content: %w", err)
		}

		_, err = tx.Exec(ctx, msgQuery,
			msg.ID,
			msg.ConversationID,
			msg.SequenceNum,
			msg.ResponseID,
			msg.PreviousResponseID,
			msgContent,
			msg.Model,
			msg.FinishReason,
			msg.PromptTokens,
			msg.CompletionTokens,
			msg.TotalTokens,
			msg.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to save message: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// GetMessages retrieves messages for a conversation with pagination and optional chronological order
func (r *PostgresConversationRepository) GetMessages(ctx context.Context, conversationID string, limit int, beforeSequence *int, isConversationHistory *bool) ([]*entities.Message, error) {
	innerQuery := `
		SELECT id, conversation_id, sequence_num, response_id, previous_response_id, messages, model, finish_reason, prompt_tokens, completion_tokens, total_tokens, created_at
		FROM messages
		WHERE conversation_id = $1
	`
	args := []interface{}{conversationID}

	if beforeSequence != nil {
		innerQuery += fmt.Sprintf(" AND sequence_num < $%d", len(args)+1)
		args = append(args, *beforeSequence)
	}

	innerQuery += fmt.Sprintf(" ORDER BY sequence_num DESC LIMIT $%d", len(args)+1)
	args = append(args, limit)

	query := innerQuery
	if isConversationHistory != nil && *isConversationHistory {
		query = fmt.Sprintf(`
			SELECT id, conversation_id, sequence_num, response_id, previous_response_id, messages, model, finish_reason, prompt_tokens, completion_tokens, total_tokens, created_at
			FROM (%s) sub
			ORDER BY sequence_num ASC
		`, innerQuery)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []*entities.Message
	for rows.Next() {
		var msg entities.Message
		var msgContent []byte
		err := rows.Scan(
			&msg.ID,
			&msg.ConversationID,
			&msg.SequenceNum,
			&msg.ResponseID,
			&msg.PreviousResponseID,
			&msgContent,
			&msg.Model,
			&msg.FinishReason,
			&msg.PromptTokens,
			&msg.CompletionTokens,
			&msg.TotalTokens,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		
		if err := json.Unmarshal(msgContent, &msg.Messages); err != nil {
			return nil, fmt.Errorf("failed to unmarshal message content: %w", err)
		}
		
		messages = append(messages, &msg)
	}

	return messages, nil
}

// GetLatestConversationID retrieves the ID of the latest conversation for a notebook
func (r *PostgresConversationRepository) GetLatestConversationID(ctx context.Context, notebookID string) (string, error) {
	query := `
		SELECT id
		FROM conversations
		WHERE notebook_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`
	var id string
	err := r.pool.QueryRow(ctx, query, notebookID).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("failed to get latest conversation ID: %w", err)
	}
	return id, nil
}

// FindByID finds a conversation by its ID
func (r *PostgresConversationRepository) FindByID(ctx context.Context, id string) (*entities.Conversation, error) {
	query := `
		SELECT id, notebook_id, metadata, created_at
		FROM conversations
		WHERE id = $1
	`

	var conversation entities.Conversation
	var metadataJSON []byte
	var createdAt time.Time

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&conversation.ID,
		&conversation.NotebookID,
		&metadataJSON,
		&createdAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find conversation by ID: %w", err)
	}

	conversation.CreatedAt = createdAt.Unix()

	if err := json.Unmarshal(metadataJSON, &conversation.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &conversation, nil
}

// FindByResponseID finds a conversation by its response ID
func (r *PostgresConversationRepository) FindByResponseID(ctx context.Context, responseID string) (*entities.Conversation, error) {
	// Note: This method needs adjustment to search by message.response_id
	query := `
		SELECT c.id, c.notebook_id, c.metadata, c.created_at
		FROM conversations c
		JOIN messages m ON c.id = m.conversation_id
		WHERE m.response_id = $1
		LIMIT 1
	`

	var conversation entities.Conversation
	var metadataJSON []byte
	var createdAt time.Time

	err := r.pool.QueryRow(ctx, query, responseID).Scan(
		&conversation.ID,
		&conversation.NotebookID,
		&metadataJSON,
		&createdAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find conversation by response ID: %w", err)
	}

	conversation.CreatedAt = createdAt.Unix()
	
	if err := json.Unmarshal(metadataJSON, &conversation.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &conversation, nil
}

// Delete deletes a conversation by response ID
func (r *PostgresConversationRepository) Delete(ctx context.Context, responseID string) error {
	// This should delete the conversation and cascaded messages
	query := `
		DELETE FROM conversations
		WHERE id = (SELECT conversation_id FROM messages WHERE response_id = $1 LIMIT 1)
	`
	_, err := r.pool.Exec(ctx, query, responseID)
	if err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}

	return nil
}

// Exists checks if a conversation exists for a response ID
func (r *PostgresConversationRepository) Exists(ctx context.Context, responseID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM messages WHERE response_id = $1)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, responseID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check conversation existence: %w", err)
	}

	return exists, nil
}

// List retrieves conversations with pagination and optional filters
func (r *PostgresConversationRepository) List(ctx context.Context, filter repositories.ConversationFilter) ([]*entities.Conversation, int, error) {
	query := `SELECT c.id, c.notebook_id, c.metadata, c.created_at FROM conversations c WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	// If filtering by message attributes, we need a JOIN
	if filter.Model != nil {
		query = `SELECT DISTINCT c.id, c.notebook_id, c.metadata, c.created_at FROM conversations c JOIN messages m ON c.id = m.conversation_id WHERE 1=1`
		if filter.Model != nil {
			query += fmt.Sprintf(" AND m.model = $%d", argIdx)
			args = append(args, *filter.Model)
			argIdx++
		}
	}

	if filter.NotebookID != nil {
		query += fmt.Sprintf(" AND c.notebook_id = $%d", argIdx)
		args = append(args, *filter.NotebookID)
		argIdx++
	}

	countQuery := "SELECT count(*) FROM (" + query + ") AS count_table"
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count conversations: %w", err)
	}

	if filter.OrderBy != "" {
		// Add table alias if not present to avoid ambiguity in JOIN
		orderBy := filter.OrderBy
		if !strings.Contains(orderBy, ".") {
			orderBy = "c." + orderBy
		}
		query += " ORDER BY " + orderBy
	} else {
		query += " ORDER BY c.created_at DESC"
	}

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, filter.Limit)
		argIdx++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, filter.Offset)
		argIdx++
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query conversations: %w", err)
	}
	defer rows.Close()

	var conversations []*entities.Conversation
	for rows.Next() {
		var conv entities.Conversation
		var metadataJSON []byte
		var createdAt time.Time
		if err := rows.Scan(&conv.ID, &conv.NotebookID, &metadataJSON, &createdAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan conversation: %w", err)
		}
		
		if metadataJSON != nil {
			if err := json.Unmarshal(metadataJSON, &conv.Metadata); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		} else {
			conv.Metadata = make(map[string]string)
		}
		conv.CreatedAt = createdAt.Unix()

		conversations = append(conversations, &conv)
	}

	return conversations, total, rows.Err()
}
