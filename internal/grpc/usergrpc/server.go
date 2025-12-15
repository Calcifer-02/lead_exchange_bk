package usergrpc

import (
	"context"
	"lead_exchange/internal/domain"
	pb "lead_exchange/pkg"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

// UserService описывает бизнес-логику работы с пользователями.
type UserService interface {
	GetProfile(ctx context.Context, userID uuid.UUID) (domain.User, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, update domain.UserFilter) (domain.User, error)
	UpdateUserStatus(ctx context.Context, userID uuid.UUID, status domain.UserStatus) (domain.User, error)
	ListUsers(ctx context.Context, filter domain.UserFilter) ([]domain.User, error)
}

// userServer реализует gRPC UserServiceServer.
type userServer struct {
	pb.UnimplementedUserServiceServer
	userService UserService
}

// RegisterUserServerGRPC регистрирует UserServiceServer в gRPC сервере.
func RegisterUserServerGRPC(server *grpc.Server, userSvc UserService) {
	pb.RegisterUserServiceServer(server, &userServer{
		userService: userSvc,
	})
}
