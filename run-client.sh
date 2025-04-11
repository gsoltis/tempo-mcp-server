#!/bin/bash

# This script runs the tempo-mcp-client and pipes its output to curl, which sends it to the MCP server

# Build the client if it doesn't exist
if [ ! -f ./tempo-mcp-client ]; then
  go build -o tempo-mcp-client cmd/client/main.go
fi

# Get the MCP server URL from environment or use default
MCP_SERVER=${MCP_SERVER:-"http://localhost:9090/v1/call-tool"}

# Run the client and pipe to curl
./tempo-mcp-client "$@" | curl -s -X POST -H "Content-Type: application/json" -d @- "$MCP_SERVER" 