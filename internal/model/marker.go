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
	ID           uuid.UUID  `json:"id"`
	ShortCode    string     `json:"short_code"`
	CreatorID    uuid.UUID  `json:"creator_id"`
	Name         string     `json:"name"`
	Description  *string    `json:"description"`
	Strain       *string    `json:"strain"`
	Quantity     *int32     `json:"quantity"`
	Latitude     string     `json:"latitude"`
	Longitude    string     `json:"longitude"`
	ImageURL     *string    `json:"image_url"`
	OwnerName    *string    `json:"owner_name"`
	OwnerContact *string    `json:"owner_contact"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
