module github.com/go-kafkify/grpc-service

go 1.21

require (
	github.com/google/uuid v1.5.0
	github.com/lib/pq v1.10.9
	github.com/prometheus/client_golang v1.18.0
	github.com/segmentio/kafka-go v0.4.47
	go.opentelemetry.io/otel v1.21.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.21.0
	go.opentelemetry.io/otel/sdk v1.21.0
	go.opentelemetry.io/otel/trace v1.21.0
	go.uber.org/zap v1.26.0
)

