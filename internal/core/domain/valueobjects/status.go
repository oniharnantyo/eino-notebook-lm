package valueobjects

import "errors"

// NotebookStatus represents the status of a notebook
type NotebookStatus string

const (
	StatusActive   NotebookStatus = "active"
	StatusArchived NotebookStatus = "archived"
	StatusDeleted  NotebookStatus = "deleted"
)

// String returns the string representation of the status
func (s NotebookStatus) String() string {
	return string(s)
}

// IsValid checks if the status is valid
func (s NotebookStatus) IsValid() bool {
	switch s {
	case StatusActive, StatusArchived, StatusDeleted:
		return true
	default:
		return false
	}
}

// Domain errors
var (
	ErrEmptyTitle    = errors.New("title cannot be empty")
	ErrTitleTooLong  = errors.New("title cannot exceed 200 characters")
	ErrInvalidStatus = errors.New("invalid notebook status")
	ErrNotFound      = errors.New("notebook not found")
	ErrAlreadyExists = errors.New("notebook already exists")
	ErrInvalidID     = errors.New("invalid notebook ID")
	ErrUnauthorized  = errors.New("unauthorized access")
	ErrForbidden     = errors.New("forbidden")
	ErrValidation    = errors.New("validation failed")
	ErrInternal      = errors.New("internal server error")
)
