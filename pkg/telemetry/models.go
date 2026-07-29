package telemetry

import "time"

// Default Broker URLs
const (
	DefaultNATSURL = "nats://localhost:4222"
	DefaultMQTTURL = "tcp://localhost:1883"
)

// MQTT Configuration
const (
	MQTTTelemetryTopic = "telemetry/#"
	// DefaultTelemetryQoS defines the delivery guarantee level for sensor streams:
	// QoS 0: At most once delivery (lowest overhead for continuous tracking frames)
	// QoS 1: At least once delivery (use if missing a single frame causes tracking failure)
	DefaultTelemetryQoS byte = 0
)

// NATS Subjects
const (
	// NATSRawTelemetryWildcard is used by consumers to listen to all raw sensor events
	NATSRawTelemetryWildcard = "telemetry.raw.>"

	// NATSRawTelemetryPrefix is used by ingestion to build specific subjects (e.g., telemetry.raw.RADAR)
	NATSRawTelemetryPrefix = "telemetry.raw."

	// NATSFusedTracksWildcard is used by downstream consumers (e.g. PostGIS, WebSockets)
	NATSFusedTracksWildcard = "tracks.fused.>"

	// NATSFusedTracksPrefix is used by fusion to publish fused updates (e.g., tracks.fused.TARGET-ALPHA-01)
	NATSFusedTracksPrefix = "tracks.fused."
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
