package metrics

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// AIMetrics — метрики для AI-вызовов (LLM, Reranker, Vision).
type AIMetrics struct {
	mu sync.RWMutex
	log *slog.Logger

	// Счётчики вызовов
	llmCallsTotal       int64
	rerankerCallsTotal  int64
	visionCallsTotal    int64
	embeddingCallsTotal int64

	// Счётчики ошибок
	llmErrorsTotal       int64
	rerankerErrorsTotal  int64
	visionErrorsTotal    int64
	embeddingErrorsTotal int64

	// Суммарная задержка (для расчёта среднего)
	llmLatencyTotalMs       int64
	rerankerLatencyTotalMs  int64
	visionLatencyTotalMs    int64
	embeddingLatencyTotalMs int64

	// Последние задержки (для мониторинга)
	llmLastLatencyMs       int64
	rerankerLastLatencyMs  int64
	visionLastLatencyMs    int64
	embeddingLastLatencyMs int64

	// Счётчики токенов (для LLM)
	llmTokensUsedTotal int64
}

var (
	globalMetrics *AIMetrics
	metricsOnce   sync.Once
)

// GetAIMetrics возвращает глобальный экземпляр метрик.
func GetAIMetrics(log *slog.Logger) *AIMetrics {
	metricsOnce.Do(func() {
		globalMetrics = &AIMetrics{log: log}
	})
	return globalMetrics
}

// ServiceType — тип AI-сервиса.
type ServiceType string

const (
	ServiceLLM       ServiceType = "llm"
	ServiceReranker  ServiceType = "reranker"
	ServiceVision    ServiceType = "vision"
	ServiceEmbedding ServiceType = "embedding"
)

// RecordCall записывает вызов AI-сервиса.
func (m *AIMetrics) RecordCall(service ServiceType, latency time.Duration, err error, tokensUsed int) {
	latencyMs := latency.Milliseconds()

	switch service {
	case ServiceLLM:
		atomic.AddInt64(&m.llmCallsTotal, 1)
		atomic.AddInt64(&m.llmLatencyTotalMs, latencyMs)
		atomic.StoreInt64(&m.llmLastLatencyMs, latencyMs)
		if tokensUsed > 0 {
			atomic.AddInt64(&m.llmTokensUsedTotal, int64(tokensUsed))
		}
		if err != nil {
			atomic.AddInt64(&m.llmErrorsTotal, 1)
		}
	case ServiceReranker:
		atomic.AddInt64(&m.rerankerCallsTotal, 1)
		atomic.AddInt64(&m.rerankerLatencyTotalMs, latencyMs)
		atomic.StoreInt64(&m.rerankerLastLatencyMs, latencyMs)
		if err != nil {
			atomic.AddInt64(&m.rerankerErrorsTotal, 1)
		}
	case ServiceVision:
		atomic.AddInt64(&m.visionCallsTotal, 1)
		atomic.AddInt64(&m.visionLatencyTotalMs, latencyMs)
		atomic.StoreInt64(&m.visionLastLatencyMs, latencyMs)
		if err != nil {
			atomic.AddInt64(&m.visionErrorsTotal, 1)
		}
	case ServiceEmbedding:
		atomic.AddInt64(&m.embeddingCallsTotal, 1)
		atomic.AddInt64(&m.embeddingLatencyTotalMs, latencyMs)
		atomic.StoreInt64(&m.embeddingLastLatencyMs, latencyMs)
		if err != nil {
			atomic.AddInt64(&m.embeddingErrorsTotal, 1)
		}
	}

	// Логируем вызов
	if m.log != nil {
		logAttrs := []any{
			slog.String("service", string(service)),
			slog.Int64("latency_ms", latencyMs),
		}
		if err != nil {
			logAttrs = append(logAttrs, slog.String("error", err.Error()))
			m.log.Warn("AI service call failed", logAttrs...)
		} else {
			m.log.Debug("AI service call completed", logAttrs...)
		}
	}
}

// AICallTimer помогает измерять время вызовов.
type AICallTimer struct {
	metrics   *AIMetrics
	service   ServiceType
	startTime time.Time
}

// StartTimer начинает измерение времени вызова.
func (m *AIMetrics) StartTimer(service ServiceType) *AICallTimer {
	return &AICallTimer{
		metrics:   m,
		service:   service,
		startTime: time.Now(),
	}
}

// Stop останавливает таймер и записывает метрики.
func (t *AICallTimer) Stop(err error, tokensUsed int) {
	latency := time.Since(t.startTime)
	t.metrics.RecordCall(t.service, latency, err, tokensUsed)
}

// Stats — текущая статистика по AI-сервисам.
type Stats struct {
	LLM       ServiceStats `json:"llm"`
	Reranker  ServiceStats `json:"reranker"`
	Vision    ServiceStats `json:"vision"`
	Embedding ServiceStats `json:"embedding"`
}

