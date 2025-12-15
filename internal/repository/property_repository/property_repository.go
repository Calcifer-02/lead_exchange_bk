package property_repository

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

type PropertyRepository struct {
	db  *pgxpool.Pool
	log *slog.Logger
}

func NewPropertyRepository(db *pgxpool.Pool, log *slog.Logger) *PropertyRepository {
	return &PropertyRepository{db: db, log: log}
}

// CreateProperty — создаёт новый объект недвижимости.
func (r *PropertyRepository) CreateProperty(ctx context.Context, property domain.Property) (uuid.UUID, error) {
	const op = "PropertyRepository.CreateProperty"

	query := `
		INSERT INTO properties (
			title, description, address, property_type,
			area, price, rooms,
			status, owner_user_id, created_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING property_id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		property.Title,
		property.Description,
		property.Address,
		property.PropertyType.String(),
		property.Area,
		property.Price,
		property.Rooms,
		property.Status.String(),
		property.OwnerUserID,
		property.CreatedUserID,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

// GetByID — получает объект недвижимости по ID.
func (r *PropertyRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Property, error) {
	const op = "PropertyRepository.GetByID"

	query := `
		SELECT
			property_id, title, description, address, property_type,
			area, price, rooms,
			status, owner_user_id, created_user_id,
			embedding::text, created_at, updated_at
		FROM properties
		WHERE property_id = $1
	`

	var p domain.Property
	var propertyTypeStr string
	var statusStr string
	var embeddingStr *string
	err := r.db.QueryRow(ctx, query, id).Scan(
		&p.ID,
		&p.Title,
		&p.Description,
		&p.Address,
		&propertyTypeStr,
		&p.Area,
		&p.Price,
		&p.Rooms,
		&statusStr,
		&p.OwnerUserID,
		&p.CreatedUserID,
		&embeddingStr,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Property{}, fmt.Errorf("%s: %w", op, repository.ErrPropertyNotFound)
		}
		return domain.Property{}, fmt.Errorf("%s: %w", op, err)
	}

	p.PropertyType = domain.PropertyType(propertyTypeStr)
	p.Status = domain.PropertyStatus(statusStr)

	// Конвертируем embedding из строки
	if embeddingStr != nil && *embeddingStr != "" {
		vec, err := repository.StringToVector(*embeddingStr)
		if err != nil {
			r.log.Warn("failed to parse embedding", "error", err)
		} else {
			p.Embedding = vec
		}
	}

	return p, nil
}

// UpdateProperty — частичное обновление данных объекта недвижимости.
func (r *PropertyRepository) UpdateProperty(ctx context.Context, propertyID uuid.UUID, update domain.PropertyFilter) error {
	const op = "PropertyRepository.UpdateProperty"

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
	if update.Address != nil {
		setClauses = append(setClauses, fmt.Sprintf("address = $%d", paramCount))
		params = append(params, *update.Address)
		paramCount++
	}
	if update.PropertyType != nil {
		setClauses = append(setClauses, fmt.Sprintf("property_type = $%d", paramCount))
		params = append(params, (*update.PropertyType).String())
		paramCount++
	}
	if update.Area != nil {
		setClauses = append(setClauses, fmt.Sprintf("area = $%d", paramCount))
		params = append(params, *update.Area)
		paramCount++
	}
	if update.Price != nil {
		setClauses = append(setClauses, fmt.Sprintf("price = $%d", paramCount))
		params = append(params, *update.Price)
		paramCount++
	}
	if update.Rooms != nil {
		setClauses = append(setClauses, fmt.Sprintf("rooms = $%d", paramCount))
		params = append(params, *update.Rooms)
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

	query := fmt.Sprintf(`UPDATE properties SET %s WHERE property_id = $%d`, strings.Join(setClauses, ", "), paramCount)
	params = append(params, propertyID)

	tag, err := r.db.Exec(ctx, query, params...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, repository.ErrPropertyNotFound)
	}

	return nil
}

// ListProperties — возвращает объекты недвижимости по фильтру.
func (r *PropertyRepository) ListProperties(ctx context.Context, filter domain.PropertyFilter) ([]domain.Property, error) {
	const op = "PropertyRepository.ListProperties"

	query := `
		SELECT
			property_id, title, description, address, property_type,
			area, price, rooms,
			status, owner_user_id, created_user_id,
			created_at, updated_at
		FROM properties
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
	if filter.PropertyType != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("property_type = $%d", paramCount))
		params = append(params, (*filter.PropertyType).String())
		paramCount++
	}
	if filter.MinRooms != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("rooms >= $%d", paramCount))
		params = append(params, *filter.MinRooms)
		paramCount++
	}
	if filter.MaxRooms != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("rooms <= $%d", paramCount))
		params = append(params, *filter.MaxRooms)
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

	var properties []domain.Property
	for rows.Next() {
		var p domain.Property
		var propertyTypeStr string
		var statusStr string
		if err := rows.Scan(
			&p.ID,
			&p.Title,
			&p.Description,
			&p.Address,
			&propertyTypeStr,
			&p.Area,
			&p.Price,
			&p.Rooms,
			&statusStr,
			&p.OwnerUserID,
			&p.CreatedUserID,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("%s: scan failed: %w", op, err)
		}
		p.PropertyType = domain.PropertyType(propertyTypeStr)
		p.Status = domain.PropertyStatus(statusStr)
		properties = append(properties, p)
	}

	return properties, rows.Err()
}

