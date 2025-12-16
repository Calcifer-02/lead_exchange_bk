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
func (m *MockPropertyRepository) ListProperties(ctx context.Context, filter domain.PropertyFilter) (*domain.PaginatedResult[domain.Property], error) {
	return &domain.PaginatedResult[domain.Property]{}, nil
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
func (m *MockPropertyRepository) MatchPropertiesWithHardFilters(ctx context.Context, leadEmbedding []float32, filter domain.PropertyFilter, hardFilters *domain.HardFilters, limit int) ([]domain.MatchedProperty, error) {
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

// TestCalcPriceScore тестирует расчёт score по цене.
func TestCalcPriceScore(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := &Service{log: log}

	tests := []struct {
		name        string
		objPrice    *int64
		targetPrice *int64
		wantMin     float64
		wantMax     float64
	}{
		{
			name:        "exact match",
			objPrice:    ptr[int64](10000000),
			targetPrice: ptr[int64](10000000),
			wantMin:     0.95,
			wantMax:     1.0,
		},
		{
			name:        "10% deviation",
			objPrice:    ptr[int64](11000000),
			targetPrice: ptr[int64](10000000),
			wantMin:     0.8,
			wantMax:     0.95,
		},
		{
			name:        "30% deviation",
			objPrice:    ptr[int64](13000000),
			targetPrice: ptr[int64](10000000),
			wantMin:     0.5,
			wantMax:     0.75,
		},
		{
			name:        "nil target",
			objPrice:    ptr[int64](10000000),
			targetPrice: nil,
			wantMin:     0.5,
			wantMax:     0.5,
		},
		{
			name:        "nil object price",
			objPrice:    nil,
			targetPrice: ptr[int64](10000000),
			wantMin:     0.5,
			wantMax:     0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var criteria *domain.SoftCriteria
			if tt.targetPrice != nil {
				criteria = &domain.SoftCriteria{TargetPrice: tt.targetPrice}
			}
			score := svc.calcPriceScore(tt.objPrice, criteria)
			if score < tt.wantMin || score > tt.wantMax {
				t.Errorf("calcPriceScore() = %v, want between %v and %v", score, tt.wantMin, tt.wantMax)
			}
		})
	}
}

// TestCalcRoomsScore тестирует расчёт score по комнатам.
func TestCalcRoomsScore(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := &Service{log: log}

	tests := []struct {
		name        string
		objRooms    *int32
		targetRooms *int32
		want        float64
	}{
		{"exact match", ptr[int32](3), ptr[int32](3), 1.0},
		{"diff 1", ptr[int32](3), ptr[int32](2), 0.6},
		{"diff 2", ptr[int32](4), ptr[int32](2), 0.3},
		{"diff 3+", ptr[int32](5), ptr[int32](1), 0.1},
		{"nil target", ptr[int32](3), nil, 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var criteria *domain.SoftCriteria
			if tt.targetRooms != nil {
				criteria = &domain.SoftCriteria{TargetRooms: tt.targetRooms}
			}
			score := svc.calcRoomsScore(tt.objRooms, criteria)
			if score != tt.want {
				t.Errorf("calcRoomsScore() = %v, want %v", score, tt.want)
			}
		})
	}
}

// TestCalcDistrictScore тестирует расчёт score по району.
func TestCalcDistrictScore(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := &Service{log: log}

	tests := []struct {
		name    string
		address string
		target  *string
		prefs   []string
		want    float64
	}{
		{"exact match", "Центральный район", ptr[string]("Центральный"), nil, 1.0},
		{"in preferred", "Арбат, Москва", nil, []string{"Арбат", "Тверской"}, 0.7},
		{"no match", "Бирюлёво", ptr[string]("Центр"), []string{"Арбат"}, 0.3},
		{"empty address", "", ptr[string]("Центр"), nil, 0.3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			criteria := &domain.SoftCriteria{
				TargetDistrict:     tt.target,
				PreferredDistricts: tt.prefs,
			}
			score := svc.calcDistrictScore(tt.address, criteria)
			if score != tt.want {
				t.Errorf("calcDistrictScore() = %v, want %v", score, tt.want)
			}
		})
	}
}

// TestRankMatches тестирует сортировку по взвешенному score.
func TestRankMatches(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := &Service{log: log}

	price1, price2, price3 := int64(10000000), int64(15000000), int64(8000000)
	rooms1, rooms2, rooms3 := int32(3), int32(2), int32(4)

	matches := []domain.MatchedProperty{
		{Property: domain.Property{ID: uuid.New(), Price: &price1, Rooms: &rooms1, Address: "Бирюлёво"}, Similarity: 0.6},
		{Property: domain.Property{ID: uuid.New(), Price: &price2, Rooms: &rooms2, Address: "Центр"}, Similarity: 0.9},
		{Property: domain.Property{ID: uuid.New(), Price: &price3, Rooms: &rooms3, Address: "Арбат"}, Similarity: 0.7},
	}

	weights := domain.MatchWeights{Price: 0.4, District: 0.3, Rooms: 0.2, Area: 0.0, Semantic: 0.1}.Normalize()
	criteria := &domain.SoftCriteria{
		TargetPrice:        ptr[int64](10000000),
		TargetRooms:        ptr[int32](3),
		TargetDistrict:     ptr[string]("Арбат"),
		PreferredDistricts: []string{"Центр"},
	}

	ranked := svc.rankMatches(matches, weights, criteria)

	// Все должны иметь TotalScore
	for i, m := range ranked {
		if m.TotalScore == nil {
			t.Errorf("match[%d] has nil TotalScore", i)
		}
	}

	// Должны быть отсортированы по убыванию TotalScore
	for i := 0; i < len(ranked)-1; i++ {
		if *ranked[i].TotalScore < *ranked[i+1].TotalScore {
			t.Errorf("not sorted: match[%d].TotalScore=%v < match[%d].TotalScore=%v",
				i, *ranked[i].TotalScore, i+1, *ranked[i+1].TotalScore)
		}
	}

	t.Logf("Ranked results:")
	for i, m := range ranked {
		t.Logf("  [%d] TotalScore=%.4f Price=%d Rooms=%d Address=%s Semantic=%.2f",
			i, *m.TotalScore, *m.Property.Price, *m.Property.Rooms, m.Property.Address, *m.SemanticScore)
	}
}

// TestWeightPresets тестирует пресеты весов.
func TestWeightPresets(t *testing.T) {
	presets := domain.GetWeightPresets()
	if len(presets) == 0 {
		t.Error("no weight presets")
	}

	for _, p := range presets {
		w := p.Weights.Normalize()
		sum := w.Price + w.District + w.Rooms + w.Area + w.Semantic
		if sum < 0.99 || sum > 1.01 {
			t.Errorf("preset %s: normalized sum = %v, want ~1.0", p.ID, sum)
		}
	}

	// Проверяем GetByID
	if preset := domain.GetWeightPresetByID("budget_first"); preset == nil {
		t.Error("GetWeightPresetByID('budget_first') returned nil")
	}
	if preset := domain.GetWeightPresetByID("nonexistent"); preset != nil {
		t.Error("GetWeightPresetByID('nonexistent') should return nil")
	}
}

func ptr[T any](v T) *T {
	return &v
}

