package handler

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/model"
	"github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// MarkerHandler handles marker-related requests
type MarkerHandler struct {
	queries *repository.Queries
}

// NewMarkerHandler creates a new MarkerHandler
func NewMarkerHandler(queries *repository.Queries) *MarkerHandler {
	return &MarkerHandler{
		queries: queries,
	}
}

// List returns all markers in lightweight format for map display
func (h *MarkerHandler) List(w http.ResponseWriter, r *http.Request) {
	markers, err := h.queries.ListMarkersLightweight(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch markers", nil)
		return
	}

	// Convert to response model
	response := make([]model.MarkerListItem, len(markers))
	for i, m := range markers {
		response[i] = model.MarkerListItem{
			ID:        m.ID,
			ShortCode: m.ShortCode,
			Name:      m.Name,
			Latitude:  m.Latitude,
			Longitude: m.Longitude,
		}
	}

	respondSuccess(w, http.StatusOK, "Markers retrieved successfully", response)
}

// GetByID returns full marker details by ID
func (h *MarkerHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid marker ID", nil)
		return
	}

	marker, err := h.queries.GetMarkerByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondError(w, http.StatusNotFound, "Marker not found", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to fetch marker", nil)
		return
	}

	// Convert to response model with nullable fields
	response := model.MarkerResponse{
		ID:        marker.ID,
		ShortCode: marker.ShortCode,
		CreatorID: marker.CreatorID,
		Name:      marker.Name,
		Latitude:  marker.Latitude,
		Longitude: marker.Longitude,
		CreatedAt: marker.CreatedAt.Time,
		UpdatedAt: marker.UpdatedAt.Time,
	}

	// Handle nullable fields
	if marker.Description.Valid {
		response.Description = &marker.Description.String
	}
	if marker.Strain.Valid {
		response.Strain = &marker.Strain.String
	}
	if marker.Quantity.Valid {
		response.Quantity = &marker.Quantity.Int32
	}
	if marker.ImageUrl.Valid {
		response.ImageURL = &marker.ImageUrl.String
	}
	if marker.OwnerName.Valid {
		response.OwnerName = &marker.OwnerName.String
	}
	if marker.OwnerContact.Valid {
		response.OwnerContact = &marker.OwnerContact.String
	}

	respondSuccess(w, http.StatusOK, "Marker retrieved successfully", response)
}
