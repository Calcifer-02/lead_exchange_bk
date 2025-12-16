package leadgrpc

import (
	"context"
	"lead_exchange/internal/domain"
	pb "lead_exchange/pkg"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

// LeadService описывает бизнес-логику для работы с лидами.
type LeadService interface {
	CreateLead(ctx context.Context, lead domain.Lead) (uuid.UUID, error)
	GetLead(ctx context.Context, id uuid.UUID) (domain.Lead, error)
	UpdateLead(ctx context.Context, id uuid.UUID, update domain.LeadFilter) (domain.Lead, error)
	ListLeads(ctx context.Context, filter domain.LeadFilter) (*domain.PaginatedResult[domain.Lead], error)
	ReindexLead(ctx context.Context, id uuid.UUID) error
}

// leadServer реализует gRPC LeadServiceServer.
type leadServer struct {
	pb.UnimplementedLeadServiceServer
	leadService LeadService
}

// RegisterLeadServerGRPC регистрирует LeadServiceServer в gRPC сервере.
func RegisterLeadServerGRPC(server *grpc.Server, svc LeadService) {
	pb.RegisterLeadServiceServer(server, &leadServer{
		leadService: svc,
	})
}
