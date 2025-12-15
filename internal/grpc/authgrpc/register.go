package authgrpc

import (
	"context"
	"errors"
	"fmt"
	"lead_exchange/internal/repository"
	pb "lead_exchange/pkg"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Register — регистрация нового пользователя.
func (s *authServer) Register(ctx context.Context, in *pb.RegisterRequest) (*emptypb.Empty, error) {
	if err := in.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	_, err := s.authService.Register(ctx, in.GetEmail(), in.GetPassword(), in.GetFirstName(), in.GetLastName())
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrUserExists):
			return nil, status.Error(codes.AlreadyExists, "user already exists")
		default:
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to register user: %v", err))
		}
	}

	return &emptypb.Empty{}, nil
}
