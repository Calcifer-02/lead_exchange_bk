//go:build integration
// +build integration

package reranker

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"lead_exchange/internal/config"
)

func getTestConfig() config.RerankerConfig {
	apiKey := os.Getenv("RERANKER_API_KEY")
	if apiKey == "" {
		apiKey = "jina_9e87aa2027dd494dbc9c39589b4a8070dKIsRi6nYah5Uli1mfwEJYJwPs-M"
	}

	return config.RerankerConfig{
		Enabled: true,
		BaseURL: "https://api.jina.ai/v1",
		APIKey:  apiKey,
		Model:   "jina-reranker-v2-base-multilingual",
		Timeout: 30 * time.Second,
		TopN:    5,
	}
}

// TestJinaRerankerAvailability проверяет доступность Jina Reranker API
func TestJinaRerankerAvailability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := getTestConfig()
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := NewClient(cfg, log)

	if !client.IsEnabled() {
		t.Fatal("Reranker client should be enabled")
	}

	t.Log("✅ Jina Reranker client initialized successfully")
}

// TestJinaRerankerBasicRerank тестирует базовое переранжирование
func TestJinaRerankerBasicRerank(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := getTestConfig()
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := NewClient(cfg, log)

	ctx := context.Background()

	query := "Ищу 2-комнатную квартиру в центре Москвы"
	documents := []string{
		"2-комнатная квартира в центре Москвы, 55 кв.м, свежий ремонт",
		"Офисное помещение в бизнес-центре Москвы",
		"3-комнатная квартира на окраине Москвы",
		"Квартира-студия в Санкт-Петербурге",
		"2-комнатная квартира рядом с метро, центр Москвы",
	}

	req := RerankRequest{
		Query:     query,
		Documents: documents,
		TopN:      5,
	}

	t.Log("Отправляем запрос на переранжирование...")
	t.Logf("Query: %s", query)
	t.Log("Documents:")
	for i, doc := range documents {
		t.Logf("  %d. %s", i+1, doc)
	}
	t.Log("")

	resp, err := client.Rerank(ctx, req)
	if err != nil {
		t.Fatalf("Failed to rerank: %v", err)
	}

	t.Log("Результаты переранжирования:")
	for _, result := range resp.Results {
		t.Logf("  Score: %.4f | Doc[%d]: %s",
			result.RelevanceScore, result.Index, truncate(documents[result.Index], 60))
	}
	t.Log("")

	// Проверяем, что результаты отсортированы по убыванию score
	for i := 0; i < len(resp.Results)-1; i++ {
		if resp.Results[i].RelevanceScore < resp.Results[i+1].RelevanceScore {
			t.Errorf("Results are not sorted by score: %.4f < %.4f",
				resp.Results[i].RelevanceScore, resp.Results[i+1].RelevanceScore)
		}
	}

	// Проверяем, что квартиры в центре Москвы имеют высокие scores
	topResult := resp.Results[0]
	if topResult.Index != 0 && topResult.Index != 4 {
		t.Logf("⚠️  Top result is not a 2-room apartment in Moscow center: index=%d", topResult.Index)
	} else {
		t.Log("✅ Top result is correctly a 2-room apartment in Moscow center")
	}

	// Офис должен иметь низкий score
	var officeScore float64
	for _, result := range resp.Results {
		if result.Index == 1 { // Офис
			officeScore = result.RelevanceScore
		}
	}
	if officeScore > resp.Results[0].RelevanceScore*0.8 {
		t.Log("⚠️  Office score is unexpectedly high")
	} else {
		t.Log("✅ Office correctly has lower score")
	}
}

