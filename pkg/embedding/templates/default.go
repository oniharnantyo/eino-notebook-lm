package templates

func init() {
	Register(DefaultTemplate)
	Register(WithInstructionTemplate)
	Register(VisionTemplate)
}

// DefaultTemplate is the standard template for llama.cpp embeddings
var DefaultTemplate = &PromptTemplate{
	Name:        "default",
	Description: "Standard llama.cpp embedding template",
	Template:    `{"prompt_string": "{{.Text}}", "multimodal_data": []}`,
	Variables:   []string{".Text"},
}

// WithInstructionTemplate is a template with a specific instruction prefix
var WithInstructionTemplate = &PromptTemplate{
	Name:        "with-instruction",
	Description: "Embedding template with instruction prefix",
	Template:    `{"prompt_string": "Embed this text: {{.Text}}", "multimodal_data": []}`,
	Variables:   []string{".Text"},
}

// VisionTemplate is a template that includes image data (multimodal placeholder)
var VisionTemplate = &PromptTemplate{
	Name:        "vision",
	Description: "Embedding template with multimodal data",
	Template:    `{"prompt_string": "{{.Text}}", "multimodal_data": [{{range $i, $m := .Media}}{{if $i}},{{end}}{"data": "{{$m.Data}}", "mime_type": "{{$m.MimeType}}"}{{end}}]}`,
	Variables:   []string{".Text", ".Media"},
}
