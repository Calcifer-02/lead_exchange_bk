package propertygrpc

import (
	"context"
	"lead_exchange/internal/domain"
	pb "lead_exchange/pkg"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

// PropertyService описывает бизнес-логику для работы с объектами недвижимости.
type PropertyService interface {
	CreateProperty(ctx context.Context, property domain.Property) (uuid.UUID, error)
	GetProperty(ctx context.Context, id uuid.UUID) (domain.Property, error)
	UpdateProperty(ctx context.Context, id uuid.UUID, update domain.PropertyFilter) (domain.Property, error)
	ListProperties(ctx context.Context, filter domain.PropertyFilter) (*domain.PaginatedResult[domain.Property], error)
	MatchProperties(ctx context.Context, leadID uuid.UUID, filter domain.PropertyFilter, limit int) ([]domain.MatchedProperty, error)
	MatchPropertiesWeighted(ctx context.Context, leadID uuid.UUID, filter domain.PropertyFilter, limit int, weights *domain.MatchWeights, criteria *domain.SoftCriteria, useWeightedRanking bool) ([]domain.MatchedProperty, error)
	ReindexProperty(ctx context.Context, id uuid.UUID) error
}

// propertyServer реализует gRPC PropertyServiceServer.
type propertyServer struct {
	pb.UnimplementedPropertyServiceServer
	propertyService PropertyService
}

// RegisterPropertyServerGRPC регистрирует PropertyServiceServer в gRPC сервере.
func RegisterPropertyServerGRPC(server *grpc.Server, svc PropertyService) {
	pb.RegisterPropertyServiceServer(server, &propertyServer{
		propertyService: svc,
	})
}
