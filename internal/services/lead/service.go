package lead

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"lead_exchange/internal/domain"
	"lead_exchange/internal/lib/logger/sl"
	"lead_exchange/internal/lib/ml"
	"lead_exchange/internal/repository"
	"log/slog"

	"github.com/google/uuid"
)

type LeadRepository interface {
	CreateLead(ctx context.Context, lead domain.Lead) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.Lead, error)
	UpdateLead(ctx context.Context, leadID uuid.UUID, update domain.LeadFilter) error
	ListLeads(ctx context.Context, filter domain.LeadFilter) (*domain.PaginatedResult[domain.Lead], error)
	UpdateEmbedding(ctx context.Context, leadID uuid.UUID, embedding []float32) error
}

type Service struct {
	log      *slog.Logger
	repo     LeadRepository
	mlClient ml.Client
}

var (
	ErrLeadNotFound = errors.New("lead not found")
)

func New(log *slog.Logger, repo LeadRepository, mlClient ml.Client) *Service {
	return &Service{
		log:      log,
		repo:     repo,
		mlClient: mlClient,
	}
}

// CreateLead — создаёт нового лида и генерирует embedding.
func (s *Service) CreateLead(ctx context.Context, lead domain.Lead) (uuid.UUID, error) {
	const op = "lead.Service.CreateLead"
	log := s.log.With(slog.String("op", op), slog.String("title", lead.Title))

	log.Info("creating new lead")

	// Сначала сохраняем лид без embedding
	id, err := s.repo.CreateLead(ctx, lead)
	if err != nil {
		log.Error("failed to create lead", sl.Err(err))
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("lead created successfully", slog.String("lead_id", id.String()))

	// Генерируем embedding асинхронно (в фоне)
	go func() {
		if err := s.generateAndUpdateEmbedding(context.Background(), id, lead); err != nil {
			s.log.Error("failed to generate embedding", slog.String("lead_id", id.String()), sl.Err(err))
		}
	}()

	return id, nil
}

// generateAndUpdateEmbedding генерирует embedding для лида и обновляет запись.
func (s *Service) generateAndUpdateEmbedding(ctx context.Context, leadID uuid.UUID, lead domain.Lead) error {
	const op = "lead.Service.generateAndUpdateEmbedding"

	// Парсим requirement из JSON
	var requirementMap map[string]interface{}
	if len(lead.Requirement) > 0 {
		if err := json.Unmarshal(lead.Requirement, &requirementMap); err != nil {
			s.log.Warn("failed to parse requirement JSON", sl.Err(err))
			requirementMap = nil
		}
	}

	// Извлекаем данные из requirement для ML сервиса
	var price *int64
	var district *string
	var rooms *int32
	var area *float64

	if requirementMap != nil {
		if p, ok := requirementMap["price"].(float64); ok {
			priceVal := int64(p)
			price = &priceVal
		}
		if d, ok := requirementMap["district"].(string); ok {
			district = &d
		}
		if r, ok := requirementMap["roomNumber"].(float64); ok {
			roomsVal := int32(r)
			rooms = &roomsVal
		}
		if a, ok := requirementMap["area"].(float64); ok {
			area = &a
		}
	}

	// Подготавливаем запрос к ML сервису
	mlReq := ml.PrepareAndEmbedRequest{
		Title:       lead.Title,
		Description: lead.Description,
		Requirement: requirementMap,
		Price:       price,
		District:    district,
		Rooms:       rooms,
		Area:        area,
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
	if err := s.repo.UpdateEmbedding(ctx, leadID, embedding); err != nil {
		return fmt.Errorf("%s: failed to update embedding: %w", op, err)
	}

	s.log.Info("embedding generated and updated", slog.String("lead_id", leadID.String()))
	return nil
}

// GetLead — получает лида по ID.
func (s *Service) GetLead(ctx context.Context, id uuid.UUID) (domain.Lead, error) {
	const op = "lead.Service.GetLead"

	lead, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrLeadNotFound) {
			s.log.Warn("lead not found", slog.String("lead_id", id.String()))
			return domain.Lead{}, fmt.Errorf("%s: %w", op, ErrLeadNotFound)
		}
		s.log.Error("failed to get lead", sl.Err(err))
		return domain.Lead{}, fmt.Errorf("%s: %w", op, err)
	}

	return lead, nil
}

