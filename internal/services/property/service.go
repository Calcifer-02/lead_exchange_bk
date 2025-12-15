package property

import (
	"context"
	"errors"
	"fmt"
	"lead_exchange/internal/domain"
	"lead_exchange/internal/lib/logger/sl"
	"lead_exchange/internal/lib/ml"
	"lead_exchange/internal/repository"
	"log/slog"

	"github.com/google/uuid"
)

type PropertyRepository interface {
	CreateProperty(ctx context.Context, property domain.Property) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.Property, error)
	UpdateProperty(ctx context.Context, propertyID uuid.UUID, update domain.PropertyFilter) error
	ListProperties(ctx context.Context, filter domain.PropertyFilter) ([]domain.Property, error)
	UpdateEmbedding(ctx context.Context, propertyID uuid.UUID, embedding []float32) error
	MatchProperties(ctx context.Context, leadEmbedding []float32, filter domain.PropertyFilter, limit int) ([]domain.MatchedProperty, error)
}

// LeadService нужен для получения embedding лида при матчинге.
type LeadService interface {
	GetLead(ctx context.Context, id uuid.UUID) (domain.Lead, error)
}

type Service struct {
	log        *slog.Logger
	repo       PropertyRepository
	mlClient   ml.Client
	leadService LeadService
}

var (
	ErrPropertyNotFound = errors.New("property not found")
)

func New(log *slog.Logger, repo PropertyRepository, mlClient ml.Client, leadService LeadService) *Service {
	return &Service{
		log:        log,
		repo:       repo,
		mlClient:   mlClient,
		leadService: leadService,
	}
}

// CreateProperty — создаёт новый объект недвижимости и генерирует embedding.
func (s *Service) CreateProperty(ctx context.Context, property domain.Property) (uuid.UUID, error) {
	const op = "property.Service.CreateProperty"
	log := s.log.With(slog.String("op", op), slog.String("title", property.Title))

	log.Info("creating new property")

	// Сначала сохраняем объект без embedding
	id, err := s.repo.CreateProperty(ctx, property)
	if err != nil {
		log.Error("failed to create property", sl.Err(err))
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("property created successfully", slog.String("property_id", id.String()))

	// Генерируем embedding асинхронно (в фоне)
	go func() {
		if err := s.generateAndUpdateEmbedding(context.Background(), id, property); err != nil {
			s.log.Error("failed to generate embedding", slog.String("property_id", id.String()), sl.Err(err))
		}
	}()

	return id, nil
}

// generateAndUpdateEmbedding генерирует embedding для объекта недвижимости и обновляет запись.
func (s *Service) generateAndUpdateEmbedding(ctx context.Context, propertyID uuid.UUID, property domain.Property) error {
	const op = "property.Service.generateAndUpdateEmbedding"

	// Подготавливаем запрос к ML сервису
	mlReq := ml.PrepareAndEmbedRequest{
		Title:       property.Title,
		Description: property.Description,
		Price:       property.Price,
		Rooms:       property.Rooms,
		Area:        property.Area,
		Address:     &property.Address,
	}

	// Получаем embedding от ML сервиса
	mlResp, err := s.mlClient.PrepareAndEmbed(ctx, mlReq)
	if err != nil {
		return fmt.Errorf("%s: failed to get embedding: %w", op, err)
	}

	// Конвертируем []float64 в []float32
	embedding := make([]float32, len(mlResp.Embedding))
	for i, v := range mlResp.Embedding {
		embedding[i] = float32(v)
	}

	// Обновляем embedding в БД
	if err := s.repo.UpdateEmbedding(ctx, propertyID, embedding); err != nil {
		return fmt.Errorf("%s: failed to update embedding: %w", op, err)
	}

	s.log.Info("embedding generated and updated", slog.String("property_id", propertyID.String()))
	return nil
}

// GetProperty — получает объект недвижимости по ID.
func (s *Service) GetProperty(ctx context.Context, id uuid.UUID) (domain.Property, error) {
	const op = "property.Service.GetProperty"

	property, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrPropertyNotFound) {
			s.log.Warn("property not found", slog.String("property_id", id.String()))
			return domain.Property{}, fmt.Errorf("%s: %w", op, ErrPropertyNotFound)
		}
		s.log.Error("failed to get property", sl.Err(err))
		return domain.Property{}, fmt.Errorf("%s: %w", op, err)
	}

	return property, nil
}

