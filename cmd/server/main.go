package main

import (
	"fmt"

	"github.com/mark3labs/mcp-go/server"
	"github.com/scottlepp/tempo-mcp-server/internal/handlers"
)

const (
	version = "0.1.0"
)

func main() {
	// Create a new MCP server
	s := server.NewMCPServer(
		"Tempo MCP Server",
		version,
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
	)

	// Add Tempo query tool
	tempoQueryTool := handlers.NewTempoQueryTool()
	s.AddTool(tempoQueryTool, handlers.HandleTempoQuery)

	// Start the server
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
