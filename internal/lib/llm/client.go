package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"lead_exchange/internal/config"
	"log/slog"
)

// Client — клиент для взаимодействия с LLM API (OpenAI, Azure OpenAI и др.).
type Client interface {
	// GenerateListingContent генерирует заголовок и описание для объекта недвижимости.
	GenerateListingContent(ctx context.Context, req GenerateListingRequest) (*GenerateListingResponse, error)
	// AnalyzeLeadIntent анализирует намерения из текста лида и определяет оптимальные веса.
	AnalyzeLeadIntent(ctx context.Context, req AnalyzeLeadRequest) (*AnalyzeLeadResponse, error)
	// GenerateClarificationQuestions генерирует уточняющие вопросы для неполного лида.
	GenerateClarificationQuestions(ctx context.Context, req ClarificationRequest) (*ClarificationResponse, error)
	// EnrichDescription обогащает описание объекта на основе структурированных данных.
	EnrichDescription(ctx context.Context, req EnrichDescriptionRequest) (*EnrichDescriptionResponse, error)
	// IsEnabled проверяет, включен ли сервис.
	IsEnabled() bool
}

// GenerateListingRequest — запрос на генерацию контента для листинга.
type GenerateListingRequest struct {
	PropertyType string   `json:"property_type"`
	Address      string   `json:"address"`
	City         string   `json:"city"`
	Price        *int64   `json:"price,omitempty"`
	Rooms        *int32   `json:"rooms,omitempty"`
	Area         *float64 `json:"area,omitempty"`
	Features     []string `json:"features,omitempty"`
	// ExistingTitle и ExistingDescription для улучшения существующего контента
	ExistingTitle       string `json:"existing_title,omitempty"`
	ExistingDescription string `json:"existing_description,omitempty"`
}

// GenerateListingResponse — ответ с сгенерированным контентом.
type GenerateListingResponse struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Keywords    []string `json:"keywords,omitempty"`
	Confidence  float64  `json:"confidence"`
}

// AnalyzeLeadRequest — запрос на анализ намерений лида.
type AnalyzeLeadRequest struct {
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Requirement map[string]interface{} `json:"requirement,omitempty"`
}

// AnalyzeLeadResponse — результат анализа лида с рекомендованными весами.
type AnalyzeLeadResponse struct {
	// RecommendedWeights — рекомендованные веса для матчинга
	RecommendedWeights WeightRecommendation `json:"recommended_weights"`
	// ExtractedCriteria — извлечённые критерии поиска
	ExtractedCriteria ExtractedCriteria `json:"extracted_criteria"`
	// LeadType — тип лида (budget_oriented, location_oriented, family_oriented, etc.)
	LeadType string `json:"lead_type"`
	// Confidence — уверенность в анализе (0-1)
	Confidence float64 `json:"confidence"`
	// Explanation — объяснение анализа
	Explanation string `json:"explanation"`
}

// WeightRecommendation — рекомендованные веса.
type WeightRecommendation struct {
	Price    float64 `json:"price"`
	District float64 `json:"district"`
	Rooms    float64 `json:"rooms"`
	Area     float64 `json:"area"`
	Semantic float64 `json:"semantic"`
}

// ExtractedCriteria — извлечённые критерии из текста лида.
type ExtractedCriteria struct {
	TargetPrice        *int64   `json:"target_price,omitempty"`
	TargetDistrict     *string  `json:"target_district,omitempty"`
	TargetRooms        *int32   `json:"target_rooms,omitempty"`
	TargetArea         *float64 `json:"target_area,omitempty"`
	PreferredDistricts []string `json:"preferred_districts,omitempty"`
	MustHaveFeatures   []string `json:"must_have_features,omitempty"`
	NiceToHaveFeatures []string `json:"nice_to_have_features,omitempty"`
}

// ClarificationRequest — запрос на генерацию уточняющих вопросов.
type ClarificationRequest struct {
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Requirement map[string]interface{} `json:"requirement,omitempty"`
	// MissingFields — поля, которые не заполнены
	MissingFields []string `json:"missing_fields,omitempty"`
}

// ClarificationResponse — уточняющие вопросы.
type ClarificationResponse struct {
	Questions []ClarificationQuestion `json:"questions"`
	Priority  string                  `json:"priority"` // high, medium, low
}

