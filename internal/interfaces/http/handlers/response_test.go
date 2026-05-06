package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/interfaces/http/sse"
	"github.com/oniharnantyo/eino-notebook/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockResponseUseCase struct {
	mock.Mock
}

func (m *mockResponseUseCase) Stream(ctx context.Context, req *dtos.ResponseRequest) (*schema.StreamReader[*schema.Message], *sse.StreamMeta, error) {
	args := m.Called(ctx, req)
	var sr *schema.StreamReader[*schema.Message]
	if args.Get(0) != nil {
		sr = args.Get(0).(*schema.StreamReader[*schema.Message])
	}
	var meta *sse.StreamMeta
	if args.Get(1) != nil {
		meta = args.Get(1).(*sse.StreamMeta)
	}
	return sr, meta, args.Error(2)
}

func TestResponseHandler_CreateResponse_Success(t *testing.T) {
	// Arrange
	mockUC := new(mockResponseUseCase)
	log := logger.NewWithWriter(io.Discard, logger.LevelInfo, "text")
	handler := NewResponseHandler(mockUC, log)

	reqBody := dtos.ResponseRequest{
		Input: "Hello",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/responses", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	// Prepare mock stream
	pr, pw := schema.Pipe[*schema.Message](10)
	go func() {
		_ = pw.Send(&schema.Message{Content: "Hi"}, nil)
		pw.Close()
	}()

	meta := &sse.StreamMeta{
		ResponseID: "resp_1",
	}

	mockUC.On("Stream", mock.Anything, mock.Anything).Return(pr, meta, nil)

	// Act
	handler.CreateResponse(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "response.created")
	assert.Contains(t, w.Body.String(), "response.completed")
	assert.Contains(t, w.Body.String(), "Hi")
}

func TestResponseHandler_CreateResponse_ValidationError(t *testing.T) {
	// Arrange
	mockUC := new(mockResponseUseCase)
	log := logger.NewWithWriter(io.Discard, logger.LevelInfo, "text")
	handler := NewResponseHandler(mockUC, log)

	// Missing input
	reqBody := dtos.ResponseRequest{}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/responses", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	// Act
	handler.CreateResponse(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	
	var errResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.NotNil(t, errResp["error"])
}
