#!/bin/bash

# This script runs a Tempo query via the MCP server using a simple Go client

# Display usage if no arguments provided
if [ $# -lt 2 ]; then
  echo "Usage:"
  echo "  ./run-client.sh tempo_query \"<query>\""
  echo ""
  echo "Examples:"
  echo "  ./run-client.sh tempo_query \"{duration>1s}\""
  echo "  ./run-client.sh tempo_query \"{service.name=\\\"frontend\\\"}\""
  echo "  ./run-client.sh tempo_query \"{duration>500ms}\""
  exit 1
fi

# Check that the first argument is tempo_query
if [ "$1" != "tempo_query" ]; then
  echo "Error: First argument must be 'tempo_query'"
  exit 1
fi

# Build the server if it doesn't exist
if [ ! -f ./tempo-mcp-server ]; then
  echo "Building server..."
  go build -o tempo-mcp-server cmd/server/main.go
  if [ $? -ne 0 ]; then
    echo "Failed to build server"
    exit 1
  fi
fi

# Build the simple client
echo "Building simple client..."
go build -o simple-client cmd/simple-client/main.go
if [ $? -ne 0 ]; then
  echo "Failed to build simple client"
  exit 1
fi

echo "Sending query to Tempo MCP server: $2"

# Run the simple client with the provided arguments
./simple-client "$@"

# Check exit status
STATUS=$?
if [ $STATUS -ne 0 ]; then
  echo "Error: Command failed with status $STATUS"
  echo "Note: For the MCP server to work, make sure Tempo is running at the URL specified by TEMPO_URL environment variable (default: http://localhost:3200)"
  exit $STATUS
fi

echo "Query completed successfully." 