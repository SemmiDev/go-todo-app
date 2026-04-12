// Package apperr defines unified application-level sentinel errors.
// All layers (domain, application, adapter) use these shared errors
// to avoid circular dependencies and duplicated sentinel definitions.
package apperr

import "errors"

var (
	ErrNotFound     = errors.New("not found")
	ErrForbidden    = errors.New("forbidden")
	ErrConflict     = errors.New("conflict")
	ErrValidation   = errors.New("validation error")
	ErrUnauthorized = errors.New("unauthorized")
	ErrInvalidToken = errors.New("invalid or expired session")
)

// FieldError represents a single property validation failure.
type FieldError struct {
	Field   string
	Message string
}

// ValidationError holds multiple field errors.
type ValidationError struct {
	Errors []FieldError
}

// Error returns the error message for validation failures.
func (e *ValidationError) Error() string {
	return "validation failed"
}
