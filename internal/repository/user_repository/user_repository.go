package user_repository

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

type UserRepository struct {
	db  *pgxpool.Pool
	log *slog.Logger
}

func NewUserRepository(db *pgxpool.Pool, log *slog.Logger) *UserRepository {
	return &UserRepository{db: db, log: log}
}

// CreateUser — создаёт нового пользователя.
func (r *UserRepository) CreateUser(ctx context.Context, email, firstName, lastName string, passwordHash []byte) (uuid.UUID, error) {
	const op = "UserRepository.CreateUser"

	query := `
		INSERT INTO users (email, password_hash, first_name, last_name, role)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING user_id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query, email, string(passwordHash), firstName, lastName, domain.UserRoleUser.String()).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return uuid.Nil, fmt.Errorf("%s: %w", op, repository.ErrUserExists)
		}
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

// GetByID — получает пользователя по ID.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	const op = "UserRepository.GetByID"

	query := `
		SELECT 
			user_id, email, password_hash, first_name, last_name,
			phone, agency_name, avatar_url, role, status, created_at
		FROM users
		WHERE user_id = $1
	`

	var u domain.User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.FirstName,
		&u.LastName,
		&u.Phone,
		&u.AgencyName,
		&u.AvatarURL,
		&u.Role,
		&u.Status,
		&u.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, fmt.Errorf("%s: %w", op, repository.ErrUserNotFound)
		}
		return domain.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return u, nil
}

// GetByEmail — получает пользователя по email.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	const op = "UserRepository.GetByEmail"

	query := `
		SELECT 
			user_id, email, password_hash, first_name, last_name,
			phone, agency_name, avatar_url, role, status, created_at
		FROM users
		WHERE email = $1
	`

	var u domain.User
	err := r.db.QueryRow(ctx, query, email).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.FirstName,
		&u.LastName,
		&u.Phone,
		&u.AgencyName,
		&u.AvatarURL,
		&u.Role,
		&u.Status,
		&u.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, fmt.Errorf("%s: %w", op, repository.ErrUserNotFound)
		}
		return domain.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return u, nil
}

// UpdateUser — обновляет профиль пользователя (частичное обновление).
func (r *UserRepository) UpdateUser(ctx context.Context, userID uuid.UUID, update domain.UserFilter) error {
	const op = "UserRepository.UpdateUser"

	setClauses := []string{}
	params := []interface{}{}
	paramCount := 1

	if update.FirstName != nil {
		setClauses = append(setClauses, fmt.Sprintf("first_name = $%d", paramCount))
		params = append(params, *update.FirstName)
		paramCount++
	}
	if update.LastName != nil {
		setClauses = append(setClauses, fmt.Sprintf("last_name = $%d", paramCount))
		params = append(params, *update.LastName)
		paramCount++
	}
	if update.Phone != nil {
		setClauses = append(setClauses, fmt.Sprintf("phone = $%d", paramCount))
		params = append(params, *update.Phone)
		paramCount++
	}
	if update.AgencyName != nil {
		setClauses = append(setClauses, fmt.Sprintf("agency_name = $%d", paramCount))
		params = append(params, *update.AgencyName)
		paramCount++
	}
	if update.Role != nil {
		setClauses = append(setClauses, fmt.Sprintf("role = $%d", paramCount))
		params = append(params, *update.Role)
		paramCount++
	}
	if update.Email != nil {
		setClauses = append(setClauses, fmt.Sprintf("email = $%d", paramCount))
		params = append(params, *update.Email)
		paramCount++
	}
	if update.AvatarURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("avatar_url = $%d", paramCount))
		params = append(params, *update.AvatarURL)
		paramCount++
	}
	if update.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", paramCount))
		params = append(params, *update.Status)
		paramCount++
	}

	if len(setClauses) == 0 {
		return fmt.Errorf("%s: %w", op, repository.ErrNoFieldsToUpdate)
	}

	query := fmt.Sprintf(`UPDATE users SET %s WHERE user_id = $%d`, strings.Join(setClauses, ", "), paramCount)
	params = append(params, userID)

	tag, err := r.db.Exec(ctx, query, params...)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("%s: %w", op, repository.ErrUserExists)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, repository.ErrUserNotFound)
	}

	return nil
}

// ListUsers — возвращает всех пользователей с опциональным фильтром по роли.
func (r *UserRepository) ListUsers(ctx context.Context, filter domain.UserFilter) ([]domain.User, error) {
	const op = "UserRepository.ListUsers"

	query := `
		SELECT 
			user_id, email, password_hash, first_name, last_name,
			phone, agency_name, avatar_url, role, status, created_at
		FROM users
	`
	params := []interface{}{}
	whereClauses := []string{}
	paramCount := 1

	if filter.Email != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("email = $%d", paramCount))
		params = append(params, *filter.Email)
		paramCount++
	}
	if filter.FirstName != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("first_name = $%d", paramCount))
		params = append(params, *filter.FirstName)
		paramCount++
	}
	if filter.LastName != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("last_name = $%d", paramCount))
		params = append(params, *filter.LastName)
		paramCount++
	}
	if filter.Phone != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("phone = $%d", paramCount))
		params = append(params, *filter.Phone)
		paramCount++
	}
	if filter.AgencyName != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("agency_name = $%d", paramCount))
		params = append(params, *filter.AgencyName)
		paramCount++
	}
	if filter.Role != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("role = $%d", paramCount))
		params = append(params, *filter.Role)
		paramCount++
	}
	if filter.Status != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", paramCount))
		params = append(params, *filter.Status)
		paramCount++
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	rows, err := r.db.Query(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("%s: query failed: %w", op, err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		err := rows.Scan(
			&u.ID,
			&u.Email,
			&u.PasswordHash,
			&u.FirstName,
			&u.LastName,
			&u.Phone,
			&u.AgencyName,
			&u.AvatarURL,
			&u.Role,
			&u.Status,
			&u.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: scan failed: %w", op, err)
		}
		users = append(users, u)
	}

	return users, rows.Err()
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
