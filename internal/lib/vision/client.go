package vision

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"lead_exchange/internal/config"
	"log/slog"
)

// Client — клиент для взаимодействия с CV API (Computer Vision).
type Client interface {
	// AnalyzeImage анализирует одно изображение.
	AnalyzeImage(ctx context.Context, imageData []byte) (*ImageAnalysis, error)
	// AnalyzeImages анализирует несколько изображений.
	AnalyzeImages(ctx context.Context, images [][]byte) (*PropertyImageAnalysis, error)
	// AnalyzeImageURL анализирует изображение по URL.
	AnalyzeImageURL(ctx context.Context, imageURL string) (*ImageAnalysis, error)
	// IsEnabled проверяет, включен ли сервис.
	IsEnabled() bool
}

// ImageAnalysis — результат анализа одного изображения.
type ImageAnalysis struct {
	// DetectedFeatures — обнаруженные особенности (балкон, панорамные окна и т.д.)
	DetectedFeatures []Feature `json:"detected_features"`
	// RoomType — тип комнаты (кухня, спальня, гостиная и т.д.)
	RoomType string `json:"room_type,omitempty"`
	// QualityScore — оценка качества отделки (0-1)
	QualityScore float64 `json:"quality_score"`
	// ViewType — тип вида из окна (город, парк, двор и т.д.)
	ViewType string `json:"view_type,omitempty"`
	// Brightness — уровень освещённости (0-1)
	Brightness float64 `json:"brightness"`
	// Tags — дополнительные теги
	Tags map[string]string `json:"tags,omitempty"`
	// Confidence — уверенность в анализе
	Confidence float64 `json:"confidence"`
}

// Feature — обнаруженная особенность.
type Feature struct {
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
	Category   string  `json:"category"` // interior, exterior, view, amenity
}

// PropertyImageAnalysis — агрегированный результат анализа всех фото объекта.
type PropertyImageAnalysis struct {
	// TotalImages — количество проанализированных изображений
	TotalImages int `json:"total_images"`
	// AverageQuality — средняя оценка качества
	AverageQuality float64 `json:"average_quality"`
	// DetectedRooms — обнаруженные типы комнат
	DetectedRooms []string `json:"detected_rooms"`
	// AllFeatures — все обнаруженные особенности (дедуплицированные)
	AllFeatures []Feature `json:"all_features"`
	// ViewTypes — типы видов из окон
	ViewTypes []string `json:"view_types"`
	// OverallAssessment — общая оценка объекта
	OverallAssessment string `json:"overall_assessment"`
	// EmbeddingFeatures — признаки для включения в эмбеддинг
	EmbeddingFeatures []float64 `json:"embedding_features"`
}

type client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	log        *slog.Logger
}

// NewClient создаёт новый клиент для CV API.
func NewClient(cfg config.VisionConfig, log *slog.Logger) Client {
	if !cfg.Enabled {
		return &noopClient{log: log}
	}

	return &client{
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		baseURL: cfg.BaseURL,
		apiKey:  cfg.APIKey,
		log:     log,
	}
}

