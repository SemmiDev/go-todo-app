package token

import (
	"time"

	"github.com/google/uuid"
)

// Maker is an interface for managing tokens.
type Maker interface {
	// CreateToken creates a new token for a specific user and duration.
	CreateToken(userID uuid.UUID, email string, duration time.Duration, tokenType TokenType) (string, *Payload, error)

	// VerifyToken checks if the token is valid or not.
	VerifyToken(token string, tokenType TokenType) (*Payload, error)
}