// TestJinaRerankerRealEstateScenario тестирует реальный сценарий для недвижимости
func TestJinaRerankerRealEstateScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := getTestConfig()
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := NewClient(cfg, log)

	ctx := context.Background()

	t.Log("╔══════════════════════════════════════════════════════════════════╗")
	t.Log("║           ТЕСТ RERANKER ДЛЯ НЕДВИЖИМОСТИ                         ║")
	t.Log("╚══════════════════════════════════════════════════════════════════╝")
	t.Log("")

	scenarios := []struct {
		name      string
		query     string
		documents []string
		expected  int // Индекс документа, который должен быть первым
	}{
		{
			name:  "Поиск квартиры по цене",
			query: "Ищу недорогую квартиру до 10 миллионов рублей",
			documents: []string{
				"Квартира за 8 млн рублей, хорошее состояние",
				"Элитная квартира премиум-класса за 50 млн",
				"Квартира за 9.5 млн, свежий ремонт",
				"Офис за 12 млн рублей",
			},
			expected: 0, // Первая или третья квартира
		},
		{
			name:  "Поиск по району",
			query: "Квартира в центре Москвы, рядом с метро Арбатская",
			documents: []string{
				"Квартира в спальном районе Бутово",
				"Квартира на Арбате, 3 минуты от метро Арбатская",
				"Квартира в центре Санкт-Петербурга",
				"Офис рядом с метро Арбатская",
			},
			expected: 1,
		},
		{
			name:  "Поиск по количеству комнат",
			query: "Нужна просторная трёхкомнатная квартира для большой семьи",
			documents: []string{
				"Студия 25 кв.м",
				"3-комнатная квартира 90 кв.м для семьи",
				"2-комнатная квартира",
				"Пентхаус 200 кв.м",
			},
			expected: 1,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Logf("Query: %s", scenario.query)
			t.Log("Documents:")
			for i, doc := range scenario.documents {
				t.Logf("  [%d] %s", i, doc)
			}

			req := RerankRequest{
				Query:     scenario.query,
				Documents: scenario.documents,
				TopN:      len(scenario.documents),
			}

			resp, err := client.Rerank(ctx, req)
			if err != nil {
				t.Fatalf("Failed to rerank: %v", err)
			}

			t.Log("")
			t.Log("Результаты:")
			for i, result := range resp.Results {
				marker := "  "
				if i == 0 {
					marker = "→ "
				}
				t.Logf("%s[%d] Score: %.4f | %s",
					marker, result.Index, result.RelevanceScore, scenario.documents[result.Index])
			}
			t.Log("")

			// Проверяем, что ожидаемый документ в топе
			if resp.Results[0].Index == scenario.expected {
				t.Log("✅ Ожидаемый документ на первом месте")
			} else {
				t.Logf("⚠️  Ожидался документ [%d], получен [%d]",
					scenario.expected, resp.Results[0].Index)
			}
		})
	}
}

// TestJinaRerankerMultilingual тестирует работу с русским языком
func TestJinaRerankerMultilingual(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := getTestConfig()
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := NewClient(cfg, log)

	ctx := context.Background()

	t.Log("Тест мультиязычности (русский язык):")
	t.Log("")

	query := "хочу купить двушку в москве недорого"
	documents := []string{
		"Продаётся 2-комнатная квартира в Москве по выгодной цене",
		"Expensive luxury penthouse in Moscow city center",
		"Дешёвая двухкомнатная квартира, срочная продажа, Москва",
		"3-комнатная квартира в элитном доме",
	}

	req := RerankRequest{
		Query:     query,
		Documents: documents,
		TopN:      4,
	}

	t.Logf("Query: %s", query)

	resp, err := client.Rerank(ctx, req)
	if err != nil {
		t.Fatalf("Failed to rerank: %v", err)
	}

	t.Log("Results:")
	for _, result := range resp.Results {
		t.Logf("  %.4f | %s", result.RelevanceScore, documents[result.Index])
	}
	t.Log("")

	// Проверяем, что русские документы о недорогих двушках имеют высокие scores
	topIndex := resp.Results[0].Index
	if topIndex == 0 || topIndex == 2 {
		t.Log("✅ Русский документ о недорогой двушке правильно определён как наиболее релевантный")
	} else {
		t.Logf("⚠️  Первым стал документ [%d]: %s", topIndex, documents[topIndex])
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