// AnalyzeImage анализирует одно изображение.
func (c *client) AnalyzeImage(ctx context.Context, imageData []byte) (*ImageAnalysis, error) {
	const op = "vision.Client.AnalyzeImage"

	// Кодируем изображение в base64
	encoded := base64.StdEncoding.EncodeToString(imageData)

	req := visionRequest{
		Image: encoded,
		Features: []string{
			"room_type",
			"quality",
			"features",
			"view",
		},
	}

	resp, err := c.sendRequest(ctx, "/analyze", req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return resp, nil
}

// AnalyzeImages анализирует несколько изображений и агрегирует результаты.
func (c *client) AnalyzeImages(ctx context.Context, images [][]byte) (*PropertyImageAnalysis, error) {
	const op = "vision.Client.AnalyzeImages"

	if len(images) == 0 {
		return &PropertyImageAnalysis{}, nil
	}

	var analyses []*ImageAnalysis
	for i, img := range images {
		analysis, err := c.AnalyzeImage(ctx, img)
		if err != nil {
			c.log.Warn("failed to analyze image",
				slog.Int("index", i),
				slog.String("error", err.Error()),
			)
			continue
		}
		analyses = append(analyses, analysis)
	}

	return c.aggregateAnalyses(analyses), nil
}

// AnalyzeImageURL анализирует изображение по URL.
func (c *client) AnalyzeImageURL(ctx context.Context, imageURL string) (*ImageAnalysis, error) {
	const op = "vision.Client.AnalyzeImageURL"

	req := visionRequest{
		URL: imageURL,
		Features: []string{
			"room_type",
			"quality",
			"features",
			"view",
		},
	}

	resp, err := c.sendRequest(ctx, "/analyze-url", req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return resp, nil
}

func (c *client) IsEnabled() bool {
	return true
}

type visionRequest struct {
	Image    string   `json:"image,omitempty"`
	URL      string   `json:"url,omitempty"`
	Features []string `json:"features"`
}

func (c *client) sendRequest(ctx context.Context, endpoint string, req visionRequest) (*ImageAnalysis, error) {
	const op = "vision.Client.sendRequest"

	url := c.baseURL + endpoint

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

	var result ImageAnalysis
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("%s: failed to decode response: %w", op, err)
	}

	return &result, nil
}

// aggregateAnalyses агрегирует результаты анализа нескольких изображений.
func (c *client) aggregateAnalyses(analyses []*ImageAnalysis) *PropertyImageAnalysis {
	if len(analyses) == 0 {
		return &PropertyImageAnalysis{}
	}

	result := &PropertyImageAnalysis{
		TotalImages: len(analyses),
	}

	featuresMap := make(map[string]Feature)
	roomsSet := make(map[string]bool)
	viewsSet := make(map[string]bool)
	var totalQuality float64

	for _, a := range analyses {
		totalQuality += a.QualityScore

		if a.RoomType != "" {
			roomsSet[a.RoomType] = true
		}

		if a.ViewType != "" {
			viewsSet[a.ViewType] = true
		}

		for _, f := range a.DetectedFeatures {
			if existing, ok := featuresMap[f.Name]; ok {
				// Обновляем confidence если новый выше
				if f.Confidence > existing.Confidence {
					featuresMap[f.Name] = f
				}
			} else {
				featuresMap[f.Name] = f
			}
		}
	}

	result.AverageQuality = totalQuality / float64(len(analyses))

	for room := range roomsSet {
		result.DetectedRooms = append(result.DetectedRooms, room)
	}

	for view := range viewsSet {
		result.ViewTypes = append(result.ViewTypes, view)
	}

	for _, f := range featuresMap {
		result.AllFeatures = append(result.AllFeatures, f)
	}

	// Генерируем общую оценку
	result.OverallAssessment = c.generateAssessment(result)

	// Генерируем embedding features
	result.EmbeddingFeatures = c.generateEmbeddingFeatures(result)

	return result
}

// generateAssessment генерирует текстовую оценку объекта.
func (c *client) generateAssessment(analysis *PropertyImageAnalysis) string {
	var parts []string

	if analysis.AverageQuality >= 0.8 {
		parts = append(parts, "отличное состояние")
	} else if analysis.AverageQuality >= 0.6 {
		parts = append(parts, "хорошее состояние")
	} else if analysis.AverageQuality >= 0.4 {
		parts = append(parts, "удовлетворительное состояние")
	} else {
		parts = append(parts, "требует ремонта")
	}

	if len(analysis.DetectedRooms) > 0 {
		parts = append(parts, fmt.Sprintf("%d типов помещений", len(analysis.DetectedRooms)))
	}

	premiumFeatures := 0
	for _, f := range analysis.AllFeatures {
		if f.Category == "premium" || f.Name == "panoramic_windows" || f.Name == "high_ceilings" {
			premiumFeatures++
		}
	}

	if premiumFeatures > 0 {
		parts = append(parts, fmt.Sprintf("%d премиум-особенностей", premiumFeatures))
	}

	if len(analysis.ViewTypes) > 0 {
		hasGoodView := false
		for _, v := range analysis.ViewTypes {
			if v == "park" || v == "river" || v == "panorama" {
				hasGoodView = true
				break
			}
		}
		if hasGoodView {
			parts = append(parts, "хороший вид")
		}
	}

	return strings.Join(parts, ", ")
}

// generateEmbeddingFeatures генерирует числовые признаки для эмбеддинга.
func (c *client) generateEmbeddingFeatures(analysis *PropertyImageAnalysis) []float64 {
	// Генерируем вектор из 16 признаков
	features := make([]float64, 16)

	// 0: Quality score
	features[0] = analysis.AverageQuality

	// 1: Number of rooms (normalized)
	features[1] = float64(len(analysis.DetectedRooms)) / 10.0
	if features[1] > 1.0 {
		features[1] = 1.0
	}

	// 2: Number of premium features (normalized)
	premiumCount := 0
	for _, f := range analysis.AllFeatures {
		if f.Category == "premium" {
			premiumCount++
		}
	}
	features[2] = float64(premiumCount) / 5.0
	if features[2] > 1.0 {
		features[2] = 1.0
	}

	// 3: Has good view
	for _, v := range analysis.ViewTypes {
		if v == "park" || v == "river" || v == "panorama" {
			features[3] = 1.0
			break
		}
	}

	// 4-15: Feature presence flags
	featureNames := []string{
		"balcony", "terrace", "panoramic_windows", "high_ceilings",
		"modern_kitchen", "master_bedroom", "walk_in_closet", "bathroom_modern",
		"parking", "security", "gym", "pool",
	}

	featureSet := make(map[string]bool)
	for _, f := range analysis.AllFeatures {
		featureSet[f.Name] = true
	}

	for i, name := range featureNames {
		if featureSet[name] {
			features[4+i] = 1.0
		}
	}

	return features
}

// noopClient — заглушка для случая, когда CV сервис отключен.
type noopClient struct {
	log *slog.Logger
}

func (c *noopClient) AnalyzeImage(ctx context.Context, imageData []byte) (*ImageAnalysis, error) {
	c.log.Debug("Vision service is disabled")
	return &ImageAnalysis{
		Confidence: 0,
	}, nil
}

func (c *noopClient) AnalyzeImages(ctx context.Context, images [][]byte) (*PropertyImageAnalysis, error) {
	c.log.Debug("Vision service is disabled")
	return &PropertyImageAnalysis{}, nil
}

func (c *noopClient) AnalyzeImageURL(ctx context.Context, imageURL string) (*ImageAnalysis, error) {
	c.log.Debug("Vision service is disabled")
	return &ImageAnalysis{
		Confidence: 0,
	}, nil
}

func (c *noopClient) IsEnabled() bool {
	return false
}

