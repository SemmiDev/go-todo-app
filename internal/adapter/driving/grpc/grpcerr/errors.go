package grpcerr

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/semmidev/todo-app/internal/common/apperr"
)

// FromAppError maps a unified application error to the appropriate gRPC status.
func FromAppError(ctx context.Context, err error) error {
	var valErr *apperr.ValidationError
	if errors.As(err, &valErr) {
		st := status.New(codes.InvalidArgument, "validation failed")
		var violations []*errdetails.BadRequest_FieldViolation
		for _, e := range valErr.Errors {
			violations = append(violations, &errdetails.BadRequest_FieldViolation{
				Field:       e.Field,
				Description: e.Message,
			})
		}
		if details, dErr := st.WithDetails(&errdetails.BadRequest{FieldViolations: violations}); dErr == nil {
			return details.Err()
		}
		return st.Err()
	}

	switch {
	case errors.Is(err, apperr.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, apperr.ErrForbidden):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, apperr.ErrConflict):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, apperr.ErrValidation):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, apperr.ErrInvalidToken), errors.Is(err, apperr.ErrUnauthorized):
		return status.Error(codes.Unauthenticated, err.Error())
	default:
		slog.ErrorContext(ctx, "internal error", slog.Any("error", err))
		return status.Error(codes.Internal, "internal server error")
	}
}

// NewUnauthenticated creates a gRPC Unauthenticated error.
func NewUnauthenticated(msg string) error {
	return status.Error(codes.Unauthenticated, msg)
}

// NewInvalidArgument creates a gRPC InvalidArgument error.
func NewInvalidArgument(msg string) error {
	return status.Error(codes.InvalidArgument, msg)
}
