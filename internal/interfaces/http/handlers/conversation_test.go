package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockConversationUseCase struct {
	mock.Mock
}

func (m *mockConversationUseCase) List(ctx context.Context, req *dtos.ListConversationsRequest) (*dtos.ListConversationsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) != nil {
		return args.Get(0).(*dtos.ListConversationsResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockConversationUseCase) GetMessages(ctx context.Context, req *dtos.GetMessagesRequest) (*dtos.GetMessagesResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) != nil {
		return args.Get(0).(*dtos.GetMessagesResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestConversationHandler_GetMessages(t *testing.T) {
	// Arrange
	mockUC := new(mockConversationUseCase)
	log := logger.New(logger.LevelInfo, "text")
	handler := NewConversationHandler(mockUC, log)

	req := httptest.NewRequest("GET", "/api/v1/notebooks/nb-123/conversations/conv-456/messages?limit=10&before_sequence=5", nil)

	// Use gorilla/mux to set path variables
	vars := map[string]string{
		"notebookId":     "nb-123",
		"conversationId": "conv-456",
	}
	req = mux.SetURLVars(req, vars)
	
	w := httptest.NewRecorder()

	expectedResp := &dtos.GetMessagesResponse{
		ConversationID: "conv-456",
		HasMore:        false,
		Messages: []dtos.MessageResponse{
			{ID: "msg-1", Messages: []*entities.StoredMessage{{Content: "Hello"}}},
		},
	}

	mockUC.On("GetMessages", mock.Anything, &dtos.GetMessagesRequest{
		NotebookID:     "nb-123",
		ConversationID: "conv-456",
		Limit:          10,
		BeforeSequence: intPtr(5),
	}).Return(expectedResp, nil)

	// Act
	handler.GetMessages(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	
	var resp dtos.GetMessagesResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, expectedResp.ConversationID, resp.ConversationID)
	assert.Len(t, resp.Messages, 1)

	mockUC.AssertExpectations(t)
}

func intPtr(i int) *int {
	return &i
}
