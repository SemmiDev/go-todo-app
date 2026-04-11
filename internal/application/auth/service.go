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

// Service implements input.AuthUseCase.
type Service struct {
	userRepo    output.UserRepository
	sessionRepo output.SessionRepository
	tokenMaker  token.Maker
	oauthCfg    *oauth2.Config
	cfg         Config
}

// Compile-time interface conformance check.
var _ input.AuthUseCase = (*Service)(nil)

func NewService(
	userRepo output.UserRepository,
	sessionRepo output.SessionRepository,
	tokenMaker token.Maker,
	cfg Config,
) *Service {
	return &Service{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		tokenMaker:  tokenMaker,
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
		cfg: cfg,
	}
}

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

	u, err := s.userRepo.GetOrCreateByEmail(ctx, info.Email, info.Name)
	if err != nil {
		return nil, fmt.Errorf("upsert user: %w", err)
	}

	// Create access token.
	accessToken, accessPayload, err := s.tokenMaker.CreateToken(
		u.ID(),
		u.Email(),
		s.cfg.AccessTokenDuration,
		token.TokenTypeAccessToken,
	)
	if err != nil {
		return nil, fmt.Errorf("create access token: %w", err)
	}

	// Create refresh token.
	refreshToken, refreshPayload, err := s.tokenMaker.CreateToken(
		u.ID(),
		u.Email(),
		s.cfg.RefreshTokenDuration,
		token.TokenTypeRefreshToken,
	)
	if err != nil {
		return nil, fmt.Errorf("create refresh token: %w", err)
	}

	// Persist session with the refresh token.
	if err := s.sessionRepo.Create(ctx, &session.Session{
		ID:           refreshPayload.ID,
		UserID:       u.ID(),
		RefreshToken: refreshToken,
		UserAgent:    p.UserAgent,
		ClientIP:     p.ClientIP,
		IsBlocked:    false,
		ExpiresAt:    refreshPayload.ExpiredAt,
		CreatedAt:    time.Now().UTC(),
	}); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
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

func (s *Service) RenewAccessToken(ctx context.Context, p input.RenewAccessTokenParams) (*input.RenewTokenResult, error) {
	refreshPayload, err := s.tokenMaker.VerifyToken(p.RefreshToken, token.TokenTypeRefreshToken)
	if err != nil {
		return nil, apperr.ErrInvalidToken
	}

	sess, err := s.sessionRepo.GetByID(ctx, refreshPayload.ID)
	if err != nil {
		return nil, apperr.ErrInvalidToken
	}

	if sess.IsBlocked {
		return nil, apperr.ErrUnauthorized
	}

	if sess.UserID != refreshPayload.UserID {
		return nil, apperr.ErrUnauthorized
	}

	if sess.RefreshToken != p.RefreshToken {
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

func (s *Service) Logout(ctx context.Context, p input.LogoutParams) error {
	return s.sessionRepo.Delete(ctx, p.SessionID)
}
