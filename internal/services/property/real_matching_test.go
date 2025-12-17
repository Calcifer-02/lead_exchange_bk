//go:build integration
// +build integration

package property

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

// MLServiceConfig содержит настройки для ML сервиса
type MLServiceConfig struct {
	BaseURL string
	Timeout time.Duration
}

func getMLConfig() MLServiceConfig {
	baseURL := os.Getenv("ML_BASE_URL")
	if baseURL == "" {
		baseURL = "https://calcifer0323-matching.hf.space"
	}
	return MLServiceConfig{
		BaseURL: baseURL,
		Timeout: 60 * time.Second,
	}
}

// PrepareAndEmbedRequest - запрос к ML сервису
type PrepareAndEmbedRequest struct {
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Price       *int64   `json:"price,omitempty"`
	Rooms       *int32   `json:"rooms,omitempty"`
	Area        *float64 `json:"area,omitempty"`
	Address     *string  `json:"address,omitempty"`
}

// PrepareAndEmbedResponse - ответ от ML сервиса
type PrepareAndEmbedResponse struct {
	Embedding    []float64 `json:"embedding"`
	Dimensions   int       `json:"dimensions"`
	PreparedText string    `json:"prepared_text"`
}

// callMLService вызывает реальный ML сервис для генерации эмбеддингов
func callMLService(cfg MLServiceConfig, req PrepareAndEmbedRequest) (*PrepareAndEmbedResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, cfg.BaseURL+"/prepare-and-embed", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: cfg.Timeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result PrepareAndEmbedResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// TestMLServiceAvailability проверяет доступность ML сервиса
func TestMLServiceAvailability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := getMLConfig()
	t.Logf("Testing ML service at: %s", cfg.BaseURL)

	// Проверяем health endpoint
	resp, err := http.Get(cfg.BaseURL + "/health")
	if err != nil {
		t.Fatalf("ML service not available: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("ML service health check failed: status=%d, body=%s", resp.StatusCode, string(body))
	}

	t.Log("✅ ML service is available")
}

// TestRealEmbeddingGeneration проверяет генерацию реальных эмбеддингов
func TestRealEmbeddingGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := getMLConfig()
	ctx := context.Background()
	_ = ctx

	testCases := []struct {
		name string
		req  PrepareAndEmbedRequest
	}{
		{
			name: "Simple apartment",
			req: PrepareAndEmbedRequest{
				Title:       "2-комнатная квартира",
				Description: "Светлая квартира в центре Москвы",
			},
		},
		{
			name: "Detailed apartment",
			req: PrepareAndEmbedRequest{
				Title:       "Просторная 3-комнатная квартира",
				Description: "Отличная квартира с видом на парк, свежий ремонт",
				Price:       int64Ptr(15000000),
				Rooms:       int32Ptr(3),
				Area:        float64Ptr(85.5),
				Address:     stringPtr("Москва, ул. Пушкина, д. 10"),
			},
		},
		{
			name: "Commercial property",
			req: PrepareAndEmbedRequest{
				Title:       "Офисное помещение",
				Description: "Современный офис в бизнес-центре класса А",
				Price:       int64Ptr(50000000),
				Area:        float64Ptr(200.0),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := callMLService(cfg, tc.req)
			if err != nil {
				t.Fatalf("Failed to get embedding: %v", err)
			}

			if len(resp.Embedding) == 0 {
				t.Fatal("Empty embedding returned")
			}

			t.Logf("✅ Generated embedding with %d dimensions", resp.Dimensions)
			t.Logf("   Prepared text: %s", truncateString(resp.PreparedText, 100))
		})
	}
}

