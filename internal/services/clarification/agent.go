package clarification

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"lead_exchange/internal/domain"
	"lead_exchange/internal/lib/llm"
	"lead_exchange/internal/services/weights"
)

// Agent — AI-агент для генерации уточняющих вопросов.
type Agent struct {
	log             *slog.Logger
	llmClient       llm.Client
	weightsAnalyzer *weights.Analyzer
}

// NewAgent создаёт нового агента для уточнения.
func NewAgent(log *slog.Logger, llmClient llm.Client, weightsAnalyzer *weights.Analyzer) *Agent {
	return &Agent{
		log:             log,
		llmClient:       llmClient,
		weightsAnalyzer: weightsAnalyzer,
	}
}

// ClarificationResult — результат работы агента.
type ClarificationResult struct {
	// NeedsClarification — нужно ли уточнение
	NeedsClarification bool `json:"needs_clarification"`
	// Questions — список уточняющих вопросов
	Questions []Question `json:"questions"`
	// Priority — приоритет уточнения (high, medium, low)
	Priority string `json:"priority"`
	// MissingFields — отсутствующие поля
	MissingFields []string `json:"missing_fields"`
	// LeadQualityScore — оценка качества лида (0-1)
	LeadQualityScore float64 `json:"lead_quality_score"`
}

// Question — уточняющий вопрос.
type Question struct {
	// Field — поле, которое уточняется
	Field string `json:"field"`
	// Question — текст вопроса
	Question string `json:"question"`
	// QuestionType — тип вопроса (open, choice, range, boolean)
	QuestionType string `json:"question_type"`
	// SuggestedOptions — предложенные варианты ответа
	SuggestedOptions []string `json:"suggested_options,omitempty"`
	// Importance — важность вопроса (required, recommended, optional)
	Importance string `json:"importance"`
}

// AnalyzeAndGenerateQuestions анализирует лид и генерирует уточняющие вопросы.
func (a *Agent) AnalyzeAndGenerateQuestions(ctx context.Context, lead domain.Lead) (*ClarificationResult, error) {
	const op = "clarification.Agent.AnalyzeAndGenerateQuestions"

	result := &ClarificationResult{
		MissingFields: a.weightsAnalyzer.GetMissingFields(lead),
	}

	// Оцениваем качество лида
	result.LeadQualityScore = a.calculateLeadQuality(lead, result.MissingFields)

	// Определяем, нужно ли уточнение
	result.NeedsClarification = a.weightsAnalyzer.IsShortLead(lead) || len(result.MissingFields) >= 3

	if !result.NeedsClarification {
		result.Priority = "low"
		return result, nil
	}

	// Определяем приоритет
	result.Priority = a.determinePriority(result.MissingFields, result.LeadQualityScore)

	// Генерируем вопросы
	if a.llmClient.IsEnabled() {
		questions, err := a.generateQuestionsWithLLM(ctx, lead, result.MissingFields)
		if err != nil {
			a.log.Warn("LLM question generation failed, using fallback",
				slog.String("lead_id", lead.ID.String()),
				slog.String("error", err.Error()),
			)
			result.Questions = a.generateFallbackQuestions(result.MissingFields)
		} else {
			result.Questions = questions
		}
	} else {
		result.Questions = a.generateFallbackQuestions(result.MissingFields)
	}

	a.log.Info("clarification analysis completed",
		slog.String("lead_id", lead.ID.String()),
		slog.Bool("needs_clarification", result.NeedsClarification),
		slog.Int("questions_count", len(result.Questions)),
		slog.String("priority", result.Priority),
	)

	return result, nil
}

// calculateLeadQuality вычисляет оценку качества лида (0-1).
func (a *Agent) calculateLeadQuality(lead domain.Lead, missingFields []string) float64 {
	score := 1.0

	// Штраф за каждое отсутствующее поле
	score -= float64(len(missingFields)) * 0.15

	// Штраф за короткое описание
	if len(lead.Description) < 30 {
		score -= 0.2
	} else if len(lead.Description) < 50 {
		score -= 0.1
	}

	// Штраф за отсутствие города
	if lead.City == nil || *lead.City == "" {
		score -= 0.15
	}

	// Бонус за подробное описание
	if len(lead.Description) > 150 {
		score += 0.1
	}

	// Нормализация
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score
}

