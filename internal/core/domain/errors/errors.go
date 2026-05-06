package errors

import (
	"errors"
	"fmt"
)

var (
	// Domain errors
	ErrNotFound         = errors.New("resource not found")
	ErrAlreadyExists    = errors.New("resource already exists")
	ErrInvalidInput     = errors.New("invalid input")
	ErrInvalidInputType = errors.New("invalid input type")
	ErrValidation       = errors.New("validation failed")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrForbidden        = errors.New("forbidden")
	ErrInternal         = errors.New("internal server error")
	ErrInvalidID        = errors.New("invalid id")
	ErrEmptyTitle       = errors.New("title cannot be empty")
	ErrTitleTooLong     = errors.New("title cannot exceed 200 characters")
	ErrInvalidStatus    = errors.New("invalid status")
)

// DomainError represents a domain error with context
type DomainError struct {
	Code    string
	Message string
	Err     error
}

// Error implements the error interface
func (e *DomainError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *DomainError) Unwrap() error {
	return e.Err
}

// NewDomainError creates a new domain error
func NewDomainError(code, message string, err error) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Common error constructors
func NewNotFoundError(resource string) *DomainError {
	return &DomainError{
		Code:    "NOT_FOUND",
		Message: fmt.Sprintf("%s not found", resource),
		Err:     ErrNotFound,
	}
}

func NewValidationError(message string) *DomainError {
	return &DomainError{
		Code:    "VALIDATION_ERROR",
		Message: message,
		Err:     ErrValidation,
	}
}

func NewUnauthorizedError(message string) *DomainError {
	return &DomainError{
		Code:    "UNAUTHORIZED",
		Message: message,
		Err:     ErrUnauthorized,
	}
}

func NewInternalError(message string, err error) *DomainError {
	return &DomainError{
		Code:    "INTERNAL_ERROR",
		Message: message,
		Err:     err,
	}
}
