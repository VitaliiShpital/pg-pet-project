package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/nats-io/nats.go"

	"pg-pet-project/pkg/telemetry"
)

func main() {
	nc, err := nats.Connect(telemetry.DefaultNATSURL)
	if err != nil {
		log.Fatalf("[INGESTION] NATS connect error: %v", err)
	}
	defer nc.Close()

	opts := mqtt.NewClientOptions().
		AddBroker(telemetry.DefaultMQTTURL).
		SetClientID("air_defense_ingestion")

	opts.OnConnect = func(client mqtt.Client) {
		log.Println("[INGESTION] Connected to MQTT broker")

		token := client.Subscribe(telemetry.MQTTTelemetryTopic, telemetry.DefaultTelemetryQoS, func(c mqtt.Client, m mqtt.Message) {
			var event telemetry.RawSensorEvent
			if err := json.Unmarshal(m.Payload(), &event); err != nil {
				log.Printf("[INGESTION] Error unmarshaling payload: %v", err)
				return
			}

			natsSubject := fmt.Sprintf("%s%s", telemetry.NATSRawTelemetryPrefix, event.SensorType)
			if err := nc.Publish(natsSubject, m.Payload()); err != nil {
				log.Printf("[INGESTION] Failed to publish to NATS: %v", err)
			}
		})

		if token.Wait() && token.Error() != nil {
			log.Fatalf("[INGESTION] MQTT subscribe error: %v", token.Error())
		}
		log.Printf("[INGESTION] Subscribed to MQTT topic: %s", telemetry.MQTTTelemetryTopic)
	}

	mqttClient := mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("[INGESTION] MQTT connect error: %v", token.Error())
	}

	waitForShutdown()
	mqttClient.Disconnect(250)
}

func waitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	log.Println("[INGESTION] Service stopped.")
}
