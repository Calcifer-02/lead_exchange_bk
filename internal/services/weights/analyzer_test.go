package weights

import (
	"context"
	"encoding/json"
	"lead_exchange/internal/config"
	"lead_exchange/internal/domain"
	"lead_exchange/internal/lib/llm"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
)

// MockLLMClient — мок для LLM клиента.
type MockLLMClient struct {
	AnalyzeLeadIntentFunc func(ctx context.Context, req llm.AnalyzeLeadRequest) (*llm.AnalyzeLeadResponse, error)
	IsEnabledValue        bool
}

func (m *MockLLMClient) GenerateListingContent(ctx context.Context, req llm.GenerateListingRequest) (*llm.GenerateListingResponse, error) {
	return nil, nil
}

func (m *MockLLMClient) AnalyzeLeadIntent(ctx context.Context, req llm.AnalyzeLeadRequest) (*llm.AnalyzeLeadResponse, error) {
	if m.AnalyzeLeadIntentFunc != nil {
		return m.AnalyzeLeadIntentFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockLLMClient) GenerateClarificationQuestions(ctx context.Context, req llm.ClarificationRequest) (*llm.ClarificationResponse, error) {
	return nil, nil
}

func (m *MockLLMClient) EnrichDescription(ctx context.Context, req llm.EnrichDescriptionRequest) (*llm.EnrichDescriptionResponse, error) {
	return nil, nil
}

func (m *MockLLMClient) IsEnabled() bool {
	return m.IsEnabledValue
}

func TestAnalyzer_HeuristicAnalysis_BudgetOriented(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	llmClient := &MockLLMClient{IsEnabledValue: false}
	cfg := config.SearchConfig{DynamicWeightsEnabled: true}

	analyzer := NewAnalyzer(log, llmClient, cfg)

	lead := domain.Lead{
		ID:          uuid.New(),
		Title:       "Ищу квартиру недорого",
		Description: "Бюджет ограничен, максимум 5 млн",
	}

	result, err := analyzer.AnalyzeLead(context.Background(), lead)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.LeadType != "budget_oriented" {
		t.Errorf("expected lead type 'budget_oriented', got %s", result.LeadType)
	}

	// Для бюджет-ориентированного лида вес цены должен быть высоким
	if result.Weights.Price < 0.3 {
		t.Errorf("expected price weight >= 0.3 for budget_oriented, got %f", result.Weights.Price)
	}

	if result.UsedLLM {
		t.Error("expected UsedLLM to be false")
	}
}

func TestAnalyzer_HeuristicAnalysis_LocationOriented(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	llmClient := &MockLLMClient{IsEnabledValue: false}
	cfg := config.SearchConfig{DynamicWeightsEnabled: true}

	analyzer := NewAnalyzer(log, llmClient, cfg)

	lead := domain.Lead{
		ID:          uuid.New(),
		Title:       "Квартира в центре",
		Description: "Хочу рядом с метро Арбат, район Арбат",
	}

	result, err := analyzer.AnalyzeLead(context.Background(), lead)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.LeadType != "location_oriented" {
		t.Errorf("expected lead type 'location_oriented', got %s", result.LeadType)
	}

	// Для локация-ориентированного лида вес района должен быть высоким
	if result.Weights.District < 0.3 {
		t.Errorf("expected district weight >= 0.3 for location_oriented, got %f", result.Weights.District)
	}
}

func TestAnalyzer_HeuristicAnalysis_FamilyOriented(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	llmClient := &MockLLMClient{IsEnabledValue: false}
	cfg := config.SearchConfig{DynamicWeightsEnabled: true}

	analyzer := NewAnalyzer(log, llmClient, cfg)

	lead := domain.Lead{
		ID:          uuid.New(),
		Title:       "Квартира для семьи",
		Description: "Двое детей, нужна школа рядом, большая площадь",
	}

	result, err := analyzer.AnalyzeLead(context.Background(), lead)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.LeadType != "family_oriented" {
		t.Errorf("expected lead type 'family_oriented', got %s", result.LeadType)
	}

	// Для семейного лида вес комнат и площади должен быть высоким
	if result.Weights.Rooms < 0.2 {
		t.Errorf("expected rooms weight >= 0.2 for family_oriented, got %f", result.Weights.Rooms)
	}
}

func TestAnalyzer_LLMAnalysis_Success(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	targetPrice := int64(10000000)
	llmClient := &MockLLMClient{
		IsEnabledValue: true,
		AnalyzeLeadIntentFunc: func(ctx context.Context, req llm.AnalyzeLeadRequest) (*llm.AnalyzeLeadResponse, error) {
			return &llm.AnalyzeLeadResponse{
				RecommendedWeights: llm.WeightRecommendation{
					Price:    0.4,
					District: 0.25,
					Rooms:    0.15,
					Area:     0.1,
					Semantic: 0.1,
				},
				ExtractedCriteria: llm.ExtractedCriteria{
					TargetPrice: &targetPrice,
				},
				LeadType:    "budget_oriented",
				Confidence:  0.9,
				Explanation: "Client is focused on budget",
			}, nil
		},
	}
	cfg := config.SearchConfig{DynamicWeightsEnabled: true}

	analyzer := NewAnalyzer(log, llmClient, cfg)

	lead := domain.Lead{
		ID:          uuid.New(),
		Title:       "Test Lead",
		Description: "Looking for apartment under 10 million",
	}

	result, err := analyzer.AnalyzeLead(context.Background(), lead)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.UsedLLM {
		t.Error("expected UsedLLM to be true")
	}

	if result.LeadType != "budget_oriented" {
		t.Errorf("expected lead type 'budget_oriented', got %s", result.LeadType)
	}

	if result.Confidence != 0.9 {
		t.Errorf("expected confidence 0.9, got %f", result.Confidence)
	}

	if result.Criteria == nil {
		t.Error("expected criteria to be set")
	} else if result.Criteria.TargetPrice == nil || *result.Criteria.TargetPrice != 10000000 {
		t.Error("expected target price to be 10000000")
	}
}

func TestAnalyzer_DynamicWeightsDisabled(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	llmClient := &MockLLMClient{IsEnabledValue: true}
	cfg := config.SearchConfig{DynamicWeightsEnabled: false}

	analyzer := NewAnalyzer(log, llmClient, cfg)

	lead := domain.Lead{
		ID:          uuid.New(),
		Title:       "Test Lead",
		Description: "Looking for apartment",
	}

	result, err := analyzer.AnalyzeLead(context.Background(), lead)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Когда динамические веса отключены, должен использоваться эвристический анализ
	if result.UsedLLM {
		t.Error("expected UsedLLM to be false when dynamic weights disabled")
	}
}

func TestAnalyzer_ExtractCriteriaFromRequirement(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	llmClient := &MockLLMClient{IsEnabledValue: false}
	cfg := config.SearchConfig{DynamicWeightsEnabled: true}

	analyzer := NewAnalyzer(log, llmClient, cfg)

	requirement := map[string]interface{}{
		"price":      float64(15000000),
		"district":   "Центр",
		"roomNumber": float64(3),
		"area":       float64(80),
	}
	reqJSON, _ := json.Marshal(requirement)

	lead := domain.Lead{
		ID:          uuid.New(),
		Title:       "Test Lead",
		Description: "Test",
		Requirement: reqJSON,
	}

	result, err := analyzer.AnalyzeLead(context.Background(), lead)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Criteria == nil {
		t.Fatal("expected criteria to be set")
	}

	if result.Criteria.TargetPrice == nil || *result.Criteria.TargetPrice != 15000000 {
		t.Errorf("expected target price 15000000, got %v", result.Criteria.TargetPrice)
	}

	if result.Criteria.TargetDistrict == nil || *result.Criteria.TargetDistrict != "Центр" {
		t.Errorf("expected target district 'Центр', got %v", result.Criteria.TargetDistrict)
	}
}

func TestMatchWeights_Normalize(t *testing.T) {
	weights := domain.MatchWeights{
		Price:    0.5,
		District: 0.5,
		Rooms:    0.5,
		Area:     0.5,
		Semantic: 0.5,
	}

	normalized := weights.Normalize()

	sum := normalized.Price + normalized.District + normalized.Rooms + normalized.Area + normalized.Semantic
	if sum < 0.99 || sum > 1.01 {
		t.Errorf("expected normalized sum to be ~1.0, got %f", sum)
	}

	// Все веса должны быть равны после нормализации
	expected := 0.2
	if normalized.Price != expected {
		t.Errorf("expected price %f, got %f", expected, normalized.Price)
	}
}

func TestDefaultWeights(t *testing.T) {
	weights := domain.DefaultWeights()

	sum := weights.Price + weights.District + weights.Rooms + weights.Area + weights.Semantic
	if sum < 0.99 || sum > 1.01 {
		t.Errorf("expected default weights sum to be ~1.0, got %f", sum)
	}
}

