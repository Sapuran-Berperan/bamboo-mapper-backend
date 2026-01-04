package model

import (
	"time"

	"github.com/google/uuid"
)

// PaginationMeta contains pagination metadata for paginated responses
type PaginationMeta struct {
	CurrentPage int   `json:"current_page"`
	PerPage     int   `json:"per_page"`
	TotalItems  int64 `json:"total_items"`
	TotalPages  int   `json:"total_pages"`
}

// ListMarkersParams contains all parameters for paginated marker listing
type ListMarkersParams struct {
	// Pagination
	Page    int
	PerPage int

	// Sorting
	SortBy  string
	SortDir string

	// Search
	Search string

	// Filters
	DateFrom  *time.Time
	DateTo    *time.Time
	CreatorID *uuid.UUID
}

// DefaultListMarkersParams returns default pagination parameters
func DefaultListMarkersParams() ListMarkersParams {
	return ListMarkersParams{
		Page:    1,
		PerPage: 10,
		SortBy:  "created_at",
		SortDir: "desc",
	}
}
