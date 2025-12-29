package model

import (
	"time"

	"github.com/google/uuid"
)

// MarkerListItem represents lightweight marker data for map display
type MarkerListItem struct {
	ID        uuid.UUID `json:"id"`
	ShortCode string    `json:"short_code"`
	Name      string    `json:"name"`
	Latitude  string    `json:"latitude"`
	Longitude string    `json:"longitude"`
}

// MarkerResponse represents full marker details
type MarkerResponse struct {
	ID           uuid.UUID `json:"id"`
	ShortCode    string    `json:"short_code"`
	CreatorID    uuid.UUID `json:"creator_id"`
	Name         string    `json:"name"`
	Description  *string   `json:"description"`
	Strain       *string   `json:"strain"`
	Quantity     *int32    `json:"quantity"`
	Latitude     string    `json:"latitude"`
	Longitude    string    `json:"longitude"`
	ImageURL     *string   `json:"image_url"`
	OwnerName    *string   `json:"owner_name"`
	OwnerContact *string   `json:"owner_contact"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CreateMarkerRequest represents the request body for creating a marker
type CreateMarkerRequest struct {
	Name         string
	Latitude     string
	Longitude    string
	Description  *string
	Strain       *string
	Quantity     *int32
	OwnerName    *string
	OwnerContact *string
}

// Validate validates the create marker request
func (r *CreateMarkerRequest) Validate() map[string]string {
	errors := make(map[string]string)

	if r.Name == "" {
		errors["name"] = "Name is required"
	}

	if r.Latitude == "" {
		errors["latitude"] = "Latitude is required"
	}

	if r.Longitude == "" {
		errors["longitude"] = "Longitude is required"
	}

	if r.Quantity != nil && *r.Quantity < 0 {
		errors["quantity"] = "Quantity must be non-negative"
	}

	return errors
}

// UpdateMarkerRequest represents the request body for updating a marker
// All fields are optional - only provided fields will be updated
type UpdateMarkerRequest struct {
	Name         *string
	Latitude     *string
	Longitude    *string
	Description  *string
	Strain       *string
	Quantity     *int32
	OwnerName    *string
	OwnerContact *string
}

// Validate validates the update marker request
func (r *UpdateMarkerRequest) Validate() map[string]string {
	errors := make(map[string]string)

	if r.Quantity != nil && *r.Quantity < 0 {
		errors["quantity"] = "Quantity must be non-negative"
	}

	return errors
}
