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
	// City — город для жёсткой фильтрации при матчинге
	City          *string
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
	// City — жёсткий фильтр по городу (критическое поле)
	City          *string
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

	// Пагинация
	Pagination    *PaginationParams
}

// MatchedProperty — результат матчинга с коэффициентом схожести.
type MatchedProperty struct {
	Property   Property
	Similarity float64 // Косинусная близость (0-1)
	// Взвешенные scores (заполняются при use_weighted_ranking=true)
	TotalScore       *float64
	PriceScore       *float64
	DistrictScore    *float64
	RoomsScore       *float64
	AreaScore        *float64
	SemanticScore    *float64
	MatchExplanation *string
}

// MatchWeights — веса для параметров матчинга (сумма должна быть ~1.0).
type MatchWeights struct {
	Price    float64 `json:"price"`    // Вес цены (default: 0.30)
	District float64 `json:"district"` // Вес района (default: 0.25)
	Rooms    float64 `json:"rooms"`    // Вес комнат (default: 0.20)
	Area     float64 `json:"area"`     // Вес площади (default: 0.10)
	Semantic float64 `json:"semantic"` // Вес семантики (default: 0.15)
}

// DefaultWeights возвращает веса по умолчанию.
func DefaultWeights() MatchWeights {
	return MatchWeights{
		Price:    0.30,
		District: 0.25,
		Rooms:    0.20,
		Area:     0.10,
		Semantic: 0.15,
	}
}

// Normalize нормализует веса чтобы сумма = 1.
func (w MatchWeights) Normalize() MatchWeights {
	total := w.Price + w.District + w.Rooms + w.Area + w.Semantic
	if total <= 0 {
		return DefaultWeights()
	}
	return MatchWeights{
		Price:    w.Price / total,
		District: w.District / total,
		Rooms:    w.Rooms / total,
		Area:     w.Area / total,
		Semantic: w.Semantic / total,
	}
}

// SoftCriteria — мягкие критерии для ранжирования (из лида).
type SoftCriteria struct {
	TargetPrice        *int64   // Желаемая цена
	TargetDistrict     *string  // Желаемый район
	TargetRooms        *int32   // Желаемое кол-во комнат
	TargetArea         *float64 // Желаемая площадь
	PreferredDistricts []string // Список предпочтительных районов
}

// HardFilters — жёсткие фильтры для критических полей матчинга.
// Объекты, не соответствующие этим критериям, исключаются из выдачи.
type HardFilters struct {
	// City — город (обязательное совпадение)
	City *string
	// PropertyType — тип недвижимости (обязательное совпадение)
	PropertyType *PropertyType
	// MinRooms / MaxRooms — диапазон комнат (жёсткий, но с допуском)
	MinRooms *int32
	MaxRooms *int32
	// MinPrice / MaxPrice — ценовой диапазон (жёсткий, но с допуском)
	MinPrice *int64
	MaxPrice *int64
}

// DefaultHardFiltersFromLead создаёт HardFilters из данных лида.
// Применяет разумные допуски: комнаты ±1, цена ±20%.
func DefaultHardFiltersFromLead(city *string, propertyType *PropertyType, targetRooms *int32, targetPrice *int64) HardFilters {
	hf := HardFilters{
		City:         city,
		PropertyType: propertyType,
	}

	// Допуск по комнатам: ±1
	if targetRooms != nil {
		minR := *targetRooms - 1
		if minR < 1 {
			minR = 1
		}
		maxR := *targetRooms + 1
		hf.MinRooms = &minR
		hf.MaxRooms = &maxR
	}

	// Допуск по цене: ±20%
	if targetPrice != nil {
		minP := int64(float64(*targetPrice) * 0.8)
		maxP := int64(float64(*targetPrice) * 1.2)
		hf.MinPrice = &minP
		hf.MaxPrice = &maxP
	}

	return hf
}

// WeightPreset — пресет весов.
type WeightPreset struct {
	ID          string
	Name        string
	Description string
	Weights     MatchWeights
}

// GetWeightPresets возвращает предустановленные наборы весов.
func GetWeightPresets() []WeightPreset {
	return []WeightPreset{
		{ID: "balanced", Name: "Сбалансированный", Description: "Равномерное распределение", Weights: MatchWeights{Price: 0.25, District: 0.25, Rooms: 0.20, Area: 0.15, Semantic: 0.15}},
		{ID: "budget_first", Name: "Бюджет важнее", Description: "Приоритет на цену", Weights: MatchWeights{Price: 0.45, District: 0.20, Rooms: 0.15, Area: 0.10, Semantic: 0.10}},
		{ID: "location_first", Name: "Локация важнее", Description: "Приоритет на район", Weights: MatchWeights{Price: 0.20, District: 0.40, Rooms: 0.15, Area: 0.10, Semantic: 0.15}},
		{ID: "family", Name: "Для семьи", Description: "Комнаты и площадь", Weights: MatchWeights{Price: 0.20, District: 0.20, Rooms: 0.30, Area: 0.20, Semantic: 0.10}},
		{ID: "semantic", Name: "Умный поиск", Description: "Приоритет на семантику", Weights: MatchWeights{Price: 0.15, District: 0.15, Rooms: 0.15, Area: 0.10, Semantic: 0.45}},
	}
}

// GetWeightPresetByID возвращает пресет по ID.
func GetWeightPresetByID(id string) *WeightPreset {
	for _, p := range GetWeightPresets() {
		if p.ID == id {
			return &p
		}
	}
	return nil
}

