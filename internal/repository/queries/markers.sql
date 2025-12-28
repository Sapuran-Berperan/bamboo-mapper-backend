-- name: ListMarkersLightweight :many
-- Returns lightweight marker data for map display
SELECT id, short_code, name, latitude, longitude
FROM markers
ORDER BY created_at DESC;

-- name: GetMarkerByID :one
-- Returns full marker details by ID
SELECT * FROM markers WHERE id = $1;

-- name: CreateMarker :one
-- Creates a new marker and returns the created record
INSERT INTO markers (
    short_code, creator_id, name, description, strain,
    quantity, latitude, longitude, image_url, owner_name, owner_contact
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;
