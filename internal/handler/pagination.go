package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/model"
	"github.com/google/uuid"
)

// Allowed sort fields for marker listing
var allowedSortFields = map[string]bool{
	"name":       true,
	"created_at": true,
	"updated_at": true,
	"strain":     true,
	"quantity":   true,
}

// Allowed sort directions
var allowedSortDirs = map[string]bool{
	"asc":  true,
	"desc": true,
}

const (
	defaultPage    = 1
	defaultPerPage = 10
	maxPerPage     = 100
	minPerPage     = 1
)

// ParseListMarkersParams parses query parameters for paginated marker listing
func ParseListMarkersParams(r *http.Request) (model.ListMarkersParams, error) {
	params := model.DefaultListMarkersParams()

	// Parse page
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = defaultPage
		}
		params.Page = page
	}

	// Parse per_page
	if perPageStr := r.URL.Query().Get("per_page"); perPageStr != "" {
		perPage, err := strconv.Atoi(perPageStr)
		if err != nil || perPage < minPerPage {
			perPage = defaultPerPage
		}
		if perPage > maxPerPage {
			perPage = maxPerPage
		}
		params.PerPage = perPage
	}

	// Parse sort_by
	if sortBy := r.URL.Query().Get("sort_by"); sortBy != "" {
		sortBy = strings.ToLower(strings.TrimSpace(sortBy))
		if allowedSortFields[sortBy] {
			params.SortBy = sortBy
		}
	}

	// Parse sort_dir
	if sortDir := r.URL.Query().Get("sort_dir"); sortDir != "" {
		sortDir = strings.ToLower(strings.TrimSpace(sortDir))
		if allowedSortDirs[sortDir] {
			params.SortDir = sortDir
		}
	}

	// Parse search
	if search := r.URL.Query().Get("search"); search != "" {
		search = strings.TrimSpace(search)
		if len(search) >= 1 {
			params.Search = search
		}
	}

	// Parse date_from
	if dateFromStr := r.URL.Query().Get("date_from"); dateFromStr != "" {
		if dateFrom, err := time.Parse("2006-01-02", dateFromStr); err == nil {
			params.DateFrom = &dateFrom
		}
	}

	// Parse date_to
	if dateToStr := r.URL.Query().Get("date_to"); dateToStr != "" {
		if dateTo, err := time.Parse("2006-01-02", dateToStr); err == nil {
			// Set to end of day for inclusive filtering
			dateTo = dateTo.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			params.DateTo = &dateTo
		}
	}

	// Parse creator_id
	if creatorIDStr := r.URL.Query().Get("creator_id"); creatorIDStr != "" {
		if creatorID, err := uuid.Parse(creatorIDStr); err == nil {
			params.CreatorID = &creatorID
		}
	}

	return params, nil
}

// CalculateTotalPages calculates total pages from total items and per page
func CalculateTotalPages(totalItems int64, perPage int) int {
	if totalItems == 0 {
		return 0
	}
	pages := int(totalItems) / perPage
	if int(totalItems)%perPage > 0 {
		pages++
	}
	return pages
}

// CalculateOffset calculates SQL offset from page and per_page
func CalculateOffset(page, perPage int) int {
	return (page - 1) * perPage
}
