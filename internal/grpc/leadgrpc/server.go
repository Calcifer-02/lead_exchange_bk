package leadgrpc

import (
	"context"
	"lead_exchange/internal/domain"
	"lead_exchange/internal/services/clarification"
	"lead_exchange/internal/services/weights"
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

// serverAPI реализует gRPC LeadServiceServer с поддержкой AI-функций.
type serverAPI struct {
	pb.UnimplementedLeadServiceServer
	leadService        LeadService
	clarificationAgent *clarification.Agent
	weightsAnalyzer    *weights.Analyzer
}

// ServerOption — опция для конфигурации сервера.
type ServerOption func(*serverAPI)

// WithClarificationAgent добавляет агента уточнения.
func WithClarificationAgent(agent *clarification.Agent) ServerOption {
	return func(s *serverAPI) {
		s.clarificationAgent = agent
	}
}

// WithWeightsAnalyzer добавляет анализатор весов.
func WithWeightsAnalyzer(analyzer *weights.Analyzer) ServerOption {
	return func(s *serverAPI) {
		s.weightsAnalyzer = analyzer
	}
}

// RegisterLeadServerGRPC регистрирует LeadServiceServer в gRPC сервере.
func RegisterLeadServerGRPC(server *grpc.Server, svc LeadService, opts ...ServerOption) {
	s := &serverAPI{
		leadService: svc,
	}

	for _, opt := range opts {
		opt(s)
	}

	pb.RegisterLeadServiceServer(server, s)
}

// Для обратной совместимости (старый leadServer удалён, используем serverAPI)
type leadServer = serverAPI
