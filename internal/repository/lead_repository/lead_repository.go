package lead_repository

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
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LeadRepository struct {
	db  *pgxpool.Pool
	log *slog.Logger
}

func NewLeadRepository(db *pgxpool.Pool, log *slog.Logger) *LeadRepository {
	return &LeadRepository{db: db, log: log}
}

// CreateLead — создаёт нового лида.
func (r *LeadRepository) CreateLead(ctx context.Context, lead domain.Lead) (uuid.UUID, error) {
	const op = "LeadRepository.CreateLead"

	query := `
		INSERT INTO leads (
			title, description, requirement,
			contact_name, contact_phone, contact_email,
			status, owner_user_id, created_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING lead_id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		lead.Title,
		lead.Description,
		lead.Requirement,
		lead.ContactName,
		lead.ContactPhone,
		lead.ContactEmail,
		lead.Status.String(),
		lead.OwnerUserID,
		lead.CreatedUserID,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

// GetByID — получает лида по ID.
func (r *LeadRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Lead, error) {
	const op = "LeadRepository.GetByID"

	query := `
		SELECT
			lead_id, title, description, requirement,
			contact_name, contact_phone, contact_email,
			status, owner_user_id, created_user_id,
			embedding::text, created_at, updated_at
		FROM leads
		WHERE lead_id = $1
	`

	var l domain.Lead
	var embeddingStr *string
	err := r.db.QueryRow(ctx, query, id).Scan(
		&l.ID,
		&l.Title,
		&l.Description,
		&l.Requirement,
		&l.ContactName,
		&l.ContactPhone,
		&l.ContactEmail,
		&l.Status,
		&l.OwnerUserID,
		&l.CreatedUserID,
		&embeddingStr,
		&l.CreatedAt,
		&l.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Lead{}, fmt.Errorf("%s: %w", op, repository.ErrLeadNotFound)
		}
		return domain.Lead{}, fmt.Errorf("%s: %w", op, err)
	}

	// Конвертируем embedding из строки
	if embeddingStr != nil && *embeddingStr != "" {
		vec, err := repository.StringToVector(*embeddingStr)
		if err != nil {
			r.log.Warn("failed to parse embedding", "error", err)
		} else {
			l.Embedding = vec
		}
	}

	return l, nil
}

// UpdateLead — частичное обновление данных лида.
func (r *LeadRepository) UpdateLead(ctx context.Context, leadID uuid.UUID, update domain.LeadFilter) error {
	const op = "LeadRepository.UpdateLead"

	setClauses := []string{}
	params := []interface{}{}
	paramCount := 1

	if update.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", paramCount))
		params = append(params, *update.Title)
		paramCount++
	}
	if update.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", paramCount))
		params = append(params, *update.Description)
		paramCount++
	}
	if update.Requirement != nil {
		setClauses = append(setClauses, fmt.Sprintf("requirement = $%d", paramCount))
		params = append(params, *update.Requirement)
		paramCount++
	}
	if update.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", paramCount))
		params = append(params, (*update.Status).String())
		paramCount++
	}
	if update.OwnerUserID != nil {
		setClauses = append(setClauses, fmt.Sprintf("owner_user_id = $%d", paramCount))
		params = append(params, *update.OwnerUserID)
		paramCount++
	}

	if len(setClauses) == 0 {
		return fmt.Errorf("%s: %w", op, repository.ErrNoFieldsToUpdate)
	}

	setClauses = append(setClauses, "updated_at = NOW()")

	query := fmt.Sprintf(`UPDATE leads SET %s WHERE lead_id = $%d`, strings.Join(setClauses, ", "), paramCount)
	params = append(params, leadID)

	tag, err := r.db.Exec(ctx, query, params...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, repository.ErrLeadNotFound)
	}

	return nil
}

// ListLeads — возвращает лидов по фильтру.
func (r *LeadRepository) ListLeads(ctx context.Context, filter domain.LeadFilter) ([]domain.Lead, error) {
	const op = "LeadRepository.ListLeads"

	query := `
		SELECT
			lead_id, title, description, requirement,
			contact_name, contact_phone, contact_email,
			status, owner_user_id, created_user_id,
			created_at, updated_at
		FROM leads
	`
	whereClauses := []string{}
	params := []interface{}{}
	paramCount := 1

	if filter.Status != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", paramCount))
		params = append(params, (*filter.Status).String())
		paramCount++
	}
	if filter.OwnerUserID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("owner_user_id = $%d", paramCount))
		params = append(params, *filter.OwnerUserID)
		paramCount++
	}
	if filter.CreatedUserID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("created_user_id = $%d", paramCount))
		params = append(params, *filter.CreatedUserID)
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

	var leads []domain.Lead
	for rows.Next() {
		var l domain.Lead
		if err := rows.Scan(
			&l.ID,
			&l.Title,
			&l.Description,
			&l.Requirement,
			&l.ContactName,
			&l.ContactPhone,
			&l.ContactEmail,
			&l.Status,
			&l.OwnerUserID,
			&l.CreatedUserID,
			&l.CreatedAt,
			&l.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("%s: scan failed: %w", op, err)
		}
		leads = append(leads, l)
	}

	return leads, rows.Err()
}

// UpdateEmbedding обновляет embedding для лида.
func (r *LeadRepository) UpdateEmbedding(ctx context.Context, leadID uuid.UUID, embedding []float32) error {
	const op = "LeadRepository.UpdateEmbedding"

	query := `
		UPDATE leads 
		SET embedding = $1::vector, updated_at = NOW()
		WHERE lead_id = $2
	`

	embeddingStr := repository.VectorToString(embedding)
	tag, err := r.db.Exec(ctx, query, embeddingStr, leadID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, repository.ErrLeadNotFound)
	}

	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
