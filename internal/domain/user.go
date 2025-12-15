package domain

import (
	"time"

	"github.com/google/uuid"
)

// User — доменная сущность пользователя.
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash []byte
	FirstName    string
	LastName     string
	Phone        *string
	AgencyName   *string
	AvatarURL    *string
	Role         UserRole
	Status       UserStatus
	CreatedAt    time.Time
}

// UserRole — тип роли пользователя.
type UserRole string

const (
	UserRoleUnspecified UserRole = ""
	UserRoleUser        UserRole = "USER"
	UserRoleAdmin       UserRole = "ADMIN"
)

func (r UserRole) String() string {
	return string(r)
}

// UserStatus — тип статуса пользователя.
type UserStatus string

const (
	UserStatusUnspecified UserStatus = ""
	UserStatusActive      UserStatus = "ACTIVE"
	UserStatusBanned      UserStatus = "BANNED"
	UserStatusSuspended   UserStatus = "SUSPENDED"
)

func (s UserStatus) String() string {
	return string(s)
}

// UserFilter — фильтр для выборок пользователей (например, в админке).
type UserFilter struct {
	Email      *string
	FirstName  *string
	LastName   *string
	Phone      *string
	AgencyName *string
	AvatarURL  *string
	Role       *UserRole
	Status     *UserStatus
}
