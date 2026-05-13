package model

// Shared types for OpenAI-compatible vision API implementations

type chatCompletionContent struct {
	Type     string               `json:"type"`
	Text     string               `json:"text,omitempty"`
	ImageURL *chatCompletionImage `json:"image_url,omitempty"`
}

type chatCompletionImage struct {
	URL string `json:"url"`
}

type chatCompletionMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type chatCompletionRequest struct {
	Model    string                  `json:"model"`
	Messages []chatCompletionMessage `json:"messages"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}
