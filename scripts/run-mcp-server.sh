#!/bin/bash

# Set default environment variables
export TEMPO_URL=${TEMPO_URL:-"http://localhost:3200"}

# Run the server
go run ../cmd/server 