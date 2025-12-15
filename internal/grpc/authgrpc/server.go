package authgrpc

import (
	"context"
	pb "lead_exchange/pkg"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

// AuthService описывает бизнес-логику авторизации и регистрации.
type AuthService interface {
	Register(ctx context.Context, email, password, firstName, lastName string) (uuid.UUID, error)
	Login(ctx context.Context, email, password string) (uuid.UUID, string, error)
}

// authServer реализует gRPC AuthServiceServer.
type authServer struct {
	pb.UnimplementedAuthServiceServer

	authService AuthService
}

// RegisterAuthServerGRPC регистрирует AuthServiceServer в gRPC сервере.
func RegisterAuthServerGRPC(server *grpc.Server, authSvc AuthService) {
	pb.RegisterAuthServiceServer(server, &authServer{
		authService: authSvc,
	})
}
