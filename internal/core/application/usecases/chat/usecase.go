package chat

import (
	"context"
	"io"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
)

// ResponseUseCase defines the interface for Responses API operations
type ResponseUseCase interface {
	// CreateResponse generates a non-streaming response
	CreateResponse(ctx context.Context, req *dtos.ResponseRequest) (*dtos.ResponseResource, error)

	// CreateResponseStream generates a streaming response
	CreateResponseStream(ctx context.Context, req *dtos.ResponseRequest) (io.ReadCloser, error)
}
