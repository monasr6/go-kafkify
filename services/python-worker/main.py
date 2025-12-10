#!/usr/bin/env python3
"""
Python Worker Service

Consumes events from Kafka topics and processes them.
Stores processed results in PostgreSQL database.
"""

import json
import logging
import os
import signal
import sys
import time
from datetime import datetime
from typing import Any, Dict

import psycopg2
from kafka import KafkaConsumer
from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.semconv.resource import ResourceAttributes
from prometheus_client import Counter, Histogram, start_http_server
from pythonjsonlogger import jsonlogger

# Metrics
messages_processed = Counter(
    "worker_messages_processed_total", "Total messages processed", ["topic", "status"]
)
processing_duration = Histogram(
    "worker_processing_duration_seconds", "Message processing duration", ["topic"]
)
db_operations = Counter(
    "worker_db_operations_total", "Total database operations", ["operation", "status"]
)

# Global variables
db_conn = None
running = True

# Setup structured logging
logHandler = logging.StreamHandler()
formatter = jsonlogger.JsonFormatter(
    "%(asctime)s %(name)s %(levelname)s %(message)s",
    rename_fields={"asctime": "timestamp", "levelname": "level"},
)
logHandler.setFormatter(formatter)
logger = logging.getLogger(__name__)
logger.addHandler(logHandler)
logger.setLevel(logging.INFO)


def init_telemetry():
    """Initialize OpenTelemetry tracing"""
    otel_endpoint = os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")

    resource = Resource(
        attributes={
            ResourceAttributes.SERVICE_NAME: "python-worker",
            ResourceAttributes.SERVICE_VERSION: "1.0.0",
        }
    )

    provider = TracerProvider(resource=resource)
    processor = BatchSpanProcessor(
        OTLPSpanExporter(endpoint=otel_endpoint, insecure=True)
    )
    provider.add_span_processor(processor)
    trace.set_tracer_provider(provider)

    logger.info("OpenTelemetry initialized", extra={"endpoint": otel_endpoint})
    return trace.get_tracer(__name__)


def init_database():
    """Initialize PostgreSQL database connection"""
    db_config = {
        "host": os.getenv("WORKER_DB_HOST", "localhost"),
        "port": os.getenv("WORKER_DB_PORT", "5432"),
        "user": os.getenv("WORKER_DB_USER", "postgres"),
        "password": os.getenv("WORKER_DB_PASSWORD", "postgres"),
        "database": os.getenv("WORKER_DB_NAME", "workerdb"),
    }

    max_retries = 30
    for attempt in range(max_retries):
        try:
            conn = psycopg2.connect(**db_config)
            conn.autocommit = False
            logger.info("Database connection established", extra=db_config)
            return conn
        except psycopg2.OperationalError:
            logger.info(f"Waiting for database... attempt {attempt + 1}/{max_retries}")
            time.sleep(2)

    logger.error("Failed to connect to database after maximum retries")
    sys.exit(1)


def init_kafka_consumer():
    """Initialize Kafka consumer"""
    kafka_brokers = os.getenv("KAFKA_BROKERS", "localhost:9092").split(",")
    consumer_group = os.getenv("KAFKA_CONSUMER_GROUP_WORKER", "python-worker-group")

    topics = [
        "task.completed",
        "resource.created",
        "resource.updated",
        "resource.deleted",
    ]

    consumer = KafkaConsumer(
        *topics,
        bootstrap_servers=kafka_brokers,
        group_id=consumer_group,
        auto_offset_reset="latest",
        enable_auto_commit=True,
        auto_commit_interval_ms=1000,
        value_deserializer=lambda m: json.loads(m.decode("utf-8")),
        key_deserializer=lambda m: m.decode("utf-8") if m else None,
    )

    logger.info(
        "Kafka consumer initialized",
        extra={"brokers": kafka_brokers, "group_id": consumer_group, "topics": topics},
    )

    return consumer


