package sse

type StreamMeta struct {
	ResponseID         string
	ModelName          string
	CreatedAt          int64
	Instructions       *string
	PreviousResponseID *string
	MaxOutputTokens    *int
	Temperature        *float64
	MaxToolCalls       *int
	Metadata           map[string]string
}
