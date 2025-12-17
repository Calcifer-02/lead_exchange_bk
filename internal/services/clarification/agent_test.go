package clarification

import (
	"context"
	"encoding/json"
	"lead_exchange/internal/config"
	"lead_exchange/internal/domain"
	"lead_exchange/internal/lib/llm"
	"lead_exchange/internal/services/weights"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
)

// MockLLMClient — мок для LLM клиента.
type MockLLMClient struct {
	GenerateClarificationQuestionsFunc func(ctx context.Context, req llm.ClarificationRequest) (*llm.ClarificationResponse, error)
	IsEnabledValue                     bool
}

func (m *MockLLMClient) GenerateListingContent(ctx context.Context, req llm.GenerateListingRequest) (*llm.GenerateListingResponse, error) {
	return nil, nil
}

func (m *MockLLMClient) AnalyzeLeadIntent(ctx context.Context, req llm.AnalyzeLeadRequest) (*llm.AnalyzeLeadResponse, error) {
	return &llm.AnalyzeLeadResponse{
		RecommendedWeights: llm.WeightRecommendation{
			Price: 0.3, District: 0.25, Rooms: 0.2, Area: 0.1, Semantic: 0.15,
		},
		LeadType:   "unknown",
		Confidence: 0.5,
	}, nil
}

func (m *MockLLMClient) GenerateClarificationQuestions(ctx context.Context, req llm.ClarificationRequest) (*llm.ClarificationResponse, error) {
	if m.GenerateClarificationQuestionsFunc != nil {
		return m.GenerateClarificationQuestionsFunc(ctx, req)
	}
	return &llm.ClarificationResponse{
		Questions: []llm.ClarificationQuestion{},
		Priority:  "low",
	}, nil
}

func (m *MockLLMClient) EnrichDescription(ctx context.Context, req llm.EnrichDescriptionRequest) (*llm.EnrichDescriptionResponse, error) {
	return nil, nil
}

func (m *MockLLMClient) IsEnabled() bool {
	return m.IsEnabledValue
}

func TestAgent_AnalyzeShortLead_NeedsClarification(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	llmClient := &MockLLMClient{IsEnabledValue: false}
	cfg := config.SearchConfig{DynamicWeightsEnabled: false}
	weightsAnalyzer := weights.NewAnalyzer(log, llmClient, cfg)

	agent := NewAgent(log, llmClient, weightsAnalyzer)

	// Короткий лид без деталей
	lead := domain.Lead{
		ID:          uuid.New(),
		Title:       "Квартира",
		Description: "хочу купить",
	}

	result, err := agent.AnalyzeAndGenerateQuestions(context.Background(), lead)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.NeedsClarification {
		t.Error("expected NeedsClarification to be true for short lead")
	}

	if len(result.Questions) == 0 {
		t.Error("expected at least one question for short lead")
	}

	if result.LeadQualityScore >= 0.5 {
		t.Errorf("expected low quality score for short lead, got %f", result.LeadQualityScore)
	}
}

func TestAgent_AnalyzeCompleteLead_NoClarification(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	llmClient := &MockLLMClient{IsEnabledValue: false}
	cfg := config.SearchConfig{DynamicWeightsEnabled: false}
	weightsAnalyzer := weights.NewAnalyzer(log, llmClient, cfg)

	agent := NewAgent(log, llmClient, weightsAnalyzer)

	// Полный лид со всеми деталями
	requirement := map[string]interface{}{
		"price":      float64(15000000),
		"district":   "Центральный",
		"roomNumber": float64(3),
		"area":       float64(80),
	}
	reqJSON, _ := json.Marshal(requirement)

	lead := domain.Lead{
		ID:          uuid.New(),
		Title:       "Ищу 3-комнатную квартиру в центре",
		Description: "Семья из 4 человек, нужна квартира около 80 кв.м. Бюджет до 15 млн. Важно чтобы был балкон и парковка.",
		Requirement: reqJSON,
	}

	result, err := agent.AnalyzeAndGenerateQuestions(context.Background(), lead)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.NeedsClarification {
		t.Error("expected NeedsClarification to be false for complete lead")
	}

	if result.Priority != "low" {
		t.Errorf("expected priority 'low', got %s", result.Priority)
	}
}

