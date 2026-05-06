package response

import (
	"strings"

	"github.com/cloudwego/eino/schema"
)

// historySavingReader wraps a StreamReader and triggers a save callback when the stream ends.
type historySavingReader struct {
	inner  *schema.StreamReader[*schema.Message]
	onSave func(accumulatedText string)
}

// NewHistorySavingReader creates a new historySavingReader.
func NewHistorySavingReader(inner *schema.StreamReader[*schema.Message], onSave func(string)) *historySavingReader {
	return &historySavingReader{
		inner:  inner,
		onSave: onSave,
	}
}

// Pipe returns a new StreamReader that forwards messages from the inner reader 
// and triggers the onSave callback when the stream is finished or closed.
func (r *historySavingReader) Pipe() *schema.StreamReader[*schema.Message] {
	sr, sw := schema.Pipe[*schema.Message](10)
	
	go func() {
		defer sw.Close()
		defer r.inner.Close()

		var accumulated strings.Builder
		for {
			msg, err := r.inner.Recv()
			if msg != nil {
				accumulated.WriteString(msg.Content)
				_ = sw.Send(msg, nil)
			}
			if err != nil {
				r.onSave(accumulated.String())
				_ = sw.Send(nil, err)
				break
			}
		}
	}()

	return sr
}
