package entities

import (
	"time"

	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// Document represents a document entity for indexing
// TODO: Implement document fields and methods based on requirements
type Document struct {
	ID        uuid.UUID
	Title     string
	Content   string
	Metadata  map[string]interface{}
	CreatedAt time.Time
	UpdatedAt time.Time
}