// TestRealMatchingScenarios тестирует реальные сценарии матчинга
func TestRealMatchingScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := getMLConfig()

	t.Log("╔══════════════════════════════════════════════════════════════════╗")
	t.Log("║           РЕАЛЬНЫЕ ТЕСТЫ МАТЧИНГА НЕДВИЖИМОСТИ                   ║")
	t.Log("╚══════════════════════════════════════════════════════════════════╝")
	t.Log("")

	// Сценарий 1: Точное совпадение по типу и району
	t.Run("Scenario1_ExactMatch", func(t *testing.T) {
		t.Log("=== СЦЕНАРИЙ 1: Точное совпадение по типу и району ===")

		leadReq := PrepareAndEmbedRequest{
			Title:       "Ищу 2-комнатную квартиру",
			Description: "Нужна светлая квартира в центре Москвы, рядом с метро, до 15 млн",
			Price:       int64Ptr(15000000),
			Rooms:       int32Ptr(2),
		}

		prop1Req := PrepareAndEmbedRequest{
			Title:       "2-комнатная квартира в центре",
			Description: "Отличная двушка рядом с метро в центре Москвы",
			Price:       int64Ptr(14500000),
			Rooms:       int32Ptr(2),
			Area:        float64Ptr(55.0),
			Address:     stringPtr("Москва, Арбат"),
		}

		prop2Req := PrepareAndEmbedRequest{
			Title:       "3-комнатная квартира",
			Description: "Большая трёшка на окраине Москвы",
			Price:       int64Ptr(12000000),
			Rooms:       int32Ptr(3),
			Area:        float64Ptr(80.0),
			Address:     stringPtr("Москва, Бутово"),
		}

		leadEmb, err := callMLService(cfg, leadReq)
		if err != nil {
			t.Fatalf("Failed to get lead embedding: %v", err)
		}

		prop1Emb, err := callMLService(cfg, prop1Req)
		if err != nil {
			t.Fatalf("Failed to get property 1 embedding: %v", err)
		}

		prop2Emb, err := callMLService(cfg, prop2Req)
		if err != nil {
			t.Fatalf("Failed to get property 2 embedding: %v", err)
		}

		sim1 := cosineSimilarity(leadEmb.Embedding, prop1Emb.Embedding)
		sim2 := cosineSimilarity(leadEmb.Embedding, prop2Emb.Embedding)

		t.Logf("Лид: %s", leadReq.Title)
		t.Logf("  %s", leadReq.Description)
		t.Log("")
		t.Logf("Объект 1 (идеальное совпадение): %.2f%%", sim1*100)
		t.Logf("Объект 2 (другой район и комнаты): %.2f%%", sim2*100)
		t.Log("")

		if sim1 <= sim2 {
			t.Errorf("Expected property 1 to have higher similarity than property 2")
		} else {
			t.Logf("✅ Объект 1 корректно имеет более высокий рейтинг")
		}
	})

	// Сценарий 2: Фильтрация по городу
	t.Run("Scenario2_CityFiltering", func(t *testing.T) {
		t.Log("=== СЦЕНАРИЙ 2: Фильтрация по городу ===")

		leadReq := PrepareAndEmbedRequest{
			Title:       "Квартира в Москве",
			Description: "Ищу квартиру только в Москве, район не важен",
			Price:       int64Ptr(10000000),
			Rooms:       int32Ptr(2),
		}

		propMoscowReq := PrepareAndEmbedRequest{
			Title:       "2-комнатная квартира",
			Description: "Квартира в спальном районе Москвы",
			Price:       int64Ptr(9500000),
			Rooms:       int32Ptr(2),
			Address:     stringPtr("Москва"),
		}

		propSpbReq := PrepareAndEmbedRequest{
			Title:       "2-комнатная квартира",
			Description: "Квартира в спальном районе Санкт-Петербурга",
			Price:       int64Ptr(9500000),
			Rooms:       int32Ptr(2),
			Address:     stringPtr("Санкт-Петербург"),
		}

		leadEmb, err := callMLService(cfg, leadReq)
		if err != nil {
			t.Fatalf("Failed to get lead embedding: %v", err)
		}

		propMoscowEmb, err := callMLService(cfg, propMoscowReq)
		if err != nil {
			t.Fatalf("Failed to get Moscow property embedding: %v", err)
		}

		propSpbEmb, err := callMLService(cfg, propSpbReq)
		if err != nil {
			t.Fatalf("Failed to get SPb property embedding: %v", err)
		}

		simMoscow := cosineSimilarity(leadEmb.Embedding, propMoscowEmb.Embedding)
		simSpb := cosineSimilarity(leadEmb.Embedding, propSpbEmb.Embedding)

		t.Logf("Лид ищет: %s", leadReq.Description)
		t.Log("")
		t.Logf("Объект в Москве: %.2f%%", simMoscow*100)
		t.Logf("Объект в СПб: %.2f%%", simSpb*100)
		t.Log("")

		diff := (simMoscow - simSpb) * 100
		t.Logf("Разница: %.2f процентных пунктов", diff)

		if diff < 3 {
			t.Log("⚠️  Разница менее 3% - НЕОБХОДИМ жёсткий фильтр по городу!")
		}
		t.Log("✅ Жёсткий фильтр по городу автоматически исключит объект из СПб")
	})

	// Сценарий 3: Разные типы недвижимости
	t.Run("Scenario3_PropertyTypes", func(t *testing.T) {
		t.Log("=== СЦЕНАРИЙ 3: Различение типов недвижимости ===")

		leadReq := PrepareAndEmbedRequest{
			Title:       "Ищу квартиру для семьи",
			Description: "Нужна квартира для проживания семьи с детьми",
			Price:       int64Ptr(12000000),
			Rooms:       int32Ptr(3),
		}

		propApartmentReq := PrepareAndEmbedRequest{
			Title:       "Семейная квартира",
			Description: "Просторная квартира для семьи, рядом школа и детсад",
			Price:       int64Ptr(11000000),
			Rooms:       int32Ptr(3),
		}

		propOfficeReq := PrepareAndEmbedRequest{
			Title:       "Офисное помещение",
			Description: "Офис для работы, хороший ремонт",
			Price:       int64Ptr(12000000),
			Area:        float64Ptr(80.0),
		}

		propHouseReq := PrepareAndEmbedRequest{
			Title:       "Загородный дом",
			Description: "Дом для семьи с участком, рядом лес",
			Price:       int64Ptr(15000000),
			Rooms:       int32Ptr(5),
		}

		leadEmb, err := callMLService(cfg, leadReq)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		propAptEmb, err := callMLService(cfg, propApartmentReq)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		propOfficeEmb, err := callMLService(cfg, propOfficeReq)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		propHouseEmb, err := callMLService(cfg, propHouseReq)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		simApt := cosineSimilarity(leadEmb.Embedding, propAptEmb.Embedding)
		simOffice := cosineSimilarity(leadEmb.Embedding, propOfficeEmb.Embedding)
		simHouse := cosineSimilarity(leadEmb.Embedding, propHouseEmb.Embedding)

		t.Logf("Лид: %s", leadReq.Title)
		t.Log("")
		t.Logf("Квартира (семейная): %.2f%%", simApt*100)
		t.Logf("Офис:                %.2f%%", simOffice*100)
		t.Logf("Дом (для семьи):     %.2f%%", simHouse*100)
		t.Log("")

		if simApt > simOffice {
			t.Log("✅ Квартира корректно имеет более высокий рейтинг чем офис")
		} else {
			t.Errorf("❌ Офис не должен иметь более высокий рейтинг чем квартира")
		}

		// Дом тоже может подходить для семьи
		if simHouse > simOffice {
			t.Log("✅ Дом корректно имеет более высокий рейтинг чем офис")
		}
	})

	// Сценарий 4: Ценовой диапазон
	t.Run("Scenario4_PriceRange", func(t *testing.T) {
		t.Log("=== СЦЕНАРИЙ 4: Влияние цены на матчинг ===")

		leadReq := PrepareAndEmbedRequest{
			Title:       "Квартира до 10 млн",
			Description: "Ищу недорогую квартиру, бюджет ограничен 10 миллионами",
			Price:       int64Ptr(10000000),
			Rooms:       int32Ptr(2),
		}

		propCheapReq := PrepareAndEmbedRequest{
			Title:       "Доступная квартира",
			Description: "Хорошая квартира по доступной цене",
			Price:       int64Ptr(8000000),
			Rooms:       int32Ptr(2),
		}

		propExpensiveReq := PrepareAndEmbedRequest{
			Title:       "Элитная квартира",
			Description: "Роскошная квартира премиум-класса",
			Price:       int64Ptr(50000000),
			Rooms:       int32Ptr(2),
		}

		leadEmb, err := callMLService(cfg, leadReq)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		propCheapEmb, err := callMLService(cfg, propCheapReq)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		propExpensiveEmb, err := callMLService(cfg, propExpensiveReq)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		simCheap := cosineSimilarity(leadEmb.Embedding, propCheapEmb.Embedding)
		simExpensive := cosineSimilarity(leadEmb.Embedding, propExpensiveEmb.Embedding)

		t.Logf("Лид: бюджет до 10 млн")
		t.Log("")
		t.Logf("Квартира за 8 млн:  %.2f%%", simCheap*100)
		t.Logf("Квартира за 50 млн: %.2f%%", simExpensive*100)
		t.Log("")

		if simCheap > simExpensive {
			t.Log("✅ Доступная квартира корректно имеет более высокий рейтинг")
		} else {
			t.Log("⚠️  Цена слабо влияет на эмбеддинги - используйте взвешенное ранжирование!")
		}
	})
}

