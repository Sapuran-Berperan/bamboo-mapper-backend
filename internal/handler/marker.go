package handler

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/middleware"
	"github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/model"
	"github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/repository"
	"github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/storage"
	"github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/util"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const maxUploadSize = 10 << 20 // 10 MB

// MarkerHandler handles marker-related requests
type MarkerHandler struct {
	queries *repository.Queries
	gdrive  *storage.GDriveService
}

// NewMarkerHandler creates a new MarkerHandler
func NewMarkerHandler(queries *repository.Queries, gdrive *storage.GDriveService) *MarkerHandler {
	return &MarkerHandler{
		queries: queries,
		gdrive:  gdrive,
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

// Create handles creating a new marker with optional image upload
func (h *MarkerHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Get creator ID from JWT context
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	// Parse multipart form with size limit
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		respondError(w, http.StatusBadRequest, "Failed to parse form data", nil)
		return
	}

	// Extract form fields
	req := model.CreateMarkerRequest{
		Name:      r.FormValue("name"),
		Latitude:  r.FormValue("latitude"),
		Longitude: r.FormValue("longitude"),
	}

	// Handle optional string fields
	if desc := r.FormValue("description"); desc != "" {
		req.Description = &desc
	}
	if strain := r.FormValue("strain"); strain != "" {
		req.Strain = &strain
	}
	if ownerName := r.FormValue("owner_name"); ownerName != "" {
		req.OwnerName = &ownerName
	}
	if ownerContact := r.FormValue("owner_contact"); ownerContact != "" {
		req.OwnerContact = &ownerContact
	}

	// Handle optional quantity field
	if qtyStr := r.FormValue("quantity"); qtyStr != "" {
		qty, err := strconv.ParseInt(qtyStr, 10, 32)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Validation failed", map[string]string{
				"quantity": "Invalid quantity format",
			})
			return
		}
		qty32 := int32(qty)
		req.Quantity = &qty32
	}

	// Validate request
	if validationErrors := req.Validate(); len(validationErrors) > 0 {
		respondError(w, http.StatusBadRequest, "Validation failed", validationErrors)
		return
	}

	// Generate short code first (needed for image filename)
	shortCode := util.GenerateShortCode()

	// Handle image upload (optional)
	var imageURL sql.NullString
	file, header, err := r.FormFile("image")
	if err == nil {
		defer file.Close()

		// Upload to Google Drive with short_code as filename
		if h.gdrive != nil {
			// Get file extension from original filename
			ext := getFileExtension(header.Filename)
			filename := shortCode + ext

			url, uploadErr := h.gdrive.UploadFile(file, filename, header.Header.Get("Content-Type"))
			if uploadErr != nil {
				log.Printf("Failed to upload image to Google Drive: %v", uploadErr)
				respondError(w, http.StatusInternalServerError, "Failed to upload image", nil)
				return
			}
			imageURL = sql.NullString{String: url, Valid: true}
		} else {
			log.Println("Image provided but Google Drive service not configured")
		}
	}

	// Create marker in database
	marker, err := h.queries.CreateMarker(r.Context(), repository.CreateMarkerParams{
		ShortCode:    shortCode,
		CreatorID:    claims.UserID,
		Name:         req.Name,
		Description:  toNullString(req.Description),
		Strain:       toNullString(req.Strain),
		Quantity:     toNullInt32(req.Quantity),
		Latitude:     req.Latitude,
		Longitude:    req.Longitude,
		ImageUrl:     imageURL,
		OwnerName:    toNullString(req.OwnerName),
		OwnerContact: toNullString(req.OwnerContact),
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create marker", nil)
		return
	}

	// Build response
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

	respondSuccess(w, http.StatusCreated, "Marker created successfully", response)
}

// Helper functions for nullable types
func toNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *s, Valid: true}
}

func toNullInt32(i *int32) sql.NullInt32 {
	if i == nil {
		return sql.NullInt32{Valid: false}
	}
	return sql.NullInt32{Int32: *i, Valid: true}
}

// getFileExtension extracts the file extension from a filename (e.g., ".jpg")
func getFileExtension(filename string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[i:]
		}
	}
	return ""
}

// extractGDriveFileID extracts the file ID from a Google Drive URL
// Format: https://drive.google.com/uc?id=FILE_ID
func extractGDriveFileID(url string) string {
	const prefix = "https://drive.google.com/uc?id="
	if len(url) > len(prefix) && url[:len(prefix)] == prefix {
		return url[len(prefix):]
	}
	return ""
}

