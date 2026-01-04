package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/model"
	"github.com/google/uuid"
)

// ListMarkersPaginatedResult contains the paginated markers and total count
type ListMarkersPaginatedResult struct {
	Markers    []Marker
	TotalCount int64
}

// ListMarkersPaginated retrieves markers with pagination, sorting, search, and filters
func (q *Queries) ListMarkersPaginated(ctx context.Context, params model.ListMarkersParams) (*ListMarkersPaginatedResult, error) {
	// Use PostgreSQL placeholder format ($1, $2, etc.)
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	// Build base WHERE conditions
	conditions := sq.And{}

	// Add search condition
	if params.Search != "" {
		searchPattern := "%" + params.Search + "%"
		searchCondition := sq.Or{
			sq.ILike{"name": searchPattern},
			sq.ILike{"description": searchPattern},
			sq.ILike{"strain": searchPattern},
			sq.ILike{"short_code": searchPattern},
			sq.ILike{"owner_name": searchPattern},
			sq.ILike{"owner_contact": searchPattern},
		}
		conditions = append(conditions, searchCondition)
	}

	// Add date_from filter
	if params.DateFrom != nil {
		conditions = append(conditions, sq.GtOrEq{"created_at": params.DateFrom})
	}

	// Add date_to filter
	if params.DateTo != nil {
		conditions = append(conditions, sq.LtOrEq{"created_at": params.DateTo})
	}

	// Add creator_id filter
	if params.CreatorID != nil {
		conditions = append(conditions, sq.Eq{"creator_id": params.CreatorID})
	}

	// Get total count first
	countQuery := psql.Select("COUNT(*)").From("markers")
	if len(conditions) > 0 {
		countQuery = countQuery.Where(conditions)
	}

	countSQL, countArgs, err := countQuery.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build count query: %w", err)
	}

	var totalCount int64
	err = q.db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to execute count query: %w", err)
	}

	// If no results, return early
	if totalCount == 0 {
		return &ListMarkersPaginatedResult{
			Markers:    []Marker{},
			TotalCount: 0,
		}, nil
	}

	// Build select query
	selectQuery := psql.Select(
		"id", "short_code", "creator_id", "name", "description",
		"strain", "quantity", "latitude", "longitude", "image_url",
		"owner_name", "owner_contact", "created_at", "updated_at",
	).From("markers")

	if len(conditions) > 0 {
		selectQuery = selectQuery.Where(conditions)
	}

	// Add ordering
	orderColumn := sanitizeSortColumn(params.SortBy)
	orderDir := strings.ToUpper(params.SortDir)
	if orderDir != "ASC" && orderDir != "DESC" {
		orderDir = "DESC"
	}
	selectQuery = selectQuery.OrderBy(fmt.Sprintf("%s %s", orderColumn, orderDir))

	// Add pagination
	offset := (params.Page - 1) * params.PerPage
	selectQuery = selectQuery.Limit(uint64(params.PerPage)).Offset(uint64(offset))

	// Execute query
	selectSQL, selectArgs, err := selectQuery.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	rows, err := q.db.QueryContext(ctx, selectSQL, selectArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute select query: %w", err)
	}
	defer rows.Close()

	var markers []Marker
	for rows.Next() {
		var m Marker
		err := rows.Scan(
			&m.ID,
			&m.ShortCode,
			&m.CreatorID,
			&m.Name,
			&m.Description,
			&m.Strain,
			&m.Quantity,
			&m.Latitude,
			&m.Longitude,
			&m.ImageUrl,
			&m.OwnerName,
			&m.OwnerContact,
			&m.CreatedAt,
			&m.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan marker row: %w", err)
		}
		markers = append(markers, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating marker rows: %w", err)
	}

	return &ListMarkersPaginatedResult{
		Markers:    markers,
		TotalCount: totalCount,
	}, nil
}

// sanitizeSortColumn ensures only allowed columns are used for sorting
func sanitizeSortColumn(column string) string {
	allowedColumns := map[string]bool{
		"name":       true,
		"created_at": true,
		"updated_at": true,
		"strain":     true,
		"quantity":   true,
	}
	if allowedColumns[column] {
		return column
	}
	return "created_at"
}

// GetDB returns the underlying database connection for custom queries
func (q *Queries) GetDB() DBTX {
	return q.db
}

// NullStringToPtr converts sql.NullString to *string
func NullStringToPtr(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

// NullInt32ToPtr converts sql.NullInt32 to *int32
func NullInt32ToPtr(ni sql.NullInt32) *int32 {
	if ni.Valid {
		return &ni.Int32
	}
	return nil
}

// NullUUIDToPtr converts uuid.NullUUID to *uuid.UUID
func NullUUIDToPtr(nu uuid.NullUUID) *uuid.UUID {
	if nu.Valid {
		return &nu.UUID
	}
	return nil
}
