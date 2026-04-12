// Package pagination provides helpers to normalize and validate pagination parameters.
package pagination

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// Normalize sanitises page_size and page values from a request,
// returning a (pageSize, offset) pair safe to hand to a repository.
func Normalize(pageSize, page int) (size, offset int) {
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	if page <= 0 {
		page = 1
	}
	return pageSize, (page - 1) * pageSize
}
