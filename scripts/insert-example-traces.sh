#!/bin/bash

# This script inserts example traces into Tempo using a Go-based trace generator

echo "Creating example traces for Tempo using Go..."

# Check if Docker is running and Tempo container exists
USING_DOCKER=false
NETWORK_NAME=""

if command -v docker &> /dev/null; then
  if docker ps | grep -q tempo; then
    USING_DOCKER=true
    echo "Tempo container detected in Docker"
    
    # Get the Docker network that Tempo is on
    NETWORK_NAME=$(docker inspect -f '{{range $k, $v := .NetworkSettings.Networks}}{{$k}}{{end}}' $(docker ps -q --filter name=tempo))
    if [ -n "$NETWORK_NAME" ]; then
      echo "Found Tempo on Docker network: $NETWORK_NAME"
    fi
  fi
fi

# Set default values based on environment
if [ "$USING_DOCKER" = "true" ]; then
  # Using Docker, access Tempo via the container name
  DEFAULT_TEMPO_HOST="tempo"
  DEFAULT_TEMPO_PORT="4317"
else
  # Not using Docker, connect to localhost
  DEFAULT_TEMPO_HOST="localhost"
  DEFAULT_TEMPO_PORT="4317"
fi

# Set up variables
TEMPO_HOST=${TEMPO_HOST:-$DEFAULT_TEMPO_HOST}
TEMPO_PORT=${TEMPO_PORT:-$DEFAULT_TEMPO_PORT}  # OTLP gRPC port (4317)
NUM_TRACES=${NUM_TRACES:-10}
SERVICE_NAMES=${SERVICE_NAMES:-"frontend,backend,database,auth-service,payment-service"}

echo "Using Tempo at $TEMPO_HOST:$TEMPO_PORT (OTLP gRPC endpoint)"
echo "API endpoint: http://$TEMPO_HOST:3200"
echo "Will generate $NUM_TRACES traces for services: $SERVICE_NAMES"

# Check if we need to add OpenTelemetry dependencies to go.mod
if ! grep -q "go.opentelemetry.io/otel" go.mod 2>/dev/null; then
  echo "Adding OpenTelemetry dependencies to go.mod..."
  
  # Add required dependencies
  go get go.opentelemetry.io/otel
  go get go.opentelemetry.io/otel/attribute
  go get go.opentelemetry.io/otel/codes
  go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc
  go get go.opentelemetry.io/otel/sdk/resource
  go get go.opentelemetry.io/otel/sdk/trace
  go get go.opentelemetry.io/otel/trace
  go get google.golang.org/grpc
  
  echo "Dependencies added"
fi

# Build the trace generator
echo "Building trace generator..."
go build -o trace-generator cmd/trace-generator/main.go

# Run the trace generator
echo "Running trace generator..."

if [ "$USING_DOCKER" = "true" ] && [ -n "$NETWORK_NAME" ]; then
  # Run in Docker on the same network as Tempo
  echo "Running trace generator in Docker container to connect to Tempo container..."
  
  # Create a temporary Dockerfile for the trace generator
  TEMP_DIR=$(mktemp -d)
  cat > "$TEMP_DIR/Dockerfile" << EOF
FROM golang:1.19

WORKDIR /app
COPY trace-generator /app/
ENTRYPOINT ["/app/trace-generator"]
EOF

  # Copy the trace generator binary to the temporary directory
  cp trace-generator "$TEMP_DIR/"
  
  # Build the Docker image
  TRACE_GENERATOR_IMAGE="tempo-trace-generator:latest"
  docker build -t "$TRACE_GENERATOR_IMAGE" "$TEMP_DIR"
  
  # Run the trace generator in Docker
  docker run --rm \
    --network "$NETWORK_NAME" \
    -e TEMPO_HOST="$TEMPO_HOST" \
    -e TEMPO_PORT="$TEMPO_PORT" \
    -e NUM_TRACES="$NUM_TRACES" \
    -e SERVICE_NAMES="$SERVICE_NAMES" \
    "$TRACE_GENERATOR_IMAGE"
    
  # Clean up
  rm -rf "$TEMP_DIR"
else
  # Run locally
  echo "Running trace generator locally..."
  TEMPO_HOST="$TEMPO_HOST" \
  TEMPO_PORT="$TEMPO_PORT" \
  NUM_TRACES="$NUM_TRACES" \
  SERVICE_NAMES="$SERVICE_NAMES" \
  ./trace-generator
fi

echo "Trace generation completed"
echo "You can query these traces in Tempo using the following queries:"
echo "  {service.name=\"frontend\"}"
echo "  {service.name=\"backend\"}"
echo "  {service.name=\"database\"}"
echo "  {duration>100ms}"
echo "  {status.code=error}"
echo ""
echo "Use ./run-client.sh to query these traces. For example:"
echo "  ./run-client.sh tempo_query \"{service.name=\\\"frontend\\\"}\"" 