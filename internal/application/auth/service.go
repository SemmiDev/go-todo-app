// Package auth provides the application logic for user authentication and session management.
// It handles OAuth2 flows, token issuance, validation, and session lifecycle.
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/semmidev/go-todo-app/internal/common/apperr"
	"github.com/semmidev/go-todo-app/internal/common/token"
	"github.com/semmidev/go-todo-app/internal/domain/session"
	"github.com/semmidev/go-todo-app/internal/domain/user"
	"github.com/semmidev/go-todo-app/internal/port/input"
	"github.com/semmidev/go-todo-app/internal/port/output"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Config holds OAuth2 and token configuration for the auth service.
type Config struct {
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
	GoogleClientID       string
	GoogleClientSecret   string
	GoogleCallbackURL    string
}

// Service implements input.AuthUseCase and manages the authentication lifecycle.
type Service struct {
	userRepo        output.UserRepository
	sessionRepo     output.SessionRepository
	tokenMaker      token.Maker
	taskDistributor output.TaskDistributor
	oauthCfg        *oauth2.Config
	cfg             Config
	transactor      output.Transactor
}

// Compile-time interface conformance check.
var _ input.AuthUseCase = (*Service)(nil)

// NewService creates a new authentication service with the provided dependencies and configuration.
func NewService(
	userRepo output.UserRepository,
	sessionRepo output.SessionRepository,
	tokenMaker token.Maker,
	taskDistributor output.TaskDistributor,
	cfg Config,
	transactor output.Transactor,
) *Service {
	return &Service{
		userRepo:        userRepo,
		sessionRepo:     sessionRepo,
		tokenMaker:      tokenMaker,
		taskDistributor: taskDistributor,
		oauthCfg: &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  cfg.GoogleCallbackURL,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
				"openid",
			},
			Endpoint: google.Endpoint,
		},
		cfg:        cfg,
		transactor: transactor,
	}
}

// BuildAuthURL generates the Google OAuth2 authorization URL.
func (s *Service) BuildAuthURL(ctx context.Context, p input.BuildAuthURLParams) string {
	return s.oauthCfg.AuthCodeURL(p.State)
}

// GoogleUserInfo is the response structure from Google's userinfo endpoint.
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
}

// ExchangeAndLogin exchanges an OAuth2 code for tokens and establishes a user session.
func (s *Service) ExchangeAndLogin(ctx context.Context, p input.ExchangeAndLoginParams) (*input.LoginResult, error) {
	tok, err := s.oauthCfg.Exchange(ctx, p.Code)
	if err != nil {
		return nil, fmt.Errorf("exchange oauth code: %w", err)
	}

	client := s.oauthCfg.Client(ctx, tok)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google responded %d", resp.StatusCode)
	}

	var info GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode user info: %w", err)
	}
	if !info.VerifiedEmail {
		return nil, fmt.Errorf("google email not verified")
	}

	var u *user.User
	var accessToken, refreshToken string
	var accessPayload, refreshPayload *token.Payload

	err = s.transactor.RunInTx(ctx, func(txCtx context.Context) error {
		var txErr error
		u, txErr = s.userRepo.GetOrCreateByEmail(txCtx, info.Email, info.Name)
		if txErr != nil {
			return fmt.Errorf("upsert user: %w", txErr)
		}

		// Create access token.
		accessToken, accessPayload, txErr = s.tokenMaker.CreateToken(
			u.ID(),
			u.Email(),
			s.cfg.AccessTokenDuration,
			token.TokenTypeAccessToken,
		)
		if txErr != nil {
			return fmt.Errorf("create access token: %w", txErr)
		}

		// Create refresh token.
		refreshToken, refreshPayload, txErr = s.tokenMaker.CreateToken(
			u.ID(),
			u.Email(),
			s.cfg.RefreshTokenDuration,
			token.TokenTypeRefreshToken,
		)
		if txErr != nil {
			return fmt.Errorf("create refresh token: %w", txErr)
		}

		// Persist session with the refresh token.
		sess := session.New(
			refreshPayload.ID, u.ID(), refreshToken,
			p.UserAgent, p.ClientIP, refreshPayload.ExpiredAt,
		)
		if txErr := s.sessionRepo.Create(txCtx, sess); txErr != nil {
			return fmt.Errorf("create session: %w", txErr)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Trigger welcome email if the user was just registered (within the last 10 seconds)
	if time.Since(u.CreatedAt()) < 10*time.Second {
		_ = s.taskDistributor.DistributeTaskSendWelcomeEmail(ctx, &output.TaskPayloadSendWelcomeEmail{
			UserID: u.ID(),
		})
	}

	return &input.LoginResult{
		SessionID:             refreshPayload.ID,
		AccessToken:           accessToken,
		RefreshToken:          refreshToken,
		AccessTokenExpiresAt:  accessPayload.ExpiredAt,
		RefreshTokenExpiresAt: refreshPayload.ExpiredAt,
		User:                  u,
	}, nil
}

// ValidateToken verifies the authenticity and expiration of an access token.
func (s *Service) ValidateToken(ctx context.Context, p input.ValidateTokenParams) (*user.User, error) {
	payload, err := s.tokenMaker.VerifyToken(p.AccessToken, token.TokenTypeAccessToken)
	if err != nil {
		return nil, apperr.ErrInvalidToken
	}
	u, err := s.userRepo.GetByID(ctx, payload.UserID)
	if err != nil {
		return nil, apperr.ErrNotFound
	}
	return u, nil
}

// RenewAccessToken issues a new access token using a valid refresh token.
func (s *Service) RenewAccessToken(ctx context.Context, p input.RenewAccessTokenParams) (*input.RenewTokenResult, error) {
	refreshPayload, err := s.tokenMaker.VerifyToken(p.RefreshToken, token.TokenTypeRefreshToken)
	if err != nil {
		return nil, apperr.ErrInvalidToken
	}

	sess, err := s.sessionRepo.GetByID(ctx, refreshPayload.ID)
	if err != nil {
		return nil, apperr.ErrInvalidToken
	}

	if sess.IsBlocked() {
		return nil, apperr.ErrUnauthorized
	}

	if sess.UserID() != refreshPayload.UserID {
		return nil, apperr.ErrUnauthorized
	}

	if sess.RefreshToken() != p.RefreshToken {
		return nil, apperr.ErrUnauthorized
	}

	accessToken, accessPayload, err := s.tokenMaker.CreateToken(
		refreshPayload.UserID,
		refreshPayload.Email,
		s.cfg.AccessTokenDuration,
		token.TokenTypeAccessToken,
	)
	if err != nil {
		return nil, fmt.Errorf("create access token: %w", err)
	}

	return &input.RenewTokenResult{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessPayload.ExpiredAt,
	}, nil
}

// Logout invalidates a user session.
func (s *Service) Logout(ctx context.Context, p input.LogoutParams) error {
	return s.sessionRepo.Delete(ctx, p.SessionID)
}

// ListSessions retrieves all active sessions for a specific user.
func (s *Service) ListSessions(ctx context.Context, p input.ListSessionsParams) ([]*session.Session, error) {
	return s.sessionRepo.ListByUserID(ctx, p.UserID)
}

// RevokeSession terminates a specific session, ensuring it belongs to the requesting user.
func (s *Service) RevokeSession(ctx context.Context, p input.RevokeSessionParams) error {
	sess, err := s.sessionRepo.GetByID(ctx, p.SessionID)
	if err != nil {
		return apperr.ErrNotFound
	}
	if sess.UserID() != p.UserID {
		return apperr.ErrUnauthorized
	}
	return s.sessionRepo.Delete(ctx, p.SessionID)
}
