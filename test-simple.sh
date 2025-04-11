#!/bin/bash

# Test client-server interaction directly without using run-client.sh

# Build the client and server
go build -o tempo-mcp-client cmd/client/main.go
go build -o tempo-mcp-server cmd/server/main.go

echo "Testing with a simple query: {duration>1s}"

# Create a request JSON manually - use single line format
REQUEST='{"id":"test-1","method":"tools/call","params":{"name":"tempo_query","arguments":{"query":"{duration>1s}","start":"-5m","end":"now","limit":20}},"jsonrpc":"2.0"}'

# Create temp files
REQUEST_FILE=$(mktemp)
RESPONSE_FILE=$(mktemp)

# Clean up on exit
cleanup() {
  rm -f "$REQUEST_FILE" "$RESPONSE_FILE"
}
trap cleanup EXIT

# Write request to file - add a newline to ensure proper parsing
echo "$REQUEST" > "$REQUEST_FILE"

# Process with server
echo "Sending request to server..."
cat "$REQUEST_FILE" | ./tempo-mcp-server > "$RESPONSE_FILE"

# Display response
echo "Response from server:"
cat "$RESPONSE_FILE"

echo "Test completed" 