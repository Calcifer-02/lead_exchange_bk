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
			title, description, address, city, property_type,
			area, price, rooms,
			status, owner_user_id, created_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING property_id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		property.Title,
		property.Description,
		property.Address,
		property.City,
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
			property_id, title, description, address, city, property_type,
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
		&p.City,
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
	if update.City != nil {
		setClauses = append(setClauses, fmt.Sprintf("city = $%d", paramCount))
		params = append(params, *update.City)
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

// ListProperties — возвращает объекты недвижимости по фильтру с пагинацией.
func (r *PropertyRepository) ListProperties(ctx context.Context, filter domain.PropertyFilter) (*domain.PaginatedResult[domain.Property], error) {
	const op = "PropertyRepository.ListProperties"

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
		case "created_at", "updated_at", "title", "price":
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
	if filter.PropertyType != nil {
		baseWhereClauses = append(baseWhereClauses, fmt.Sprintf("property_type = $%d", paramCount))
		baseParams = append(baseParams, (*filter.PropertyType).String())
		paramCount++
	}
	if filter.MinRooms != nil {
		baseWhereClauses = append(baseWhereClauses, fmt.Sprintf("rooms >= $%d", paramCount))
		baseParams = append(baseParams, *filter.MinRooms)
		paramCount++
	}
	if filter.MaxRooms != nil {
		baseWhereClauses = append(baseWhereClauses, fmt.Sprintf("rooms <= $%d", paramCount))
		baseParams = append(baseParams, *filter.MaxRooms)
		paramCount++
	}
	if filter.MinPrice != nil {
		baseWhereClauses = append(baseWhereClauses, fmt.Sprintf("price >= $%d", paramCount))
		baseParams = append(baseParams, *filter.MinPrice)
		paramCount++
	}
	if filter.MaxPrice != nil {
		baseWhereClauses = append(baseWhereClauses, fmt.Sprintf("price <= $%d", paramCount))
		baseParams = append(baseParams, *filter.MaxPrice)
		paramCount++
	}
	if filter.City != nil {
		baseWhereClauses = append(baseWhereClauses, fmt.Sprintf("LOWER(city) = LOWER($%d)", paramCount))
		baseParams = append(baseParams, *filter.City)
		paramCount++
	}

	// Получаем total count
	countQuery := "SELECT COUNT(*) FROM properties"
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
		if orderDir == domain.OrderDesc {
			whereClauses = append(whereClauses,
				fmt.Sprintf("(%s, property_id) < ($%d, $%d)", orderBy, paramCount, paramCount+1))
		} else {
			whereClauses = append(whereClauses,
				fmt.Sprintf("(%s, property_id) > ($%d, $%d)", orderBy, paramCount, paramCount+1))
		}
		params = append(params, cursor.LastCreatedAt, cursor.LastID)
		paramCount += 2
	}

	// Собираем основной запрос
	query := `
		SELECT
			property_id, title, description, address, city, property_type,
			area, price, rooms,
			status, owner_user_id, created_user_id,
			created_at, updated_at
		FROM properties
	`
	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// ORDER BY с direction
	dirStr := "DESC"
	if orderDir == domain.OrderAsc {
		dirStr = "ASC"
	}
	query += fmt.Sprintf(" ORDER BY %s %s, property_id %s", orderBy, dirStr, dirStr)

	// LIMIT +1 для определения has_more
	query += fmt.Sprintf(" LIMIT $%d", paramCount)
	params = append(params, pageSize+1)

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
			&p.City,
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

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows error: %w", op, err)
	}

	// Определяем hasMore и обрезаем до pageSize
	hasMore := len(properties) > pageSize
	if hasMore {
		properties = properties[:pageSize]
	}

	// Генерируем next cursor
	var nextPageToken string
	if hasMore && len(properties) > 0 {
		lastProp := properties[len(properties)-1]
		nextCursor := &domain.PageCursor{
			LastID:        lastProp.ID,
			LastCreatedAt: lastProp.CreatedAt,
		}
		nextPageToken = nextCursor.Encode()
	}

	return &domain.PaginatedResult[domain.Property]{
		Items:         properties,
		NextPageToken: nextPageToken,
		TotalCount:    totalCount,
		HasMore:       hasMore,
	}, nil
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
	return r.MatchPropertiesWithHardFilters(ctx, leadEmbedding, filter, nil, limit)
}

