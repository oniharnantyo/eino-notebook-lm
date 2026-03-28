package kreuzberg

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestKreuzbergParser_Parse_Metadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Errorf("expected multipart/form-data content type, got %s", r.Header.Get("Content-Type"))
		}

		response := []KreuzbergExtractResponse{
			{
				Content: "Title\nBody text",
				Metadata: map[string]interface{}{
					"mime_type": "application/pdf",
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	ctx := context.Background()
	kp, err := NewKreuzbergParser(&Config{
		ServiceURL: server.URL,
	})
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	reader := strings.NewReader("dummy content")
	docs, err := kp.Parse(ctx, reader)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if len(docs) != 1 {
		t.Fatalf("expected 1 document, got %d", len(docs))
	}

	doc := docs[0]
	if doc.Content != "Title\nBody text" {
		t.Errorf("expected content 'Title\\nBody text', got %q", doc.Content)
	}

	mime, ok := doc.MetaData["mime_type"].(string)
	if !ok || mime != "application/pdf" {
		t.Errorf("expected mime_type 'application/pdf', got %v", doc.MetaData["mime_type"])
	}
}
