package domain

const (
	// DefaultPerPage кол-во записей на странице по умолчанию
	DefaultPerPage = 25
	// MaxPerPage максимальное кол-во записей на странице
	MaxPerPage = 10000
)

type Pager struct {
	page, perPage int32
}

func NewPager(page int32, perPage int32) *Pager {
	return &Pager{page: page, perPage: perPage}
}

// Limit вернет SQL LIMIT
func (p *Pager) Limit() int64 {
	if p == nil || p.perPage == 0 {
		return DefaultPerPage
	}

	return min(MaxPerPage, int64(p.perPage))
}

// Offset вернет для SQL OFFSET
func (p *Pager) Offset() int64 {
	if p == nil || p.page == 0 {
		return 0
	}
	return int64((p.page - 1) * p.perPage)
}
