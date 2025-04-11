#!/bin/bash

# This script runs a Tempo query via the MCP server using a simplified approach

# Display usage if no arguments provided
if [ $# -lt 2 ]; then
  echo "Usage:"
  echo "  ./run-client.sh tempo_query \"<query>\""
  echo ""
  echo "Examples:"
  echo "  ./run-client.sh tempo_query \"{duration>1s}\""
  echo "  ./run-client.sh tempo_query \"{resource.service.name=\\\"example-service\\\"}\""
  echo "  ./run-client.sh tempo_query \"{duration>500ms}\""
  exit 1
fi

# Check that the first argument is tempo_query
if [ "$1" != "tempo_query" ]; then
  echo "Error: First argument must be 'tempo_query'"
  exit 1
fi

# Build the original client if it doesn't exist
if [ ! -f ./tempo-mcp-client ]; then
  echo "Building client..."
  go build -o tempo-mcp-client cmd/client/main.go
  if [ $? -ne 0 ]; then
    echo "Failed to build client"
    exit 1
  fi
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

echo "Starting with query: ${2}"

# Set environment variables
export TEMPO_URL="http://localhost:3200"

# Extract parameters
QUERY="$2"
START="-5m"
END="now"
LIMIT=20

# If additional arguments are provided, use them
if [ $# -ge 3 ]; then
  START="$3"
fi

if [ $# -ge 4 ]; then
  END="$4"
fi

if [ $# -ge 5 ]; then
  LIMIT="$5"
fi

# Create the request JSON directly
REQUEST="{\"id\":\"client-1\",\"method\":\"tools/call\",\"params\":{\"name\":\"tempo_query\",\"arguments\":{\"query\":\"$QUERY\",\"start\":\"$START\",\"end\":\"$END\",\"limit\":$LIMIT}},\"jsonrpc\":\"2.0\"}"

# Create temp files
REQUEST_FILE=$(mktemp)
RESPONSE_FILE=$(mktemp)

# Clean up function to remove temp files on exit
cleanup() {
  rm -f "$REQUEST_FILE" "$RESPONSE_FILE"
}
trap cleanup EXIT

# Write the request to a file
echo "$REQUEST" > "$REQUEST_FILE"

# Process with server
echo "Sending request to server..."
cat "$REQUEST_FILE" | ./tempo-mcp-server > "$RESPONSE_FILE"

# Display the response in a readable format
echo "Processing response..."
RESPONSE_JSON=$(cat "$RESPONSE_FILE")

# Check if the response contains an error
if echo "$RESPONSE_JSON" | grep -q "error"; then
  ERROR_MSG=$(echo "$RESPONSE_JSON" | grep -o '"message":"[^"]*"' | cut -d'"' -f4)
  echo "Error: $ERROR_MSG"
  exit 1
fi

# Otherwise, pretty print the result
echo "$RESPONSE_JSON" | grep -o '"text":"[^"]*"' | cut -d'"' -f4

echo "Query completed successfully." 