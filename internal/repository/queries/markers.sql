-- name: ListMarkersLightweight :many
-- Returns lightweight marker data for map display
SELECT id, short_code, name, latitude, longitude
FROM markers
ORDER BY created_at DESC;

-- name: GetMarkerByID :one
-- Returns full marker details by ID
SELECT * FROM markers WHERE id = $1;
