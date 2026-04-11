// Package session models the user session domain entity.
package session

import (
	"time"

	"github.com/google/uuid"
)

// Session represents an authenticated user session backed by a refresh token.
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
