package interceptor

import (
	"context"
	"log/slog"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/semmidev/go-todo-app/internal/common/wideevent"
	"github.com/semmidev/go-todo-app/internal/domain/user"
	"github.com/semmidev/go-todo-app/internal/port/input"
)

// Public methods that do NOT require authentication.
var publicMethods = map[string]bool{
	"/todo.v1.AuthService/GetAuthURL":       true,
	"/todo.v1.AuthService/ExchangeCode":     true,
	"/todo.v1.AuthService/RenewAccessToken": true,
}

type contextKey string

const userContextKey contextKey = "authenticated_user"

// ContextWithUser stores the authenticated user in the context.
func ContextWithUser(ctx context.Context, u *user.User) context.Context {
	return context.WithValue(ctx, userContextKey, u)
}

// UserFromContext retrieves the authenticated user from the context.
func UserFromContext(ctx context.Context) (*user.User, bool) {
	u, ok := ctx.Value(userContextKey).(*user.User)
	return u, ok && u != nil
}

// AuthUnaryInterceptor validates Bearer tokens on every non-public gRPC call.
// After a successful validation it enriches the request-scoped WideEvent with
// user context (user_id, user_email) so the canonical log line captures who
// made the request — without emitting any extra log lines.
func AuthUnaryInterceptor(authSvc input.AuthUseCase) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if publicMethods[info.FullMethod] {
			// Mark public endpoints clearly in the wide event
			wideevent.Add(ctx, slog.Bool("public_endpoint", true))
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		accessToken := extractBearerToken(md)
		if accessToken == "" {
			return nil, status.Error(codes.Unauthenticated, "missing or invalid authorization header")
		}

		u, err := authSvc.ValidateToken(ctx, input.ValidateTokenParams{
			AccessToken: accessToken,
		})
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
		}

		// Enrich the wide event with user identity — no extra log line.
		wideevent.Add(ctx,
			slog.String("user_id", u.ID().String()),
			slog.String("user_email", u.Email()),
		)

		ctx = ContextWithUser(ctx, u)
		return handler(ctx, req)
	}
}

func extractBearerToken(md metadata.MD) string {
	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return ""
	}

	authHeader := authHeaders[0]
	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return ""
	}

	return authHeader[len(prefix):]
}
