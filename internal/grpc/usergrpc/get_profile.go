package usergrpc

import (
	"context"
	"fmt"
	"lead_exchange/internal/middleware"
	pb "lead_exchange/pkg"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GetProfile — получение профиля текущего пользователя.
func (s *userServer) GetProfile(ctx context.Context, _ *emptypb.Empty) (*pb.UserProfile, error) {
	userID, ok := middleware.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not found in context")
	}

	user, err := s.userService.GetProfile(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get profile: %v", err))
	}

	return userDomainToProto(user), nil
}
