package internal

// Pager holds metadata for paginated results.
type Pager struct {
	CurrentPage int
	PerPage     int
	TotalItems  int
	TotalPages  int
}

// NewPager creates a new Pager for the given current page, items per page, and total items.
func NewPager(currentPage, perPage, totalItems int) *Pager {
	if perPage <= 0 {
		perPage = 1
	}
	totalPages := (totalItems + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}
	if currentPage < 1 {
		currentPage = 1
	} else if currentPage > totalPages {
		currentPage = totalPages
	}
	return &Pager{
		CurrentPage: currentPage,
		PerPage:     perPage,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
	}
}

// HasPrev returns true if there is a previous page.
func (p *Pager) HasPrev() bool {
	return p.CurrentPage > 1
}

// HasNext returns true if there is a next page.
func (p *Pager) HasNext() bool {
	return p.CurrentPage < p.TotalPages
}

// PrevPage returns the previous page number, or 1 if on the first page.
func (p *Pager) PrevPage() int {
	if p.HasPrev() {
		return p.CurrentPage - 1
	}
	return 1
}

// NextPage returns the next page number, or the last page if on the final page.
func (p *Pager) NextPage() int {
	if p.HasNext() {
		return p.CurrentPage + 1
	}
	return p.TotalPages
}

// Page returns the current page number.
func (p *Pager) Page() int {
	return p.CurrentPage
}

// PageNums returns the total number of pages.
func (p *Pager) PageNums() int {
	return p.TotalPages
}

// Nums returns the total number of items.
func (p *Pager) Nums() int {
	return p.TotalItems
}
