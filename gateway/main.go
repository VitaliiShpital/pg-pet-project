package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MQTT config
const (
	MQTTBrokerHost = "tcp://localhost:1883"
	MQTTClientID   = "air-defense-ingestion-gateway"
	MQTTTopic      = "telemetry/#" // captures all sensor sub-topics
	MQTTQoS        = 1
)

// SensorTelemetry represents the incoming telemetry JSON schema.
type SensorTelemetry struct {
	SensorID       string    `json:"sensor_id"`
	SensorType     string    `json:"sensor_type"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	AltitudeMeters float64   `json:"altitude_meters"`
	SpeedKMH       float64   `json:"speed_kmh"`
	AzimuthDegrees float64   `json:"azimuth_degrees"`
	Confidence     float64   `json:"confidence"`
	DetectedAt     time.Time `json:"detected_at"`

	// Ingestion metadata added by this gateway.
	IngestedAt time.Time `json:"ingested_at"`
}

// OnMessageReceived handles incoming MQTT messages concurrently.
func OnMessageReceived(client mqtt.Client, msg mqtt.Message) {
	var telemetry SensorTelemetry
	if err := json.Unmarshal(msg.Payload(), &telemetry); err != nil {
		log.Printf("[ERROR] Failed to unmarshal message from topic %s: %v", msg.Topic(), err)
		return
	}

	telemetry.IngestedAt = time.Now().UTC()
	latencyMs := telemetry.IngestedAt.Sub(telemetry.DetectedAt).Milliseconds()

	fmt.Printf("[%s] Sensor: %-22s | Type: %-15s | Pos: (%.6f, %.6f) | Alt: %.1fm | Latency: %d ms\n",
		telemetry.IngestedAt.Format("15:04:05.000"),
		telemetry.SensorID,
		telemetry.SensorType,
		telemetry.Latitude,
		telemetry.Longitude,
		telemetry.AltitudeMeters,
		latencyMs,
	)
}

func main() {
	log.Println("Initializing ingestion gateway...")

	opts := mqtt.NewClientOptions()
	opts.AddBroker(MQTTBrokerHost)
	opts.SetClientID(MQTTClientID)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(2 * time.Second)

	opts.SetDefaultPublishHandler(OnMessageReceived)
	opts.OnConnect = func(c mqtt.Client) {
		log.Println("Successfully connected to MQTT broker.")

		token := c.Subscribe(MQTTTopic, MQTTQoS, OnMessageReceived)
		if token.Wait() && token.Error() != nil {
			log.Fatalf("[FATAL] Failed to subscribe to topic %s: %v", MQTTTopic, token.Error())
		}
		log.Printf("Subscribed to MQTT topic pattern: '%s'", MQTTTopic)
	}

	opts.OnConnectionLost = func(c mqtt.Client, err error) {
		log.Printf("[WARN] Connection lost to MQTT Broker: %v. Reconnecting...", err)
	}

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("[FATAL] Connection failed: %v", token.Error())
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Println("\nShutting down gateway gracefully...")
	client.Disconnect(250)
	log.Println("Gateway stopped.")
}