def process_message(tracer, topic: str, key: str, value: Dict[Any, Any]) -> bool:
    """Process a single Kafka message"""
    with tracer.start_as_current_span("process_message") as span:
        span.set_attribute("topic", topic)
        span.set_attribute("key", key)

        start_time = time.time()

        try:
            logger.info(
                "Processing message",
                extra={"topic": topic, "key": key, "payload": value},
            )

            # Process based on topic
            if topic == "task.completed":
                result = process_task_completed(value)
            elif topic.startswith("resource."):
                result = process_resource_event(topic, value)
            else:
                logger.warning(f"Unknown topic: {topic}")
                result = False

            duration = time.time() - start_time
            processing_duration.labels(topic=topic).observe(duration)

            status = "success" if result else "failed"
            messages_processed.labels(topic=topic, status=status).inc()

            logger.info(
                "Message processed",
                extra={
                    "topic": topic,
                    "key": key,
                    "status": status,
                    "duration_seconds": duration,
                },
            )

            return result

        except Exception as e:
            logger.error(
                "Error processing message",
                extra={"topic": topic, "key": key, "error": str(e)},
                exc_info=True,
            )
            messages_processed.labels(topic=topic, status="error").inc()
            return False


def process_task_completed(payload: Dict[Any, Any]) -> bool:
    """Process task completed events"""
    task_id = payload.get("task_id")
    resource_id = payload.get("resource_id")
    action = payload.get("action")

    if not all([task_id, resource_id, action]):
        logger.error("Missing required fields in task.completed payload", extra=payload)
        return False

    # Store in database
    cursor = db_conn.cursor()
    try:
        query = """
        INSERT INTO processed_events (id, event_type, resource_id, task_id, action, payload, processed_at)
        VALUES (gen_random_uuid(), %s, %s, %s, %s, %s, %s)
        """
        cursor.execute(
            query,
            (
                "task.completed",
                resource_id,
                task_id,
                action,
                json.dumps(payload),
                datetime.utcnow(),
            ),
        )
        db_conn.commit()
        db_operations.labels(operation="insert", status="success").inc()
        return True
    except Exception as e:
        db_conn.rollback()
        db_operations.labels(operation="insert", status="error").inc()
        logger.error(
            "Database error processing task.completed", extra={"error": str(e)}
        )
        return False
    finally:
        cursor.close()


def process_resource_event(topic: str, payload: Dict[Any, Any]) -> bool:
    """Process resource events (created, updated, deleted)"""
    resource_id = payload.get("id")

    if not resource_id:
        logger.error("Missing resource ID in payload", extra=payload)
        return False

    event_type = topic
    action = topic.split(".")[-1]  # Extract 'created', 'updated', or 'deleted'

    cursor = db_conn.cursor()
    try:
        query = """
        INSERT INTO processed_events (id, event_type, resource_id, action, payload, processed_at)
        VALUES (gen_random_uuid(), %s, %s, %s, %s, %s)
        """
        cursor.execute(
            query,
            (event_type, resource_id, action, json.dumps(payload), datetime.utcnow()),
        )
        db_conn.commit()
        db_operations.labels(operation="insert", status="success").inc()
        return True
    except Exception as e:
        db_conn.rollback()
        db_operations.labels(operation="insert", status="error").inc()
        logger.error(
            "Database error processing resource event", extra={"error": str(e)}
        )
        return False
    finally:
        cursor.close()


def signal_handler(signum, frame):
    """Handle shutdown signals gracefully"""
    global running
    logger.info(f"Received signal {signum}, shutting down...")
    running = False


def main():
    """Main worker loop"""
    global db_conn, running

    logger.info("Starting Python Worker Service")

    # Initialize components
    tracer = init_telemetry()
    db_conn = init_database()
    consumer = init_kafka_consumer()

    # Start Prometheus metrics server
    metrics_port = int(os.getenv("METRICS_PORT", "9092"))
    start_http_server(metrics_port)
    logger.info(f"Metrics server started on port {metrics_port}")

    # Setup signal handlers
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)

    # Main processing loop
    logger.info("Starting message processing loop")
    try:
        while running:
            # Poll for messages with timeout
            messages = consumer.poll(timeout_ms=1000)

            for topic_partition, records in messages.items():
                for record in records:
                    process_message(tracer, record.topic, record.key, record.value)

    except Exception as e:
        logger.error(
            "Unexpected error in main loop", extra={"error": str(e)}, exc_info=True
        )

    finally:
        logger.info("Cleaning up resources")
        consumer.close()
        if db_conn:
            db_conn.close()
        logger.info("Python Worker Service stopped")


if __name__ == "__main__":
    main()
