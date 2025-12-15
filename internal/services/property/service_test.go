package property

import (
	"context"
	"lead_exchange/internal/domain"
	"lead_exchange/internal/lib/ml"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
)

// MockPropertyRepository
type MockPropertyRepository struct {
	GetByIDFunc         func(ctx context.Context, id uuid.UUID) (domain.Property, error)
	UpdateEmbeddingFunc func(ctx context.Context, propertyID uuid.UUID, embedding []float32) error
}

func (m *MockPropertyRepository) CreateProperty(ctx context.Context, property domain.Property) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *MockPropertyRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Property, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return domain.Property{}, nil
}
func (m *MockPropertyRepository) UpdateProperty(ctx context.Context, propertyID uuid.UUID, update domain.PropertyFilter) error {
	return nil
}
func (m *MockPropertyRepository) ListProperties(ctx context.Context, filter domain.PropertyFilter) ([]domain.Property, error) {
	return nil, nil
}
func (m *MockPropertyRepository) UpdateEmbedding(ctx context.Context, propertyID uuid.UUID, embedding []float32) error {
	if m.UpdateEmbeddingFunc != nil {
		return m.UpdateEmbeddingFunc(ctx, propertyID, embedding)
	}
	return nil
}
func (m *MockPropertyRepository) MatchProperties(ctx context.Context, leadEmbedding []float32, filter domain.PropertyFilter, limit int) ([]domain.MatchedProperty, error) {
	return nil, nil
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

// MockLeadService
type MockLeadService struct{}

func (m *MockLeadService) GetLead(ctx context.Context, id uuid.UUID) (domain.Lead, error) {
	return domain.Lead{}, nil
}

func TestService_ReindexProperty(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	propertyID := uuid.New()

	repo := &MockPropertyRepository{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (domain.Property, error) {
			if id != propertyID {
				t.Errorf("expected propertyID %s, got %s", propertyID, id)
			}
			return domain.Property{
				ID:          propertyID,
				Title:       "Test Property",
				Description: "Test Description",
				Address:     "Test Address",
			}, nil
		},
		UpdateEmbeddingFunc: func(ctx context.Context, id uuid.UUID, embedding []float32) error {
			if id != propertyID {
				t.Errorf("expected propertyID %s, got %s", propertyID, id)
			}
			if len(embedding) != 2 {
				t.Errorf("expected embedding length 2, got %d", len(embedding))
			}
			return nil
		},
	}

	mlClient := &MockMLClient{
		ReindexFunc: func(ctx context.Context, req ml.ReindexRequest) (*ml.ReindexResponse, error) {
			if req.EntityID != propertyID.String() {
				t.Errorf("expected EntityID %s, got %s", propertyID.String(), req.EntityID)
			}
			return &ml.ReindexResponse{
				EntityID:  propertyID.String(),
				Embedding: []float64{0.1, 0.2},
			}, nil
		},
	}

	leadService := &MockLeadService{}

	svc := New(log, repo, mlClient, leadService)

	err := svc.ReindexProperty(context.Background(), propertyID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

