package authgrpc

import (
	"context"
	"errors"
	"fmt"
	"lead_exchange/internal/services/user"
	pb "lead_exchange/pkg"

	"lead_exchange/internal/repository"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Login — авторизация пользователя.
func (s *authServer) Login(ctx context.Context, in *pb.LoginRequest) (*pb.AuthResponse, error) {
	if err := in.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	_, token, err := s.authService.Login(ctx, in.GetEmail(), in.GetPassword())
	if err != nil {
		switch {
		case errors.Is(err, user.ErrInvalidCredentials):
			return nil, status.Error(codes.Unauthenticated, "invalid email or password")
		case errors.Is(err, repository.ErrUserNotFound):
			return nil, status.Error(codes.NotFound, "user not found")
		default:
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to login: %v", err))
		}
	}

	return &pb.AuthResponse{Token: token}, nil
}