// UpdateProperty — частичное обновление данных объекта недвижимости.
func (s *Service) UpdateProperty(ctx context.Context, propertyID uuid.UUID, update domain.PropertyFilter) (domain.Property, error) {
	const op = "property.Service.UpdateProperty"

	err := s.repo.UpdateProperty(ctx, propertyID, update)
	if err != nil {
		if errors.Is(err, repository.ErrPropertyNotFound) {
			return domain.Property{}, fmt.Errorf("%s: %w", op, ErrPropertyNotFound)
		}
		return domain.Property{}, fmt.Errorf("%s: %w", op, err)
	}

	updated, err := s.repo.GetByID(ctx, propertyID)
	if err != nil {
		return domain.Property{}, fmt.Errorf("%s: failed to fetch updated property: %w", op, err)
	}

	// Переиндексируем embedding асинхронно, если изменились данные, влияющие на matching
	if update.Title != nil || update.Description != nil || update.Address != nil ||
		update.Price != nil || update.Rooms != nil || update.Area != nil {
		go func() {
			if err := s.reindexProperty(context.Background(), propertyID, updated); err != nil {
				s.log.Error("failed to reindex property", slog.String("property_id", propertyID.String()), sl.Err(err))
			}
		}()
	}

	return updated, nil
}

// ReindexProperty — публичный метод для ручной переиндексации объекта недвижимости.
func (s *Service) ReindexProperty(ctx context.Context, propertyID uuid.UUID) error {
	const op = "property.Service.ReindexProperty"

	property, err := s.repo.GetByID(ctx, propertyID)
	if err != nil {
		if errors.Is(err, repository.ErrPropertyNotFound) {
			return fmt.Errorf("%s: %w", op, ErrPropertyNotFound)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	if err := s.reindexProperty(ctx, propertyID, property); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// reindexProperty переиндексирует embedding для объекта недвижимости после обновления.
func (s *Service) reindexProperty(ctx context.Context, propertyID uuid.UUID, property domain.Property) error {
	const op = "property.Service.reindexProperty"

	// Подготавливаем запрос к ML сервису для переиндексации
	mlReq := ml.ReindexRequest{
		EntityID:    propertyID.String(),
		EntityType:  "property",
		Title:       property.Title,
		Description: property.Description,
		Price:       property.Price,
		Rooms:       property.Rooms,
		Area:        property.Area,
		Address:     &property.Address,
	}

	// Получаем новый embedding от ML сервиса
	mlResp, err := s.mlClient.Reindex(ctx, mlReq)
	if err != nil {
		return fmt.Errorf("%s: failed to reindex: %w", op, err)
	}

	// Конвертируем []float64 в []float32
	embedding := make([]float32, len(mlResp.Embedding))
	for i, v := range mlResp.Embedding {
		embedding[i] = float32(v)
	}

	// Обновляем embedding в БД
	if err := s.repo.UpdateEmbedding(ctx, propertyID, embedding); err != nil {
		return fmt.Errorf("%s: failed to update embedding: %w", op, err)
	}

	s.log.Info("property reindexed successfully", slog.String("property_id", propertyID.String()))
	return nil
}

// ListProperties — возвращает объекты недвижимости по фильтру.
func (s *Service) ListProperties(ctx context.Context, filter domain.PropertyFilter) ([]domain.Property, error) {
	const op = "property.Service.ListProperties"

	properties, err := s.repo.ListProperties(ctx, filter)
	if err != nil {
		s.log.Error("failed to list properties", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return properties, nil
}

// MatchProperties находит подходящие объекты недвижимости для лида по векторному сходству.
func (s *Service) MatchProperties(ctx context.Context, leadID uuid.UUID, filter domain.PropertyFilter, limit int) ([]domain.MatchedProperty, error) {
	return s.MatchPropertiesWeighted(ctx, leadID, filter, limit, nil, nil, false)
}

// MatchPropertiesWeighted находит объекты с поддержкой взвешенного ранжирования.
func (s *Service) MatchPropertiesWeighted(
	ctx context.Context,
	leadID uuid.UUID,
	filter domain.PropertyFilter,
	limit int,
	weights *domain.MatchWeights,
	criteria *domain.SoftCriteria,
	useWeightedRanking bool,
) ([]domain.MatchedProperty, error) {
	const op = "property.Service.MatchPropertiesWeighted"

	lead, err := s.leadService.GetLead(ctx, leadID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get lead: %w", op, err)
	}

	if len(lead.Embedding) == 0 {
		return nil, fmt.Errorf("%s: lead has no embedding", op)
	}

	if limit <= 0 {
		limit = 10
	}

	// Для взвешенного ранжирования получаем больше результатов
	fetchLimit := limit
	if useWeightedRanking {
		fetchLimit = limit * 5
		if fetchLimit > 100 {
			fetchLimit = 100
		}
	}

	matches, err := s.repo.MatchProperties(ctx, lead.Embedding, filter, fetchLimit)
	if err != nil {
		s.log.Error("failed to match properties", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if useWeightedRanking && len(matches) > 0 {
		w := domain.DefaultWeights()
		if weights != nil {
			w = weights.Normalize()
		}
		matches = s.rankMatches(matches, w, criteria)
		if len(matches) > limit {
			matches = matches[:limit]
		}
	}

	return matches, nil
}

// rankMatches применяет взвешенное ранжирование к результатам.
func (s *Service) rankMatches(matches []domain.MatchedProperty, w domain.MatchWeights, criteria *domain.SoftCriteria) []domain.MatchedProperty {
	for i := range matches {
		s.calculateScores(&matches[i], w, criteria)
	}
	// Сортируем по TotalScore (убывание)
	for i := 0; i < len(matches)-1; i++ {
		for j := i + 1; j < len(matches); j++ {
			scoreI, scoreJ := 0.0, 0.0
			if matches[i].TotalScore != nil {
				scoreI = *matches[i].TotalScore
			}
			if matches[j].TotalScore != nil {
				scoreJ = *matches[j].TotalScore
			}
			if scoreJ > scoreI {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}
	return matches
}

// calculateScores вычисляет все scores для одного матча.
func (s *Service) calculateScores(m *domain.MatchedProperty, w domain.MatchWeights, criteria *domain.SoftCriteria) {
	p := m.Property

	// Semantic score (косинусная близость уже 0-1)
	semantic := m.Similarity
	if semantic < 0 {
		semantic = (semantic + 1) / 2
	}

	// Price score
	price := s.calcPriceScore(p.Price, criteria)

	// District score
	district := s.calcDistrictScore(p.Address, criteria)

	// Rooms score
	rooms := s.calcRoomsScore(p.Rooms, criteria)

	// Area score
	area := s.calcAreaScore(p.Area, criteria)

	// Total weighted score
	total := w.Price*price + w.District*district + w.Rooms*rooms + w.Area*area + w.Semantic*semantic

	m.TotalScore = &total
	m.PriceScore = &price
	m.DistrictScore = &district
	m.RoomsScore = &rooms
	m.AreaScore = &area
	m.SemanticScore = &semantic

	// Генерируем объяснение
	expl := s.generateExplanation(m)
	m.MatchExplanation = &expl
}

func (s *Service) calcPriceScore(objPrice *int64, c *domain.SoftCriteria) float64 {
	if objPrice == nil || c == nil || c.TargetPrice == nil {
		return 0.5
	}
	target := float64(*c.TargetPrice)
	price := float64(*objPrice)
	if target == 0 {
		return 0.5
	}
	dev := abs(price-target) / target * 100
	if dev <= 20 {
		return 1.0 - (dev/20)*0.3
	}
	return max(0.0, 0.7-(dev-20)/100*0.7)
}

func (s *Service) calcDistrictScore(address string, c *domain.SoftCriteria) float64 {
	if c == nil || address == "" {
		return 0.3
	}
	addrLower := toLower(address)
	if c.TargetDistrict != nil {
		if contains(addrLower, toLower(*c.TargetDistrict)) {
			return 1.0
		}
	}
	for _, pref := range c.PreferredDistricts {
		if contains(addrLower, toLower(pref)) {
			return 0.7
		}
	}
	return 0.3
}

func (s *Service) calcRoomsScore(objRooms *int32, c *domain.SoftCriteria) float64 {
	if objRooms == nil || c == nil || c.TargetRooms == nil {
		return 0.5
	}
	diff := absInt(*objRooms - *c.TargetRooms)
	switch diff {
	case 0:
		return 1.0
	case 1:
		return 0.6
	case 2:
		return 0.3
	default:
		return 0.1
	}
}

func (s *Service) calcAreaScore(objArea *float64, c *domain.SoftCriteria) float64 {
	if objArea == nil || c == nil || c.TargetArea == nil || *c.TargetArea == 0 {
		return 0.5
	}
	dev := abs(*objArea-*c.TargetArea) / *c.TargetArea * 100
	if dev <= 15 {
		return 1.0 - (dev/15)*0.3
	}
	return max(0.0, 0.7-(dev-15)/50*0.7)
}

func (s *Service) generateExplanation(m *domain.MatchedProperty) string {
	var parts []string
	if m.PriceScore != nil && *m.PriceScore >= 0.7 && m.Property.Price != nil {
		parts = append(parts, fmt.Sprintf("цена %d₽ подходит", *m.Property.Price))
	}
	if m.DistrictScore != nil && *m.DistrictScore >= 0.7 {
		parts = append(parts, "район подходит")
	}
	if m.RoomsScore != nil && *m.RoomsScore >= 0.7 && m.Property.Rooms != nil {
		parts = append(parts, fmt.Sprintf("%d комн.", *m.Property.Rooms))
	}
	if m.AreaScore != nil && *m.AreaScore >= 0.7 && m.Property.Area != nil {
		parts = append(parts, fmt.Sprintf("%.0f м²", *m.Property.Area))
	}
	if m.SemanticScore != nil && *m.SemanticScore >= 0.6 {
		parts = append(parts, "описание соответствует")
	}
	if len(parts) == 0 {
		return "частичное совпадение"
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += "; " + parts[i]
	}
	return result
}

// Вспомогательные функции
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func absInt(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
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

