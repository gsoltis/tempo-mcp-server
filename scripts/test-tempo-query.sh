#!/bin/bash

# Default values
QUERY="{}"
START="-15m"
END="now"
LIMIT=20
TEMPO_URL=${TEMPO_URL:-"http://localhost:3200"}

# Parse command line arguments
if [ "$#" -ge 1 ]; then
  QUERY="$1"
fi

if [ "$#" -ge 2 ]; then
  START="$2"
fi

if [ "$#" -ge 3 ]; then
  END="$3"
fi

if [ "$#" -ge 4 ]; then
  LIMIT="$4"
fi

echo "Querying Tempo:"
echo "  URL: $TEMPO_URL"
echo "  Query: $QUERY"
echo "  Time Range: $START to $END"
echo "  Limit: $LIMIT"
echo ""

# Construct the curl command
CURL_CMD="curl -s \"$TEMPO_URL/api/search?q=$QUERY&start=$(date -u -v$START +%s%N)&end=$(date -u -v$END +%s%N)&limit=$LIMIT\""

# Execute the command
echo "Executing: $CURL_CMD"
echo ""
eval $CURL_CMD | jq '.'

# Exit with the status of the curl command
exit ${PIPESTATUS[0]} 