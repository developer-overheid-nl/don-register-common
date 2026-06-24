package pagination

import "math"

type Pagination struct {
	Next           *int
	Previous       *int
	CurrentPage    int
	RecordsPerPage int
	TotalPages     int
	TotalRecords   int
}

func Normalize(page, perPage, defaultPerPage, maxPerPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = defaultPerPage
	}
	if maxPerPage > 0 && perPage > maxPerPage {
		perPage = maxPerPage
	}
	return page, perPage
}

func New(page, perPage, totalRecords int) Pagination {
	totalPages := 0
	if totalRecords > 0 && perPage > 0 {
		totalPages = int(math.Ceil(float64(totalRecords) / float64(perPage)))
	}
	p := Pagination{
		CurrentPage:    page,
		RecordsPerPage: perPage,
		TotalPages:     totalPages,
		TotalRecords:   totalRecords,
	}
	if page < totalPages {
		next := page + 1
		p.Next = &next
	}
	if page > 1 && totalPages > 0 {
		prev := page - 1
		p.Previous = &prev
	}
	return p
}