// Update handles updating an existing marker
func (h *MarkerHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Ensure user is authenticated
	_, ok := middleware.GetClaims(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	// Get marker ID from URL
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid marker ID", nil)
		return
	}

	// Fetch existing marker
	existingMarker, err := h.queries.GetMarkerByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondError(w, http.StatusNotFound, "Marker not found", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to fetch marker", nil)
		return
	}

	// Parse multipart form with size limit
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		respondError(w, http.StatusBadRequest, "Failed to parse form data", nil)
		return
	}

	// Build update request with optional fields
	var req model.UpdateMarkerRequest

	if name := r.FormValue("name"); name != "" {
		req.Name = &name
	}
	if lat := r.FormValue("latitude"); lat != "" {
		req.Latitude = &lat
	}
	if lng := r.FormValue("longitude"); lng != "" {
		req.Longitude = &lng
	}
	if desc := r.FormValue("description"); desc != "" {
		req.Description = &desc
	}
	if strain := r.FormValue("strain"); strain != "" {
		req.Strain = &strain
	}
	if ownerName := r.FormValue("owner_name"); ownerName != "" {
		req.OwnerName = &ownerName
	}
	if ownerContact := r.FormValue("owner_contact"); ownerContact != "" {
		req.OwnerContact = &ownerContact
	}

	// Handle optional quantity field
	if qtyStr := r.FormValue("quantity"); qtyStr != "" {
		qty, err := strconv.ParseInt(qtyStr, 10, 32)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Validation failed", map[string]string{
				"quantity": "Invalid quantity format",
			})
			return
		}
		qty32 := int32(qty)
		req.Quantity = &qty32
	}

	// Validate request
	if validationErrors := req.Validate(); len(validationErrors) > 0 {
		respondError(w, http.StatusBadRequest, "Validation failed", validationErrors)
		return
	}

	// Prepare update params - use existing values for fields not provided
	updateParams := repository.UpdateMarkerParams{
		ID:           id,
		Name:         existingMarker.Name,
		Description:  existingMarker.Description,
		Strain:       existingMarker.Strain,
		Quantity:     existingMarker.Quantity,
		Latitude:     existingMarker.Latitude,
		Longitude:    existingMarker.Longitude,
		ImageUrl:     existingMarker.ImageUrl,
		OwnerName:    existingMarker.OwnerName,
		OwnerContact: existingMarker.OwnerContact,
	}

	// Override with provided values
	if req.Name != nil {
		updateParams.Name = *req.Name
	}
	if req.Latitude != nil {
		updateParams.Latitude = *req.Latitude
	}
	if req.Longitude != nil {
		updateParams.Longitude = *req.Longitude
	}
	if req.Description != nil {
		updateParams.Description = toNullString(req.Description)
	}
	if req.Strain != nil {
		updateParams.Strain = toNullString(req.Strain)
	}
	if req.Quantity != nil {
		updateParams.Quantity = toNullInt32(req.Quantity)
	}
	if req.OwnerName != nil {
		updateParams.OwnerName = toNullString(req.OwnerName)
	}
	if req.OwnerContact != nil {
		updateParams.OwnerContact = toNullString(req.OwnerContact)
	}

	// Handle image upload (optional)
	file, header, err := r.FormFile("image")
	if err == nil {
		defer file.Close()

		if h.gdrive != nil {
			// Delete old image if exists
			if existingMarker.ImageUrl.Valid {
				oldFileID := extractGDriveFileID(existingMarker.ImageUrl.String)
				if oldFileID != "" {
					if deleteErr := h.gdrive.DeleteFile(oldFileID); deleteErr != nil {
						log.Printf("Failed to delete old image from Google Drive: %v", deleteErr)
						// Continue anyway - old file might already be deleted
					}
				}
			}

			// Upload new image with short_code as filename
			ext := getFileExtension(header.Filename)
			filename := existingMarker.ShortCode + ext

			url, uploadErr := h.gdrive.UploadFile(file, filename, header.Header.Get("Content-Type"))
			if uploadErr != nil {
				log.Printf("Failed to upload image to Google Drive: %v", uploadErr)
				respondError(w, http.StatusInternalServerError, "Failed to upload image", nil)
				return
			}
			updateParams.ImageUrl = sql.NullString{String: url, Valid: true}
		} else {
			log.Println("Image provided but Google Drive service not configured")
		}
	}

	// Update marker in database
	marker, err := h.queries.UpdateMarker(r.Context(), updateParams)
	if err != nil {
		log.Printf("Failed to update marker: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to update marker", nil)
		return
	}

	// Build response
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

	respondSuccess(w, http.StatusOK, "Marker updated successfully", response)
}

// Delete handles deleting an existing marker
func (h *MarkerHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Ensure user is authenticated
	_, ok := middleware.GetClaims(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	// Get marker ID from URL
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid marker ID", nil)
		return
	}

	// Fetch existing marker to get image URL for cleanup
	existingMarker, err := h.queries.GetMarkerByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondError(w, http.StatusNotFound, "Marker not found", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to fetch marker", nil)
		return
	}

	// Delete image from Google Drive if exists
	if existingMarker.ImageUrl.Valid && h.gdrive != nil {
		fileID := extractGDriveFileID(existingMarker.ImageUrl.String)
		if fileID != "" {
			if deleteErr := h.gdrive.DeleteFile(fileID); deleteErr != nil {
				log.Printf("Failed to delete image from Google Drive: %v", deleteErr)
				// Continue anyway - don't fail the delete operation
			}
		}
	}

	// Delete marker from database
	if err := h.queries.DeleteMarker(r.Context(), id); err != nil {
		log.Printf("Failed to delete marker: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to delete marker", nil)
		return
	}

	respondSuccess(w, http.StatusOK, "Marker deleted successfully", nil)
}
