package propertygrpc

import (
	"context"
	"lead_exchange/internal/domain"
	"lead_exchange/internal/lib/llm"
	"lead_exchange/internal/lib/vision"
	pb "lead_exchange/pkg"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

// LLMClient — интерфейс для LLM клиента (используется для type assertion в grpc/app.go).
type LLMClient = llm.Client

// VisionClient — интерфейс для Vision клиента (используется для type assertion в grpc/app.go).
type VisionClient = vision.Client

// PropertyService описывает бизнес-логику для работы с объектами недвижимости.
type PropertyService interface {
	CreateProperty(ctx context.Context, property domain.Property) (uuid.UUID, error)
	GetProperty(ctx context.Context, id uuid.UUID) (domain.Property, error)
	UpdateProperty(ctx context.Context, id uuid.UUID, update domain.PropertyFilter) (domain.Property, error)
	ListProperties(ctx context.Context, filter domain.PropertyFilter) (*domain.PaginatedResult[domain.Property], error)
	MatchProperties(ctx context.Context, leadID uuid.UUID, filter domain.PropertyFilter, limit int) ([]domain.MatchedProperty, error)
	MatchPropertiesWeighted(ctx context.Context, leadID uuid.UUID, filter domain.PropertyFilter, limit int, weights *domain.MatchWeights, criteria *domain.SoftCriteria, useWeightedRanking bool) ([]domain.MatchedProperty, error)
	MatchPropertiesAdvanced(ctx context.Context, leadID uuid.UUID, filter domain.PropertyFilter, limit int) ([]domain.MatchedProperty, error)
	ReindexProperty(ctx context.Context, id uuid.UUID) error
}

// serverAPI реализует gRPC PropertyServiceServer с поддержкой AI-функций.
type serverAPI struct {
	pb.UnimplementedPropertyServiceServer
	propertyService PropertyService
	llmClient       llm.Client
	visionClient    vision.Client
}

// ServerOption — опция для конфигурации сервера.
type ServerOption func(*serverAPI)

// WithLLMClient добавляет LLM клиент.
func WithLLMClient(client llm.Client) ServerOption {
	return func(s *serverAPI) {
		s.llmClient = client
	}
}

// WithVisionClient добавляет Vision клиент.
func WithVisionClient(client vision.Client) ServerOption {
	return func(s *serverAPI) {
		s.visionClient = client
	}
}

// RegisterPropertyServerGRPC регистрирует PropertyServiceServer в gRPC сервере.
func RegisterPropertyServerGRPC(server *grpc.Server, svc PropertyService, opts ...ServerOption) {
	s := &serverAPI{
		propertyService: svc,
	}

	for _, opt := range opts {
		opt(s)
	}

	pb.RegisterPropertyServiceServer(server, s)
}

// Для обратной совместимости
type propertyServer = serverAPI