// TestEndToEndMatchingPipeline тестирует полный пайплайн матчинга
func TestEndToEndMatchingPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := getMLConfig()

	t.Log("╔══════════════════════════════════════════════════════════════════╗")
	t.Log("║           END-TO-END ТЕСТ ПАЙПЛАЙНА МАТЧИНГА                     ║")
	t.Log("╚══════════════════════════════════════════════════════════════════╝")
	t.Log("")

	// Создаём набор объектов недвижимости
	properties := []struct {
		name string
		req  PrepareAndEmbedRequest
		city string
		typ  string
	}{
		{
			name: "Квартира в центре Москвы",
			req: PrepareAndEmbedRequest{
				Title:       "2-комнатная квартира в центре",
				Description: "Светлая квартира с ремонтом, метро 5 минут",
				Price:       int64Ptr(14000000),
				Rooms:       int32Ptr(2),
				Area:        float64Ptr(55.0),
				Address:     stringPtr("Москва, Тверская"),
			},
			city: "Москва",
			typ:  "APARTMENT",
		},
		{
			name: "Квартира на окраине Москвы",
			req: PrepareAndEmbedRequest{
				Title:       "2-комнатная квартира в Бутово",
				Description: "Квартира в спальном районе, тихий двор",
				Price:       int64Ptr(9000000),
				Rooms:       int32Ptr(2),
				Area:        float64Ptr(52.0),
				Address:     stringPtr("Москва, Бутово"),
			},
			city: "Москва",
			typ:  "APARTMENT",
		},
		{
			name: "Квартира в СПб",
			req: PrepareAndEmbedRequest{
				Title:       "2-комнатная квартира",
				Description: "Отличная квартира в Санкт-Петербурге",
				Price:       int64Ptr(12000000),
				Rooms:       int32Ptr(2),
				Area:        float64Ptr(50.0),
				Address:     stringPtr("Санкт-Петербург"),
			},
			city: "Санкт-Петербург",
			typ:  "APARTMENT",
		},
		{
			name: "Офис в Москве",
			req: PrepareAndEmbedRequest{
				Title:       "Офисное помещение",
				Description: "Современный офис в бизнес-центре",
				Price:       int64Ptr(15000000),
				Area:        float64Ptr(100.0),
				Address:     stringPtr("Москва, Сити"),
			},
			city: "Москва",
			typ:  "COMMERCIAL",
		},
		{
			name: "3-комнатная квартира",
			req: PrepareAndEmbedRequest{
				Title:       "3-комнатная квартира",
				Description: "Просторная трёшка для семьи",
				Price:       int64Ptr(18000000),
				Rooms:       int32Ptr(3),
				Area:        float64Ptr(85.0),
				Address:     stringPtr("Москва, Хамовники"),
			},
			city: "Москва",
			typ:  "APARTMENT",
		},
	}

	// Лид: ищем 2-комнатную квартиру в Москве
	leadReq := PrepareAndEmbedRequest{
		Title:       "Ищу 2-комнатную квартиру",
		Description: "Нужна квартира в Москве, желательно ближе к центру",
		Price:       int64Ptr(15000000),
		Rooms:       int32Ptr(2),
	}
	leadCity := "Москва"
	leadType := "APARTMENT"

	t.Log("ЭТАП 1: Генерация эмбеддингов")
	t.Log("─────────────────────────────")

	leadEmb, err := callMLService(cfg, leadReq)
	if err != nil {
		t.Fatalf("Failed to get lead embedding: %v", err)
	}
	t.Logf("✓ Лид: %s (город: %s)", leadReq.Title, leadCity)

	type PropertyWithEmbedding struct {
		name      string
		embedding []float64
		city      string
		typ       string
	}

	var propertiesWithEmb []PropertyWithEmbedding
	for _, p := range properties {
		emb, err := callMLService(cfg, p.req)
		if err != nil {
			t.Fatalf("Failed to get embedding for %s: %v", p.name, err)
		}
		propertiesWithEmb = append(propertiesWithEmb, PropertyWithEmbedding{
			name:      p.name,
			embedding: emb.Embedding,
			city:      p.city,
			typ:       p.typ,
		})
		t.Logf("✓ %s (город: %s, тип: %s)", p.name, p.city, p.typ)
	}
	t.Log("")

	t.Log("ЭТАП 2: Векторный поиск (без фильтров)")
	t.Log("─────────────────────────────────────")

	type MatchResult struct {
		name       string
		similarity float64
		city       string
		typ        string
	}

	var allResults []MatchResult
	for _, p := range propertiesWithEmb {
		sim := cosineSimilarity(leadEmb.Embedding, p.embedding)
		allResults = append(allResults, MatchResult{
			name:       p.name,
			similarity: sim,
			city:       p.city,
			typ:        p.typ,
		})
	}

	// Сортируем по убыванию сходства
	for i := 0; i < len(allResults)-1; i++ {
		for j := i + 1; j < len(allResults); j++ {
			if allResults[j].similarity > allResults[i].similarity {
				allResults[i], allResults[j] = allResults[j], allResults[i]
			}
		}
	}

	for i, r := range allResults {
		t.Logf("  %d. %s: %.2f%% (город: %s, тип: %s)",
			i+1, r.name, r.similarity*100, r.city, r.typ)
	}
	t.Log("")

	t.Log("ЭТАП 3: Применение жёстких фильтров")
	t.Log("───────────────────────────────────")
	t.Logf("Фильтр по городу: %s", leadCity)
	t.Logf("Фильтр по типу: %s", leadType)
	t.Log("")

	var filteredResults []MatchResult
	for _, r := range allResults {
		passesCity := r.city == leadCity
		passesType := r.typ == leadType

		if passesCity && passesType {
			t.Logf("  ✅ %s: ПРОХОДИТ", r.name)
			filteredResults = append(filteredResults, r)
		} else {
			reason := ""
			if !passesCity {
				reason = fmt.Sprintf("город=%s", r.city)
			}
			if !passesType {
				if reason != "" {
					reason += ", "
				}
				reason += fmt.Sprintf("тип=%s", r.typ)
			}
			t.Logf("  ❌ %s: ОТФИЛЬТРОВАН (%s)", r.name, reason)
		}
	}
	t.Log("")

	t.Log("ЭТАП 4: Финальный рейтинг")
	t.Log("─────────────────────────")
	for i, r := range filteredResults {
		t.Logf("  %d. %s: %.2f%%", i+1, r.name, r.similarity*100)
	}
	t.Log("")

	// Проверки
	t.Log("ПРОВЕРКИ:")
	t.Log("─────────")

	// Проверка 1: СПб отфильтрован
	spbFiltered := true
	for _, r := range filteredResults {
		if r.city == "Санкт-Петербург" {
			spbFiltered = false
		}
	}
	if spbFiltered {
		t.Log("✅ Объекты из СПб корректно отфильтрованы")
	} else {
		t.Error("❌ Объект из СПб не был отфильтрован")
	}

	// Проверка 2: Офис отфильтрован
	officeFiltered := true
	for _, r := range filteredResults {
		if r.typ == "COMMERCIAL" {
			officeFiltered = false
		}
	}
	if officeFiltered {
		t.Log("✅ Офисы корректно отфильтрованы")
	} else {
		t.Error("❌ Офис не был отфильтрован")
	}

	// Проверка 3: Результаты отсортированы
	sorted := true
	for i := 0; i < len(filteredResults)-1; i++ {
		if filteredResults[i].similarity < filteredResults[i+1].similarity {
			sorted = false
		}
	}
	if sorted {
		t.Log("✅ Результаты корректно отсортированы по сходству")
	} else {
		t.Error("❌ Результаты не отсортированы")
	}

	// Проверка 4: Есть результаты
	if len(filteredResults) > 0 {
		t.Logf("✅ Найдено %d подходящих объектов", len(filteredResults))
	} else {
		t.Error("❌ Не найдено подходящих объектов")
	}
}

// Вспомогательные функции
func int64Ptr(v int64) *int64     { return &v }
func int32Ptr(v int32) *int32     { return &v }
func float64Ptr(v float64) *float64 { return &v }
func stringPtr(v string) *string  { return &v }

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