func TestAgent_GenerateQuestionsWithLLM(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	llmClient := &MockLLMClient{
		IsEnabledValue: true,
		GenerateClarificationQuestionsFunc: func(ctx context.Context, req llm.ClarificationRequest) (*llm.ClarificationResponse, error) {
			return &llm.ClarificationResponse{
				Questions: []llm.ClarificationQuestion{
					{
						Field:            "price",
						Question:         "Какой у вас примерный бюджет?",
						QuestionType:     "range",
						SuggestedOptions: []string{"до 5 млн", "5-10 млн", "10-15 млн"},
						Importance:       "required",
					},
					{
						Field:        "district",
						Question:     "В каком районе вы хотите жить?",
						QuestionType: "open",
						Importance:   "recommended",
					},
				},
				Priority: "high",
			}, nil
		},
	}
	cfg := config.SearchConfig{DynamicWeightsEnabled: false}
	weightsAnalyzer := weights.NewAnalyzer(log, llmClient, cfg)

	agent := NewAgent(log, llmClient, weightsAnalyzer)

	lead := domain.Lead{
		ID:          uuid.New(),
		Title:       "Квартира",
		Description: "хочу",
	}

	result, err := agent.AnalyzeAndGenerateQuestions(context.Background(), lead)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Questions) != 2 {
		t.Errorf("expected 2 questions from LLM, got %d", len(result.Questions))
	}

	// Проверяем что первый вопрос о бюджете
	if len(result.Questions) > 0 && result.Questions[0].Field != "price" {
		t.Errorf("expected first question field 'price', got %s", result.Questions[0].Field)
	}
}

func TestAgent_FallbackQuestions(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	llmClient := &MockLLMClient{IsEnabledValue: false}
	cfg := config.SearchConfig{DynamicWeightsEnabled: false}
	weightsAnalyzer := weights.NewAnalyzer(log, llmClient, cfg)

	agent := NewAgent(log, llmClient, weightsAnalyzer)

	// Тестируем генерацию fallback вопросов
	missingFields := []string{"price", "district", "roomNumber"}
	questions := agent.generateFallbackQuestions(missingFields)

	if len(questions) == 0 {
		t.Error("expected fallback questions to be generated")
	}

	// Проверяем что вопросы соответствуют missing fields
	fieldSet := make(map[string]bool)
	for _, q := range questions {
		fieldSet[q.Field] = true
	}

	for _, field := range missingFields {
		if !fieldSet[field] {
			t.Errorf("expected question for missing field %s", field)
		}
	}
}

func TestAgent_CalculateLeadQuality(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	llmClient := &MockLLMClient{IsEnabledValue: false}
	cfg := config.SearchConfig{DynamicWeightsEnabled: false}
	weightsAnalyzer := weights.NewAnalyzer(log, llmClient, cfg)

	agent := NewAgent(log, llmClient, weightsAnalyzer)

	tests := []struct {
		name          string
		lead          domain.Lead
		missingFields []string
		minQuality    float64
		maxQuality    float64
	}{
		{
			name: "complete lead",
			lead: domain.Lead{
				Title:       "3к квартира в центре до 15 млн",
				Description: "Семья из 4 человек, нужна квартира с ремонтом около 80 кв.м",
			},
			missingFields: []string{},
			minQuality:    0.6,
			maxQuality:    1.0,
		},
		{
			name: "minimal lead",
			lead: domain.Lead{
				Title:       "кв",
				Description: "",
			},
			missingFields: []string{"price", "district", "rooms", "area"},
			minQuality:    0.0,
			maxQuality:    0.3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := agent.calculateLeadQuality(tt.lead, tt.missingFields)
			if score < tt.minQuality || score > tt.maxQuality {
				t.Errorf("calculateLeadQuality() = %f, want between %f and %f", score, tt.minQuality, tt.maxQuality)
			}
		})
	}
}

func TestAgent_DeterminePriority(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	llmClient := &MockLLMClient{IsEnabledValue: false}
	cfg := config.SearchConfig{DynamicWeightsEnabled: false}
	weightsAnalyzer := weights.NewAnalyzer(log, llmClient, cfg)

	agent := NewAgent(log, llmClient, weightsAnalyzer)

	tests := []struct {
		name          string
		missingFields []string
		qualityScore  float64
		expected      string
	}{
		{
			name:          "many missing with low quality",
			missingFields: []string{"price", "district", "rooms", "area"},
			qualityScore:  0.2,
			expected:      "high",
		},
		{
			name:          "few missing with medium quality",
			missingFields: []string{"price"},
			qualityScore:  0.6,
			expected:      "medium",
		},
		{
			name:          "none missing with high quality",
			missingFields: []string{},
			qualityScore:  0.9,
			expected:      "low",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			priority := agent.determinePriority(tt.missingFields, tt.qualityScore)
			if priority != tt.expected {
				t.Errorf("determinePriority() = %s, want %s", priority, tt.expected)
			}
		})
	}
}

