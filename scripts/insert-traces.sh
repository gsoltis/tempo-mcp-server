#!/bin/bash

# Simple script to insert example traces into an existing Tempo instance

# Default settings - use the service name 'tempo' from docker-compose for Docker Compose networking
TEMPO_HOST=${TEMPO_HOST:-"tempo"}
TEMPO_PORT=${TEMPO_PORT:-"4317"}
NUM_TRACES=${NUM_TRACES:-10}
SERVICE_NAMES=${SERVICE_NAMES:-"frontend,backend,database,auth-service,payment-service"}
TIMEOUT_DURATION=${TIMEOUT_DURATION:-30}

echo "Inserting $NUM_TRACES traces to Tempo at $TEMPO_HOST:$TEMPO_PORT"
echo "Services: $SERVICE_NAMES"

# Check Docker networking mode
DOCKER_COMPOSE_NETWORK="tempo-mcp-server_default"
IN_DOCKER=false
if [ -f /.dockerenv ] || [ -f /proc/self/cgroup ] && grep -q docker /proc/self/cgroup; then
  IN_DOCKER=true
  echo "Running inside Docker container"
fi

# Check Tempo connection first
echo "Checking connection to Tempo..."
if nc -z -w 2 $TEMPO_HOST $TEMPO_PORT; then
  echo "✅ Successfully connected to Tempo at $TEMPO_HOST:$TEMPO_PORT"
else
  echo "⚠️  WARNING: Cannot connect to Tempo at $TEMPO_HOST:$TEMPO_PORT"
  
  if [ "$IN_DOCKER" = true ]; then
    echo "Since you're running inside Docker, use the service name from docker-compose:"
    echo "  TEMPO_HOST=tempo ./insert-traces.sh"
  else 
    # Not in Docker
    if [ "$TEMPO_HOST" = "tempo" ]; then
      echo "You're trying to connect to 'tempo' from outside Docker."
      echo "If running from your Mac, try: TEMPO_HOST=localhost ./insert-traces.sh"
    elif [ "$TEMPO_HOST" = "localhost" ]; then
      echo "Make sure Tempo is running locally on port $TEMPO_PORT"
      echo "Verify ports are exposed in docker-compose.yml: '4317:4317'"
    elif [ "$TEMPO_HOST" = "host.docker.internal" ]; then
      echo "Docker networking issue detected. Try these options:"
      echo "1. Use port forwarding: TEMPO_HOST=localhost TEMPO_PORT=4317 ./insert-traces.sh"
      echo "2. Make sure you're in the $DOCKER_COMPOSE_NETWORK network"
    fi
  fi
  
  echo "Continuing anyway..."
fi

# Build the trace generator if needed
if [ ! -f ./trace-generator ] || [ cmd/trace-generator/main.go -nt ./trace-generator ]; then
  echo "Building trace generator..."
  go build -o trace-generator cmd/trace-generator/main.go
fi

# Run the trace generator with cross-platform timeout
echo "Generating traces (timeout: ${TIMEOUT_DURATION}s)..."
if command -v timeout >/dev/null 2>&1; then
  # Linux has timeout command
  timeout ${TIMEOUT_DURATION}s env \
  TEMPO_HOST="$TEMPO_HOST" \
  TEMPO_PORT="$TEMPO_PORT" \
  NUM_TRACES="$NUM_TRACES" \
  SERVICE_NAMES="$SERVICE_NAMES" \
  ./trace-generator
elif command -v gtimeout >/dev/null 2>&1; then
  # macOS with homebrew coreutils installed
  gtimeout ${TIMEOUT_DURATION}s env \
  TEMPO_HOST="$TEMPO_HOST" \
  TEMPO_PORT="$TEMPO_PORT" \
  NUM_TRACES="$NUM_TRACES" \
  SERVICE_NAMES="$SERVICE_NAMES" \
  ./trace-generator
else
  # No timeout command available - run without timeout
  env \
  TEMPO_HOST="$TEMPO_HOST" \
  TEMPO_PORT="$TEMPO_PORT" \
  NUM_TRACES="$NUM_TRACES" \
  SERVICE_NAMES="$SERVICE_NAMES" \
  ./trace-generator
fi

# Generate query URL based on TEMPO_HOST
if [ "$IN_DOCKER" = true ]; then
  # Inside Docker, use service name from docker-compose
  if [ "$TEMPO_HOST" = "tempo" ]; then
    TEMPO_QUERY_URL="http://tempo:3200"
  else
    TEMPO_QUERY_URL="http://$TEMPO_HOST:3200"
  fi
else
  # Outside Docker on host machine
  if [ "$TEMPO_HOST" = "localhost" ] || [ "$TEMPO_HOST" = "tempo" ] || [ "$TEMPO_HOST" = "host.docker.internal" ]; then
    TEMPO_QUERY_URL="http://localhost:3200"
  else
    TEMPO_QUERY_URL="http://$TEMPO_HOST:3200"
  fi
fi

echo "Done! Query examples (set TEMPO_URL=$TEMPO_QUERY_URL in your environment):"
echo "  ./run-client.sh tempo_query \"{service.name=\\\"frontend\\\"}\""
echo "  ./run-client.sh tempo_query \"{duration>100ms}\""
echo "  ./run-client.sh tempo_query \"{status.code=error}\""
echo ""
echo "Note: Make sure your MCP server has TEMPO_URL=$TEMPO_QUERY_URL in its environment."
echo "      Tempo uses port 4317 for sending traces (gRPC) but port 3200 for queries (HTTP)." 