package uuid

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// UUID represents a unique identifier
type UUID string

// New generates a new UUID
func New() UUID {
	return UUID(uuid.New().String())
}

// Parse parses a UUID string
func Parse(s string) (UUID, error) {
	uid, err := uuid.Parse(s)
	if err != nil {
		return "", fmt.Errorf("invalid UUID: %w", err)
	}
	return UUID(uid.String()), nil
}

// String returns the string representation of the UUID
func (u UUID) String() string {
	return string(u)
}

// IsValid checks if the UUID is valid
func (u UUID) IsValid() bool {
	_, err := uuid.Parse(string(u))
	return err == nil
}

// IsEmpty checks if the UUID is empty
func (u UUID) IsEmpty() bool {
	return string(u) == ""
}

// Equals checks if two UUIDs are equal
func (u UUID) Equals(other UUID) bool {
	return strings.EqualFold(string(u), string(other))
}

// MarshalJSON implements json.Marshaler
func (u UUID) MarshalJSON() ([]byte, error) {
	if u.IsEmpty() {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf(`"%s"`, u)), nil
}

// UnmarshalJSON implements json.Unmarshaler
func (u *UUID) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*u = ""
		return nil
	}
	s := strings.Trim(string(data), `"`)
	uid, err := Parse(s)
	if err != nil {
		return err
	}
	*u = uid
	return nil
}
