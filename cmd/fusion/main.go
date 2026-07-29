package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"

	"pg-pet-project/pkg/fusion"
	"pg-pet-project/pkg/telemetry"
)

func main() {
	nc, err := nats.Connect(telemetry.DefaultNATSURL)
	if err != nil {
		log.Fatalf("[FUSION] NATS connect error: %v", err)
	}
	defer nc.Close()

	smoother := fusion.NewTrackSmoother()

	_, err = nc.Subscribe(telemetry.NATSRawTelemetryWildcard, func(msg *nats.Msg) {
		var reading telemetry.RawSensorEvent
		if err := json.Unmarshal(msg.Data, &reading); err != nil {
			log.Printf("[FUSION] Error decoding event: %v", err)
			return
		}

		fusedTrack := smoother.ProcessMeasurement(reading)

		fmt.Printf("[TRACK #%-4d] Pos: (%.6f, %.6f) | Speed: %5.1f km/h | Heading: %5.1f° | Source: %-15s\n",
			fusedTrack.UpdateCount,
			fusedTrack.Latitude,
			fusedTrack.Longitude,
			fusedTrack.EstimatedSpeed,
			fusedTrack.HeadingDegrees,
			reading.SensorType,
		)

		fusedData, err := json.Marshal(fusedTrack)
		if err != nil {
			log.Printf("[FUSION] Failed to marshal fused track: %v", err)
			return
		}

		subject := fmt.Sprintf("%s%s", telemetry.NATSFusedTracksPrefix, fusedTrack.TrackID)
		if err := nc.Publish(subject, fusedData); err != nil {
			log.Printf("[FUSION] Failed to publish fused track to NATS: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("[FUSION] NATS subscribe error: %v", err)
	}

	log.Printf("[FUSION] Service listening on NATS %s", telemetry.NATSRawTelemetryWildcard)
	waitForShutdown()
}

func waitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	log.Println("[FUSION] Service stopped.")
}
