package dtos

import (
	"time"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// ArtifactResponse represents an artifact response
type ArtifactResponse struct {
	ID         uuid.UUID              `json:"id"`
	NotebookID uuid.UUID              `json:"notebook_id"`
	Title      string                 `json:"title"`
	Type       string                 `json:"type"`
	Status     string                 `json:"status"`
	Format     string                 `json:"format"`
	Content    string                 `json:"content,omitempty"`
	SourceIDs  []uuid.UUID            `json:"source_ids,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Error      *string                `json:"error,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// ArtifactListResponse represents a lightweight artifact for list responses
type ArtifactListResponse struct {
	ID         uuid.UUID `json:"id"`
	NotebookID uuid.UUID `json:"notebook_id"`
	Title      string    `json:"title"`
	Type       string    `json:"type"`
	Status     string    `json:"status"`
	Format     string    `json:"format"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ListArtifactsRequest represents a request to list artifacts
type ListArtifactsRequest struct {
	NotebookID uuid.UUID `json:"notebook_id" validate:"required"`
	Type       string    `json:"type" validate:"omitempty,oneof=mindmap podcast slides"`
	Status     string    `json:"status" validate:"omitempty,oneof=pending processing completed failed"`
	Page       int       `json:"page" validate:"min=1"`
	Limit      int       `json:"limit" validate:"min=1,max=100"`
}

// ListArtifactsResponse represents a paginated list of artifacts
type ListArtifactsResponse struct {
	Artifacts  []ArtifactListResponse `json:"artifacts"`
	Total      int64                  `json:"total"`
	Page       int                    `json:"page"`
	Limit      int                    `json:"limit"`
	TotalPages int                    `json:"total_pages"`
}

// TriggerMindmapRequest represents a request to trigger mindmap generation
type TriggerMindmapRequest struct {
	NotebookID uuid.UUID   `json:"notebook_id" validate:"required"`
	SourceIDs  []uuid.UUID `json:"source_ids" validate:"required,min=1"`
	Title      string      `json:"title" validate:"omitempty,min=1,max=500"`
}

// TriggerMindmapResponse represents the response from triggering mindmap generation
type TriggerMindmapResponse struct {
	ArtifactID uuid.UUID `json:"artifact_id"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

// ParseArtifactType parses a string to ArtifactType
func ParseArtifactType(artifactType string) entities.ArtifactType {
	switch artifactType {
	case "mindmap":
		return entities.ArtifactTypeMindmap
	case "podcast":
		return entities.ArtifactTypePodcast
	case "slides":
		return entities.ArtifactTypeSlides
	default:
		return entities.ArtifactTypeMindmap // default
	}
}

// ParseArtifactStatus parses a string to ArtifactStatus
func ParseArtifactStatus(status string) entities.ArtifactStatus {
	switch status {
	case "pending":
		return entities.ArtifactStatusPending
	case "processing":
		return entities.ArtifactStatusProcessing
	case "completed":
		return entities.ArtifactStatusCompleted
	case "failed":
		return entities.ArtifactStatusFailed
	default:
		return entities.ArtifactStatusPending // default
	}
}
