package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"lead_exchange/internal/domain"
	"lead_exchange/internal/lib/jwt"
	"lead_exchange/internal/lib/logger/sl"
	"lead_exchange/internal/repository"
)

type UserRepository interface {
	CreateUser(ctx context.Context, email, firstName, lastName string, passwordHash []byte) (uuid.UUID, error)
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.User, error)
	UpdateUser(ctx context.Context, userID uuid.UUID, update domain.UserFilter) error
	ListUsers(ctx context.Context, filter domain.UserFilter) ([]domain.User, error)
}

type Service struct {
	log      *slog.Logger
	repo     UserRepository
	tokenTTL time.Duration
	secret   string
}

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserExists         = errors.New("user already exists")
)

func New(log *slog.Logger, repo UserRepository, tokenTTL time.Duration, secret string) *Service {
	return &Service{
		log:      log,
		repo:     repo,
		tokenTTL: tokenTTL,
		secret:   secret,
	}
}

// Register — регистрация нового пользователя.
func (s *Service) Register(ctx context.Context, email, password, firstName, lastName string) (uuid.UUID, error) {
	const op = "user.Service.Register"
	log := s.log.With(slog.String("op", op), slog.String("email", email))

	log.Info("registering user")

	passHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to generate password hash", sl.Err(err))
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	id, err := s.repo.CreateUser(ctx, email, firstName, lastName, passHash)
	if err != nil {
		if errors.Is(err, repository.ErrUserExists) {
			return uuid.Nil, fmt.Errorf("%s: %w", op, ErrUserExists)
		}
		log.Error("failed to save user", sl.Err(err))
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("user registered successfully", slog.String("user_id", id.String()))
	return id, nil
}

// Login — аутентификация пользователя и выдача JWT-токена.
func (s *Service) Login(ctx context.Context, email, password string) (uuid.UUID, string, error) {
	const op = "user.Service.Login"
	log := s.log.With(slog.String("op", op), slog.String("email", email))

	log.Info("attempting login")

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			log.Warn("user not found", sl.Err(err))
			return uuid.Nil, "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
		}
		log.Error("failed to fetch user", sl.Err(err))
		return uuid.Nil, "", fmt.Errorf("%s: %w", op, err)
	}

	if err := bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password)); err != nil {
		log.Info("invalid password", sl.Err(err))
		return uuid.Nil, "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}

	token, err := jwt.NewToken(user, s.secret, s.tokenTTL)
	if err != nil {
		log.Error("failed to generate token", sl.Err(err))
		return uuid.Nil, "", fmt.Errorf("%s: %w", op, err)
	}

	log.Info("login successful")
	return user.ID, token, nil
}

// GetProfile — возвращает профиль пользователя по ID.
func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (domain.User, error) {
	const op = "user.Service.GetProfile"

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			s.log.Warn("user not found", sl.Err(err))
			return domain.User{}, fmt.Errorf("%s: %w", op, repository.ErrUserNotFound)
		}
		s.log.Error("failed to get user", sl.Err(err))
		return domain.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return user, nil
}

// UpdateProfile — частичное обновление данных профиля.
func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, update domain.UserFilter) (domain.User, error) {
	const op = "user.Service.UpdateProfile"

	err := s.repo.UpdateUser(ctx, userID, update)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return domain.User{}, fmt.Errorf("%s: %w", op, repository.ErrUserNotFound)
		}
		return domain.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return s.repo.GetByID(ctx, userID)
}

// ListUsers — возвращает пользователей по фильтру (например, для админа).
func (s *Service) ListUsers(ctx context.Context, filter domain.UserFilter) ([]domain.User, error) {
	return s.repo.ListUsers(ctx, filter)
}

// UpdateUserStatus — обновляет статус пользователя.
func (s *Service) UpdateUserStatus(ctx context.Context, userID uuid.UUID, status domain.UserStatus) (domain.User, error) {
	const op = "user.Service.UpdateUserStatus"

	filter := domain.UserFilter{
		Status: &status,
	}

	err := s.repo.UpdateUser(ctx, userID, filter)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return domain.User{}, fmt.Errorf("%s: %w", op, repository.ErrUserNotFound)
		}
		return domain.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return s.repo.GetByID(ctx, userID)
}
