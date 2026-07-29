package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pg-pet-project/pkg/telemetry"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/nats-io/nats.go"
)

// SQL query to upsert fused tracks into tracked_targets
const upsertTrackedTargetSQL = `
	INSERT INTO tracked_targets (
		id,
		track_number,
		current_location,
		current_speed_kmh,
		current_heading_degrees,
		first_detected_at,
		last_updated_at
	) VALUES (
		$1,
		$2,
		ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography,
		$5,
		$6,
		$7,
		$7
	)
	ON CONFLICT (id) DO UPDATE SET
		current_location = EXCLUDED.current_location,
		current_speed_kmh = EXCLUDED.current_speed_kmh,
		current_heading_degrees = EXCLUDED.current_heading_degrees,
		last_updated_at = EXCLUDED.last_updated_at;
`

func main() {
	// 1. Connect to PostGIS Database
	db, err := sql.Open("postgres", telemetry.DefaultPostgresDSN)
	if err != nil {
		log.Fatalf("[STORAGE] Failed to open DB connection: %v", err)
	}
	defer safeClose("Postgres DB connection", db.Close)

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatalf("[STORAGE] Failed to ping PostGIS database: %v", err)
	}
	log.Println("[STORAGE] Connected to PostGIS database")

	// 2. Connect to NATS
	nc, err := nats.Connect(telemetry.DefaultNATSURL)
	if err != nil {
		log.Fatalf("[STORAGE] NATS connect error: %v", err)
	}
	defer nc.Close()

	// 3. Prepare SQL Statement
	stmt, err := db.Prepare(upsertTrackedTargetSQL)
	if err != nil {
		log.Fatalf("[STORAGE] Failed to prepare insert query: %v", err)
	}
	defer safeClose("Prepared SQL statement", stmt.Close)

	// 4. Subscribe to Fused Tracks on NATS
	_, err = nc.Subscribe(telemetry.NATSFusedTracksWildcard, func(msg *nats.Msg) {
		var track telemetry.FusedTrack
		if err := json.Unmarshal(msg.Data, &track); err != nil {
			log.Printf("[STORAGE] Error unmarshaling fused track: %v", err)
			return
		}

		// Execute Upsert Query
		// ST_MakePoint order: (Longitude, Latitude)
		_, err := stmt.Exec(
			stringToUUID(track.TrackID),
			track.UpdateCount, // Mapping track update counter or ID to track_number
			track.Longitude,
			track.Latitude,
			track.EstimatedSpeed,
			track.HeadingDegrees,
			track.LastUpdatedAt,
		)
		if err != nil {
			log.Printf("[STORAGE] Failed to upsert track %s into PostGIS: %v", track.TrackID, err)
			return
		}

		log.Printf("[STORAGE] Saved track #%d (%s) at (%.6f, %.6f)",
			track.UpdateCount, track.TrackID, track.Latitude, track.Longitude)
	})
	if err != nil {
		log.Fatalf("[STORAGE] Failed to subscribe to NATS subject: %v", err)
	}

	log.Printf("[STORAGE] Storage worker listening on NATS subject: %s", telemetry.NATSFusedTracksWildcard)
	waitForShutdown()
}

// Generate a deterministic UUID from string (e.g. "TARGET-ALPHA-01")
func stringToUUID(input string) string {
	// Uses MD5-based deterministic UUID v3 under a fixed namespace
	return uuid.NewMD5(uuid.NameSpaceDNS, []byte(input)).String()
}

func safeClose(resource string, closeFn func() error) {
	if err := closeFn(); err != nil {
		log.Printf("[STORAGE] Error closing %s: %v", resource, err)
	}
}

func waitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	log.Println("[STORAGE] Service shutting down...")
}
