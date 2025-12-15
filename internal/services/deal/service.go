package deal

import (
	"context"
	"errors"
	"fmt"
	"lead_exchange/internal/domain"
	"lead_exchange/internal/lib/logger/sl"
	"lead_exchange/internal/repository"
	"log/slog"

	"github.com/google/uuid"
	"github.com/samber/lo"
)

type DealRepository interface {
	CreateDeal(ctx context.Context, deal domain.Deal) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.Deal, error)
	UpdateDeal(ctx context.Context, dealID uuid.UUID, update domain.DealFilter) error
	ListDeals(ctx context.Context, filter domain.DealFilter) ([]domain.Deal, error)
}

type Service struct {
	log  *slog.Logger
	repo DealRepository
}

var (
	ErrDealNotFound = errors.New("deal not found")
)

func New(log *slog.Logger, repo DealRepository) *Service {
	return &Service{
		log:  log,
		repo: repo,
	}
}

// CreateDeal — создаёт новую сделку.
func (s *Service) CreateDeal(ctx context.Context, deal domain.Deal) (uuid.UUID, error) {
	const op = "deal.Service.CreateDeal"
	log := s.log.With(slog.String("op", op), slog.String("lead_id", deal.LeadID.String()))

	log.Info("creating new deal")

	id, err := s.repo.CreateDeal(ctx, deal)
	if err != nil {
		log.Error("failed to create deal", sl.Err(err))
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("deal created successfully", slog.String("deal_id", id.String()))
	return id, nil
}

// GetDeal — получает сделку по ID.
func (s *Service) GetDeal(ctx context.Context, id uuid.UUID) (domain.Deal, error) {
	const op = "deal.Service.GetDeal"

	deal, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrDealNotFound) {
			s.log.Warn("deal not found", slog.String("deal_id", id.String()))
			return domain.Deal{}, fmt.Errorf("%s: %w", op, ErrDealNotFound)
		}
		s.log.Error("failed to get deal", sl.Err(err))
		return domain.Deal{}, fmt.Errorf("%s: %w", op, err)
	}

	return deal, nil
}

// UpdateDeal — частичное обновление данных сделки.
func (s *Service) UpdateDeal(ctx context.Context, dealID uuid.UUID, update domain.DealFilter) (domain.Deal, error) {
	const op = "deal.Service.UpdateDeal"

	err := s.repo.UpdateDeal(ctx, dealID, update)
	if err != nil {
		if errors.Is(err, repository.ErrDealNotFound) {
			return domain.Deal{}, fmt.Errorf("%s: %w", op, ErrDealNotFound)
		}
		return domain.Deal{}, fmt.Errorf("%s: %w", op, err)
	}

	updated, err := s.repo.GetByID(ctx, dealID)
	if err != nil {
		return domain.Deal{}, fmt.Errorf("%s: failed to fetch updated deal: %w", op, err)
	}

	return updated, nil
}

// ListDeals — возвращает сделки по фильтру.
func (s *Service) ListDeals(ctx context.Context, filter domain.DealFilter) ([]domain.Deal, error) {
	const op = "deal.Service.ListDeals"

	deals, err := s.repo.ListDeals(ctx, filter)
	if err != nil {
		s.log.Error("failed to list deals", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return deals, nil
}

// AcceptDeal — принимает сделку (покупатель принимает предложение).
func (s *Service) AcceptDeal(ctx context.Context, dealID uuid.UUID, buyerUserID uuid.UUID) (domain.Deal, error) {
	const op = "deal.Service.AcceptDeal"

	// Получаем текущую сделку
	deal, err := s.repo.GetByID(ctx, dealID)
	if err != nil {
		if errors.Is(err, repository.ErrDealNotFound) {
			return domain.Deal{}, fmt.Errorf("%s: %w", op, ErrDealNotFound)
		}
		return domain.Deal{}, fmt.Errorf("%s: %w", op, err)
	}

	// Проверяем, что сделка в статусе PENDING
	if deal.Status != domain.DealStatusPending {
		return domain.Deal{}, fmt.Errorf("%s: deal is not in PENDING status", op)
	}

	// Обновляем сделку: устанавливаем покупателя и статус ACCEPTED
	update := domain.DealFilter{
		BuyerUserID: &buyerUserID,
		Status:      lo.ToPtr(domain.DealStatusAccepted),
	}

	return s.UpdateDeal(ctx, dealID, update)
}