// ServiceStats — статистика по одному сервису.
type ServiceStats struct {
	CallsTotal       int64   `json:"calls_total"`
	ErrorsTotal      int64   `json:"errors_total"`
	ErrorRate        float64 `json:"error_rate"`
	AvgLatencyMs     float64 `json:"avg_latency_ms"`
	LastLatencyMs    int64   `json:"last_latency_ms"`
	TokensUsedTotal  int64   `json:"tokens_used_total,omitempty"`
}

// GetStats возвращает текущую статистику.
func (m *AIMetrics) GetStats() Stats {
	return Stats{
		LLM:       m.getServiceStats(ServiceLLM),
		Reranker:  m.getServiceStats(ServiceReranker),
		Vision:    m.getServiceStats(ServiceVision),
		Embedding: m.getServiceStats(ServiceEmbedding),
	}
}

func (m *AIMetrics) getServiceStats(service ServiceType) ServiceStats {
	var calls, errors, latencyTotal, lastLatency, tokens int64

	switch service {
	case ServiceLLM:
		calls = atomic.LoadInt64(&m.llmCallsTotal)
		errors = atomic.LoadInt64(&m.llmErrorsTotal)
		latencyTotal = atomic.LoadInt64(&m.llmLatencyTotalMs)
		lastLatency = atomic.LoadInt64(&m.llmLastLatencyMs)
		tokens = atomic.LoadInt64(&m.llmTokensUsedTotal)
	case ServiceReranker:
		calls = atomic.LoadInt64(&m.rerankerCallsTotal)
		errors = atomic.LoadInt64(&m.rerankerErrorsTotal)
		latencyTotal = atomic.LoadInt64(&m.rerankerLatencyTotalMs)
		lastLatency = atomic.LoadInt64(&m.rerankerLastLatencyMs)
	case ServiceVision:
		calls = atomic.LoadInt64(&m.visionCallsTotal)
		errors = atomic.LoadInt64(&m.visionErrorsTotal)
		latencyTotal = atomic.LoadInt64(&m.visionLatencyTotalMs)
		lastLatency = atomic.LoadInt64(&m.visionLastLatencyMs)
	case ServiceEmbedding:
		calls = atomic.LoadInt64(&m.embeddingCallsTotal)
		errors = atomic.LoadInt64(&m.embeddingErrorsTotal)
		latencyTotal = atomic.LoadInt64(&m.embeddingLatencyTotalMs)
		lastLatency = atomic.LoadInt64(&m.embeddingLastLatencyMs)
	}

	var errorRate, avgLatency float64
	if calls > 0 {
		errorRate = float64(errors) / float64(calls)
		avgLatency = float64(latencyTotal) / float64(calls)
	}

	return ServiceStats{
		CallsTotal:      calls,
		ErrorsTotal:     errors,
		ErrorRate:       errorRate,
		AvgLatencyMs:    avgLatency,
		LastLatencyMs:   lastLatency,
		TokensUsedTotal: tokens,
	}
}

// Reset сбрасывает все метрики.
func (m *AIMetrics) Reset() {
	atomic.StoreInt64(&m.llmCallsTotal, 0)
	atomic.StoreInt64(&m.rerankerCallsTotal, 0)
	atomic.StoreInt64(&m.visionCallsTotal, 0)
	atomic.StoreInt64(&m.embeddingCallsTotal, 0)
	atomic.StoreInt64(&m.llmErrorsTotal, 0)
	atomic.StoreInt64(&m.rerankerErrorsTotal, 0)
	atomic.StoreInt64(&m.visionErrorsTotal, 0)
	atomic.StoreInt64(&m.embeddingErrorsTotal, 0)
	atomic.StoreInt64(&m.llmLatencyTotalMs, 0)
	atomic.StoreInt64(&m.rerankerLatencyTotalMs, 0)
	atomic.StoreInt64(&m.visionLatencyTotalMs, 0)
	atomic.StoreInt64(&m.embeddingLatencyTotalMs, 0)
	atomic.StoreInt64(&m.llmLastLatencyMs, 0)
	atomic.StoreInt64(&m.rerankerLastLatencyMs, 0)
	atomic.StoreInt64(&m.visionLastLatencyMs, 0)
	atomic.StoreInt64(&m.embeddingLastLatencyMs, 0)
	atomic.StoreInt64(&m.llmTokensUsedTotal, 0)
}

// WrapWithMetrics оборачивает функцию для автоматического сбора метрик.
func WrapWithMetrics[T any](
	ctx context.Context,
	m *AIMetrics,
	service ServiceType,
	fn func(ctx context.Context) (T, error),
) (T, error) {
	timer := m.StartTimer(service)
	result, err := fn(ctx)
	timer.Stop(err, 0)
	return result, err
}