// UpdateLead — частичное обновление данных лида.
func (s *Service) UpdateLead(ctx context.Context, leadID uuid.UUID, update domain.LeadFilter) (domain.Lead, error) {
	const op = "lead.Service.UpdateLead"

	err := s.repo.UpdateLead(ctx, leadID, update)
	if err != nil {
		if errors.Is(err, repository.ErrLeadNotFound) {
			return domain.Lead{}, fmt.Errorf("%s: %w", op, ErrLeadNotFound)
		}
		return domain.Lead{}, fmt.Errorf("%s: %w", op, err)
	}

	updated, err := s.repo.GetByID(ctx, leadID)
	if err != nil {
		return domain.Lead{}, fmt.Errorf("%s: failed to fetch updated lead: %w", op, err)
	}

	// Переиндексируем embedding асинхронно, если изменились данные, влияющие на matching
	if update.Title != nil || update.Description != nil || update.Requirement != nil {
		go func() {
			if err := s.reindexLead(context.Background(), leadID, updated); err != nil {
				s.log.Error("failed to reindex lead", slog.String("lead_id", leadID.String()), sl.Err(err))
			}
		}()
	}

	return updated, nil
}

// ReindexLead — публичный метод для ручной переиндексации лида.
func (s *Service) ReindexLead(ctx context.Context, leadID uuid.UUID) error {
	const op = "lead.Service.ReindexLead"

	lead, err := s.repo.GetByID(ctx, leadID)
	if err != nil {
		if errors.Is(err, repository.ErrLeadNotFound) {
			return fmt.Errorf("%s: %w", op, ErrLeadNotFound)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	if err := s.reindexLead(ctx, leadID, lead); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// reindexLead переиндексирует embedding для лида после обновления.
func (s *Service) reindexLead(ctx context.Context, leadID uuid.UUID, lead domain.Lead) error {
	const op = "lead.Service.reindexLead"

	// Парсим requirement из JSON
	var requirementMap map[string]interface{}
	if len(lead.Requirement) > 0 {
		if err := json.Unmarshal(lead.Requirement, &requirementMap); err != nil {
			s.log.Warn("failed to parse requirement JSON", sl.Err(err))
			requirementMap = nil
		}
	}

	// Извлекаем данные из requirement для ML сервиса
	var price *int64
	var district *string
	var rooms *int32
	var area *float64

	if requirementMap != nil {
		if p, ok := requirementMap["price"].(float64); ok {
			priceVal := int64(p)
			price = &priceVal
		}
		if d, ok := requirementMap["district"].(string); ok {
			district = &d
		}
		if r, ok := requirementMap["roomNumber"].(float64); ok {
			roomsVal := int32(r)
			rooms = &roomsVal
		}
		if a, ok := requirementMap["area"].(float64); ok {
			area = &a
		}
	}

	// Подготавливаем запрос к ML сервису для переиндексации
	mlReq := ml.ReindexRequest{
		EntityID:    leadID.String(),
		EntityType:  "lead",
		Title:       lead.Title,
		Description: lead.Description,
		Price:       price,
		District:    district,
		Rooms:       rooms,
		Area:        area,
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
	if err := s.repo.UpdateEmbedding(ctx, leadID, embedding); err != nil {
		return fmt.Errorf("%s: failed to update embedding: %w", op, err)
	}

	s.log.Info("lead reindexed successfully", slog.String("lead_id", leadID.String()))
	return nil
}

// ListLeads — возвращает лидов по фильтру с пагинацией.
func (s *Service) ListLeads(ctx context.Context, filter domain.LeadFilter) (*domain.PaginatedResult[domain.Lead], error) {
	const op = "lead.Service.ListLeads"

	result, err := s.repo.ListLeads(ctx, filter)
	if err != nil {
		s.log.Error("failed to list leads", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return result, nil
}
