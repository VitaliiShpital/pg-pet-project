-- description: create initial schema for Air Defense Multi-Source Sensor Fusion

-- enable PostGIS extension (spatial DB capabilities)
CREATE EXTENSION IF NOT EXISTS postgis;

-- sensor types
CREATE TYPE sensor_type AS ENUM (
    'RADAR',           -- primary air-defense radar
    'ACOUSTIC',        -- acoustic sensor (microphone array network?)
    'OPTICAL',         -- optical sensor (thermal camera network?)
    'VISUAL_OBSERVER', -- mobile fire group (human observers)
    'SIGINT'           -- signals intelligence (radio frequency?)
);

-- target types
CREATE TYPE target_classification AS ENUM (
    'UNKNOWN',           -- unknown target
    'UAV_SHAHEAD_136',   -- low-speed kamikaze drone
    'UAV_RECON',         -- high-altitude reconnaissance UAV
    'CRUISE_MISSILE',    -- high-speed low-altitude cruise missile
    'BALLISTIC_MISSILE', -- high-speed high-altitude ballistic trajectory
    'AIRCRAFT_FRIENDLY'  -- friendly air force asset
);

-- raw sensor data table
CREATE TABLE raw_sensor_detections (
    id BIGSERIAL PRIMARY KEY,

    -- (e.g., 'radar-kyiv-north-01', 'acoustic-post-402')
    sensor_id VARCHAR(64) NOT NULL,
    sensor_type sensor_type NOT NULL,

    -- spatial location
    -- WGS84 EPSG:4326 - GPS standard latitude/longitude coordinates
    location GEOGRAPHY(Point, 4326) NOT NULL,

    -- height above mean sea level
    altitude_meters REAL,
    -- estimated ground speed
    speed_kmh REAL,
    -- heading in degrees (0.0 to 360.0)
    azimuth_degrees REAL,

    -- data quality and confidence (source reliability?) (0.0 - 1.0)
    confidence REAL DEFAULT 1.0,

    -- exact sensor measurement timestamp at the edge
    detected_at TIMESTAMPTZ NOT NULL,
    -- timestamp when received by backend
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- tracked targets table
CREATE TABLE tracked_targets (
    -- uuid because it can be cross-referenced across multiple systems and databases.
    id UUID PRIMARY KEY,
    -- human-readable identifier (e.g., target #104)
    track_number INT UNIQUE NOT NULL,

    classification target_classification DEFAULT 'UNKNOWN',

    -- current estimated state
    current_location GEOGRAPHY(Point, 4326) NOT NULL,
    current_altitude_meters REAL,
    current_speed_kmh REAL,
    current_heading_degrees REAL,

    -- active target lifecycle status
    -- could also be enum (status) but a boolean is_active flag is simpler for now
    is_active BOOLEAN DEFAULT TRUE,

    first_detected_at TIMESTAMPTZ NOT NULL,
    last_updated_at TIMESTAMPTZ NOT NULL
);

-- air defense geofences table
CREATE TABLE air_defense_zones (
    id SERIAL PRIMARY KEY,
    zone_name VARCHAR(128) NOT NULL,
    -- e.g., 'CRITICAL_INFRASTRUCTURE', 'INTERCEPTION_ZONE'
    zone_type VARCHAR(32) NOT NULL,

    -- spatial Polygon in WGS84
    polygon GEOGRAPHY(Polygon, 4326) NOT NULL,

    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- BRIN index for time-series range queries on raw detections
-- extremely compact, low overhead during heavy write throughput
CREATE INDEX idx_raw_detections_time_brin
    ON raw_sensor_detections USING BRIN (detected_at);

-- GiST index for spatial proximity searches on raw detections
CREATE INDEX idx_raw_detections_location_gist
    ON raw_sensor_detections USING GIST (location);

-- GiST index for active targets location lookups
CREATE INDEX idx_tracked_targets_location_gist
    ON tracked_targets USING GIST (current_location);

-- GiST index for polygon intersection checks (Geofencing)
CREATE INDEX idx_zones_polygon_gist
    ON air_defense_zones USING GIST (polygon);
