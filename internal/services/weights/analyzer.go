package weights

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"lead_exchange/internal/config"
	"lead_exchange/internal/domain"
	"lead_exchange/internal/lib/llm"
)

// Analyzer — сервис для динамического определения весов матчинга.
type Analyzer struct {
	log       *slog.Logger
	llmClient llm.Client
	cfg       config.SearchConfig
}

// NewAnalyzer создаёт новый анализатор весов.
func NewAnalyzer(log *slog.Logger, llmClient llm.Client, cfg config.SearchConfig) *Analyzer {
	return &Analyzer{
		log:       log,
		llmClient: llmClient,
		cfg:       cfg,
	}
}

// AnalyzeResult — результат анализа лида.
type AnalyzeResult struct {
	// Weights — рекомендованные веса для матчинга
	Weights domain.MatchWeights
	// Criteria — извлечённые мягкие критерии
	Criteria *domain.SoftCriteria
	// LeadType — тип лида
	LeadType string
	// Confidence — уверенность в анализе (0-1)
	Confidence float64
	// Explanation — объяснение выбора весов
	Explanation string
	// UsedLLM — использовался ли LLM для анализа
	UsedLLM bool
}

// AnalyzeLead анализирует лид и определяет оптимальные веса для матчинга.
// Если LLM недоступен или отключен, используется эвристический анализ.
func (a *Analyzer) AnalyzeLead(ctx context.Context, lead domain.Lead) (*AnalyzeResult, error) {
	const op = "weights.Analyzer.AnalyzeLead"

	// Если динамические веса отключены, возвращаем веса по умолчанию
	if !a.cfg.DynamicWeightsEnabled {
		return a.heuristicAnalysis(lead), nil
	}

	// Пытаемся использовать LLM
	if a.llmClient.IsEnabled() {
		result, err := a.llmAnalysis(ctx, lead)
		if err != nil {
			a.log.Warn("LLM analysis failed, falling back to heuristic",
				slog.String("lead_id", lead.ID.String()),
				slog.String("error", err.Error()),
			)
			return a.heuristicAnalysis(lead), nil
		}
		return result, nil
	}

	return a.heuristicAnalysis(lead), nil
}

