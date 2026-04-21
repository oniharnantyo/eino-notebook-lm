package templates

import (
	"fmt"
	"sync"
	"text/template"
)

// PromptTemplate holds template information for embedding requests
type PromptTemplate struct {
	Name        string
	Description string
	Template    string   // Go template syntax
	Variables   []string // Required variables
}

var (
	registry = make(map[string]*PromptTemplate)
	mu       sync.RWMutex
)

// Register registers a new prompt template
func Register(tmpl *PromptTemplate) {
	mu.Lock()
	defer mu.Unlock()
	registry[tmpl.Name] = tmpl
}

// GetTemplate returns a template by name or the default template if not found
func GetTemplate(name string) *PromptTemplate {
	mu.RLock()
	defer mu.RUnlock()
	if tmpl, ok := registry[name]; ok {
		return tmpl
	}
	// Fallback to default
	return registry["default"]
}

// ValidateTemplate validates that the template string is a valid Go template
func ValidateTemplate(tmplStr string) error {
	_, err := template.New("validate").Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("invalid template syntax: %w", err)
	}
	return nil
}
