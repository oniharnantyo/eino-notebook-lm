package entities

import (
	"time"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/valueobjects"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// Notebook represents the core domain entity
type Notebook struct {
	ID          uuid.UUID
	Title       string
	Description string
	Content     string
	Status      valueobjects.NotebookStatus
	Tags        []string
	Metadata    map[string]interface{}
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// NewNotebook creates a new notebook entity
func NewNotebook(title, description, content string, tags []string) (*Notebook, error) {
	notebook := &Notebook{
		ID:          uuid.New(),
		Title:       title,
		Description: description,
		Content:     content,
		Status:      valueobjects.StatusActive,
		Tags:        tags,
		Metadata:    make(map[string]interface{}),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := notebook.Validate(); err != nil {
		return nil, err
	}

	return notebook, nil
}

// Validate validates the notebook entity
func (n *Notebook) Validate() error {
	if n.Title == "" {
		return valueobjects.ErrEmptyTitle
	}
	if len(n.Title) > 200 {
		return valueobjects.ErrTitleTooLong
	}
	return nil
}

// Update updates the notebook fields
func (n *Notebook) Update(title, description, content string, tags []string) error {
	n.Title = title
	n.Description = description
	n.Content = content
	n.Tags = tags
	n.UpdatedAt = time.Now()

	return n.Validate()
}

// Archive marks the notebook as archived
func (n *Notebook) Archive() {
	n.Status = valueobjects.StatusArchived
	n.UpdatedAt = time.Now()
}

// Activate marks the notebook as active
func (n *Notebook) Activate() {
	n.Status = valueobjects.StatusActive
	n.UpdatedAt = time.Now()
}

// SoftDelete marks the notebook as deleted
func (n *Notebook) SoftDelete() {
	now := time.Now()
	n.DeletedAt = &now
	n.Status = valueobjects.StatusDeleted
	n.UpdatedAt = now
}

// AddMetadata adds metadata to the notebook
func (n *Notebook) AddMetadata(key string, value interface{}) {
	if n.Metadata == nil {
		n.Metadata = make(map[string]interface{})
	}
	n.Metadata[key] = value
	n.UpdatedAt = time.Now()
}

// IsDeleted checks if the notebook is deleted
func (n *Notebook) IsDeleted() bool {
	return n.DeletedAt != nil || n.Status == valueobjects.StatusDeleted
}