// ClarificationQuestion — один уточняющий вопрос.
type ClarificationQuestion struct {
	Field          string   `json:"field"`           // Поле, которое уточняется
	Question       string   `json:"question"`        // Текст вопроса
	QuestionType   string   `json:"question_type"`   // open, choice, range
	SuggestedOptions []string `json:"suggested_options,omitempty"`
	Importance     string   `json:"importance"`      // required, recommended, optional
}

// EnrichDescriptionRequest — запрос на обогащение описания.
type EnrichDescriptionRequest struct {
	CurrentDescription string                 `json:"current_description"`
	StructuredData     map[string]interface{} `json:"structured_data"`
	ImageAnalysis      *ImageAnalysisResult   `json:"image_analysis,omitempty"`
}

// ImageAnalysisResult — результат анализа изображений (от CV).
type ImageAnalysisResult struct {
	DetectedFeatures []string          `json:"detected_features"`
	RoomTypes        []string          `json:"room_types"`
	QualityScore     float64           `json:"quality_score"`
	Tags             map[string]string `json:"tags"`
}

// EnrichDescriptionResponse — обогащённое описание.
type EnrichDescriptionResponse struct {
	EnrichedDescription string   `json:"enriched_description"`
	AddedFeatures       []string `json:"added_features"`
	Confidence          float64  `json:"confidence"`
}

type client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	model      string
	log        *slog.Logger
}

