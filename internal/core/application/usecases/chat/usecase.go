package chat

import (
	"context"

	"github.com/cloudwego/eino/schema"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/interfaces/http/sse"
)

// ResponseUseCase defines the interface for Responses API operations
type ResponseUseCase interface {
	Stream(ctx context.Context, req *dtos.ResponseRequest) (*schema.StreamReader[*schema.Message], *sse.StreamMeta, error)
}
