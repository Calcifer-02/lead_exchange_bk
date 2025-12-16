package domain

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	// DefaultPageSize кол-во записей на странице по умолчанию
	DefaultPageSize = 20
	// MaxPageSize максимальное кол-во записей на странице
	MaxPageSize = 10000
)

// OrderDirection направление сортировки
type OrderDirection string

const (
	OrderAsc  OrderDirection = "asc"
	OrderDesc OrderDirection = "desc"
)

// PaginationParams параметры пагинации для запроса
type PaginationParams struct {
	PageSize       int32
	PageToken      string // cursor для cursor-based пагинации
	OrderBy        string
	OrderDirection OrderDirection
}

// PageCursor курсор для cursor-based пагинации
type PageCursor struct {
	LastID        uuid.UUID `json:"id"`
	LastCreatedAt time.Time `json:"ca"`
	LastValue     string    `json:"v,omitempty"` // для сортировки по другим полям
}

// Encode кодирует курсор в base64 строку
func (c *PageCursor) Encode() string {
	if c == nil {
		return ""
	}
	data, err := json.Marshal(c)
	if err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(data)
}

// DecodePageCursor декодирует курсор из base64 строки
func DecodePageCursor(token string) (*PageCursor, error) {
	if token == "" {
		return nil, nil
	}
	data, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return nil, err
	}
	var cursor PageCursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return nil, err
	}
	return &cursor, nil
}

// PaginatedResult результат пагинированного запроса
type PaginatedResult[T any] struct {
	Items         []T
	NextPageToken string
	TotalCount    int32
	HasMore       bool
}

// NormalizePageSize нормализует размер страницы
func NormalizePageSize(size int32) int32 {
	if size <= 0 {
		return DefaultPageSize
	}
	if size > MaxPageSize {
		return MaxPageSize
	}
	return size
}

// NormalizeOrderDirection нормализует направление сортировки
func NormalizeOrderDirection(dir string) OrderDirection {
	if dir == "asc" || dir == "ASC" {
		return OrderAsc
	}
	return OrderDesc
}

// Pager старая структура для обратной совместимости
type Pager struct {
	page, perPage int32
}

func NewPager(page int32, perPage int32) *Pager {
	return &Pager{page: page, perPage: perPage}
}

// Limit вернет SQL LIMIT
func (p *Pager) Limit() int64 {
	if p == nil || p.perPage == 0 {
		return DefaultPageSize
	}

	return min(MaxPageSize, int64(p.perPage))
}

// Offset вернет для SQL OFFSET
func (p *Pager) Offset() int64 {
	if p == nil || p.page == 0 {
		return 0
	}
	return int64((p.page - 1) * p.perPage)
}
