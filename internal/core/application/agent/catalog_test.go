package agent

import (
	"context"
	"fmt"
	"testing"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/mocks/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBuildCatalog(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(repositories.MockSourceRepository)

	tests := []struct {
		name      string
		sourceIDs []uuid.UUID
		mockSetup func()
		expected  string
		wantErr   bool
	}{
		{
			name:      "empty source IDs",
			sourceIDs: []uuid.UUID{},
			mockSetup: func() {},
			expected:  "No sources available.",
			wantErr:   false,
		},
		{
			name:      "nil source IDs",
			sourceIDs: nil,
			mockSetup: func() {},
			expected:  "No sources available.",
			wantErr:   false,
		},
		{
			name:      "repository error",
			sourceIDs: []uuid.UUID{uuid.New()},
			mockSetup: func() {
				mockRepo.On("ListSourceSummariesByID", ctx, mock.Anything).Return(nil, fmt.Errorf("db error")).Once()
			},
			expected: "",
			wantErr:  true,
		},
		{
			name:      "no sources found in repository",
			sourceIDs: []uuid.UUID{uuid.New()},
			mockSetup: func() {
				mockRepo.On("ListSourceSummariesByID", ctx, mock.Anything).Return([]*entities.Source{}, nil).Once()
			},
			expected: "No sources available.",
			wantErr:  false,
		},
		{
			name:      "single source catalog",
			sourceIDs: []uuid.UUID{uuid.New()},
			mockSetup: func() {
				id := uuid.New()
				mockRepo.On("ListSourceSummariesByID", ctx, mock.Anything).Return([]*entities.Source{
					{ID: id, Title: "Test Source", Status: entities.SourceStatusCompleted},
				}, nil).Once()
			},
			expected: "Available Sources:\n- [completed] ID: %s, Title: Test Source\n",
			wantErr:  false,
		},
		{
			name:      "multiple sources catalog",
			sourceIDs: []uuid.UUID{uuid.New(), uuid.New()},
			mockSetup: func() {
				id1 := uuid.New()
				id2 := uuid.New()
				mockRepo.On("ListSourceSummariesByID", ctx, mock.Anything).Return([]*entities.Source{
					{ID: id1, Title: "Source 1", Status: entities.SourceStatusCompleted},
					{ID: id2, Title: "Source 2", Status: entities.SourceStatusProcessing},
				}, nil).Once()
			},
			expected: "Available Sources:\n- [completed] ID: %s, Title: Source 1\n- [processing] ID: %s, Title: Source 2\n",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			result, err := BuildCatalog(ctx, mockRepo, tt.sourceIDs)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if len(tt.sourceIDs) > 0 && tt.expected != "No sources available." {
					assert.Contains(t, result, "Available Sources:")
					assert.Contains(t, result, "ID:")
					assert.Contains(t, result, "Title:")
				} else {
					assert.Equal(t, tt.expected, result)
				}
			}
			mockRepo.AssertExpectations(t)
		})
	}
}
