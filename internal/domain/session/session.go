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
	id           uuid.UUID
	userID       uuid.UUID
	refreshToken string
	userAgent    string
	clientIP     string
	isBlocked    bool
	expiresAt    time.Time
	createdAt    time.Time
}

// New creates a new Session with the provided values.
func New(id, userID uuid.UUID, refreshToken, userAgent, clientIP string, expiresAt time.Time) *Session {
	return &Session{
		id:           id,
		userID:       userID,
		refreshToken: refreshToken,
		userAgent:    userAgent,
		clientIP:     clientIP,
		isBlocked:    false,
		expiresAt:    expiresAt,
		createdAt:    time.Now().UTC(),
	}
}

// Reconstitute creates a Session from existing data (e.g., from a database).
func Reconstitute(id, userID uuid.UUID, refreshToken, userAgent, clientIP string, isBlocked bool, expiresAt, createdAt time.Time) *Session {
	return &Session{
		id:           id,
		userID:       userID,
		refreshToken: refreshToken,
		userAgent:    userAgent,
		clientIP:     clientIP,
		isBlocked:    isBlocked,
		expiresAt:    expiresAt,
		createdAt:    createdAt,
	}
}

// ID returns the unique identifier for the session.
func (s *Session) ID() uuid.UUID { return s.id }

// UserID returns the owner's identifier.
func (s *Session) UserID() uuid.UUID { return s.userID }

// RefreshToken returns the session's refresh token string.
func (s *Session) RefreshToken() string { return s.refreshToken }

// UserAgent returns the client's user agent string.
func (s *Session) UserAgent() string { return s.userAgent }

// ClientIP returns the client's IP address.
func (s *Session) ClientIP() string { return s.clientIP }

// IsBlocked returns whether the session has been blocked.
func (s *Session) IsBlocked() bool { return s.isBlocked }

// ExpiresAt returns the session's expiration timestamp.
func (s *Session) ExpiresAt() time.Time { return s.expiresAt }

// CreatedAt returns the timestamp when the session was created.
func (s *Session) CreatedAt() time.Time { return s.createdAt }