// MatchPropertiesWithHardFilters находит объекты с применением жёстких фильтров (город, тип недвижимости и др.).
// Жёсткие фильтры применяются ДО векторного поиска, исключая нерелевантные объекты.
func (r *PropertyRepository) MatchPropertiesWithHardFilters(
	ctx context.Context,
	leadEmbedding []float32,
	filter domain.PropertyFilter,
	hardFilters *domain.HardFilters,
	limit int,
) ([]domain.MatchedProperty, error) {
	const op = "PropertyRepository.MatchPropertiesWithHardFilters"

	embeddingStr := repository.VectorToString(leadEmbedding)

	query := `
		SELECT
			property_id, title, description, address, city, property_type,
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

	// ===== ЖЁСТКИЕ ФИЛЬТРЫ (критические поля) =====
	if hardFilters != nil {
		// Город — обязательное совпадение (case-insensitive)
		if hardFilters.City != nil && *hardFilters.City != "" {
			whereClauses = append(whereClauses, fmt.Sprintf("LOWER(city) = LOWER($%d)", paramCount))
			params = append(params, *hardFilters.City)
			paramCount++
		}
		// Тип недвижимости — обязательное совпадение
		if hardFilters.PropertyType != nil {
			whereClauses = append(whereClauses, fmt.Sprintf("property_type = $%d", paramCount))
			params = append(params, (*hardFilters.PropertyType).String())
			paramCount++
		}
		// Комнаты — диапазон (жёсткий)
		if hardFilters.MinRooms != nil {
			whereClauses = append(whereClauses, fmt.Sprintf("(rooms >= $%d OR rooms IS NULL)", paramCount))
			params = append(params, *hardFilters.MinRooms)
			paramCount++
		}
		if hardFilters.MaxRooms != nil {
			whereClauses = append(whereClauses, fmt.Sprintf("(rooms <= $%d OR rooms IS NULL)", paramCount))
			params = append(params, *hardFilters.MaxRooms)
			paramCount++
		}
		// Цена — диапазон (жёсткий)
		if hardFilters.MinPrice != nil {
			whereClauses = append(whereClauses, fmt.Sprintf("(price >= $%d OR price IS NULL)", paramCount))
			params = append(params, *hardFilters.MinPrice)
			paramCount++
		}
		if hardFilters.MaxPrice != nil {
			whereClauses = append(whereClauses, fmt.Sprintf("(price <= $%d OR price IS NULL)", paramCount))
			params = append(params, *hardFilters.MaxPrice)
			paramCount++
		}
	}

	// ===== МЯГКИЕ ФИЛЬТРЫ (из PropertyFilter) =====
	if filter.Status != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", paramCount))
		params = append(params, (*filter.Status).String())
		paramCount++
	}
	if filter.City != nil && (hardFilters == nil || hardFilters.City == nil) {
		whereClauses = append(whereClauses, fmt.Sprintf("LOWER(city) = LOWER($%d)", paramCount))
		params = append(params, *filter.City)
		paramCount++
	}
	if filter.MinPrice != nil && (hardFilters == nil || hardFilters.MinPrice == nil) {
		whereClauses = append(whereClauses, fmt.Sprintf("price >= $%d", paramCount))
		params = append(params, *filter.MinPrice)
		paramCount++
	}
	if filter.MaxPrice != nil && (hardFilters == nil || hardFilters.MaxPrice == nil) {
		whereClauses = append(whereClauses, fmt.Sprintf("price <= $%d", paramCount))
		params = append(params, *filter.MaxPrice)
		paramCount++
	}
	if filter.PropertyType != nil && (hardFilters == nil || hardFilters.PropertyType == nil) {
		whereClauses = append(whereClauses, fmt.Sprintf("property_type = $%d", paramCount))
		params = append(params, (*filter.PropertyType).String())
		paramCount++
	}
	if filter.MinRooms != nil && (hardFilters == nil || hardFilters.MinRooms == nil) {
		whereClauses = append(whereClauses, fmt.Sprintf("rooms >= $%d", paramCount))
		params = append(params, *filter.MinRooms)
		paramCount++
	}
	if filter.MaxRooms != nil && (hardFilters == nil || hardFilters.MaxRooms == nil) {
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
			&p.City,
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

// HybridSearchParams — параметры гибридного поиска.
type HybridSearchParams struct {
	// LeadEmbedding — эмбеддинг лида для векторного поиска
	LeadEmbedding []float32
	// SearchQuery — текстовый запрос для полнотекстового поиска
	SearchQuery string
	// VectorWeight — вес векторного поиска (0-1)
	VectorWeight float64
	// FulltextWeight — вес полнотекстового поиска (0-1)
	FulltextWeight float64
	// Filter — фильтры
	Filter domain.PropertyFilter
	// HardFilters — жёсткие фильтры
	HardFilters *domain.HardFilters
	// Limit — количество результатов
	Limit int
}

// HybridSearch выполняет гибридный поиск: комбинирует векторный и полнотекстовый поиск.
// Использует Reciprocal Rank Fusion (RRF) для объединения результатов.
func (r *PropertyRepository) HybridSearch(ctx context.Context, params HybridSearchParams) ([]domain.MatchedProperty, error) {
	const op = "PropertyRepository.HybridSearch"

	// Если нет текстового запроса или полнотекстовый поиск отключен, используем только векторный
	if params.SearchQuery == "" || params.FulltextWeight <= 0 {
		return r.MatchPropertiesWithHardFilters(ctx, params.LeadEmbedding, params.Filter, params.HardFilters, params.Limit)
	}

	embeddingStr := repository.VectorToString(params.LeadEmbedding)

	// CTE для векторного поиска
	// CTE для полнотекстового поиска
	// Объединение с RRF (Reciprocal Rank Fusion)
	query := `
		WITH vector_search AS (
			SELECT
				property_id,
				ROW_NUMBER() OVER (ORDER BY embedding <=> $1::vector) as vector_rank,
				1 - (embedding <=> $1::vector) as vector_similarity
			FROM properties
			WHERE embedding IS NOT NULL
			%s
			LIMIT $2
		),
		fulltext_search AS (
			SELECT
				property_id,
				ROW_NUMBER() OVER (ORDER BY ts_rank(search_vector, plainto_tsquery('russian', $3)) DESC) as fts_rank,
				ts_rank(search_vector, plainto_tsquery('russian', $3)) as fts_score
			FROM properties
			WHERE search_vector @@ plainto_tsquery('russian', $3)
			%s
			LIMIT $2
		),
		combined AS (
			SELECT
				COALESCE(v.property_id, f.property_id) as property_id,
				-- RRF score: 1/(k + rank)
				COALESCE($4::float / (60.0 + v.vector_rank), 0) +
				COALESCE($5::float / (60.0 + f.fts_rank), 0) as rrf_score,
				COALESCE(v.vector_similarity, 0) as vector_similarity,
				COALESCE(f.fts_score, 0) as fts_score
			FROM vector_search v
			FULL OUTER JOIN fulltext_search f ON v.property_id = f.property_id
		)
		SELECT
			p.property_id, p.title, p.description, p.address, p.city, p.property_type,
			p.area, p.price, p.rooms,
			p.status, p.owner_user_id, p.created_user_id,
			p.embedding::text, p.created_at, p.updated_at,
			c.rrf_score,
			c.vector_similarity,
			c.fts_score
		FROM combined c
		JOIN properties p ON p.property_id = c.property_id
		ORDER BY c.rrf_score DESC
		LIMIT $6
	`

	// Собираем WHERE условия для CTE
	whereClauses := []string{}
	params_list := []interface{}{embeddingStr, params.Limit * 2, params.SearchQuery, params.VectorWeight, params.FulltextWeight}
	paramCount := 6

	// Жёсткие фильтры
	if params.HardFilters != nil {
		if params.HardFilters.City != nil && *params.HardFilters.City != "" {
			whereClauses = append(whereClauses, fmt.Sprintf("AND LOWER(city) = LOWER($%d)", paramCount))
			params_list = append(params_list, *params.HardFilters.City)
			paramCount++
		}
		if params.HardFilters.PropertyType != nil {
			whereClauses = append(whereClauses, fmt.Sprintf("AND property_type = $%d", paramCount))
			params_list = append(params_list, (*params.HardFilters.PropertyType).String())
			paramCount++
		}
		if params.HardFilters.MinRooms != nil {
			whereClauses = append(whereClauses, fmt.Sprintf("AND (rooms >= $%d OR rooms IS NULL)", paramCount))
			params_list = append(params_list, *params.HardFilters.MinRooms)
			paramCount++
		}
		if params.HardFilters.MaxRooms != nil {
			whereClauses = append(whereClauses, fmt.Sprintf("AND (rooms <= $%d OR rooms IS NULL)", paramCount))
			params_list = append(params_list, *params.HardFilters.MaxRooms)
			paramCount++
		}
		if params.HardFilters.MinPrice != nil {
			whereClauses = append(whereClauses, fmt.Sprintf("AND (price >= $%d OR price IS NULL)", paramCount))
			params_list = append(params_list, *params.HardFilters.MinPrice)
			paramCount++
		}
		if params.HardFilters.MaxPrice != nil {
			whereClauses = append(whereClauses, fmt.Sprintf("AND (price <= $%d OR price IS NULL)", paramCount))
			params_list = append(params_list, *params.HardFilters.MaxPrice)
			paramCount++
		}
	}

	// Мягкие фильтры
	if params.Filter.Status != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("AND status = $%d", paramCount))
		params_list = append(params_list, (*params.Filter.Status).String())
		paramCount++
	}

	whereStr := strings.Join(whereClauses, " ")
	params_list = append(params_list, params.Limit)

	// Форматируем запрос с WHERE условиями
	query = fmt.Sprintf(query, whereStr, whereStr)

	rows, err := r.db.Query(ctx, query, params_list...)
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
		var rrfScore float64
		var vectorSimilarity float64
		var ftsScore float64

		if err := rows.Scan(
			&p.ID,
			&p.Title,
			&p.Description,
			&p.Address,
			&p.City,
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
			&rrfScore,
			&vectorSimilarity,
			&ftsScore,
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
			Similarity: vectorSimilarity, // Используем vector similarity как основной score
		})
	}

	return matches, rows.Err()
}

// FulltextSearch выполняет только полнотекстовый поиск.
func (r *PropertyRepository) FulltextSearch(ctx context.Context, query string, filter domain.PropertyFilter, limit int) ([]domain.MatchedProperty, error) {
	const op = "PropertyRepository.FulltextSearch"

	sqlQuery := `
		SELECT
			property_id, title, description, address, city, property_type,
			area, price, rooms,
			status, owner_user_id, created_user_id,
			created_at, updated_at,
			ts_rank(search_vector, plainto_tsquery('russian', $1)) as rank
		FROM properties
		WHERE search_vector @@ plainto_tsquery('russian', $1)
	`

	whereClauses := []string{}
	params := []interface{}{query}
	paramCount := 2

	if filter.Status != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("AND status = $%d", paramCount))
		params = append(params, (*filter.Status).String())
		paramCount++
	}
	if filter.City != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("AND LOWER(city) = LOWER($%d)", paramCount))
		params = append(params, *filter.City)
		paramCount++
	}

	if len(whereClauses) > 0 {
		sqlQuery += " " + strings.Join(whereClauses, " ")
	}

	sqlQuery += fmt.Sprintf(" ORDER BY rank DESC LIMIT $%d", paramCount)
	params = append(params, limit)

	rows, err := r.db.Query(ctx, sqlQuery, params...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var matches []domain.MatchedProperty
	for rows.Next() {
		var p domain.Property
		var propertyTypeStr string
		var statusStr string
		var rank float64

		if err := rows.Scan(
			&p.ID,
			&p.Title,
			&p.Description,
			&p.Address,
			&p.City,
			&propertyTypeStr,
			&p.Area,
			&p.Price,
			&p.Rooms,
			&statusStr,
			&p.OwnerUserID,
			&p.CreatedUserID,
			&p.CreatedAt,
			&p.UpdatedAt,
			&rank,
		); err != nil {
			return nil, fmt.Errorf("%s: scan failed: %w", op, err)
		}

		p.PropertyType = domain.PropertyType(propertyTypeStr)
		p.Status = domain.PropertyStatus(statusStr)

		matches = append(matches, domain.MatchedProperty{
			Property:   p,
			Similarity: rank,
		})
	}

	return matches, rows.Err()
}
