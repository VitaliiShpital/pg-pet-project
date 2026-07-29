"""
Air defense multi-source sensor simulator.
It simulates 5 different sensors (radar, acoustic, optical, observer, SIGINT)
tracking an incoming target towards Kyiv over MQTT.
"""
import asyncio
import json
import random
from datetime import datetime, timezone
from time import time
from typing import Dict, Any, Tuple

import paho.mqtt.client as mqtt
from paho.mqtt.client import Client

# MQTT broker conf
MQTT_HOST: str = "localhost"
MQTT_PORT: int = 1883
MQTT_KEEPALIVE_SECONDS: int = 60

# MQTT Quality of Service (QoS) level:
# 0 = at most once (fire-and-forget, risk of data loss)
# 1 = at least once (guaranteed delivery)
# 2 = exactly once (4-way handshake, higher latency overhead)
MQTT_QOS_AT_LEAST_ONCE: int = 1

# Target initial position (moving South-West towards Kyiv)
TARGET_START_LAT: float = 51.5000
TARGET_START_LNG: float = 32.5000
TARGET_SPEED_KMH: float = 180.0    # typical shahed cruise speed (probably :))
TARGET_HEADING_DEG: float = 225.0  # heading South-West (225d)
TARGET_BASE_ALTITUDE_METERS: float = 450.0

# 1 degree of Latitude on Earth is approximately 111.13 km.
# calculation: 1 / 111.13 km ~= 0.00900009 degrees per km.
# example: moving 1 km North/South changes Latitude by ~0.009
KM_TO_DEGREE_FACTOR: float = 0.009
# Degrees of Longitude shrink as you move away from the Equator towards the poles.
# at Kyiv's latitude (~50.45d N), cos(50.45d) ~= 0.6367.
# example: 1 degree of Longitude at Kyiv's latitude is ~70.7 km instead of 111 km.
KYIV_LATITUDE_COSINE_SCALING: float = 0.6367

# Heading 225° lies at a 45° angle relative to West and South.
# sin(45°) = cos(45°) = √2 / 2 ~= 0.7071
# example: if target travels 100 meters diagonally (SW) it moves 70.7m West and 70.7m South.
SOUTH_WEST_VECTOR_COMPONENT: float = 0.7071

# Simulated altitude sensor error (+-10m)
ALTITUDE_JITTER_METERS: float = 10.0
# Simulated speed calculation error (+-5 km/h)
SPEED_JITTER_KMH: float = 5.0

SECONDS_PER_HOUR: float = 3600.0

# heterogeneous sensors setup
SENSORS: list[Dict[str, Any]] = [
    {
        "id": "radar-north-01",
        "type": "RADAR",
        "freq_hz": 2.0,       # reports 2 times per second (high frequency)
        "noise_meters": 5.0,  # highly accurate position
        "confidence": 0.95
    },
    {
        "id": "acoustic-post-402",
        "type": "ACOUSTIC",
        "freq_hz": 0.5,        # reports once every 2 seconds
        "noise_meters": 150.0, # less accurate spatial position
        "confidence": 0.60
    },
    {
        "id": "thermal-cam-07",
        "type": "OPTICAL",
        "freq_hz": 1.0,       # reports 1 time per second
        "noise_meters": 20.0, # good accuracy in visual range
        "confidence": 0.85
    },
    {
        "id": "mobile-fire-group-12",
        "type": "VISUAL_OBSERVER",
        "freq_hz": 0.2,        # human report every 5 seconds
        "noise_meters": 300.0, # high positional error
        "confidence": 0.50
    },
    {
        "id": "sigint-array-03",
        "type": "SIGINT",
        "freq_hz": 1.0,
        "noise_meters": 50.0,
        "confidence": 0.80
    }
]

def add_spatial_noise(lat: float, lng: float, noise_meters: float) -> Tuple[float, float]:
    """
    Adds some random offsets to simulate real-world sensor inaccuracy.
    1 degree lat = ~111,130m
    """
    degree_lat: float = 111130.0
    lat_offset: float = (random.uniform(-noise_meters, noise_meters)) / degree_lat
    lng_offset: float = (random.uniform(-noise_meters, noise_meters)) / (degree_lat * KYIV_LATITUDE_COSINE_SCALING)

    return round(lat + lat_offset, 6), round(lng + lng_offset, 6)

async def run_sensor(client: mqtt.Client, sensor: Dict[str, Any]) -> None:
    """Simulates a single sensor publishing some telemetry with its own interval."""

    sensor_id: str = str(sensor["id"])
    sensor_type: str = str(sensor["type"])
    freq_hz: float = float(sensor["freq_hz"])
    noise_meters: float = float(sensor["noise_meters"])
    confidence: float = float(sensor["confidence"])

    print(f"Started sensor loop: {sensor_id} ({sensor_type}) @ {freq_hz} Hz")

    start_time: float = time()

    while True:
        elapsed_seconds = time() - start_time

        distance_moved_km: float = (TARGET_SPEED_KMH / SECONDS_PER_HOUR) * elapsed_seconds
        distance_moved_degrees: float = distance_moved_km * KM_TO_DEGREE_FACTOR

        # calculate true trajectory coordinates moving South-West (225° Heading):
        # Latitude decreases going South (-), Longitude decreases going West (-)
        true_lat: float = TARGET_START_LAT - (distance_moved_degrees * SOUTH_WEST_VECTOR_COMPONENT)
        true_lng: float = TARGET_START_LNG - (distance_moved_degrees * SOUTH_WEST_VECTOR_COMPONENT)

        noisy_lat, noisy_lng = add_spatial_noise(true_lat, true_lng, noise_meters)

        payload: Dict[str, Any] = {
            "sensor_id": sensor_id,
            "sensor_type": sensor_type,
            "latitude": noisy_lat,
            "longitude": noisy_lng,
            "altitude_meters": round(TARGET_BASE_ALTITUDE_METERS + random.uniform(-ALTITUDE_JITTER_METERS, ALTITUDE_JITTER_METERS), 1),
            "speed_kmh": round(TARGET_SPEED_KMH + random.uniform(-SPEED_JITTER_KMH, SPEED_JITTER_KMH), 1),
            "azimuth_degrees": TARGET_HEADING_DEG,
            "confidence": confidence,
            "detected_at": datetime.now(timezone.utc).isoformat()
        }

        topic: str = f"telemetry/{sensor_type.lower()}/{sensor_id}"

        client.publish(topic, json.dumps(payload), qos=MQTT_QOS_AT_LEAST_ONCE)
        print(f"[{sensor_type}] -> Topic: {topic} | Position: ({noisy_lat}, {noisy_lng})")

        await asyncio.sleep(1.0 / freq_hz)

async def main() -> None:
    client: mqtt.Client = mqtt.Client()
    client.connect(MQTT_HOST, MQTT_PORT, keepalive=MQTT_KEEPALIVE_SECONDS)
    client.loop_start()

    print(f"Successfully connected to Broker at {MQTT_HOST}:{MQTT_PORT}")

    tasks = [run_sensor(client, sensor) for sensor in SENSORS]
    await asyncio.gather(*tasks)

if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print("\nSensor Simulator shut down successfully.")
