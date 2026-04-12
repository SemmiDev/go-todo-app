// Package user defines the User domain model.
// It represents the core identity of a person using the system.
package user

import (
	"time"

	"github.com/google/uuid"
)

// User represents a registered person in the system.
// It contains core identity information and audit timestamps.
type User struct {
	id        uuid.UUID
	email     string
	fullName  string
	createdAt time.Time
	updatedAt time.Time
}

// New creates a new User with a fresh UUID and current timestamps.
func New(email, fullName string) *User {
	now := time.Now().UTC()
	return &User{
		id: uuid.Must(uuid.NewV7()), email: email,
		fullName: fullName, createdAt: now, updatedAt: now,
	}
}

// Reconstitute creates a User from existing data (e.g., from a database).
func Reconstitute(id uuid.UUID, email, fullName string, createdAt, updatedAt time.Time) *User {
	return &User{id: id, email: email, fullName: fullName, createdAt: createdAt, updatedAt: updatedAt}
}

// ID returns the unique identifier for the user.
func (u *User) ID() uuid.UUID        { return u.id }

// Email returns the user's primary email address.
func (u *User) Email() string        { return u.email }

// FullName returns the user's full name.
func (u *User) FullName() string     { return u.fullName }

// CreatedAt returns the timestamp when the user was registered.
func (u *User) CreatedAt() time.Time { return u.createdAt }

// UpdatedAt returns the timestamp when the user profile was last modified.
func (u *User) UpdatedAt() time.Time { return u.updatedAt }
