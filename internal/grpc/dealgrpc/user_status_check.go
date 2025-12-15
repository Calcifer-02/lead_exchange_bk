package dealgrpc

import (
	"context"
	"fmt"
	"lead_exchange/internal/domain"
	"lead_exchange/internal/middleware"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// checkUserStatus проверяет, что пользователь не забанен и не приостановлен.
// Возвращает ошибку, если пользователь BANNED или SUSPENDED.
func (s *dealServer) checkUserStatus(ctx context.Context) error {
	userID, ok := middleware.FromContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "user not found in context")
	}

	user, err := s.userService.GetProfile(ctx, userID)
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to get user profile: %v", err))
	}

	if user.Status == domain.UserStatusBanned {
		return status.Error(codes.PermissionDenied, "banned users cannot access deals")
	}

	if user.Status == domain.UserStatusSuspended {
		return status.Error(codes.PermissionDenied, "suspended users cannot access deals")
	}

	return nil
}
