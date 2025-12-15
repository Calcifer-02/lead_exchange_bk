package dealgrpc

import (
	"context"
	"lead_exchange/internal/domain"
	pb "lead_exchange/pkg"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

// DealService описывает бизнес-логику для работы со сделками.
type DealService interface {
	CreateDeal(ctx context.Context, deal domain.Deal) (uuid.UUID, error)
	GetDeal(ctx context.Context, id uuid.UUID) (domain.Deal, error)
	UpdateDeal(ctx context.Context, id uuid.UUID, update domain.DealFilter) (domain.Deal, error)
	ListDeals(ctx context.Context, filter domain.DealFilter) ([]domain.Deal, error)
	AcceptDeal(ctx context.Context, dealID uuid.UUID, buyerUserID uuid.UUID) (domain.Deal, error)
}

// UserService описывает бизнес-логику работы с пользователями (для проверки статуса).
type UserService interface {
	GetProfile(ctx context.Context, userID uuid.UUID) (domain.User, error)
}

// dealServer реализует gRPC DealServiceServer.
type dealServer struct {
	pb.UnimplementedDealServiceServer
	dealService DealService
	userService UserService
}

// RegisterDealServerGRPC регистрирует DealServiceServer в gRPC сервере.
func RegisterDealServerGRPC(server *grpc.Server, svc DealService, userSvc UserService) {
	pb.RegisterDealServiceServer(server, &dealServer{
		dealService: svc,
		userService: userSvc,
	})
}
