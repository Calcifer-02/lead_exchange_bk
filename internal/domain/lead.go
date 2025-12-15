package domain

import (
	"time"

	"github.com/google/uuid"
)

// Lead — доменная сущность лида.
type Lead struct {
	ID          uuid.UUID
	Title       string
	Description string
	// JSON с предпочтениями ("roomNumber": 3, "price": "5000000")
	Requirement   []byte
	ContactName   string
	ContactPhone  string
	ContactEmail  *string
	Status        LeadStatus
	OwnerUserID   uuid.UUID
	CreatedUserID uuid.UUID
	// Embedding — векторное представление для матчинга (pgvector)
	Embedding     []float32
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// LeadStatus — статус лида.
type LeadStatus string

const (
	LeadStatusUnspecified LeadStatus = ""
	LeadStatusNew         LeadStatus = "NEW"       // Создан, виден только создателю
	LeadStatusPublished   LeadStatus = "PUBLISHED" // Опубликован, доступен всем
	LeadStatusPurchased   LeadStatus = "PURCHASED" // Куплен
	LeadStatusDeleted     LeadStatus = "DELETED"   // Удалён админом
)

func (s LeadStatus) String() string {
	return string(s)
}

// LeadFilter — фильтр для выборок или обновлений лидов.
type LeadFilter struct {
	Title         *string
	Description   *string
	Requirement   *[]byte
	Status        *LeadStatus
	OwnerUserID   *uuid.UUID
	CreatedUserID *uuid.UUID
}
