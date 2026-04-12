// Package session defines the User Session domain model.
// Sessions are used to track authenticated users and manage refresh tokens.
package session

import (
	"time"

	"github.com/google/uuid"
)

// Session represents an active authentication session for a user.
// It tracks the refresh token, client metadata, and expiration state.
type Session struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	RefreshToken string
	UserAgent    string
	ClientIP     string
	IsBlocked    bool
	ExpiresAt    time.Time
	CreatedAt    time.Time
}