// NewClient создаёт новый клиент для LLM API.
func NewClient(cfg config.LLMConfig, log *slog.Logger) Client {
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

// GenerateListingContent генерирует контент для листинга.
func (c *client) GenerateListingContent(ctx context.Context, req GenerateListingRequest) (*GenerateListingResponse, error) {
	const op = "llm.Client.GenerateListingContent"

	prompt := buildListingPrompt(req)

	chatReq := ChatCompletionRequest{
		Model: c.model,
		Messages: []ChatMessage{
			{
				Role:    "system",
				Content: "Ты — эксперт по недвижимости. Создавай привлекательные, информативные и точные заголовки и описания для объектов недвижимости. Ответ давай строго в формате JSON.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.7,
		MaxTokens:   500,
	}

	resp, err := c.sendChatRequest(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var result GenerateListingResponse
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		// Пытаемся извлечь JSON из текста
		jsonStr := extractJSON(resp.Content)
		if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
			return nil, fmt.Errorf("%s: failed to parse response: %w", op, err)
		}
	}

	return &result, nil
}

// AnalyzeLeadIntent анализирует намерения лида.
func (c *client) AnalyzeLeadIntent(ctx context.Context, req AnalyzeLeadRequest) (*AnalyzeLeadResponse, error) {
	const op = "llm.Client.AnalyzeLeadIntent"

	prompt := buildLeadAnalysisPrompt(req)

	chatReq := ChatCompletionRequest{
		Model: c.model,
		Messages: []ChatMessage{
			{
				Role: "system",
				Content: `Ты — AI-аналитик запросов на недвижимость. Анализируй текст лида и определяй:
1. Приоритеты клиента (бюджет, локация, размер и т.д.)
2. Рекомендованные веса для поиска (сумма = 1.0)
3. Извлечённые критерии поиска
4. Тип лида (budget_oriented, location_oriented, family_oriented, investor, luxury, first_time_buyer)
Ответ строго в формате JSON.`,
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.3,
		MaxTokens:   800,
	}

	resp, err := c.sendChatRequest(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var result AnalyzeLeadResponse
	jsonStr := extractJSON(resp.Content)
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("%s: failed to parse response: %w", op, err)
	}

	return &result, nil
}

// GenerateClarificationQuestions генерирует уточняющие вопросы.
func (c *client) GenerateClarificationQuestions(ctx context.Context, req ClarificationRequest) (*ClarificationResponse, error) {
	const op = "llm.Client.GenerateClarificationQuestions"

	prompt := buildClarificationPrompt(req)

	chatReq := ChatCompletionRequest{
		Model: c.model,
		Messages: []ChatMessage{
			{
				Role: "system",
				Content: `Ты — AI-ассистент риелтора. Генерируй релевантные уточняющие вопросы для клиентов,
чтобы лучше понять их потребности. Вопросы должны быть вежливыми, конкретными и помогать
найти идеальный объект недвижимости. Ответ строго в формате JSON.`,
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.5,
		MaxTokens:   600,
	}

	resp, err := c.sendChatRequest(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var result ClarificationResponse
	jsonStr := extractJSON(resp.Content)
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("%s: failed to parse response: %w", op, err)
	}

	return &result, nil
}

// EnrichDescription обогащает описание объекта.
func (c *client) EnrichDescription(ctx context.Context, req EnrichDescriptionRequest) (*EnrichDescriptionResponse, error) {
	const op = "llm.Client.EnrichDescription"

	prompt := buildEnrichmentPrompt(req)

	chatReq := ChatCompletionRequest{
		Model: c.model,
		Messages: []ChatMessage{
			{
				Role: "system",
				Content: `Ты — эксперт по созданию описаний недвижимости. Обогащай существующие описания,
добавляя релевантную информацию из структурированных данных и результатов анализа фотографий.
Сохраняй стиль оригинального описания. Ответ строго в формате JSON.`,
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.6,
		MaxTokens:   800,
	}

	resp, err := c.sendChatRequest(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var result EnrichDescriptionResponse
	jsonStr := extractJSON(resp.Content)
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("%s: failed to parse response: %w", op, err)
	}

	return &result, nil
}

func (c *client) IsEnabled() bool {
	return true
}

// ChatCompletionRequest — запрос к Chat Completion API.
type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

// ChatMessage — сообщение в чате.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionResponse — ответ от Chat Completion API.
type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
}

type simplifiedResponse struct {
	Content string
}

func (c *client) sendChatRequest(ctx context.Context, req ChatCompletionRequest) (*simplifiedResponse, error) {
	const op = "llm.Client.sendChatRequest"

	url := fmt.Sprintf("%s/chat/completions", c.baseURL)

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

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to send request: %w", op, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%s: unexpected status code %d: %s", op, resp.StatusCode, string(body))
	}

	var chatResp ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("%s: failed to decode response: %w", op, err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("%s: no choices in response", op)
	}

	return &simplifiedResponse{
		Content: chatResp.Choices[0].Message.Content,
	}, nil
}

// Вспомогательные функции для построения промптов

func buildListingPrompt(req GenerateListingRequest) string {
	var sb strings.Builder
	sb.WriteString("Создай привлекательный заголовок и описание для объекта недвижимости:\n\n")
	sb.WriteString(fmt.Sprintf("Тип: %s\n", req.PropertyType))
	sb.WriteString(fmt.Sprintf("Адрес: %s\n", req.Address))
	sb.WriteString(fmt.Sprintf("Город: %s\n", req.City))

	if req.Price != nil {
		sb.WriteString(fmt.Sprintf("Цена: %d руб.\n", *req.Price))
	}
	if req.Rooms != nil {
		sb.WriteString(fmt.Sprintf("Комнат: %d\n", *req.Rooms))
	}
	if req.Area != nil {
		sb.WriteString(fmt.Sprintf("Площадь: %.1f м²\n", *req.Area))
	}
	if len(req.Features) > 0 {
		sb.WriteString(fmt.Sprintf("Особенности: %s\n", strings.Join(req.Features, ", ")))
	}
	if req.ExistingTitle != "" {
		sb.WriteString(fmt.Sprintf("\nТекущий заголовок (улучши): %s\n", req.ExistingTitle))
	}
	if req.ExistingDescription != "" {
		sb.WriteString(fmt.Sprintf("Текущее описание (улучши): %s\n", req.ExistingDescription))
	}

	sb.WriteString("\nОтвет в формате JSON: {\"title\": \"...\", \"description\": \"...\", \"keywords\": [...], \"confidence\": 0.9}")
	return sb.String()
}

func buildLeadAnalysisPrompt(req AnalyzeLeadRequest) string {
	var sb strings.Builder
	sb.WriteString("Проанализируй запрос клиента на недвижимость:\n\n")
	sb.WriteString(fmt.Sprintf("Заголовок: %s\n", req.Title))
	sb.WriteString(fmt.Sprintf("Описание: %s\n", req.Description))

	if req.Requirement != nil {
		reqJSON, _ := json.Marshal(req.Requirement)
		sb.WriteString(fmt.Sprintf("Требования: %s\n", string(reqJSON)))
	}

	sb.WriteString(`
Определи:
1. recommended_weights — веса для поиска (price, district, rooms, area, semantic), сумма = 1.0
2. extracted_criteria — извлечённые критерии (target_price, target_district, target_rooms, target_area, preferred_districts, must_have_features, nice_to_have_features)
3. lead_type — тип клиента (budget_oriented, location_oriented, family_oriented, investor, luxury, first_time_buyer)
4. confidence — уверенность анализа (0-1)
5. explanation — краткое объяснение

Ответ в формате JSON.`)
	return sb.String()
}

func buildClarificationPrompt(req ClarificationRequest) string {
	var sb strings.Builder
	sb.WriteString("Клиент оставил запрос на недвижимость с недостаточной информацией:\n\n")
	sb.WriteString(fmt.Sprintf("Заголовок: %s\n", req.Title))
	sb.WriteString(fmt.Sprintf("Описание: %s\n", req.Description))

	if len(req.MissingFields) > 0 {
		sb.WriteString(fmt.Sprintf("Незаполненные поля: %s\n", strings.Join(req.MissingFields, ", ")))
	}

	sb.WriteString(`
Сгенерируй уточняющие вопросы в формате JSON:
{
  "questions": [
    {
      "field": "price",
      "question": "Какой у вас примерный бюджет?",
      "question_type": "range",
      "suggested_options": ["до 5 млн", "5-10 млн", "10-15 млн", "от 15 млн"],
      "importance": "required"
    }
  ],
  "priority": "high"
}`)
	return sb.String()
}

func buildEnrichmentPrompt(req EnrichDescriptionRequest) string {
	var sb strings.Builder
	sb.WriteString("Обогати описание объекта недвижимости:\n\n")
	sb.WriteString(fmt.Sprintf("Текущее описание: %s\n", req.CurrentDescription))

	if req.StructuredData != nil {
		dataJSON, _ := json.Marshal(req.StructuredData)
		sb.WriteString(fmt.Sprintf("Структурированные данные: %s\n", string(dataJSON)))
	}

	if req.ImageAnalysis != nil {
		sb.WriteString(fmt.Sprintf("Результаты анализа фото:\n"))
		sb.WriteString(fmt.Sprintf("- Обнаруженные особенности: %s\n", strings.Join(req.ImageAnalysis.DetectedFeatures, ", ")))
		sb.WriteString(fmt.Sprintf("- Типы комнат: %s\n", strings.Join(req.ImageAnalysis.RoomTypes, ", ")))
		sb.WriteString(fmt.Sprintf("- Оценка качества: %.2f\n", req.ImageAnalysis.QualityScore))
	}

	sb.WriteString(`
Ответ в формате JSON:
{
  "enriched_description": "...",
  "added_features": ["..."],
  "confidence": 0.9
}`)
	return sb.String()
}

// extractJSON извлекает JSON из текста ответа LLM.
func extractJSON(text string) string {
	// Ищем первую { и последнюю }
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")

	if start != -1 && end != -1 && end > start {
		return text[start : end+1]
	}
	return text
}

// noopClient — заглушка для случая, когда LLM отключен.
type noopClient struct {
	log *slog.Logger
}

func (c *noopClient) GenerateListingContent(ctx context.Context, req GenerateListingRequest) (*GenerateListingResponse, error) {
	c.log.Debug("LLM service is disabled")
	return &GenerateListingResponse{
		Title:       req.ExistingTitle,
		Description: req.ExistingDescription,
		Confidence:  0,
	}, nil
}

func (c *noopClient) AnalyzeLeadIntent(ctx context.Context, req AnalyzeLeadRequest) (*AnalyzeLeadResponse, error) {
	c.log.Debug("LLM service is disabled, returning default weights")
	return &AnalyzeLeadResponse{
		RecommendedWeights: WeightRecommendation{
			Price:    0.30,
			District: 0.25,
			Rooms:    0.20,
			Area:     0.10,
			Semantic: 0.15,
		},
		LeadType:   "unknown",
		Confidence: 0,
		Explanation: "LLM service disabled",
	}, nil
}

func (c *noopClient) GenerateClarificationQuestions(ctx context.Context, req ClarificationRequest) (*ClarificationResponse, error) {
	c.log.Debug("LLM service is disabled")
	return &ClarificationResponse{
		Questions: []ClarificationQuestion{},
		Priority:  "low",
	}, nil
}

func (c *noopClient) EnrichDescription(ctx context.Context, req EnrichDescriptionRequest) (*EnrichDescriptionResponse, error) {
	c.log.Debug("LLM service is disabled")
	return &EnrichDescriptionResponse{
		EnrichedDescription: req.CurrentDescription,
		Confidence:          0,
	}, nil
}

func (c *noopClient) IsEnabled() bool {
	return false
}

