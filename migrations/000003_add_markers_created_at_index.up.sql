-- Add index on created_at for efficient pagination sorting
CREATE INDEX IF NOT EXISTS idx_markers_created_at ON markers(created_at DESC);
