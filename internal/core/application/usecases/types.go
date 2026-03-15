package usecases

import "io"

// ContentType represents the type of content source
// Open/Closed Principle: Enum is closed for modification but new types can be added
type ContentType string

const (
	ContentTypeFile  ContentType = "file"
	ContentTypeURL   ContentType = "url"
	ContentTypeText  ContentType = "text"
	ContentTypeOther ContentType = "other"
)

// ContentSource represents a source of content
type ContentSource struct {
	Type     ContentType
	Reader   io.Reader
	Filename string
	URL      string
	Text     string
	Metadata map[string]interface{}
}
