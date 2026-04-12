// Package input defines the input ports (use cases) of the application.
// These interfaces are implemented by the application layer and called by the driving adapters.
package input

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/semmidev/go-todo-app/internal/domain/session"
	"github.com/semmidev/go-todo-app/internal/domain/user"
)

// ─── Auth Params ───────────────────────────────────────────────────────────────

// BuildAuthURLParams holds parameters for building the Google Auth URL.
type BuildAuthURLParams struct {
	// State is an OAuth2 state string used for CSRF protection.
	State string `json:"state" validate:"required"`
}

// ExchangeAndLoginParams holds parameters for exchanging an OAuth code and logging in.
type ExchangeAndLoginParams struct {
	// Code is the OAuth2 authorization code received from the provider.
	Code string `json:"code" validate:"required"`
	// UserAgent is the client's user agent string for session tracking.
	UserAgent string `json:"user_agent" validate:"required"`
	// ClientIP is the client's IP address for session tracking.
	ClientIP string `json:"client_ip" validate:"required"`
}

// ValidateTokenParams holds parameters for validating an access token.
type ValidateTokenParams struct {
	// AccessToken is the PASETO/JWT token to be validated.
	AccessToken string `json:"access_token" validate:"required"`
}

// RenewAccessTokenParams holds parameters for renewing an access token.
type RenewAccessTokenParams struct {
	// RefreshToken is the valid refresh token used to generate a new access token.
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// LogoutParams holds parameters for logging out a user session.
type LogoutParams struct {
	// SessionID is the unique identifier of the session to be terminated.
	SessionID uuid.UUID `json:"session_id" validate:"required,uuid"`
}

// ListSessionsParams holds parameters for listing user sessions.
type ListSessionsParams struct {
	// UserID is the unique identifier of the user whose sessions are being listed.
	UserID uuid.UUID `json:"user_id" validate:"required,uuid"`
}

// RevokeSessionParams holds parameters for revoking a user session.
type RevokeSessionParams struct {
	// SessionID is the unique identifier of the session to be revoked.
	SessionID uuid.UUID `json:"session_id" validate:"required,uuid"`
	// UserID is the unique identifier of the user who owns the session.
	UserID uuid.UUID `json:"user_id" validate:"required,uuid"`
}

// ─── Auth Results ──────────────────────────────────────────────────────────────

// LoginResult holds the result of a successful OAuth exchange and login.
type LoginResult struct {
	SessionID             uuid.UUID
	AccessToken           string
	RefreshToken          string
	AccessTokenExpiresAt  time.Time
	RefreshTokenExpiresAt time.Time
	User                  *user.User
}

// RenewTokenResult holds the result of a successful access-token renewal.
type RenewTokenResult struct {
	AccessToken          string
	AccessTokenExpiresAt time.Time
}

// ─── Use-case interfaces ─────────────────────────────────────────────────────

// AuthUseCase is the driving port for authentication operations.
type AuthUseCase interface {
	// BuildAuthURL generates a URL to redirect the user to for OAuth2 authentication.
	BuildAuthURL(ctx context.Context, p BuildAuthURLParams) string
	// ExchangeAndLogin exchanges an OAuth2 code for tokens and creates a new session.
	ExchangeAndLogin(ctx context.Context, p ExchangeAndLoginParams) (*LoginResult, error)
	// ValidateToken checks if an access token is valid and returns the associated user.
	ValidateToken(ctx context.Context, p ValidateTokenParams) (*user.User, error)
	// RenewAccessToken uses a refresh token to generate a new access token.
	RenewAccessToken(ctx context.Context, p RenewAccessTokenParams) (*RenewTokenResult, error)
	// Logout terminates a user session by its ID.
	Logout(ctx context.Context, p LogoutParams) error
	// ListSessions retrieves all active sessions for a specific user.
	ListSessions(ctx context.Context, p ListSessionsParams) ([]*session.Session, error)
	// RevokeSession revokes a specific session for a user.
	RevokeSession(ctx context.Context, p RevokeSessionParams) error
}