// llmAnalysis использует LLM для анализа лида.
func (a *Analyzer) llmAnalysis(ctx context.Context, lead domain.Lead) (*AnalyzeResult, error) {
	const op = "weights.Analyzer.llmAnalysis"

	// Парсим requirement
	var requirementMap map[string]interface{}
	if len(lead.Requirement) > 0 {
		if err := json.Unmarshal(lead.Requirement, &requirementMap); err != nil {
			a.log.Warn("failed to parse requirement JSON", slog.String("error", err.Error()))
		}
	}

	req := llm.AnalyzeLeadRequest{
		Title:       lead.Title,
		Description: lead.Description,
		Requirement: requirementMap,
	}

	resp, err := a.llmClient.AnalyzeLeadIntent(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Конвертируем результат LLM в наш формат
	result := &AnalyzeResult{
		Weights: domain.MatchWeights{
			Price:    resp.RecommendedWeights.Price,
			District: resp.RecommendedWeights.District,
			Rooms:    resp.RecommendedWeights.Rooms,
			Area:     resp.RecommendedWeights.Area,
			Semantic: resp.RecommendedWeights.Semantic,
		},
		LeadType:    resp.LeadType,
		Confidence:  resp.Confidence,
		Explanation: resp.Explanation,
		UsedLLM:     true,
	}

	// Конвертируем извлечённые критерии
	if resp.ExtractedCriteria.TargetPrice != nil ||
	   resp.ExtractedCriteria.TargetDistrict != nil ||
	   resp.ExtractedCriteria.TargetRooms != nil ||
	   resp.ExtractedCriteria.TargetArea != nil {
		result.Criteria = &domain.SoftCriteria{
			TargetPrice:        resp.ExtractedCriteria.TargetPrice,
			TargetDistrict:     resp.ExtractedCriteria.TargetDistrict,
			TargetRooms:        resp.ExtractedCriteria.TargetRooms,
			TargetArea:         resp.ExtractedCriteria.TargetArea,
			PreferredDistricts: resp.ExtractedCriteria.PreferredDistricts,
		}
	}

	// Нормализуем веса
	result.Weights = result.Weights.Normalize()

	a.log.Info("LLM analysis completed",
		slog.String("lead_id", lead.ID.String()),
		slog.String("lead_type", result.LeadType),
		slog.Float64("confidence", result.Confidence),
	)

	return result, nil
}

// heuristicAnalysis использует эвристики для определения весов.
func (a *Analyzer) heuristicAnalysis(lead domain.Lead) *AnalyzeResult {
	result := &AnalyzeResult{
		Weights:     domain.DefaultWeights(),
		LeadType:    "unknown",
		Confidence:  0.5,
		Explanation: "Эвристический анализ на основе ключевых слов",
		UsedLLM:     false,
	}

	text := strings.ToLower(lead.Title + " " + lead.Description)

	// Определяем тип лида по ключевым словам
	leadType, weights := a.detectLeadType(text)
	result.LeadType = leadType
	result.Weights = weights

	// Извлекаем критерии из requirement
	result.Criteria = a.extractCriteriaFromRequirement(lead)

	// Корректируем веса на основе заполненности данных
	result.Weights = a.adjustWeightsBasedOnData(result.Weights, lead, result.Criteria)

	// Генерируем объяснение
	result.Explanation = a.generateExplanation(result)

	return result
}

// detectLeadType определяет тип лида по ключевым словам.
func (a *Analyzer) detectLeadType(text string) (string, domain.MatchWeights) {
	// Ключевые слова для разных типов лидов
	budgetKeywords := []string{"бюджет", "недорого", "дешево", "эконом", "до ", "не более", "максимум"}
	locationKeywords := []string{"район", "рядом с", "около", "центр", "метро", "улица", "ЖК", "жилой комплекс"}
	familyKeywords := []string{"семья", "дети", "школ", "детский сад", "площадка", "большая", "просторн"}
	investorKeywords := []string{"инвест", "аренд", "доход", "окупаем", "сдавать", "бизнес"}
	luxuryKeywords := []string{"элит", "премиум", "люкс", "пентхаус", "вид", "панорам", "террас"}

	budgetScore := a.countKeywords(text, budgetKeywords)
	locationScore := a.countKeywords(text, locationKeywords)
	familyScore := a.countKeywords(text, familyKeywords)
	investorScore := a.countKeywords(text, investorKeywords)
	luxuryScore := a.countKeywords(text, luxuryKeywords)

	// Определяем доминирующий тип
	maxScore := budgetScore
	leadType := "budget_oriented"
	weights := domain.MatchWeights{Price: 0.45, District: 0.20, Rooms: 0.15, Area: 0.10, Semantic: 0.10}

	if locationScore > maxScore {
		maxScore = locationScore
		leadType = "location_oriented"
		weights = domain.MatchWeights{Price: 0.20, District: 0.40, Rooms: 0.15, Area: 0.10, Semantic: 0.15}
	}

	if familyScore > maxScore {
		maxScore = familyScore
		leadType = "family_oriented"
		weights = domain.MatchWeights{Price: 0.20, District: 0.20, Rooms: 0.30, Area: 0.20, Semantic: 0.10}
	}

	if investorScore > maxScore {
		maxScore = investorScore
		leadType = "investor"
		weights = domain.MatchWeights{Price: 0.35, District: 0.30, Rooms: 0.10, Area: 0.10, Semantic: 0.15}
	}

	if luxuryScore > maxScore {
		leadType = "luxury"
		weights = domain.MatchWeights{Price: 0.10, District: 0.25, Rooms: 0.15, Area: 0.20, Semantic: 0.30}
	}

	// Если ни один тип не определён явно
	if maxScore == 0 {
		leadType = "balanced"
		weights = domain.DefaultWeights()
	}

	return leadType, weights
}

// countKeywords подсчитывает количество вхождений ключевых слов.
func (a *Analyzer) countKeywords(text string, keywords []string) int {
	count := 0
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			count++
		}
	}
	return count
}

// extractCriteriaFromRequirement извлекает критерии из JSON requirement.
func (a *Analyzer) extractCriteriaFromRequirement(lead domain.Lead) *domain.SoftCriteria {
	if len(lead.Requirement) == 0 {
		return nil
	}

	var reqMap map[string]interface{}
	if err := json.Unmarshal(lead.Requirement, &reqMap); err != nil {
		return nil
	}

	criteria := &domain.SoftCriteria{}
	hasData := false

	if price, ok := reqMap["price"].(float64); ok {
		p := int64(price)
		criteria.TargetPrice = &p
		hasData = true
	}

	if district, ok := reqMap["district"].(string); ok {
		criteria.TargetDistrict = &district
		hasData = true
	}

	if rooms, ok := reqMap["roomNumber"].(float64); ok {
		r := int32(rooms)
		criteria.TargetRooms = &r
		hasData = true
	}

	if area, ok := reqMap["area"].(float64); ok {
		criteria.TargetArea = &area
		hasData = true
	}

	if !hasData {
		return nil
	}

	return criteria
}