// determinePriority определяет приоритет уточнения.
func (a *Agent) determinePriority(missingFields []string, qualityScore float64) string {
	// Критические поля
	criticalMissing := 0
	for _, f := range missingFields {
		if f == "price" || f == "city" || f == "roomNumber" {
			criticalMissing++
		}
	}

	if qualityScore < 0.3 || criticalMissing >= 2 {
		return "high"
	}

	if qualityScore < 0.6 || criticalMissing >= 1 {
		return "medium"
	}

	return "low"
}

// generateQuestionsWithLLM генерирует вопросы с помощью LLM.
func (a *Agent) generateQuestionsWithLLM(ctx context.Context, lead domain.Lead, missingFields []string) ([]Question, error) {
	const op = "clarification.Agent.generateQuestionsWithLLM"

	var reqMap map[string]interface{}
	if len(lead.Requirement) > 0 {
		json.Unmarshal(lead.Requirement, &reqMap)
	}

	req := llm.ClarificationRequest{
		Title:         lead.Title,
		Description:   lead.Description,
		Requirement:   reqMap,
		MissingFields: missingFields,
	}

	resp, err := a.llmClient.GenerateClarificationQuestions(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Конвертируем ответ LLM в наш формат
	questions := make([]Question, len(resp.Questions))
	for i, q := range resp.Questions {
		questions[i] = Question{
			Field:            q.Field,
			Question:         q.Question,
			QuestionType:     q.QuestionType,
			SuggestedOptions: q.SuggestedOptions,
			Importance:       q.Importance,
		}
	}

	return questions, nil
}

// generateFallbackQuestions генерирует вопросы без LLM.
func (a *Agent) generateFallbackQuestions(missingFields []string) []Question {
	// Предопределённые вопросы для каждого поля
	questionTemplates := map[string]Question{
		"price": {
			Field:        "price",
			Question:     "Какой у вас примерный бюджет на покупку?",
			QuestionType: "range",
			SuggestedOptions: []string{
				"до 5 млн ₽",
				"5-10 млн ₽",
				"10-15 млн ₽",
				"15-25 млн ₽",
				"от 25 млн ₽",
			},
			Importance: "required",
		},
		"roomNumber": {
			Field:        "roomNumber",
			Question:     "Сколько комнат вам нужно?",
			QuestionType: "choice",
			SuggestedOptions: []string{
				"Студия",
				"1 комната",
				"2 комнаты",
				"3 комнаты",
				"4+ комнаты",
			},
			Importance: "required",
		},
		"city": {
			Field:        "city",
			Question:     "В каком городе вы ищете недвижимость?",
			QuestionType: "open",
			Importance:   "required",
		},
		"district": {
			Field:        "district",
			Question:     "Какой район или районы вы рассматриваете?",
			QuestionType: "open",
			SuggestedOptions: []string{
				"Центральный",
				"Спальный район",
				"Новостройки",
				"Рядом с метро",
				"Любой",
			},
			Importance: "recommended",
		},
		"area": {
			Field:        "area",
			Question:     "Какая минимальная площадь вас интересует?",
			QuestionType: "range",
			SuggestedOptions: []string{
				"до 40 м²",
				"40-60 м²",
				"60-80 м²",
				"80-100 м²",
				"от 100 м²",
			},
			Importance: "recommended",
		},
	}

	var questions []Question

	// Сначала добавляем обязательные вопросы
	for _, field := range missingFields {
		if q, ok := questionTemplates[field]; ok {
			if q.Importance == "required" {
				questions = append(questions, q)
			}
		}
	}

	// Затем рекомендованные
	for _, field := range missingFields {
		if q, ok := questionTemplates[field]; ok {
			if q.Importance == "recommended" {
				questions = append(questions, q)
			}
		}
	}

	// Ограничиваем количество вопросов
	if len(questions) > 5 {
		questions = questions[:5]
	}

	return questions
}

// ApplyClarificationAnswers применяет ответы на уточняющие вопросы к лиду.
// Возвращает обновлённый requirement JSON.
func (a *Agent) ApplyClarificationAnswers(lead domain.Lead, answers map[string]interface{}) (json.RawMessage, error) {
	const op = "clarification.Agent.ApplyClarificationAnswers"

	var reqMap map[string]interface{}
	if len(lead.Requirement) > 0 {
		if err := json.Unmarshal(lead.Requirement, &reqMap); err != nil {
			reqMap = make(map[string]interface{})
		}
	} else {
		reqMap = make(map[string]interface{})
	}

	// Применяем ответы
	for field, value := range answers {
		switch field {
		case "price":
			// Парсим бюджет из текстового ответа
			if priceStr, ok := value.(string); ok {
				if price := a.parsePriceRange(priceStr); price > 0 {
					reqMap["price"] = price
				}
			} else if price, ok := value.(float64); ok {
				reqMap["price"] = int64(price)
			}
		case "roomNumber":
			// Парсим количество комнат
			if roomsStr, ok := value.(string); ok {
				if rooms := a.parseRooms(roomsStr); rooms > 0 {
					reqMap["roomNumber"] = rooms
				}
			} else if rooms, ok := value.(float64); ok {
				reqMap["roomNumber"] = int32(rooms)
			}
		case "area":
			// Парсим площадь
			if areaStr, ok := value.(string); ok {
				if area := a.parseArea(areaStr); area > 0 {
					reqMap["area"] = area
				}
			} else if area, ok := value.(float64); ok {
				reqMap["area"] = area
			}
		case "district":
			if district, ok := value.(string); ok && district != "" && district != "Любой" {
				reqMap["district"] = district
			}
		default:
			// Другие поля добавляем как есть
			reqMap[field] = value
		}
	}

	return json.Marshal(reqMap)
}

// parsePriceRange парсит бюджет из текстового ответа.
func (a *Agent) parsePriceRange(text string) int64 {
	priceRanges := map[string]int64{
		"до 5 млн":    5000000,
		"5-10 млн":    7500000,
		"10-15 млн":   12500000,
		"15-25 млн":   20000000,
		"от 25 млн":   30000000,
	}

	for pattern, price := range priceRanges {
		if containsIgnoreCase(text, pattern) {
			return price
		}
	}

	return 0
}

// parseRooms парсит количество комнат из текстового ответа.
func (a *Agent) parseRooms(text string) int32 {
	roomPatterns := map[string]int32{
		"студия":    0, // или 1, зависит от логики
		"1 комнат":  1,
		"2 комнат":  2,
		"3 комнат":  3,
		"4+ комнат": 4,
		"4 комнат":  4,
	}

	for pattern, rooms := range roomPatterns {
		if containsIgnoreCase(text, pattern) {
			return rooms
		}
	}

	return 0
}

// parseArea парсит площадь из текстового ответа.
func (a *Agent) parseArea(text string) float64 {
	areaRanges := map[string]float64{
		"до 40":    35.0,
		"40-60":    50.0,
		"60-80":    70.0,
		"80-100":   90.0,
		"от 100":   120.0,
	}

	for pattern, area := range areaRanges {
		if containsIgnoreCase(text, pattern) {
			return area
		}
	}

	return 0
}

func containsIgnoreCase(s, substr string) bool {
	// Простая проверка вхождения без учёта регистра
	sl := toLower(s)
	substrL := toLower(substr)
	return contains(sl, substrL)
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		result[i] = c
	}
	return string(result)
}

func contains(s, substr string) bool {
	return len(substr) <= len(s) && findSubstring(s, substr) >= 0
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

