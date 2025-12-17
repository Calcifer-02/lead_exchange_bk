package metrics

import (
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestAIMetrics_RecordCall(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	m := &AIMetrics{log: log}

	// Сбрасываем глобальные метрики для теста
	m.Reset()

	// Тест успешного вызова LLM
	m.RecordCall(ServiceLLM, 100*time.Millisecond, nil, 150)

	stats := m.GetStats()
	if stats.LLM.CallsTotal != 1 {
		t.Errorf("expected 1 LLM call, got %d", stats.LLM.CallsTotal)
	}
	if stats.LLM.ErrorsTotal != 0 {
		t.Errorf("expected 0 LLM errors, got %d", stats.LLM.ErrorsTotal)
	}
	if stats.LLM.TokensUsedTotal != 150 {
		t.Errorf("expected 150 tokens, got %d", stats.LLM.TokensUsedTotal)
	}

	// Тест вызова с ошибкой
	m.RecordCall(ServiceLLM, 50*time.Millisecond, errors.New("test error"), 0)

	stats = m.GetStats()
	if stats.LLM.CallsTotal != 2 {
		t.Errorf("expected 2 LLM calls, got %d", stats.LLM.CallsTotal)
	}
	if stats.LLM.ErrorsTotal != 1 {
		t.Errorf("expected 1 LLM error, got %d", stats.LLM.ErrorsTotal)
	}
}

func TestAIMetrics_RecordCall_AllServices(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	m := &AIMetrics{log: log}
	m.Reset()

	// Записываем вызовы для всех сервисов
	m.RecordCall(ServiceLLM, 100*time.Millisecond, nil, 100)
	m.RecordCall(ServiceReranker, 50*time.Millisecond, nil, 0)
	m.RecordCall(ServiceVision, 200*time.Millisecond, nil, 0)
	m.RecordCall(ServiceEmbedding, 30*time.Millisecond, nil, 0)

	stats := m.GetStats()

	if stats.LLM.CallsTotal != 1 {
		t.Errorf("expected 1 LLM call, got %d", stats.LLM.CallsTotal)
	}
	if stats.Reranker.CallsTotal != 1 {
		t.Errorf("expected 1 Reranker call, got %d", stats.Reranker.CallsTotal)
	}
	if stats.Vision.CallsTotal != 1 {
		t.Errorf("expected 1 Vision call, got %d", stats.Vision.CallsTotal)
	}
	if stats.Embedding.CallsTotal != 1 {
		t.Errorf("expected 1 Embedding call, got %d", stats.Embedding.CallsTotal)
	}
}

func TestAIMetrics_Timer(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	m := &AIMetrics{log: log}
	m.Reset()

	// Тестируем таймер
	timer := m.StartTimer(ServiceLLM)
	time.Sleep(10 * time.Millisecond)
	timer.Stop(nil, 50)

	stats := m.GetStats()
	if stats.LLM.CallsTotal != 1 {
		t.Errorf("expected 1 LLM call, got %d", stats.LLM.CallsTotal)
	}
	if stats.LLM.LastLatencyMs < 10 {
		t.Errorf("expected latency >= 10ms, got %d", stats.LLM.LastLatencyMs)
	}
	if stats.LLM.TokensUsedTotal != 50 {
		t.Errorf("expected 50 tokens, got %d", stats.LLM.TokensUsedTotal)
	}
}

func TestAIMetrics_ErrorRate(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	m := &AIMetrics{log: log}
	m.Reset()

	// 3 успешных вызова, 1 ошибка = error rate 25%
	m.RecordCall(ServiceReranker, 10*time.Millisecond, nil, 0)
	m.RecordCall(ServiceReranker, 10*time.Millisecond, nil, 0)
	m.RecordCall(ServiceReranker, 10*time.Millisecond, nil, 0)
	m.RecordCall(ServiceReranker, 10*time.Millisecond, errors.New("error"), 0)

	stats := m.GetStats()
	expectedErrorRate := 0.25

	if stats.Reranker.ErrorRate != expectedErrorRate {
		t.Errorf("expected error rate %.2f, got %.2f", expectedErrorRate, stats.Reranker.ErrorRate)
	}
}

func TestAIMetrics_AvgLatency(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	m := &AIMetrics{log: log}
	m.Reset()

	// Записываем вызовы с разной задержкой
	m.RecordCall(ServiceVision, 100*time.Millisecond, nil, 0)
	m.RecordCall(ServiceVision, 200*time.Millisecond, nil, 0)

	stats := m.GetStats()
	expectedAvgLatency := 150.0

	if stats.Vision.AvgLatencyMs != expectedAvgLatency {
		t.Errorf("expected avg latency %.2f, got %.2f", expectedAvgLatency, stats.Vision.AvgLatencyMs)
	}
}

func TestAIMetrics_Reset(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	m := &AIMetrics{log: log}

	// Записываем вызовы
	m.RecordCall(ServiceLLM, 100*time.Millisecond, nil, 100)
	m.RecordCall(ServiceReranker, 50*time.Millisecond, nil, 0)

	// Сбрасываем
	m.Reset()

	stats := m.GetStats()
	if stats.LLM.CallsTotal != 0 {
		t.Errorf("expected 0 LLM calls after reset, got %d", stats.LLM.CallsTotal)
	}
	if stats.Reranker.CallsTotal != 0 {
		t.Errorf("expected 0 Reranker calls after reset, got %d", stats.Reranker.CallsTotal)
	}
}

func TestGetAIMetrics_Singleton(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Получаем метрики дважды - должен быть один и тот же экземпляр
	m1 := GetAIMetrics(log)
	m2 := GetAIMetrics(log)

	if m1 != m2 {
		t.Error("expected GetAIMetrics to return singleton instance")
	}
}

