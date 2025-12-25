-- Drop triggers
DROP TRIGGER IF EXISTS markers_updated_at ON markers;
DROP TRIGGER IF EXISTS users_updated_at ON users;

-- Drop tables
DROP TABLE IF EXISTS markers;
DROP TABLE IF EXISTS users;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at();

-- Drop extension (optional, usually keep it)
-- DROP EXTENSION IF EXISTS "uuid-ossp";
