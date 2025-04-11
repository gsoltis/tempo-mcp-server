package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// MCPRequest represents a JSON-RPC request for MCP
type MCPRequest struct {
	ID      string                 `json:"id"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params"`
	JsonRPC string                 `json:"jsonrpc"`
}

func main() {
	// Check arguments
	if len(os.Args) < 3 {
		fmt.Println("Usage: simple-client <tool> <query>")
		fmt.Println("Example: simple-client tempo_query '{duration>1s}'")
		os.Exit(1)
	}

	tool := os.Args[1]
	query := os.Args[2]

	// Create request - using the tool name directly as the method
	request := MCPRequest{
		ID:      "simple-client-1",
		Method:  tool,
		Params:  map[string]interface{}{"query": query},
		JsonRPC: "2.0",
	}

	// Convert to JSON
	requestJSON, err := json.Marshal(request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling request: %v\n", err)
		os.Exit(1)
	}

	// Print the request for debugging
	fmt.Printf("Sending request: %s\n", requestJSON)

	// Start the server process
	cmd := exec.Command("./tempo-mcp-server")

	// Get stdin pipe
	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting stdin pipe: %v\n", err)
		os.Exit(1)
	}

	// Set stdout and stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
		os.Exit(1)
	}

	// Write request to server - add a newline at the end
	if _, err := stdin.Write(append(requestJSON, '\n')); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing to server: %v\n", err)
		os.Exit(1)
	}

	// Close stdin to signal EOF
	stdin.Close()

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
