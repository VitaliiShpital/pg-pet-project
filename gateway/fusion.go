package main

import (
	"math"
	"sync"
	"time"
)

type FusedTrack struct {
	TrackID        string    `json:"track_id"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	EstimatedSpeed float64   `json:"estimated_speed_kmh"`
	HeadingDegrees float64   `json:"heading_degrees"`
	LastUpdated    time.Time `json:"last_updated"`
	UpdateCount    uint64    `json:"update_count"`
}

type SimpleTrackSmoother struct {
	mu          sync.Mutex
	lat         float64
	lng         float64
	speed       float64
	heading     float64
	lastTime    time.Time
	isInit      bool
	updateCount uint64
}

func NewTrackSmoother() *SimpleTrackSmoother {
	return &SimpleTrackSmoother{}
}

func (s *SimpleTrackSmoother) ProcessMeasurement(reading SensorTelemetry) FusedTrack {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Initial State
	if !s.isInit {
		s.lat = reading.Latitude
		s.lng = reading.Longitude
		s.speed = reading.SpeedKMH
		s.heading = reading.AzimuthDegrees
		s.lastTime = reading.DetectedAt
		s.isInit = true
		s.updateCount = 1

		return FusedTrack{
			TrackID:        "TARGET-ALPHA-01",
			Latitude:       s.lat,
			Longitude:      s.lng,
			EstimatedSpeed: s.speed,
			HeadingDegrees: s.heading,
			LastUpdated:    reading.DetectedAt,
			UpdateCount:    1,
		}
	}

	// 2. High-confidence sensors (Radar) pull position more; low-confidence (Optical) less
	weight := 0.1 + (reading.Confidence * 0.2) // Range: 0.15 to 0.30

	// 3. Smooth Latitude & Longitude
	s.lat = s.lat + weight*(reading.Latitude-s.lat)
	s.lng = s.lng + weight*(reading.Longitude-s.lng)

	// 4. Smooth Speed using reported sensor speed (prevents timing division bugs)
	s.speed = s.speed + 0.15*(reading.SpeedKMH-s.speed)

	// 5. Smooth Heading
	s.heading = s.heading + 0.15*(reading.AzimuthDegrees-s.heading)

	s.lastTime = reading.DetectedAt
	s.updateCount++

	return FusedTrack{
		TrackID:        "TARGET-ALPHA-01",
		Latitude:       s.lat,
		Longitude:      s.lng,
		EstimatedSpeed: math.Round(s.speed*10) / 10,
		HeadingDegrees: math.Round(s.heading*10) / 10,
		LastUpdated:    reading.DetectedAt,
		UpdateCount:    s.updateCount,
	}
}
