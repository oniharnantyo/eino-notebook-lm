package validator

import (
	"fmt"
	"reflect"
	"strings"
)

// Validator provides validation functionality
type Validator struct{}

// New creates a new validator
func New() *Validator {
	return &Validator{}
}

// Validate validates a struct based on validation tags
func (v *Validator) Validate(s interface{}) error {
	val := reflect.ValueOf(s)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("input must be a struct or pointer to struct")
	}

	typ := val.Type()
	var errors []string

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)
		tag := fieldType.Tag.Get("validate")

		if tag == "" {
			continue
		}

		if err := v.validateField(field, tag, fieldType.Name); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// validateField validates a single field based on validation tags
func (v *Validator) validateField(field reflect.Value, tag, fieldName string) error {
	rules := strings.Split(tag, ",")

	for _, rule := range rules {
		parts := strings.Split(rule, "=")
		name := parts[0]

		switch name {
		case "required":
			if v.isEmpty(field) {
				return fmt.Errorf("%s is required", fieldName)
			}
		case "min":
			if len(parts) < 2 {
				continue
			}
			min := parts[1]
			if err := v.validateMin(field, fieldName, min); err != nil {
				return err
			}
		case "max":
			if len(parts) < 2 {
				continue
			}
			max := parts[1]
			if err := v.validateMax(field, fieldName, max); err != nil {
				return err
			}
		case "oneof":
			if len(parts) < 2 {
				continue
			}
			allowed := strings.Split(parts[1], " ")
			if err := v.validateOneOf(field, fieldName, allowed); err != nil {
				return err
			}
		}
	}

	return nil
}

// isEmpty checks if a field is empty
func (v *Validator) isEmpty(field reflect.Value) bool {
	switch field.Kind() {
	case reflect.String:
		return field.String() == ""
	case reflect.Slice, reflect.Array:
		return field.Len() == 0
	case reflect.Interface, reflect.Ptr:
		return field.IsNil()
	default:
		return false
	}
}

// validateMin validates minimum length/value
func (v *Validator) validateMin(field reflect.Value, fieldName, minStr string) error {
	switch field.Kind() {
	case reflect.String:
		if len(field.String()) < parseMin(minStr) {
			return fmt.Errorf("%s must be at least %s characters", fieldName, minStr)
		}
	case reflect.Slice, reflect.Array:
		if field.Len() < parseMin(minStr) {
			return fmt.Errorf("%s must have at least %s items", fieldName, minStr)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		min := parseMin(minStr)
		if field.Int() < int64(min) {
			return fmt.Errorf("%s must be at least %s", fieldName, minStr)
		}
	}
	return nil
}

// validateMax validates maximum length/value
func (v *Validator) validateMax(field reflect.Value, fieldName, maxStr string) error {
	switch field.Kind() {
	case reflect.String:
		if len(field.String()) > parseMax(maxStr) {
			return fmt.Errorf("%s must be at most %s characters", fieldName, maxStr)
		}
	case reflect.Slice, reflect.Array:
		if field.Len() > parseMax(maxStr) {
			return fmt.Errorf("%s must have at most %s items", fieldName, maxStr)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		max := parseMax(maxStr)
		if field.Int() > int64(max) {
			return fmt.Errorf("%s must be at most %s", fieldName, maxStr)
		}
	}
	return nil
}

// validateOneOf validates that the value is one of the allowed values
func (v *Validator) validateOneOf(field reflect.Value, fieldName string, allowed []string) error {
	if field.Kind() != reflect.String {
		return nil
	}

	value := field.String()
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}

	return fmt.Errorf("%s must be one of: %s", fieldName, strings.Join(allowed, ", "))
}

func parseMin(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}

func parseMax(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}
