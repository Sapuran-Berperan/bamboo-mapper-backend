package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/model"
)

// Meta contains response metadata
type Meta struct {
	Success bool              `json:"success"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// Response is the standard API response structure
type Response struct {
	Meta Meta        `json:"meta"`
	Data interface{} `json:"data"`
}

// respondJSON writes a JSON response with the given status code
func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

// respondSuccess sends a success response
func respondSuccess(w http.ResponseWriter, status int, message string, data interface{}) {
	respondJSON(w, status, Response{
		Meta: Meta{
			Success: true,
			Message: message,
		},
		Data: data,
	})
}

// respondError sends an error response with data: null
func respondError(w http.ResponseWriter, status int, message string, details map[string]string) {
	respondJSON(w, status, Response{
		Meta: Meta{
			Success: false,
			Message: message,
			Details: details,
		},
		Data: nil,
	})
}

// PaginatedMeta contains response metadata with pagination info
type PaginatedMeta struct {
	Success    bool                  `json:"success"`
	Message    string                `json:"message"`
	Details    map[string]string     `json:"details,omitempty"`
	Pagination *model.PaginationMeta `json:"pagination,omitempty"`
}

// PaginatedResponse is the API response structure for paginated endpoints
type PaginatedResponse struct {
	Meta PaginatedMeta `json:"meta"`
	Data interface{}   `json:"data"`
}

// respondPaginated sends a success response with pagination metadata
func respondPaginated(w http.ResponseWriter, status int, message string, data interface{}, pagination model.PaginationMeta) {
	respondJSON(w, status, PaginatedResponse{
		Meta: PaginatedMeta{
			Success:    true,
			Message:    message,
			Pagination: &pagination,
		},
		Data: data,
	})
}
