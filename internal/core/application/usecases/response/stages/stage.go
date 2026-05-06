package stages

import "context"

// Stage defines the interface for a pipeline stage
type Stage[I any, O any] interface {
	Execute(ctx context.Context, input I) (O, error)
}
