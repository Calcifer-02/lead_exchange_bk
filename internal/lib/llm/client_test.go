package llm

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"lead_exchange/internal/config"
)

func TestNewClient_Disabled(t *testing.T) {
	cfg := config.LLMConfig{
		Enabled: false,
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	c := NewClient(cfg, log)

	if c.IsEnabled() {
		t.Error("expected client to be disabled")
	}
}

func TestNewClient_Enabled(t *testing.T) {
	cfg := config.LLMConfig{
		Enabled: true,
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "test-key",
		Model:   "gpt-4",
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	c := NewClient(cfg, log)

	if !c.IsEnabled() {
		t.Error("expected client to be enabled")
	}
}

func TestNoopClient_GenerateListingContent(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	c := &noopClient{log: log}

	req := GenerateListingRequest{
		PropertyType:        "apartment",
		Address:             "ул. Тестовая, 1",
		ExistingTitle:       "Existing Title",
		ExistingDescription: "Existing Description",
	}

	resp, err := c.GenerateListingContent(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Title != req.ExistingTitle {
		t.Errorf("expected title %s, got %s", req.ExistingTitle, resp.Title)
	}
	if resp.Confidence != 0 {
		t.Errorf("expected confidence 0, got %f", resp.Confidence)
	}
}

func TestNoopClient_AnalyzeLeadIntent(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	c := &noopClient{log: log}

	req := AnalyzeLeadRequest{
		Title:       "Test Lead",
		Description: "Looking for apartment",
	}

	resp, err := c.AnalyzeLeadIntent(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Проверяем что возвращаются дефолтные веса
	if resp.RecommendedWeights.Price != 0.30 {
		t.Errorf("expected price weight 0.30, got %f", resp.RecommendedWeights.Price)
	}
	if resp.LeadType != "unknown" {
		t.Errorf("expected lead type 'unknown', got %s", resp.LeadType)
	}
}

func TestNoopClient_GenerateClarificationQuestions(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	c := &noopClient{log: log}

	req := ClarificationRequest{
		Title:         "Test",
		Description:   "Test",
		MissingFields: []string{"price", "district"},
	}

	resp, err := c.GenerateClarificationQuestions(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Questions) != 0 {
		t.Errorf("expected 0 questions, got %d", len(resp.Questions))
	}
	if resp.Priority != "low" {
		t.Errorf("expected priority 'low', got %s", resp.Priority)
	}
}

func TestNoopClient_EnrichDescription(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	c := &noopClient{log: log}

	req := EnrichDescriptionRequest{
		CurrentDescription: "Original description",
	}

	resp, err := c.EnrichDescription(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.EnrichedDescription != req.CurrentDescription {
		t.Errorf("expected description %s, got %s", req.CurrentDescription, resp.EnrichedDescription)
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "pure JSON",
			input:    `{"title": "test"}`,
			expected: `{"title": "test"}`,
		},
		{
			name:     "JSON with text before",
			input:    `Here is the response: {"title": "test"}`,
			expected: `{"title": "test"}`,
		},
		{
			name:     "JSON with text after",
			input:    `{"title": "test"} That's all.`,
			expected: `{"title": "test"}`,
		},
		{
			name:     "nested JSON",
			input:    `{"outer": {"inner": "value"}}`,
			expected: `{"outer": {"inner": "value"}}`,
		},
		{
			name:     "no JSON",
			input:    `just text`,
			expected: `just text`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			if result != tt.expected {
				t.Errorf("extractJSON(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestClient_GenerateListingContent_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.NotFound(w, r)
			return
		}

		response := ChatCompletionResponse{
			ID: "test-id",
			Choices: []struct {
				Message ChatMessage `json:"message"`
			}{
				{
					Message: ChatMessage{
						Role:    "assistant",
						Content: `{"title": "Уютная 2к квартира", "description": "Прекрасная квартира", "keywords": ["уютная", "центр"], "confidence": 0.9}`,
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	c := &client{
		httpClient: server.Client(),
		baseURL:    server.URL,
		apiKey:     "test-key",
		model:      "gpt-4",
		log:        log,
	}

	req := GenerateListingRequest{
		PropertyType: "apartment",
		Address:      "ул. Тестовая, 1",
		City:         "Москва",
	}

	resp, err := c.GenerateListingContent(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Title != "Уютная 2к квартира" {
		t.Errorf("expected title 'Уютная 2к квартира', got %s", resp.Title)
	}
	if resp.Confidence != 0.9 {
		t.Errorf("expected confidence 0.9, got %f", resp.Confidence)
	}
}

func TestClient_AnalyzeLeadIntent_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := ChatCompletionResponse{
			ID: "test-id",
			Choices: []struct {
				Message ChatMessage `json:"message"`
			}{
				{
					Message: ChatMessage{
						Role: "assistant",
						Content: `{
							"recommended_weights": {"price": 0.4, "district": 0.3, "rooms": 0.15, "area": 0.1, "semantic": 0.05},
							"extracted_criteria": {"target_price": 10000000},
							"lead_type": "budget_oriented",
							"confidence": 0.85,
							"explanation": "Client focused on budget"
						}`,
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	c := &client{
		httpClient: server.Client(),
		baseURL:    server.URL,
		apiKey:     "test-key",
		model:      "gpt-4",
		log:        log,
	}

	req := AnalyzeLeadRequest{
		Title:       "Ищу квартиру до 10 млн",
		Description: "Бюджет ограничен",
	}

	resp, err := c.AnalyzeLeadIntent(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.LeadType != "budget_oriented" {
		t.Errorf("expected lead type 'budget_oriented', got %s", resp.LeadType)
	}
	if resp.RecommendedWeights.Price != 0.4 {
		t.Errorf("expected price weight 0.4, got %f", resp.RecommendedWeights.Price)
	}
}

func TestClient_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	c := &client{
		httpClient: server.Client(),
		baseURL:    server.URL,
		apiKey:     "test-key",
		model:      "gpt-4",
		log:        log,
	}

	req := GenerateListingRequest{
		PropertyType: "apartment",
	}

	_, err := c.GenerateListingContent(context.Background(), req)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestBuildListingPrompt(t *testing.T) {
	price := int64(10000000)
	rooms := int32(3)
	area := float64(75.5)

	req := GenerateListingRequest{
		PropertyType:        "apartment",
		Address:             "ул. Ленина, 1",
		City:                "Москва",
		Price:               &price,
		Rooms:               &rooms,
		Area:                &area,
		Features:            []string{"балкон", "парковка"},
		ExistingTitle:       "Старый заголовок",
		ExistingDescription: "Старое описание",
	}

	prompt := buildListingPrompt(req)

	// Проверяем что все поля включены в промпт
	if !contains(prompt, "apartment") {
		t.Error("prompt should contain property type")
	}
	if !contains(prompt, "Москва") {
		t.Error("prompt should contain city")
	}
	if !contains(prompt, "10000000") {
		t.Error("prompt should contain price")
	}
	if !contains(prompt, "балкон") {
		t.Error("prompt should contain features")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

