package lead

import (
	"context"
	"lead_exchange/internal/domain"
	"lead_exchange/internal/lib/ml"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
)

// MockLeadRepository
type MockLeadRepository struct {
	GetByIDFunc         func(ctx context.Context, id uuid.UUID) (domain.Lead, error)
	UpdateEmbeddingFunc func(ctx context.Context, leadID uuid.UUID, embedding []float32) error
	// other methods not needed for this test
}

func (m *MockLeadRepository) CreateLead(ctx context.Context, lead domain.Lead) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *MockLeadRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Lead, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return domain.Lead{}, nil
}
func (m *MockLeadRepository) UpdateLead(ctx context.Context, leadID uuid.UUID, update domain.LeadFilter) error {
	return nil
}
func (m *MockLeadRepository) ListLeads(ctx context.Context, filter domain.LeadFilter) (*domain.PaginatedResult[domain.Lead], error) {
	return &domain.PaginatedResult[domain.Lead]{}, nil
}
func (m *MockLeadRepository) UpdateEmbedding(ctx context.Context, leadID uuid.UUID, embedding []float32) error {
	if m.UpdateEmbeddingFunc != nil {
		return m.UpdateEmbeddingFunc(ctx, leadID, embedding)
	}
	return nil
}

// MockMLClient
type MockMLClient struct {
	ReindexFunc func(ctx context.Context, req ml.ReindexRequest) (*ml.ReindexResponse, error)
}

func (m *MockMLClient) PrepareAndEmbed(ctx context.Context, req ml.PrepareAndEmbedRequest) (*ml.PrepareAndEmbedResponse, error) {
	return nil, nil
}
func (m *MockMLClient) Reindex(ctx context.Context, req ml.ReindexRequest) (*ml.ReindexResponse, error) {
	if m.ReindexFunc != nil {
		return m.ReindexFunc(ctx, req)
	}
	return nil, nil
}
func (m *MockMLClient) ReindexBatch(ctx context.Context, req ml.ReindexBatchRequest) (*ml.ReindexBatchResponse, error) {
	return nil, nil
}
func (m *MockMLClient) GetModelInfo(ctx context.Context) (*ml.ModelInfo, error) {
	return nil, nil
}

func TestService_ReindexLead(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	leadID := uuid.New()

	repo := &MockLeadRepository{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (domain.Lead, error) {
			if id != leadID {
				t.Errorf("expected leadID %s, got %s", leadID, id)
			}
			return domain.Lead{
				ID:          leadID,
				Title:       "Test Lead",
				Description: "Test Description",
			}, nil
		},
		UpdateEmbeddingFunc: func(ctx context.Context, id uuid.UUID, embedding []float32) error {
			if id != leadID {
				t.Errorf("expected leadID %s, got %s", leadID, id)
			}
			if len(embedding) != 2 {
				t.Errorf("expected embedding length 2, got %d", len(embedding))
			}
			return nil
		},
	}

	mlClient := &MockMLClient{
		ReindexFunc: func(ctx context.Context, req ml.ReindexRequest) (*ml.ReindexResponse, error) {
			if req.EntityID != leadID.String() {
				t.Errorf("expected EntityID %s, got %s", leadID.String(), req.EntityID)
			}
			return &ml.ReindexResponse{
				EntityID:  leadID.String(),
				Embedding: []float64{0.1, 0.2},
			}, nil
		},
	}

	svc := New(log, repo, mlClient)

	err := svc.ReindexLead(context.Background(), leadID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

