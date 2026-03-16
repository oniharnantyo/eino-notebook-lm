package entities

import (
	"time"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/valueobjects"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// ContentType represents the type of content in a source
type ContentType string

const (
	ContentTypePDF      ContentType = "application/pdf"
	ContentTypeDocx     ContentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	ContentTypeWebsite  ContentType = "text/html"
	ContentTypeText     ContentType = "text/plain"
	ContentTypeMarkdown ContentType = "text/markdown"
	ContentTypeAPI      ContentType = "application/json"
	ContentTypeOther    ContentType = "application/octet-stream"
)

// Source represents an ingestible asset that produces knowledge chunks
type Source struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	NotebookID  uuid.UUID              `json:"notebook_id" db:"notebook_id"`
	Title       string                 `json:"title" db:"title"`
	URI         string                 `json:"uri,omitempty" db:"uri"`
	ContentType ContentType            `json:"content_type" db:"content_type"`
	Content     string                 `json:"content,omitempty" db:"content"`
	ChunkCount  int                    `json:"chunk_count" db:"chunk_count"`
	TotalSize   int                    `json:"total_size,omitempty" db:"total_size"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
	DeletedAt   *time.Time             `json:"deleted_at,omitempty" db:"deleted_at"`
}

// NewSource creates a new source entity
func NewSource(notebookID uuid.UUID, title, uri string, contentType ContentType) (*Source, error) {
	source := &Source{
		ID:          uuid.New(),
		NotebookID:  notebookID,
		Title:       title,
		URI:         uri,
		ContentType: contentType,
		ChunkCount:  0,
		Metadata:    make(map[string]interface{}),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := source.Validate(); err != nil {
		return nil, err
	}

	return source, nil
}

// Validate validates the source entity
func (s *Source) Validate() error {
	if s.Title == "" {
		return valueobjects.ErrEmptyTitle
	}
	if len(s.Title) > 500 {
		return valueobjects.ErrTitleTooLong
	}
	if s.NotebookID.IsEmpty() {
		return valueobjects.ErrInvalidID
	}
	return nil
}

// SetContent sets the extracted content
func (s *Source) SetContent(content string, size int) {
	s.Content = content
	s.TotalSize = size
	s.UpdatedAt = time.Now()
}

// IncrementChunkCount increments the chunk counter
func (s *Source) IncrementChunkCount() {
	s.ChunkCount++
	s.UpdatedAt = time.Now()
}

// SetMetadata sets metadata for the source
func (s *Source) SetMetadata(key string, value interface{}) {
	if s.Metadata == nil {
		s.Metadata = make(map[string]interface{})
	}
	s.Metadata[key] = value
	s.UpdatedAt = time.Now()
}

// IsPDF checks if source is a PDF
func (s *Source) IsPDF() bool {
	return s.ContentType == ContentTypePDF
}

// IsWebsite checks if source is a website
func (s *Source) IsWebsite() bool {
	return s.ContentType == ContentTypeWebsite
}

// IsText checks if source is plain text
func (s *Source) IsText() bool {
	return s.ContentType == ContentTypeText
}

// HasContent checks if content has been extracted
func (s *Source) HasContent() bool {
	return s.Content != ""
}

// SoftDelete marks the source as deleted
func (s *Source) SoftDelete() {
	now := time.Now()
	s.DeletedAt = &now
	s.UpdatedAt = now
}