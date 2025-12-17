package reranker

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
	cfg := config.RerankerConfig{
		Enabled: false,
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	c := NewClient(cfg, log)

	if c.IsEnabled() {
		t.Error("expected client to be disabled")
	}
}

func TestNewClient_Enabled(t *testing.T) {
	cfg := config.RerankerConfig{
		Enabled: true,
		BaseURL: "https://api.example.com",
		APIKey:  "test-key",
		Model:   "test-model",
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	c := NewClient(cfg, log)

	if !c.IsEnabled() {
		t.Error("expected client to be enabled")
	}
}

func TestNoopClient_Rerank(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	c := &noopClient{log: log}

	req := RerankRequest{
		Query:     "test query",
		Documents: []string{"doc1", "doc2", "doc3"},
		TopN:      3,
	}

	resp, err := c.Rerank(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Results) != 3 {
		t.Errorf("expected 3 results, got %d", len(resp.Results))
	}

	// Проверяем что результаты в исходном порядке
	for i, r := range resp.Results {
		if r.Index != i {
			t.Errorf("expected index %d, got %d", i, r.Index)
		}
		if r.Document != req.Documents[i] {
			t.Errorf("expected document %s, got %s", req.Documents[i], r.Document)
		}
	}
}

func TestClient_Rerank_Success(t *testing.T) {
	// Создаём мок сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rerank" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Errorf("unexpected auth header: %s", auth)
		}

		response := RerankResponse{
			Results: []RerankResult{
				{Index: 1, RelevanceScore: 0.95, Document: "doc2"},
				{Index: 0, RelevanceScore: 0.85, Document: "doc1"},
				{Index: 2, RelevanceScore: 0.70, Document: "doc3"},
			},
			Model: "test-model",
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
		model:      "test-model",
		log:        log,
	}

	req := RerankRequest{
		Query:     "test query",
		Documents: []string{"doc1", "doc2", "doc3"},
		TopN:      3,
	}

	resp, err := c.Rerank(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Results) != 3 {
		t.Errorf("expected 3 results, got %d", len(resp.Results))
	}

	// Проверяем что первый результат имеет индекс 1 (переранжирован)
	if resp.Results[0].Index != 1 {
		t.Errorf("expected first result index 1, got %d", resp.Results[0].Index)
	}

	if resp.Results[0].RelevanceScore != 0.95 {
		t.Errorf("expected relevance score 0.95, got %f", resp.Results[0].RelevanceScore)
	}
}

func TestClient_Rerank_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	c := &client{
		httpClient: server.Client(),
		baseURL:    server.URL,
		apiKey:     "test-key",
		model:      "test-model",
		log:        log,
	}

	req := RerankRequest{
		Query:     "test query",
		Documents: []string{"doc1"},
	}

	_, err := c.Rerank(context.Background(), req)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestNewJinaClient_EmptyAPIKey(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	c := NewJinaClient("", log)

	if c.IsEnabled() {
		t.Error("expected client to be disabled with empty API key")
	}
}

func TestNewJinaClient_WithAPIKey(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	c := NewJinaClient("test-key", log)

	if !c.IsEnabled() {
		t.Error("expected client to be enabled with API key")
	}
}

