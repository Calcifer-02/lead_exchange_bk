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
			city, status, owner_user_id, created_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
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
		lead.City,
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
			city, status, owner_user_id, created_user_id,
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
		&l.City,
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
	if update.City != nil {
		setClauses = append(setClauses, fmt.Sprintf("city = $%d", paramCount))
		params = append(params, *update.City)
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

// ListLeads — возвращает лидов по фильтру с пагинацией.
func (r *LeadRepository) ListLeads(ctx context.Context, filter domain.LeadFilter) (*domain.PaginatedResult[domain.Lead], error) {
	const op = "LeadRepository.ListLeads"

	// Нормализуем параметры пагинации
	pageSize := int(domain.DefaultPageSize)
	var cursor *domain.PageCursor
	orderBy := "created_at"
	orderDir := domain.OrderDesc

	if filter.Pagination != nil {
		pageSize = int(domain.NormalizePageSize(filter.Pagination.PageSize))
		orderDir = domain.NormalizeOrderDirection(string(filter.Pagination.OrderDirection))

		// Валидация и установка поля сортировки
		switch filter.Pagination.OrderBy {
		case "created_at", "updated_at", "title":
			orderBy = filter.Pagination.OrderBy
		}

		// Декодируем курсор
		if filter.Pagination.PageToken != "" {
			var err error
			cursor, err = domain.DecodePageCursor(filter.Pagination.PageToken)
			if err != nil {
				r.log.Warn("failed to decode page cursor, starting from beginning", "error", err)
				cursor = nil
			}
		}
	}

	// Базовые WHERE условия (без cursor)
	baseWhereClauses := []string{}
	baseParams := []interface{}{}
	paramCount := 1

	if filter.Status != nil {
		baseWhereClauses = append(baseWhereClauses, fmt.Sprintf("status = $%d", paramCount))
		baseParams = append(baseParams, (*filter.Status).String())
		paramCount++
	}
	if filter.OwnerUserID != nil {
		baseWhereClauses = append(baseWhereClauses, fmt.Sprintf("owner_user_id = $%d", paramCount))
		baseParams = append(baseParams, *filter.OwnerUserID)
		paramCount++
	}
	if filter.CreatedUserID != nil {
		baseWhereClauses = append(baseWhereClauses, fmt.Sprintf("created_user_id = $%d", paramCount))
		baseParams = append(baseParams, *filter.CreatedUserID)
		paramCount++
	}
	if filter.City != nil {
		baseWhereClauses = append(baseWhereClauses, fmt.Sprintf("LOWER(city) = LOWER($%d)", paramCount))
		baseParams = append(baseParams, *filter.City)
		paramCount++
	}

	// Получаем total count
	countQuery := "SELECT COUNT(*) FROM leads"
	if len(baseWhereClauses) > 0 {
		countQuery += " WHERE " + strings.Join(baseWhereClauses, " AND ")
	}

	var totalCount int32
	err := r.db.QueryRow(ctx, countQuery, baseParams...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("%s: count failed: %w", op, err)
	}

	// Копируем для основного запроса
	whereClauses := append([]string{}, baseWhereClauses...)
	params := append([]interface{}{}, baseParams...)

	// Применяем cursor-based пагинацию
	if cursor != nil {
		// Keyset pagination: (created_at, lead_id) < или > в зависимости от направления
		if orderDir == domain.OrderDesc {
			whereClauses = append(whereClauses,
				fmt.Sprintf("(%s, lead_id) < ($%d, $%d)", orderBy, paramCount, paramCount+1))
		} else {
			whereClauses = append(whereClauses,
				fmt.Sprintf("(%s, lead_id) > ($%d, $%d)", orderBy, paramCount, paramCount+1))
		}
		params = append(params, cursor.LastCreatedAt, cursor.LastID)
		paramCount += 2
	}

	// Собираем основной запрос
	query := `
		SELECT
			lead_id, title, description, requirement,
			contact_name, contact_phone, contact_email,
			city, status, owner_user_id, created_user_id,
			created_at, updated_at
		FROM leads
	`
	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// ORDER BY с direction
	dirStr := "DESC"
	if orderDir == domain.OrderAsc {
		dirStr = "ASC"
	}
	query += fmt.Sprintf(" ORDER BY %s %s, lead_id %s", orderBy, dirStr, dirStr)

	// LIMIT +1 для определения has_more
	query += fmt.Sprintf(" LIMIT $%d", paramCount)
	params = append(params, pageSize+1)

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
			&l.City,
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

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows error: %w", op, err)
	}

	// Определяем hasMore и обрезаем до pageSize
	hasMore := len(leads) > pageSize
	if hasMore {
		leads = leads[:pageSize]
	}

	// Генерируем next cursor
	var nextPageToken string
	if hasMore && len(leads) > 0 {
		lastLead := leads[len(leads)-1]
		nextCursor := &domain.PageCursor{
			LastID:        lastLead.ID,
			LastCreatedAt: lastLead.CreatedAt,
		}
		nextPageToken = nextCursor.Encode()
	}

	return &domain.PaginatedResult[domain.Lead]{
		Items:         leads,
		NextPageToken: nextPageToken,
		TotalCount:    totalCount,
		HasMore:       hasMore,
	}, nil
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
