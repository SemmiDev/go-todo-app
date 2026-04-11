// Package token provides PASETO-based token creation and verification.
package token

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Different types of error returned by the VerifyToken function.
var (
	ErrInvalidToken = errors.New("token is invalid")
	ErrExpiredToken = errors.New("token has expired")
)

// TokenType distinguishes access tokens from refresh tokens.
type TokenType byte

const (
	TokenTypeAccessToken  TokenType = 1
	TokenTypeRefreshToken TokenType = 2
)

// Payload contains the payload data of the token.
type Payload struct {
	ID        uuid.UUID `json:"id"`
	Type      TokenType `json:"token_type"`
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiredAt time.Time `json:"expired_at"`
}

// NewPayload creates a new token payload with a specific user ID, email and duration.
func NewPayload(userID uuid.UUID, email string, duration time.Duration, tokenType TokenType) (*Payload, error) {
	tokenID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	payload := &Payload{
		ID:        tokenID,
		Type:      tokenType,
		UserID:    userID,
		Email:     email,
		IssuedAt:  time.Now(),
		ExpiredAt: time.Now().Add(duration),
	}
	return payload, nil
}

// Valid checks if the token payload is valid or not.
func (payload *Payload) Valid(tokenType TokenType) error {
	if payload.Type != tokenType {
		return ErrInvalidToken
	}
	if time.Now().After(payload.ExpiredAt) {
		return ErrExpiredToken
	}
	return nil
}
