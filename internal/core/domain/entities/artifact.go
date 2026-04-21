package entities

import (
	"time"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/valueobjects"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// ArtifactType represents the type of artifact
type ArtifactType string

const (
	ArtifactTypeMindmap ArtifactType = "mindmap"
	ArtifactTypePodcast ArtifactType = "podcast"
	ArtifactTypeSlides  ArtifactType = "slides"
)

// ArtifactStatus represents the generation status of an artifact
type ArtifactStatus string

const (
	ArtifactStatusPending    ArtifactStatus = "pending"
	ArtifactStatusProcessing ArtifactStatus = "processing"
	ArtifactStatusCompleted  ArtifactStatus = "completed"
	ArtifactStatusFailed     ArtifactStatus = "failed"
)

// ArtifactFormat represents the output format of an artifact
type ArtifactFormat string

const (
	ArtifactFormatJSON ArtifactFormat = "json"
	ArtifactFormatPNG  ArtifactFormat = "png"
	ArtifactFormatSVG  ArtifactFormat = "svg"
	ArtifactFormatText ArtifactFormat = "text"
)

// Artifact represents a generated output from knowledge sources
type Artifact struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	NotebookID  uuid.UUID              `json:"notebook_id" db:"notebook_id"`
	Title       string                 `json:"title" db:"title"`
	Type        ArtifactType           `json:"type" db:"type"`
	Status      ArtifactStatus         `json:"status" db:"status"`
	Format      ArtifactFormat         `json:"format" db:"format"`
	Content     string                 `json:"content,omitempty" db:"content"`
	SourceIDs   []uuid.UUID            `json:"source_ids,omitempty" db:"source_ids"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	Error       *string                `json:"error,omitempty" db:"error"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
	DeletedAt   *time.Time             `json:"deleted_at,omitempty" db:"deleted_at"`
}

// NewArtifact creates a new artifact entity
func NewArtifact(notebookID uuid.UUID, title string, artifactType ArtifactType, format ArtifactFormat) (*Artifact, error) {
	artifact := &Artifact{
		ID:          uuid.New(),
		NotebookID:  notebookID,
		Title:       title,
		Type:        artifactType,
		Status:      ArtifactStatusPending,
		Format:      format,
		SourceIDs:   make([]uuid.UUID, 0),
		Metadata:    make(map[string]interface{}),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := artifact.Validate(); err != nil {
		return nil, err
	}

	return artifact, nil
}

// Validate validates the artifact entity
func (a *Artifact) Validate() error {
	if a.Title == "" {
		return valueobjects.ErrEmptyTitle
	}
	if len(a.Title) > 500 {
		return valueobjects.ErrTitleTooLong
	}
	if a.NotebookID.IsEmpty() {
		return valueobjects.ErrInvalidID
	}
	if !a.Type.IsValid() {
		return valueobjects.ErrInvalidStatus
	}
	if !a.Status.IsValid() {
		return valueobjects.ErrInvalidStatus
	}
	if !a.Format.IsValid() {
		return valueobjects.ErrInvalidStatus
	}
	return nil
}

// SetContent sets the generated content
func (a *Artifact) SetContent(content string) {
	a.Content = content
	a.UpdatedAt = time.Now()
}

// AddSourceID adds a source ID to the artifact
func (a *Artifact) AddSourceID(sourceID uuid.UUID) {
	a.SourceIDs = append(a.SourceIDs, sourceID)
	a.UpdatedAt = time.Now()
}

// SetSourceIDs sets all source IDs for the artifact
func (a *Artifact) SetSourceIDs(sourceIDs []uuid.UUID) {
	a.SourceIDs = sourceIDs
	a.UpdatedAt = time.Now()
}

// SetMetadata sets metadata for the artifact
func (a *Artifact) SetMetadata(key string, value interface{}) {
	if a.Metadata == nil {
		a.Metadata = make(map[string]interface{})
	}
	a.Metadata[key] = value
	a.UpdatedAt = time.Now()
}

// SoftDelete marks the artifact as deleted
func (a *Artifact) SoftDelete() {
	now := time.Now()
	a.DeletedAt = &now
	a.UpdatedAt = now
}

// MarkProcessing marks the artifact as being processed
func (a *Artifact) MarkProcessing() {
	a.Status = ArtifactStatusProcessing
	a.Error = nil
	a.UpdatedAt = time.Now()
}

// MarkCompleted marks the artifact as successfully completed
func (a *Artifact) MarkCompleted() {
	a.Status = ArtifactStatusCompleted
	a.Error = nil
	a.UpdatedAt = time.Now()
}

// MarkFailed marks the artifact as failed with an error message
func (a *Artifact) MarkFailed(err error) {
	a.Status = ArtifactStatusFailed
	if err != nil {
		errMsg := err.Error()
		a.Error = &errMsg
	}
	a.UpdatedAt = time.Now()
}

// IsMindmap checks if artifact is a mindmap
func (a *Artifact) IsMindmap() bool {
	return a.Type == ArtifactTypeMindmap
}

// IsPodcast checks if artifact is a podcast
func (a *Artifact) IsPodcast() bool {
	return a.Type == ArtifactTypePodcast
}

// IsSlides checks if artifact is slides
func (a *Artifact) IsSlides() bool {
	return a.Type == ArtifactTypeSlides
}

// HasContent checks if content has been generated
func (a *Artifact) HasContent() bool {
	return a.Content != ""
}

// IsCompleted checks if artifact generation is completed
func (a *Artifact) IsCompleted() bool {
	return a.Status == ArtifactStatusCompleted
}

// IsFailed checks if artifact generation failed
func (a *Artifact) IsFailed() bool {
	return a.Status == ArtifactStatusFailed
}

// IsValid checks if the artifact type is valid
func (t ArtifactType) IsValid() bool {
	switch t {
	case ArtifactTypeMindmap, ArtifactTypePodcast, ArtifactTypeSlides:
		return true
	default:
		return false
	}
}

// IsValid checks if the artifact status is valid
func (s ArtifactStatus) IsValid() bool {
	switch s {
	case ArtifactStatusPending, ArtifactStatusProcessing, ArtifactStatusCompleted, ArtifactStatusFailed:
		return true
	default:
		return false
	}
}

// IsValid checks if the artifact format is valid
func (f ArtifactFormat) IsValid() bool {
	switch f {
	case ArtifactFormatJSON, ArtifactFormatPNG, ArtifactFormatSVG, ArtifactFormatText:
		return true
	default:
		return false
	}
}

// String returns the string representation of the artifact type
func (t ArtifactType) String() string {
	return string(t)
}

// String returns the string representation of the artifact status
func (s ArtifactStatus) String() string {
	return string(s)
}

// String returns the string representation of the artifact format
func (f ArtifactFormat) String() string {
	return string(f)
}
