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

// ListUsers — получение списка пользователей.
func (s *userServer) ListUsers(ctx context.Context, in *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	// Проверяем, что пользователь аутентифицирован
	_, ok := middleware.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not found in context")
	}

	// Преобразуем фильтр из proto в доменную модель
	filter := domain.UserFilter{}
	if in.Filter != nil {
		filter.Email = in.Filter.Email
		filter.FirstName = in.Filter.FirstName
		filter.LastName = in.Filter.LastName
		filter.Phone = in.Filter.Phone
		filter.AgencyName = in.Filter.AgencyName
		if in.Filter.Role != nil {
			role := userRoleProtoToDomain(*in.Filter.Role)
			filter.Role = &role
		}
		if in.Filter.Status != nil {
			status := userStatusProtoToDomain(*in.Filter.Status)
			filter.Status = &status
		}
	}

	// Получаем список пользователей
	users, err := s.userService.ListUsers(ctx, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list users: %v", err))
	}

	// Преобразуем в proto
	protoUsers := make([]*pb.UserProfile, 0, len(users))
	for _, user := range users {
		protoUsers = append(protoUsers, userDomainToProto(user))
	}

	return &pb.ListUsersResponse{
		Users: protoUsers,
	}, nil
}
