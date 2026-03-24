package unit_test

import (
	"errors"
	"testing"
	"time"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

func TestSource_NewSource(t *testing.T) {
	notebookID := uuid.New()

	source, err := entities.NewSource(notebookID, "Test Source", "https://example.com", entities.ContentTypeWebsite)

	if err != nil {
		t.Fatalf("failed to create source: %v", err)
	}

	if source == nil {
		t.Fatal("expected source to be returned")
	}

	if source.Title != "Test Source" {
		t.Errorf("expected title 'Test Source', got '%s'", source.Title)
	}

	if source.Status != entities.SourceStatusPending {
		t.Errorf("expected status '%s', got '%s'", entities.SourceStatusPending, source.Status)
	}

	if source.Error != nil {
		t.Errorf("expected error to be nil, got '%v'", source.Error)
	}

	if source.ChunkCount != 0 {
		t.Errorf("expected chunk count 0, got %d", source.ChunkCount)
	}

	if source.ID.IsEmpty() {
		t.Error("expected ID to be set")
	}

	if source.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}

	if source.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestSource_MarkProcessing(t *testing.T) {
	source := &entities.Source{
		ID:         uuid.New(),
		NotebookID: uuid.New(),
		Title:      "Test Source",
		Status:     entities.SourceStatusPending,
		Error:      stringPtr("previous error"),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	oldUpdatedAt := source.UpdatedAt
	time.Sleep(10 * time.Millisecond) // Ensure time difference

	source.MarkProcessing()

	if source.Status != entities.SourceStatusProcessing {
		t.Errorf("expected status '%s', got '%s'", entities.SourceStatusProcessing, source.Status)
	}

	if source.Error != nil {
		t.Errorf("expected error to be nil after MarkProcessing, got '%v'", source.Error)
	}

	if !source.UpdatedAt.After(oldUpdatedAt) {
		t.Error("expected UpdatedAt to be updated")
	}
}

func TestSource_MarkCompleted(t *testing.T) {
	source := &entities.Source{
		ID:         uuid.New(),
		NotebookID: uuid.New(),
		Title:      "Test Source",
		Status:     entities.SourceStatusProcessing,
		Error:      stringPtr("previous error"),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	oldUpdatedAt := source.UpdatedAt
	time.Sleep(10 * time.Millisecond) // Ensure time difference

	source.MarkCompleted()

	if source.Status != entities.SourceStatusCompleted {
		t.Errorf("expected status '%s', got '%s'", entities.SourceStatusCompleted, source.Status)
	}

	if source.Error != nil {
		t.Errorf("expected error to be nil after MarkCompleted, got '%v'", source.Error)
	}

	if !source.UpdatedAt.After(oldUpdatedAt) {
		t.Error("expected UpdatedAt to be updated")
	}
}

func TestSource_MarkFailed(t *testing.T) {
	source := &entities.Source{
		ID:         uuid.New(),
		NotebookID: uuid.New(),
		Title:      "Test Source",
		Status:     entities.SourceStatusProcessing,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	oldUpdatedAt := source.UpdatedAt
	time.Sleep(10 * time.Millisecond) // Ensure time difference
	testErr := errors.New("processing failed")

	source.MarkFailed(testErr)

	if source.Status != entities.SourceStatusFailed {
		t.Errorf("expected status '%s', got '%s'", entities.SourceStatusFailed, source.Status)
	}

	if source.Error == nil {
		t.Fatal("expected error to be set after MarkFailed")
	}

	if *source.Error != testErr.Error() {
		t.Errorf("expected error message '%s', got '%s'", testErr.Error(), *source.Error)
	}

	if !source.UpdatedAt.After(oldUpdatedAt) {
		t.Error("expected UpdatedAt to be updated")
	}
}

func TestSource_MarkFailed_NilError(t *testing.T) {
	source := &entities.Source{
		ID:         uuid.New(),
		NotebookID: uuid.New(),
		Title:      "Test Source",
		Status:     entities.SourceStatusProcessing,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	source.MarkFailed(nil)

	if source.Status != entities.SourceStatusFailed {
		t.Errorf("expected status '%s', got '%s'", entities.SourceStatusFailed, source.Status)
	}

	if source.Error != nil {
		t.Errorf("expected error to be nil when nil error passed, got '%v'", source.Error)
	}
}

func TestSource_StatusTransitions(t *testing.T) {
	source := &entities.Source{
		ID:         uuid.New(),
		NotebookID: uuid.New(),
		Title:      "Test Source",
		Status:     entities.SourceStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Test normal flow: pending -> processing -> completed
	if source.Status != entities.SourceStatusPending {
		t.Errorf("expected initial status '%s', got '%s'", entities.SourceStatusPending, source.Status)
	}

	source.MarkProcessing()
	if source.Status != entities.SourceStatusProcessing {
		t.Errorf("expected status '%s' after MarkProcessing, got '%s'", entities.SourceStatusProcessing, source.Status)
	}

	source.MarkCompleted()
	if source.Status != entities.SourceStatusCompleted {
		t.Errorf("expected status '%s' after MarkCompleted, got '%s'", entities.SourceStatusCompleted, source.Status)
	}
}

func TestSource_StatusTransitions_WithFailure(t *testing.T) {
	source := &entities.Source{
		ID:         uuid.New(),
		NotebookID: uuid.New(),
		Title:      "Test Source",
		Status:     entities.SourceStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Test failure flow: pending -> processing -> failed
	source.MarkProcessing()
	if source.Status != entities.SourceStatusProcessing {
		t.Errorf("expected status '%s' after MarkProcessing, got '%s'", entities.SourceStatusProcessing, source.Status)
	}

	source.MarkFailed(errors.New("test error"))
	if source.Status != entities.SourceStatusFailed {
		t.Errorf("expected status '%s' after MarkFailed, got '%s'", entities.SourceStatusFailed, source.Status)
	}

	if source.Error == nil {
		t.Fatal("expected error to be set")
	}

	// Verify error is cleared on retry
	source.MarkProcessing()
	if source.Error != nil {
		t.Errorf("expected error to be cleared on retry, got '%v'", source.Error)
	}
}

func TestSource_Validate(t *testing.T) {
	tests := []struct {
		name    string
		source  *entities.Source
		wantErr bool
	}{
		{
			name: "valid source",
			source: &entities.Source{
				ID:         uuid.New(),
				NotebookID: uuid.New(),
				Title:      "Valid Source",
				Status:     entities.SourceStatusPending,
			},
			wantErr: false,
		},
		{
			name: "empty title",
			source: &entities.Source{
				ID:         uuid.New(),
				NotebookID: uuid.New(),
				Title:      "",
				Status:     entities.SourceStatusPending,
			},
			wantErr: true,
		},
		{
			name: "title too long",
			source: &entities.Source{
				ID:         uuid.New(),
				NotebookID: uuid.New(),
				Title:      string(make([]byte, 501)), // 501 characters
				Status:     entities.SourceStatusPending,
			},
			wantErr: true,
		},
		{
			name: "invalid notebook ID",
			source: &entities.Source{
				ID:         uuid.New(),
				NotebookID: "", // empty UUID
				Title:      "Valid Source",
				Status:     entities.SourceStatusPending,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.source.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
