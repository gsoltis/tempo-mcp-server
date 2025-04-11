#!/bin/bash

# Build the server and client if needed
if [ ! -f ./tempo-mcp-server ] || [ ! -f ./tempo-mcp-client ]; then
  echo "Building executables..."
  go build -o tempo-mcp-server cmd/server/main.go
  go build -o tempo-mcp-client cmd/client/main.go
fi

# Create a sample JSON-RPC request
QUERY="{resource.service.name=\"example-service\"}"
REQUEST="{\"id\":\"test-1\",\"method\":\"tools/call\",\"params\":{\"name\":\"tempo_query\",\"arguments\":{\"query\":\"$QUERY\"}},\"jsonrpc\":\"2.0\"}"

echo "Request: $REQUEST"

# Run the server with the request and capture the response
RESPONSE=$(echo -e "$REQUEST" | ./tempo-mcp-server)

echo "Response received:"
echo "$RESPONSE"

# Write the response to a file for the client to read
echo "$RESPONSE" > temp_response.json

# Run the client with the saved response
cat temp_response.json | ./tempo-mcp-client tempo_query "$QUERY"

# Clean up
rm temp_response.json

echo "Test completed" 