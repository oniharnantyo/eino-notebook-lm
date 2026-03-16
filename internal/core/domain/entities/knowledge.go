package entities

import (
	"time"

	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// KnowledgeSource represents the source type of knowledge
type KnowledgeSource string

const (
	SourceDocument KnowledgeSource = "document"
	SourceWebsite  KnowledgeSource = "website"
	SourceText     KnowledgeSource = "text"
	SourceAPI      KnowledgeSource = "api"
	SourceOther    KnowledgeSource = "other"
)

// Knowledge represents a knowledge entity for indexing
// Knowledge can come from various sources: documents, websites, APIs, etc.
type Knowledge struct {
	KnowledgeID uuid.UUID        `json:"knowledge_id" db:"knowledge_id"`
	SourceID    uuid.UUID        `json:"source_id" db:"source_id"`
	Title       string           `json:"title" db:"title"`
	Content     string           `json:"content" db:"content"`
	SourceType  KnowledgeSource  `json:"source_type" db:"source_type"`
	Metadata    map[string]any   `json:"metadata,omitempty" db:"metadata"`
	SubIndexes  []string         `json:"sub_indexes,omitempty" db:"sub_indexes"`
	CreatedAt   time.Time        `json:"created_at" db:"created_at"`
}

// NewKnowledge creates a new knowledge entity with a source reference
func NewKnowledge(sourceID uuid.UUID, title, content string, sourceType KnowledgeSource, metadata map[string]any) (*Knowledge, error) {
	knowledge := &Knowledge{
		KnowledgeID: uuid.New(),
		SourceID:    sourceID,
		Title:       title,
		Content:     content,
		SourceType:  sourceType,
		Metadata:    metadata,
		SubIndexes:  []string{},
		CreatedAt:   time.Now(),
	}

	// Set default source type if not provided
	if knowledge.SourceType == "" {
		knowledge.SourceType = SourceDocument
	}

	return knowledge, nil
}

// IsDocument checks if knowledge is from a document
func (k *Knowledge) IsDocument() bool {
	return k.SourceType == SourceDocument
}

// IsWebsite checks if knowledge is from a website
func (k *Knowledge) IsWebsite() bool {
	return k.SourceType == SourceWebsite
}

// IsText checks if knowledge is from text input
func (k *Knowledge) IsText() bool {
	return k.SourceType == SourceText
}

// IsAPI checks if knowledge is from an API
func (k *Knowledge) IsAPI() bool {
	return k.SourceType == SourceAPI
}

// AddSubIndex adds a sub-index to the knowledge
func (k *Knowledge) AddSubIndex(index string) {
	for _, idx := range k.SubIndexes {
		if idx == index {
			return // Already exists
		}
	}
	k.SubIndexes = append(k.SubIndexes, index)
}

// RemoveSubIndex removes a sub-index from the knowledge
func (k *Knowledge) RemoveSubIndex(index string) {
	for i, idx := range k.SubIndexes {
		if idx == index {
			k.SubIndexes = append(k.SubIndexes[:i], k.SubIndexes[i+1:]...)
			return
		}
	}
}
