-- name: ListMarkersLightweight :many
-- Returns lightweight marker data for map display
SELECT id, short_code, name, latitude, longitude
FROM markers
ORDER BY created_at DESC;

-- name: GetMarkerByID :one
-- Returns full marker details by ID
SELECT * FROM markers WHERE id = $1;

-- name: GetMarkerByShortCode :one
-- Returns full marker details by short_code (for QR code scanning)
SELECT * FROM markers WHERE short_code = $1;

-- name: CreateMarker :one
-- Creates a new marker and returns the created record
INSERT INTO markers (
    short_code, creator_id, name, description, strain,
    quantity, latitude, longitude, image_url, owner_name, owner_contact
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: UpdateMarker :one
-- Updates an existing marker and returns the updated record
UPDATE markers SET
    name = $2,
    description = $3,
    strain = $4,
    quantity = $5,
    latitude = $6,
    longitude = $7,
    image_url = $8,
    owner_name = $9,
    owner_contact = $10,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteMarker :exec
-- Deletes a marker by ID
DELETE FROM markers WHERE id = $1;
