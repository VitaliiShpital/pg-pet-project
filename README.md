# 🛰️ Air Defense Multi-Source Sensor Fusion Platform

A distributed, real-time event-driven telemetry ingestion and multi-sensor target tracking platform built with **Go**, **NATS JetStream**, **MQTT (Mosquitto)**, **PostGIS**, and **Docker**.

The platform ingests raw sensor observations (Radar, Acoustic, Optical, SIGINT, Mobile Observers) from edge brokers, smooths positions and trajectory tracks using real-time data fusion, and persists spatial trajectories in PostGIS.

---

## 🏗️ System Architecture

```text
  [ Python Edge Simulator ]
             │
        (MQTT: QoS 0)
             ▼
┌─────────────────────────┐
│     cmd/ingestion       │  ── Edge Ingestion Engine (MQTT -> NATS translator)
└─────────────────────────┘
             │
       (NATS: telemetry.raw.<type>)
             ▼
┌─────────────────────────┐
│       cmd/fusion        │  ── Track Smoothing & Multi-Sensor Fusion Service
└─────────────────────────┘
             │
       (NATS: tracks.fused.<track_id>)
             │
             ├───> [ cmd/storage ] ───────> [ PostGIS: `tracked_targets` ]
             │
             └───> [ WebSocket Gateway ] ──> [ Live Interactive Map UI ] (Planned)
```

---

## 📦 Key Components

| Component / Service | Description | Tech Stack |
| :--- | :--- | :--- |
| `cmd/ingestion` | Consumes raw MQTT edge observations and routes normalized events to NATS subjects. | Go, MQTT (Paho), NATS |
| `cmd/fusion` | Consumes raw detections, runs tracking algorithms/smoothing, and publishes fused state tracks. | Go, NATS |
| `cmd/storage` | Subscribes to fused target updates and upserts geographic trajectories into PostGIS with spatial indexing. | Go, PostGIS (`lib/pq`), Google UUID |
| `pkg/telemetry` | Shared data models, messaging subject constants, database query definitions, and environment bindings. | Go |
| `simulator/` | High-frequency edge sensor telemetry generator simulating multi-source target detections. | Python |

---

## 🛠️ Prerequisites

* [Docker](https://www.docker.com/) & [Docker Compose](https://docs.docker.com/compose/)
* [Go 1.25+](https://go.dev/) (for local native development)
* [Python 3.10+](https://www.python.org/) (for running the standalone simulator)

---

## 🚀 Quick Start (Docker Compose)

Spin up the infrastructure (PostGIS, Mosquitto, NATS) and all microservices in containers with a single command:

```bash
# 1. Clone the repository
git clone https://github.com/your-username/pg-pet-project.git
cd pg-pet-project

# 2. Build and start all services
docker compose up --build
```

---

## 💻 Local Development Setup

If you prefer running Go microservices natively for debugging while hosting brokers in Docker:

### Step 1: Start Infrastructure Containers
```bash
docker compose up -d postgres mosquitto nats
```

### Step 2: Launch Go Microservices (In Separate Terminals)

```bash
# Terminal 1: Storage Worker
go run ./cmd/storage

# Terminal 2: Fusion Engine
go run ./cmd/fusion

# Terminal 3: MQTT Ingestion
go run ./cmd/ingestion
```

### Step 3: Run the Telemetry Simulator
```bash
cd simulator
python -m venv venv && source venv/bin/activate
pip install -r requirements.txt
python main.py
```

---

## 🗺️ Database Schema & PostGIS Spatial Queries

Database migrations run automatically on startup (`docker/migrations/000001_init_air_defense_schema.up.sql`).

### Core Spatial Tables:
* `raw_sensor_detections`: Raw observation points (`GEOGRAPHY(Point, 4326)`).
* `tracked_targets`: Upserted fused tracks with target classification and spatial geometry (`GEOGRAPHY(Point, 4326)`).
* `air_defense_zones`: Geofencing polygons (`GEOGRAPHY(Polygon, 4326)`).

### Verify Live Targets in PostGIS:
To query active tracked targets in real-time from your terminal:

```bash
docker exec -it air_defense_postgres psql -U postgres -d air_defense -c "
SELECT 
    id, 
    track_number, 
    ST_AsText(current_location::geometry) AS position_wkt, 
    current_speed_kmh, 
    current_heading_degrees, 
    last_updated_at 
FROM tracked_targets;
"
```

---

## 🔧 Environment Variables

Environment variables allow seamless switching between local execution and containerized networking:

| Variable | Default (Local) | Docker Compose Value |
| :--- | :--- | :--- |
| `NATS_URL` | `nats://localhost:4222` | `nats://nats:4222` |
| `MQTT_URL` | `tcp://localhost:1883` | `tcp://mosquitto:1883` |
| `POSTGRES_DSN` | `postgres://postgres:postgres_password@localhost:5432/air_defense?sslmode=disable` | `postgres://postgres:postgres_password@postgres:5432/air_defense?sslmode=disable` |

---

## 🤝 Project Structure

```text
├── cmd/
│   ├── fusion/         # Fusion engine entrypoint
│   ├── ingestion/      # MQTT-to-NATS edge ingestion entrypoint
│   └── storage/        # PostGIS storage worker entrypoint
├── docker/
│   ├── migrations/     # PostGIS SQL initial schema migrations
│   ├── mosquitto/      # Mosquitto broker configuration
│   └── Dockerfile      # Multi-stage Go production builder
├── pkg/
│   ├── fusion/         # Track smoothing and estimation algorithms
│   └── telemetry/      # Shared models, DTOs, and constants
├── simulator/          # Multi-sensor Python telemetry generator
├── .dockerignore
├── docker-compose.yaml
├── go.mod
└── go.sum
```
