package input

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/semmidev/go-todo-app/internal/domain/user"
)

// ─── Auth Params ───────────────────────────────────────────────────────────────

// BuildAuthURLParams holds parameters for building the Google Auth URL.
type BuildAuthURLParams struct {
	State string `json:"state" validate:"required"`
}

// ExchangeAndLoginParams holds parameters for exchanging an OAuth code and logging in.
type ExchangeAndLoginParams struct {
	Code      string `json:"code" validate:"required"`
	UserAgent string `json:"user_agent" validate:"required"`
	ClientIP  string `json:"client_ip" validate:"required"`
}

// ValidateTokenParams holds parameters for validating an access token.
type ValidateTokenParams struct {
	AccessToken string `json:"access_token" validate:"required"`
}

// RenewAccessTokenParams holds parameters for renewing an access token.
type RenewAccessTokenParams struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// LogoutParams holds parameters for logging out a user session.
type LogoutParams struct {
	SessionID uuid.UUID `json:"session_id" validate:"required,uuid"`
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
	BuildAuthURL(ctx context.Context, p BuildAuthURLParams) string
	ExchangeAndLogin(ctx context.Context, p ExchangeAndLoginParams) (*LoginResult, error)
	ValidateToken(ctx context.Context, p ValidateTokenParams) (*user.User, error)
	RenewAccessToken(ctx context.Context, p RenewAccessTokenParams) (*RenewTokenResult, error)
	Logout(ctx context.Context, p LogoutParams) error
}
