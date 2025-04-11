# Tempo Trace Generator and Query Tools

This directory contains tools for generating test traces in Tempo and querying them through the MCP server.

## Setup

1. **Start Tempo and Grafana:**
   ```
   docker-compose -f docker-compose.test.yml up -d
   ```

   This starts:
   - Tempo on port 3200 (HTTP API)
   - Tempo on port 4317 (OTLP gRPC receiver)
   - Grafana on port 3000

2. **Generate trace data:**
   ```
   ./insert-example-traces.sh
   ```

   By default, it generates 10 traces spread across several services.
   You can configure:
   - `NUM_TRACES=20 ./insert-example-traces.sh` (change trace count)
   - `SERVICE_NAMES="service1,service2" ./insert-example-traces.sh` (custom service names)

3. **Query traces using the MCP server:**
   ```
   ./run-client.sh tempo_query "{service.name=\"frontend\"}"
   ```

   Other useful queries:
   - `./run-client.sh tempo_query "{duration>100ms}"`
   - `./run-client.sh tempo_query "{service.name=\"backend\"}"`
   - `./run-client.sh tempo_query "{status.code=error}"`

## Trace Generator

The trace generator is a Go program that creates realistic distributed traces:

- Creates spans for multiple services (frontend, backend, database, etc.)
- Adds parent-child relationships between spans
- Includes errors in some spans (20% chance per child span)
- Simulates different operation types (compute, io, network, database, external)
- Sets appropriate attributes on spans
- Creates distributed traces that span multiple services

## Visual UI

You can access the Grafana UI at http://localhost:3000 to visually explore the traces.
Tempo should be pre-configured as a data source.

## Troubleshooting

1. **Cannot connect to Tempo:**
   The script will automatically run in Docker if Tempo is detected in Docker. Otherwise,
   it will try to connect to the host directly.

2. **No traces appear:**
   Check Tempo logs:
   ```
   docker-compose -f docker-compose.test.yml logs tempo
   ```

3. **MCP server error:**
   Make sure to build the MCP server: `go build -o tempo-mcp-server cmd/server/main.go` 