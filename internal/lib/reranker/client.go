package reranker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"lead_exchange/internal/config"
	"log/slog"
)

// Client — клиент для взаимодействия с Reranker API (Jina AI, Cohere и др.).
type Client interface {
	// Rerank переранжирует документы относительно запроса.
	Rerank(ctx context.Context, req RerankRequest) (*RerankResponse, error)
	// IsEnabled проверяет, включен ли сервис.
	IsEnabled() bool
}

// RerankRequest — запрос на переранжирование документов.
type RerankRequest struct {
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopN      int      `json:"top_n,omitempty"`
	Model     string   `json:"model,omitempty"`
}

// RerankResponse — ответ от reranker API.
type RerankResponse struct {
	Results []RerankResult `json:"results"`
	Model   string         `json:"model,omitempty"`
	Usage   *Usage         `json:"usage,omitempty"`
}

// RerankResult — один результат переранжирования.
type RerankResult struct {
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
	Document       string  `json:"document,omitempty"`
}

// Usage — информация об использовании API.
type Usage struct {
	TotalTokens int `json:"total_tokens,omitempty"`
}

type client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	model      string
	log        *slog.Logger
}

// NewClient создаёт новый клиент для Reranker API.
func NewClient(cfg config.RerankerConfig, log *slog.Logger) Client {
	if !cfg.Enabled {
		return &noopClient{log: log}
	}

	return &client{
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		baseURL: cfg.BaseURL,
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
		log:     log,
	}
}

// Rerank отправляет запрос на переранжирование.
func (c *client) Rerank(ctx context.Context, req RerankRequest) (*RerankResponse, error) {
	const op = "reranker.Client.Rerank"

	if req.Model == "" {
		req.Model = c.model
	}

	url := fmt.Sprintf("%s/rerank", c.baseURL)

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to marshal request: %w", op, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("%s: failed to create request: %w", op, err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	c.log.Debug("sending rerank request",
		slog.Int("documents_count", len(req.Documents)),
		slog.Int("top_n", req.TopN),
	)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to send request: %w", op, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%s: unexpected status code %d: %s", op, resp.StatusCode, string(body))
	}

	var result RerankResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("%s: failed to decode response: %w", op, err)
	}

	c.log.Debug("rerank completed",
		slog.Int("results_count", len(result.Results)),
	)

	return &result, nil
}

func (c *client) IsEnabled() bool {
	return true
}

// noopClient — заглушка для случая, когда Reranker отключен.
type noopClient struct {
	log *slog.Logger
}

func (c *noopClient) Rerank(ctx context.Context, req RerankRequest) (*RerankResponse, error) {
	c.log.Debug("reranker is disabled, returning original order")

	// Возвращаем результаты в исходном порядке с убывающими score
	results := make([]RerankResult, len(req.Documents))
	for i, doc := range req.Documents {
		results[i] = RerankResult{
			Index:          i,
			RelevanceScore: 1.0 - float64(i)*0.01, // Убывающий score
			Document:       doc,
		}
	}

	return &RerankResponse{
		Results: results,
		Model:   "disabled",
	}, nil
}

func (c *noopClient) IsEnabled() bool {
	return false
}

// JinaRerankRequest — специфичный формат для Jina AI Reranker.
type JinaRerankRequest struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopN      int      `json:"top_n,omitempty"`
}

// NewJinaClient создаёт клиент для Jina AI Reranker.
func NewJinaClient(apiKey string, log *slog.Logger) Client {
	if apiKey == "" {
		return &noopClient{log: log}
	}

	return &client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://api.jina.ai/v1",
		apiKey:  apiKey,
		model:   "jina-reranker-v2-base-multilingual",
		log:     log,
	}
}

