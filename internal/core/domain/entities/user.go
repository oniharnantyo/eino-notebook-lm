package entities

import (
	"time"

	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// User represents a user entity
type User struct {
	ID        uuid.UUID
	Email     string
	Name      string
	Password  string // hashed
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// NewUser creates a new user entity
func NewUser(email, name, hashedPassword string) (*User, error) {
	user := &User{
		ID:        uuid.New(),
		Email:     email,
		Name:      name,
		Password:  hashedPassword,
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return user, nil
}

// IsDeleted checks if the user is deleted
func (u *User) IsDeleted() bool {
	return u.DeletedAt != nil
}
