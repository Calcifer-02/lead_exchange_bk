package usergrpc

import (
	"context"
	"errors"
	"fmt"
	"lead_exchange/internal/middleware"
	"lead_exchange/internal/repository"
	pb "lead_exchange/pkg"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UpdateUserStatus — изменение статуса пользователя.
func (s *userServer) UpdateUserStatus(ctx context.Context, in *pb.UpdateUserStatusRequest) (*pb.UserProfile, error) {
	if err := in.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Проверяем, что пользователь аутентифицирован
	_, ok := middleware.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not found in context")
	}

	// Парсим ID пользователя, статус которого нужно изменить
	targetUserID, err := uuid.Parse(in.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid user_id: %v", err))
	}

	// Преобразуем статус из proto в доменную модель
	userStatus := userStatusProtoToDomain(in.Status)

	// Обновляем статус
	updatedUser, err := s.userService.UpdateUserStatus(ctx, targetUserID, userStatus)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to update user status: %v", err))
	}

	return userDomainToProto(updatedUser), nil
}
