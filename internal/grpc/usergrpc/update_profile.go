package usergrpc

import (
	"context"
	"fmt"
	"lead_exchange/internal/domain"
	"lead_exchange/internal/middleware"
	pb "lead_exchange/pkg"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UpdateProfile — обновление профиля пользователя.
func (s *userServer) UpdateProfile(ctx context.Context, in *pb.UpdateProfileRequest) (*pb.UserProfile, error) {
	if err := in.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	userID, ok := middleware.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not found in context")
	}

	filter := domain.UserFilter{
		FirstName:  in.FirstName,
		LastName:   in.LastName,
		Phone:      in.Phone,
		AgencyName: in.AgencyName,
		AvatarURL:  in.AvatarUrl,
	}

	updatedUser, err := s.userService.UpdateProfile(ctx, userID, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to update profile: %v", err))
	}

	return userDomainToProto(updatedUser), nil
}
