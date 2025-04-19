package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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

	// Get SSE port from environment variable or use default
	ssePort := os.Getenv("SSE_PORT")
	if ssePort == "" {
		ssePort = "8080"
	}

	// Create SSE server for HTTP/SSE connections
	sseServer := server.NewSSEServer(s,
		server.WithSSEEndpoint("/sse"),
		server.WithMessageEndpoint("/mcp"),
	)

	// Create a channel to handle shutdown signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start HTTP server in a goroutine
	go func() {
		addr := fmt.Sprintf(":%s", ssePort)
		log.Printf("Starting SSE server on http://localhost%s", addr)
		log.Printf("SSE Endpoint: http://localhost%s/sse", addr)
		log.Printf("MCP Endpoint: http://localhost%s/mcp", addr)

		if err := http.ListenAndServe(addr, sseServer); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// For backward compatibility, also serve via stdio
	go func() {
		log.Println("Starting stdio server")
		if err := server.ServeStdio(s); err != nil {
			log.Printf("Stdio server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-stop
	log.Println("Shutting down servers...")
}
