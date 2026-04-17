// Package grpchandler provides gRPC implementations of the application services.
package grpchandler

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"errors"
	"github.com/google/uuid"

	pb "github.com/semmidev/go-todo-app/gen/todo/v1"
	"github.com/semmidev/go-todo-app/internal/adapter/driving/grpc/grpcerr"
	"github.com/semmidev/go-todo-app/internal/adapter/driving/grpc/interceptor"
	"github.com/semmidev/go-todo-app/internal/common/apperr"
	"github.com/semmidev/go-todo-app/internal/common/validation"
	"github.com/semmidev/go-todo-app/internal/port/input"
)

// AuthServer handles authentication and session management via gRPC.
// It depends on the input.AuthUseCase interface, NOT the concrete auth service.
type AuthServer struct {
	pb.UnimplementedAuthServiceServer
	svc       input.AuthUseCase
	validator *validation.Validator
}

// NewAuthServer creates a new AuthServer with the given use case and validator.
func NewAuthServer(svc input.AuthUseCase, validator *validation.Validator) *AuthServer {
	return &AuthServer{svc: svc, validator: validator}
}

// GetAuthURL returns the OAuth2 authorization URL for the client to redirect the user.
func (s *AuthServer) GetAuthURL(ctx context.Context, req *pb.GetAuthURLRequest) (*pb.GetAuthURLResponse, error) {
	params := input.BuildAuthURLParams{State: req.GetState()}
	if err := s.validator.Struct(params); err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}
	url := s.svc.BuildAuthURL(ctx, params)
	return &pb.GetAuthURLResponse{Url: url}, nil
}

// ExchangeCode handles the OAuth2 callback by exchanging the code for tokens and creating a session.
func (s *AuthServer) ExchangeCode(ctx context.Context, req *pb.ExchangeCodeRequest) (*pb.LoginResponse, error) {
	userAgent, clientIP := extractClientMetadata(ctx)
	params := input.ExchangeAndLoginParams{
		Code:      req.GetCode(),
		UserAgent: userAgent,
		ClientIP:  clientIP,
	}
	if err := s.validator.Struct(params); err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}
	result, err := s.svc.ExchangeAndLogin(ctx, params)
	if err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}

	return &pb.LoginResponse{
		User: &pb.UserInfo{
			UserId:   result.User.ID().String(),
			Email:    result.User.Email(),
			FullName: result.User.FullName(),
		},
		SessionId:             result.SessionID.String(),
		AccessToken:           result.AccessToken,
		RefreshToken:          result.RefreshToken,
		AccessTokenExpiresAt:  timestamppb.New(result.AccessTokenExpiresAt),
		RefreshTokenExpiresAt: timestamppb.New(result.RefreshTokenExpiresAt),
	}, nil
}

// RenewAccessToken issues a new access token using a valid refresh token.
func (s *AuthServer) RenewAccessToken(ctx context.Context, req *pb.RenewAccessTokenRequest) (*pb.RenewAccessTokenResponse, error) {
	params := input.RenewAccessTokenParams{
		RefreshToken: req.GetRefreshToken(),
	}
	if err := s.validator.Struct(params); err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}
	result, err := s.svc.RenewAccessToken(ctx, params)
	if err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}

	return &pb.RenewAccessTokenResponse{
		AccessToken:          result.AccessToken,
		AccessTokenExpiresAt: timestamppb.New(result.AccessTokenExpiresAt),
	}, nil
}

// Logout terminates the user session.
func (s *AuthServer) Logout(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	// With PASETO and Bearer tokens, we don't use cookies to logout.
	// If the client wants to revoke a session, they could send a session_id.
	// For now we leave it empty to succeed without error.
	return &emptypb.Empty{}, nil
}

// GetMe returns the current authenticated user's information.
func (s *AuthServer) GetMe(ctx context.Context, _ *emptypb.Empty) (*pb.GetMeResponse, error) {
	u, ok := interceptor.UserFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "not authenticated")
	}
	return &pb.GetMeResponse{
		User: &pb.UserInfo{
			UserId:   u.ID().String(),
			Email:    u.Email(),
			FullName: u.FullName(),
		},
	}, nil
}

// ListSessions returns all active sessions for the current user.
func (s *AuthServer) ListSessions(ctx context.Context, _ *emptypb.Empty) (*pb.ListSessionsResponse, error) {
	u, ok := interceptor.UserFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	sessions, err := s.svc.ListSessions(ctx, input.ListSessionsParams{
		UserID: u.ID(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list sessions: %v", err)
	}

	var pbSessions []*pb.Session
	for _, sess := range sessions {
		pbSessions = append(pbSessions, &pb.Session{
			Id:        sess.ID.String(),
			UserAgent: sess.UserAgent,
			ClientIp:  sess.ClientIP,
			IsCurrent: false, // Keep it false for now, or match it if we extract session ID later
			CreatedAt: timestamppb.New(sess.CreatedAt),
			ExpiresAt: timestamppb.New(sess.ExpiresAt),
		})
	}

	return &pb.ListSessionsResponse{
		Sessions: pbSessions,
	}, nil
}

// RevokeSession terminates a specific session by its ID.
func (s *AuthServer) RevokeSession(ctx context.Context, req *pb.RevokeSessionRequest) (*emptypb.Empty, error) {
	u, ok := interceptor.UserFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	sessionID, err := uuid.Parse(req.SessionId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid session ID format")
	}

	err = s.svc.RevokeSession(ctx, input.RevokeSessionParams{
		SessionID: sessionID,
		UserID:    u.ID(),
	})
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) || errors.Is(err, apperr.ErrUnauthorized) {
			return nil, status.Errorf(codes.NotFound, "session not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to revoke session: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// extractClientMetadata retrieves the user agent and client IP from the gRPC context metadata.
func extractClientMetadata(ctx context.Context) (userAgent, clientIP string) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", ""
	}
	if userAgents := md.Get("grpcgateway-user-agent"); len(userAgents) > 0 {
		userAgent = userAgents[0]
	} else if userAgents := md.Get("user-agent"); len(userAgents) > 0 {
		userAgent = userAgents[0]
	}

	if clientIPs := md.Get("x-forwarded-for"); len(clientIPs) > 0 {
		clientIP = clientIPs[0]
	}
	return userAgent, clientIP
}

var _ pb.AuthServiceServer = (*AuthServer)(nil)
