// Package filter provides the canonical Filter and Paging types for all list
// Every list operation must use Filter for input and Paging for output metadata.
// Never parse query params or write raw LIMIT/OFFSET outside this package.
package filter

import (
	"errors"
	"strings"
)

const (
	AscDirection  = "asc"
	DescDirection = "desc"

	DefaultPageLimit   = 20
	DefaultCurrentPage = 1
	MaxPageLimit       = 100

	// UnlimitedPage disables LIMIT/OFFSET — use carefully, only on small datasets.
	UnlimitedPage = -1
)

// Filter represents the canonical query parameters for every list/table endpoint.
// It is populated from gRPC request fields (page, per_page, keyword, sort_by, sort_direction).
type Filter struct {
	// Pagination
	CurrentPage int `json:"current_page"`
	PerPage     int `json:"per_page"`

	// Global keyword search
	Keyword string `json:"keyword"`

	// Sorting
	SortBy        string `json:"sort_by"`
	SortDirection string `json:"sort_direction"`
}

// NewFilter returns a Filter with safe defaults applied.
func NewFilter() Filter {
	return Filter{
		CurrentPage:   DefaultCurrentPage,
		PerPage:       DefaultPageLimit,
		SortDirection: AscDirection,
	}
}

// Validate enforces safe defaults and clamps out-of-bounds values in-place.
func (f *Filter) Validate() {
	if f.CurrentPage < 1 {
		f.CurrentPage = DefaultCurrentPage
	}
	if f.PerPage < 1 && f.PerPage != UnlimitedPage {
		f.PerPage = DefaultPageLimit
	}
	if f.PerPage > MaxPageLimit {
		f.PerPage = MaxPageLimit
	}
	if f.SortDirection != AscDirection && f.SortDirection != DescDirection {
		f.SortDirection = AscDirection
	}
	f.Keyword = strings.TrimSpace(f.Keyword)
}

// GetLimit returns the SQL LIMIT value (-1 = no limit).
func (f *Filter) GetLimit() int {
	if f.PerPage == UnlimitedPage {
		return UnlimitedPage
	}
	return f.PerPage
}

// GetOffset returns the SQL OFFSET value.
func (f *Filter) GetOffset() int {
	if f.PerPage == UnlimitedPage {
		return 0
	}
	return (f.CurrentPage - 1) * f.PerPage
}

func (f *Filter) HasKeyword() bool  { return strings.TrimSpace(f.Keyword) != "" }
func (f *Filter) HasSort() bool     { return strings.TrimSpace(f.SortBy) != "" }
func (f *Filter) IsDesc() bool      { return strings.EqualFold(f.SortDirection, DescDirection) }
func (f *Filter) IsUnlimited() bool { return f.PerPage == UnlimitedPage }

// ── Paging ─────────────────────────────────────────────────────────────────────

// Paging is the standard frontend-friendly pagination metadata returned in
// every list response. It provides everything a data-table needs to render
// navigation controls (prev/next buttons, total, page range).
type Paging struct {
	HasPreviousPage        bool `json:"has_previous_page"`
	HasNextPage            bool `json:"has_next_page"`
	CurrentPage            int  `json:"current_page"`
	PerPage                int  `json:"per_page"`
	TotalData              int  `json:"total_data"`
	TotalDataInCurrentPage int  `json:"total_data_in_current_page"`
	LastPage               int  `json:"last_page"`
	From                   int  `json:"from"`
	To                     int  `json:"to"`
}

// ErrInvalidPaging is returned when pagination parameters are logically invalid.
var ErrInvalidPaging = errors.New("per_page must be > 0 and current_page must be >= 1")

// NewPaging computes the full Paging metadata from the request filter and the
// total row count returned by the database COUNT query.
func NewPaging(currentPage, perPage, totalData int) (*Paging, error) {
	// Unlimited mode — all data in one page
	if perPage == UnlimitedPage {
		return &Paging{
			CurrentPage:            currentPage,
			PerPage:                perPage,
			TotalData:              totalData,
			LastPage:               1,
			From:                   1,
			To:                     totalData,
			TotalDataInCurrentPage: totalData,
		}, nil
	}

	if totalData == 0 {
		return &Paging{
			CurrentPage: 1,
			PerPage:     perPage,
			TotalData:   0,
			LastPage:    1,
		}, nil
	}

	offset := (currentPage - 1) * perPage
	if perPage <= 0 || offset < 0 {
		return nil, ErrInvalidPaging
	}

	lastPage := totalData / perPage
	if totalData%perPage != 0 {
		lastPage++
	}

	to := min(offset+perPage, totalData)
	from := 0
	if to > offset {
		from = offset + 1
	}

	return &Paging{
		HasPreviousPage:        currentPage > 1,
		HasNextPage:            currentPage < lastPage,
		CurrentPage:            currentPage,
		PerPage:                perPage,
		TotalData:              totalData,
		LastPage:               lastPage,
		From:                   from,
		To:                     to,
		TotalDataInCurrentPage: to - offset,
	}, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