// UpdateEmbedding обновляет embedding для объекта недвижимости.
func (r *PropertyRepository) UpdateEmbedding(ctx context.Context, propertyID uuid.UUID, embedding []float32) error {
	const op = "PropertyRepository.UpdateEmbedding"

	query := `
		UPDATE properties 
		SET embedding = $1::vector, updated_at = NOW()
		WHERE property_id = $2
	`

	embeddingStr := repository.VectorToString(embedding)
	tag, err := r.db.Exec(ctx, query, embeddingStr, propertyID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, repository.ErrPropertyNotFound)
	}

	return nil
}

// MatchProperties находит подходящие объекты недвижимости для лида по косинусному расстоянию.
func (r *PropertyRepository) MatchProperties(ctx context.Context, leadEmbedding []float32, filter domain.PropertyFilter, limit int) ([]domain.MatchedProperty, error) {
	const op = "PropertyRepository.MatchProperties"

	embeddingStr := repository.VectorToString(leadEmbedding)

	query := `
		SELECT
			property_id, title, description, address, property_type,
			area, price, rooms,
			status, owner_user_id, created_user_id,
			embedding::text, created_at, updated_at,
			1 - (embedding <=> $1::vector) as similarity
		FROM properties
		WHERE embedding IS NOT NULL
	`

	whereClauses := []string{}
	params := []interface{}{embeddingStr}
	paramCount := 2

	// Добавляем фильтры
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
	if filter.PropertyType != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("property_type = $%d", paramCount))
		params = append(params, (*filter.PropertyType).String())
		paramCount++
	}
	if filter.MinRooms != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("rooms >= $%d", paramCount))
		params = append(params, *filter.MinRooms)
		paramCount++
	}
	if filter.MaxRooms != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("rooms <= $%d", paramCount))
		params = append(params, *filter.MaxRooms)
		paramCount++
	}

	if len(whereClauses) > 0 {
		query += " AND " + strings.Join(whereClauses, " AND ")
	}

	query += " ORDER BY embedding <=> $1::vector LIMIT $%d"
	query = fmt.Sprintf(query, paramCount)
	params = append(params, limit)

	rows, err := r.db.Query(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var matches []domain.MatchedProperty
	for rows.Next() {
		var p domain.Property
		var propertyTypeStr string
		var statusStr string
		var embeddingStr *string
		var similarity float64

		if err := rows.Scan(
			&p.ID,
			&p.Title,
			&p.Description,
			&p.Address,
			&propertyTypeStr,
			&p.Area,
			&p.Price,
			&p.Rooms,
			&statusStr,
			&p.OwnerUserID,
			&p.CreatedUserID,
			&embeddingStr,
			&p.CreatedAt,
			&p.UpdatedAt,
			&similarity,
		); err != nil {
			return nil, fmt.Errorf("%s: scan failed: %w", op, err)
		}

		p.PropertyType = domain.PropertyType(propertyTypeStr)
		p.Status = domain.PropertyStatus(statusStr)

		if embeddingStr != nil && *embeddingStr != "" {
			vec, err := repository.StringToVector(*embeddingStr)
			if err != nil {
				r.log.Warn("failed to parse embedding", "error", err)
			} else {
				p.Embedding = vec
			}
		}

		matches = append(matches, domain.MatchedProperty{
			Property:   p,
			Similarity: similarity,
		})
	}

	return matches, rows.Err()
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

