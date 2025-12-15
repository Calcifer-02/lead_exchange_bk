package deal_repository

import (
	"context"
	"errors"
	"fmt"
	"lead_exchange/internal/domain"
	"lead_exchange/internal/repository"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DealRepository struct {
	db  *pgxpool.Pool
	log *slog.Logger
}

func NewDealRepository(db *pgxpool.Pool, log *slog.Logger) *DealRepository {
	return &DealRepository{db: db, log: log}
}

// CreateDeal — создаёт новую сделку.
func (r *DealRepository) CreateDeal(ctx context.Context, deal domain.Deal) (uuid.UUID, error) {
	const op = "DealRepository.CreateDeal"

	query := `
		INSERT INTO deals (
			lead_id, seller_user_id, buyer_user_id,
			price, status
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING deal_id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		deal.LeadID,
		deal.SellerUserID,
		deal.BuyerUserID,
		deal.Price,
		deal.Status.String(),
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

// GetByID — получает сделку по ID.
func (r *DealRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Deal, error) {
	const op = "DealRepository.GetByID"

	query := `
		SELECT
			deal_id, lead_id, seller_user_id, buyer_user_id,
			price, status, created_at, updated_at
		FROM deals
		WHERE deal_id = $1
	`

	var d domain.Deal
	var buyerUserID *uuid.UUID
	err := r.db.QueryRow(ctx, query, id).Scan(
		&d.ID,
		&d.LeadID,
		&d.SellerUserID,
		&buyerUserID,
		&d.Price,
		&d.Status,
		&d.CreatedAt,
		&d.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Deal{}, fmt.Errorf("%s: %w", op, repository.ErrDealNotFound)
		}
		return domain.Deal{}, fmt.Errorf("%s: %w", op, err)
	}

	d.BuyerUserID = buyerUserID
	return d, nil
}

// UpdateDeal — частичное обновление данных сделки.
func (r *DealRepository) UpdateDeal(ctx context.Context, dealID uuid.UUID, update domain.DealFilter) error {
	const op = "DealRepository.UpdateDeal"

	setClauses := []string{}
	params := []interface{}{}
	paramCount := 1

	if update.BuyerUserID != nil {
		setClauses = append(setClauses, fmt.Sprintf("buyer_user_id = $%d", paramCount))
		params = append(params, *update.BuyerUserID)
		paramCount++
	}
	if update.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", paramCount))
		params = append(params, (*update.Status).String())
		paramCount++
	}
	if update.Price != nil {
		setClauses = append(setClauses, fmt.Sprintf("price = $%d", paramCount))
		params = append(params, *update.Price)
		paramCount++
	}

	if len(setClauses) == 0 {
		return fmt.Errorf("%s: %w", op, repository.ErrNoFieldsToUpdate)
	}

	setClauses = append(setClauses, "updated_at = NOW()")

	query := fmt.Sprintf(`UPDATE deals SET %s WHERE deal_id = $%d`, strings.Join(setClauses, ", "), paramCount)
	params = append(params, dealID)

	tag, err := r.db.Exec(ctx, query, params...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, repository.ErrDealNotFound)
	}

	return nil
}

// ListDeals — возвращает сделки по фильтру.
func (r *DealRepository) ListDeals(ctx context.Context, filter domain.DealFilter) ([]domain.Deal, error) {
	const op = "DealRepository.ListDeals"

	query := `
		SELECT
			deal_id, lead_id, seller_user_id, buyer_user_id,
			price, status, created_at, updated_at
		FROM deals
	`
	whereClauses := []string{}
	params := []interface{}{}
	paramCount := 1

	if filter.LeadID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("lead_id = $%d", paramCount))
		params = append(params, *filter.LeadID)
		paramCount++
	}
	if filter.SellerUserID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("seller_user_id = $%d", paramCount))
		params = append(params, *filter.SellerUserID)
		paramCount++
	}
	if filter.BuyerUserID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("buyer_user_id = $%d", paramCount))
		params = append(params, *filter.BuyerUserID)
		paramCount++
	}
	if filter.Status != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", paramCount))
		params = append(params, (*filter.Status).String())
		paramCount++
	}
	if filter.MinPrice != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("price >= $%d", paramCount))
		params = append(params, *filter.MinPrice)
		paramCount++
	}
	if filter.MaxPrice != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("price <= $%d", paramCount))
		params = append(params, *filter.MaxPrice)
		paramCount++
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	rows, err := r.db.Query(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var deals []domain.Deal
	for rows.Next() {
		var d domain.Deal
		var buyerUserID *uuid.UUID
		if err := rows.Scan(
			&d.ID,
			&d.LeadID,
			&d.SellerUserID,
			&buyerUserID,
			&d.Price,
			&d.Status,
			&d.CreatedAt,
			&d.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("%s: scan failed: %w", op, err)
		}
		d.BuyerUserID = buyerUserID
		deals = append(deals, d)
	}

	return deals, rows.Err()
}
