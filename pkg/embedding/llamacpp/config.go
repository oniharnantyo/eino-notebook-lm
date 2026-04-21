package llamacpp

import "time"

// Config holds configuration for the llama.cpp embedder
type Config struct {
	BaseURL        string
	APIKey         string
	Model          string
	Dimension      int
	PromptTemplate string
	Timeout        time.Duration
}