// adjustWeightsBasedOnData корректирует веса на основе заполненности данных.
func (a *Analyzer) adjustWeightsBasedOnData(weights domain.MatchWeights, lead domain.Lead, criteria *domain.SoftCriteria) domain.MatchWeights {
	// Если у лида указан конкретный город, повышаем вес локации
	if lead.City != nil && *lead.City != "" {
		weights.District *= 1.2
	}

	// Если указан бюджет в criteria, повышаем вес цены
	if criteria != nil && criteria.TargetPrice != nil {
		weights.Price *= 1.2
	}

	// Если указан район в criteria, ещё больше повышаем вес локации
	if criteria != nil && criteria.TargetDistrict != nil {
		weights.District *= 1.3
	}

	// Если описание подробное (> 100 символов), повышаем вес семантики
	if len(lead.Description) > 100 {
		weights.Semantic *= 1.3
	}

	// Нормализуем веса
	return weights.Normalize()
}

// generateExplanation генерирует текстовое объяснение.
func (a *Analyzer) generateExplanation(result *AnalyzeResult) string {
	var parts []string

	switch result.LeadType {
	case "budget_oriented":
		parts = append(parts, "Клиент ориентирован на бюджет")
	case "location_oriented":
		parts = append(parts, "Локация — приоритет")
	case "family_oriented":
		parts = append(parts, "Запрос для семьи (важны комнаты и площадь)")
	case "investor":
		parts = append(parts, "Инвестиционный запрос")
	case "luxury":
		parts = append(parts, "Премиум-сегмент")
	default:
		parts = append(parts, "Сбалансированный запрос")
	}

	if result.Criteria != nil {
		if result.Criteria.TargetPrice != nil {
			parts = append(parts, fmt.Sprintf("бюджет ~%d₽", *result.Criteria.TargetPrice))
		}
		if result.Criteria.TargetDistrict != nil {
			parts = append(parts, fmt.Sprintf("район: %s", *result.Criteria.TargetDistrict))
		}
		if result.Criteria.TargetRooms != nil {
			parts = append(parts, fmt.Sprintf("%d комн.", *result.Criteria.TargetRooms))
		}
	}

	return strings.Join(parts, "; ")
}

// GetPresetByLeadType возвращает пресет весов по типу лида.
func (a *Analyzer) GetPresetByLeadType(leadType string) *domain.WeightPreset {
	presets := domain.GetWeightPresets()

	presetMap := map[string]string{
		"budget_oriented":   "budget_first",
		"location_oriented": "location_first",
		"family_oriented":   "family",
		"investor":          "budget_first",
		"luxury":            "semantic",
		"balanced":          "balanced",
	}

	presetID, ok := presetMap[leadType]
	if !ok {
		presetID = "balanced"
	}

	for _, p := range presets {
		if p.ID == presetID {
			return &p
		}
	}

	return nil
}

// IsShortLead определяет, является ли лид "коротким" (требует уточнения).
func (a *Analyzer) IsShortLead(lead domain.Lead) bool {
	// Лид считается коротким, если:
	// 1. Описание меньше 30 символов
	// 2. Нет requirement или он пустой
	// 3. Нет указания на цену, район или комнаты

	if len(lead.Description) < 30 {
		return true
	}

	if len(lead.Requirement) == 0 {
		return true
	}

	var reqMap map[string]interface{}
	if err := json.Unmarshal(lead.Requirement, &reqMap); err != nil {
		return true
	}

	// Проверяем наличие ключевых полей
	hasPrice := false
	hasRooms := false

	if _, ok := reqMap["price"]; ok {
		hasPrice = true
	}
	if _, ok := reqMap["roomNumber"]; ok {
		hasRooms = true
	}

	return !hasPrice && !hasRooms
}

// GetMissingFields возвращает список незаполненных важных полей.
func (a *Analyzer) GetMissingFields(lead domain.Lead) []string {
	var missing []string

	if lead.City == nil || *lead.City == "" {
		missing = append(missing, "city")
	}

	var reqMap map[string]interface{}
	if len(lead.Requirement) > 0 {
		json.Unmarshal(lead.Requirement, &reqMap)
	}

	if reqMap == nil {
		return append(missing, "price", "roomNumber", "area", "district")
	}

	if _, ok := reqMap["price"]; !ok {
		missing = append(missing, "price")
	}
	if _, ok := reqMap["roomNumber"]; !ok {
		missing = append(missing, "roomNumber")
	}
	if _, ok := reqMap["area"]; !ok {
		missing = append(missing, "area")
	}
	if _, ok := reqMap["district"]; !ok {
		missing = append(missing, "district")
	}

	return missing
}

