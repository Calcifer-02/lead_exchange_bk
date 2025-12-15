package domain

import (
	"time"

	"github.com/google/uuid"
)

// Deal — доменная сущность сделки.
type Deal struct {
	ID           uuid.UUID
	LeadID       uuid.UUID
	SellerUserID uuid.UUID
	BuyerUserID  *uuid.UUID // nil пока не найден покупатель
	Price        float64    // цена сделки
	Status       DealStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// DealStatus — статус сделки.
type DealStatus string

const (
	DealStatusUnspecified DealStatus = ""
	DealStatusPending     DealStatus = "PENDING"   // Создана, ожидает покупателя
	DealStatusAccepted    DealStatus = "ACCEPTED"  // Принята покупателем
	DealStatusCompleted   DealStatus = "COMPLETED" // Завершена (лид передан)
	DealStatusCancelled   DealStatus = "CANCELLED" // Отменена продавцом
	DealStatusRejected    DealStatus = "REJECTED"  // Отклонена покупателем
)

func (s DealStatus) String() string {
	return string(s)
}

// DealFilter — фильтр для выборок или обновлений сделок.
type DealFilter struct {
	LeadID       *uuid.UUID
	SellerUserID *uuid.UUID
	BuyerUserID  *uuid.UUID
	Status       *DealStatus
	Price        *float64 // для обновления цены
	MinPrice     *float64 // для фильтрации
	MaxPrice     *float64 // для фильтрации
}
