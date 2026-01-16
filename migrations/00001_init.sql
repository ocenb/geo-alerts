-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS postgis;

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language plpgsql;

CREATE TABLE IF NOT EXISTS incidents (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
		location GEOMETRY(Point, 4326) NOT NULL, 
    radius_meters INTEGER NOT NULL CHECK (radius_meters > 0),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_incidents_location ON incidents USING GIST (location);
CREATE INDEX idx_incidents_active_id ON incidents (id) WHERE is_active = TRUE;

CREATE TRIGGER update_incidents_updated_at
    BEFORE UPDATE ON incidents
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE IF NOT EXISTS location_checks (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL, -- UUID or DeviceID
		location GEOMETRY(Point, 4326) NOT NULL, 
    has_danger BOOLEAN NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_location_checks_location ON location_checks USING GIST (location);
CREATE INDEX idx_location_checks_created_at ON location_checks (created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_incidents_updated_at ON incidents;
DROP FUNCTION IF EXISTS update_updated_at_column;
DROP TABLE IF EXISTS location_checks;
DROP TABLE IF EXISTS incidents;
-- +goose StatementEnd
