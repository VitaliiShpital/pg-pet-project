package telemetry

import (
	"os"
	"time"
)

// Helper function to read environment variables with fallback
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// Broker & DB Connections with Environment Overrides
var (
	DefaultNATSURL     = getEnv("NATS_URL", "nats://localhost:4222")
	DefaultMQTTURL     = getEnv("MQTT_URL", "tcp://localhost:1883")
	DefaultPostgresDSN = getEnv("POSTGRES_DSN", "postgres://postgres:postgres_password@localhost:5432/air_defense?sslmode=disable")
)

// MQTT Configuration
const (
	MQTTTelemetryTopic  = "telemetry/#"
	DefaultTelemetryQoS = 0
)

// NATS Subjects
const (
	NATSRawTelemetryWildcard = "telemetry.raw.>"
	NATSRawTelemetryPrefix   = "telemetry.raw."

	NATSFusedTracksWildcard = "tracks.fused.>"
	NATSFusedTracksPrefix   = "tracks.fused."
)

// RawSensorEvent represents incoming raw telemetry from edge sensors
type RawSensorEvent struct {
	SensorID       string    `json:"sensor_id"`
	SensorType     string    `json:"sensor_type"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	AzimuthDegrees float64   `json:"azimuth_degrees"`
	SpeedKMH       float64   `json:"speed_kmh"`
	Confidence     float64   `json:"confidence"`
	DetectedAt     time.Time `json:"detected_at"`
}

// FusedTrack represents the smoothed target track position and trajectory
type FusedTrack struct {
	TrackID        string    `json:"track_id"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	EstimatedSpeed float64   `json:"estimated_speed"`
	HeadingDegrees float64   `json:"heading_degrees"`
	UpdateCount    uint64    `json:"update_count"`
	LastUpdatedAt  time.Time `json:"last_updated_at"`
}
