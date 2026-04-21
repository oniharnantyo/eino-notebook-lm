package templates

import (
	"testing"
)

func TestValidateTemplate(t *testing.T) {
	t.Run("ValidTemplate", func(t *testing.T) {
		tmpl := `{"prompt_string": "{{.Text}}"}`
		if err := ValidateTemplate(tmpl); err != nil {
			t.Errorf("expected no error for valid template, got: %v", err)
		}
	})

	t.Run("InvalidTemplate", func(t *testing.T) {
		tmpl := `{"prompt_string": "{{.Text"`
		if err := ValidateTemplate(tmpl); err == nil {
			t.Errorf("expected error for invalid template, got nil")
		}
	})

	t.Run("DefaultTemplates", func(t *testing.T) {
		for _, tmpl := range registry {
			if err := ValidateTemplate(tmpl.Template); err != nil {
				t.Errorf("default template %s is invalid: %v", tmpl.Name, err)
			}
		}
	})
}

func TestGetTemplate(t *testing.T) {
	t.Run("ExistingTemplate", func(t *testing.T) {
		tmpl := GetTemplate("default")
		if tmpl == nil {
			t.Errorf("expected default template, got nil")
		}
		if tmpl.Name != "default" {
			t.Errorf("expected default template name, got: %s", tmpl.Name)
		}
	})

	t.Run("NonExistentTemplateFallback", func(t *testing.T) {
		tmpl := GetTemplate("non-existent")
		if tmpl == nil {
			t.Errorf("expected fallback to default template, got nil")
		}
		if tmpl.Name != "default" {
			t.Errorf("expected fallback to default template, got: %s", tmpl.Name)
		}
	})
}
