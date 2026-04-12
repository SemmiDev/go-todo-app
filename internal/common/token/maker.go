// Package token provides the common interface and primitives for token management.
package token

import (
	"time"

	"github.com/google/uuid"
)

// Maker is the core interface for creating and verifying tokens.
type Maker interface {
	// CreateToken generates a signed token for a specific user.
	CreateToken(userID uuid.UUID, email string, duration time.Duration, tokenType TokenType) (string, *Payload, error)

	// VerifyToken parses and validates a signed token.
	VerifyToken(token string, tokenType TokenType) (*Payload, error)
}
