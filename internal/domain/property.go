package domain

import (
	"time"

	"github.com/google/uuid"
)

// Property — доменная сущность объекта недвижимости.
type Property struct {
	ID            uuid.UUID
	Title         string
	Description   string
	Address       string
	PropertyType  PropertyType
	Area          *float64
	Price         *int64
	Rooms         *int32
	Status        PropertyStatus
	OwnerUserID   uuid.UUID
	CreatedUserID uuid.UUID
	// Embedding — векторное представление для матчинга (pgvector)
	Embedding     []float32
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// PropertyType — тип недвижимости.
type PropertyType string

const (
	PropertyTypeUnspecified PropertyType = ""
	PropertyTypeApartment   PropertyType = "APARTMENT"   // Квартира
	PropertyTypeHouse       PropertyType = "HOUSE"       // Дом
	PropertyTypeCommercial  PropertyType = "COMMERCIAL"   // Коммерческая недвижимость
	PropertyTypeLand        PropertyType = "LAND"        // Земельный участок
)

func (t PropertyType) String() string {
	return string(t)
}

// PropertyStatus — статус объекта недвижимости.
type PropertyStatus string

const (
	PropertyStatusUnspecified PropertyStatus = ""
	PropertyStatusNew         PropertyStatus = "NEW"       // Создан, виден только создателю
	PropertyStatusPublished   PropertyStatus = "PUBLISHED" // Опубликован, доступен всем
	PropertyStatusSold        PropertyStatus = "SOLD"      // Продан
	PropertyStatusDeleted     PropertyStatus = "DELETED"   // Удалён админом
)

func (s PropertyStatus) String() string {
	return string(s)
}

// PropertyFilter — фильтр для выборок или обновлений объектов недвижимости.
type PropertyFilter struct {
	Title         *string
	Description   *string
	Address       *string
	PropertyType  *PropertyType
	Area          *float64
	Price         *int64
	Rooms         *int32
	MinRooms      *int32
	MaxRooms      *int32
	MinPrice      *int64
	MaxPrice      *int64
	Status        *PropertyStatus
	OwnerUserID   *uuid.UUID
	CreatedUserID *uuid.UUID
}

// MatchedProperty — результат матчинга с коэффициентом схожести.
type MatchedProperty struct {
	Property   Property
	Similarity float64 // Коэффициент схожести (1 - расстояние)
}

