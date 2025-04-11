#!/bin/bash

# This script runs a Tempo query via the MCP server

# Display usage if no arguments provided
if [ $# -lt 2 ]; then
  echo "Usage:"
  echo "  ./run-client.sh tempo_query \"<query>\""
  echo ""
  echo "Examples:"
  echo "  ./run-client.sh tempo_query \"{duration>100ms}\""
  echo "  ./run-client.sh tempo_query \"{service.name=\\\"frontend\\\"}\""
  echo "  ./run-client.sh tempo_query \"{service.name=\\\"backend\\\"}\""
  echo "  ./run-client.sh tempo_query \"{status.code=error}\""
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

echo "Starting query with: ${2}"

# Set environment variables
export TEMPO_URL="${TEMPO_URL:-http://localhost:3200}"
echo "Using Tempo URL: $TEMPO_URL"

# Extract query parameter
QUERY="$2"

# Create a request with proper escaping
REQUEST=$(cat <<EOF
{
  "id": "client-request",
  "method": "tools/call",
  "params": {
    "name": "tempo_query",
    "arguments": {
      "query": "$QUERY",
      "start": "-15m",
      "end": "now",
      "limit": 50
    }
  },
  "jsonrpc": "2.0"
}
EOF
)

# Create temp files
REQUEST_FILE=$(mktemp)
RESPONSE_FILE=$(mktemp)

# Clean up function
cleanup() {
  rm -f "$REQUEST_FILE" "$RESPONSE_FILE"
}
trap cleanup EXIT

# Write the request to a file
echo "$REQUEST" > "$REQUEST_FILE"

# Process with server
echo "Sending request to server..."
cat "$REQUEST_FILE" | ./tempo-mcp-server > "$RESPONSE_FILE"

# Display the response
echo "Response from server:"
RESPONSE=$(cat "$RESPONSE_FILE")

# Check for errors
if echo "$RESPONSE" | grep -q '"error":'; then
  ERROR_MSG=$(echo "$RESPONSE" | grep -o '"message":"[^"]*"' | head -1 | cut -d'"' -f4)
  echo "Error: $ERROR_MSG"
  exit 1
fi

# Extract and display the text content from the response
TEXT_CONTENT=$(echo "$RESPONSE" | grep -o '"text":"[^"]*"' | sed 's/"text":"//g' | sed 's/"//g')
if [ -n "$TEXT_CONTENT" ]; then
  echo "$TEXT_CONTENT"
else
  # If no text found, show raw response
  echo "Raw response (could not parse text content):"
  echo "$RESPONSE"
fi

echo "Query completed." 