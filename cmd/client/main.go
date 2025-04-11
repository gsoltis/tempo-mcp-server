package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

// MCPRequest represents the outgoing MCP request structure
type MCPRequest struct {
	ID      string      `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	JsonRPC string      `json:"jsonrpc"`
}

// MCPResponse represents the incoming MCP response structure
type MCPResponse struct {
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *MCPError       `json:"error,omitempty"`
	JsonRPC string          `json:"jsonrpc"`
}

// MCPError represents an error in the MCP protocol
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  tempo-mcp-client tempo_query \"<query>\"")
		fmt.Println("  tempo-mcp-client tempo_query \"<query>\" \"<start>\" \"<end>\" <limit>")
		fmt.Println("  tempo-mcp-client tempo_query \"<url>\" \"<query>\"")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  tempo-mcp-client tempo_query \"{duration>1s}\"")
		fmt.Println("  tempo-mcp-client tempo_query \"{service.name=\\\"frontend\\\"}\"")
		fmt.Println("  tempo-mcp-client tempo_query \"{duration>500ms}\" \"-30m\" \"now\" 50")
		os.Exit(1)
	}

	method := os.Args[1]

	// Prepare the request parameters based on the method
	var params interface{}

	switch method {
	case "tempo_query":
		params = parseTempoQueryParams(os.Args[2:])
	default:
		log.Fatalf("Unknown method: %s", method)
	}

	// Create the request
	request := MCPRequest{
		ID:      "test-1",
		Method:  method,
		Params:  params,
		JsonRPC: "2.0",
	}

	// Encode the request
	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(request); err != nil {
		log.Fatalf("Error encoding request: %v", err)
	}

	// Read the response
	decoder := json.NewDecoder(os.Stdin)
	var response MCPResponse
	if err := decoder.Decode(&response); err != nil {
		log.Fatalf("Error decoding response: %v", err)
	}

	// Check for errors
	if response.Error != nil {
		log.Fatalf("MCP error: [%d] %s", response.Error.Code, response.Error.Message)
	}

	// Pretty print the result
	var result interface{}
	if err := json.Unmarshal(response.Result, &result); err != nil {
		log.Fatalf("Error parsing result: %v", err)
	}

	prettyResult, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatalf("Error formatting result: %v", err)
	}

	fmt.Println(string(prettyResult))
}

// parseTempoQueryParams parses command line arguments for Tempo query
func parseTempoQueryParams(args []string) map[string]interface{} {
	params := make(map[string]interface{})

	// Check if the first argument is a URL
	if len(args) >= 2 && strings.HasPrefix(args[0], "http") {
		params["url"] = args[0]
		params["query"] = args[1]
		args = args[2:]
	} else if len(args) >= 1 {
		params["query"] = args[0]
		args = args[1:]
	} else {
		log.Fatal("Query is required")
	}

	// Optional start time
	if len(args) >= 1 {
		params["start"] = args[0]
	}

	// Optional end time
	if len(args) >= 2 {
		params["end"] = args[1]
	}

	// Optional limit
	if len(args) >= 3 {
		var limit float64
		_, err := fmt.Sscanf(args[2], "%f", &limit)
		if err != nil {
			log.Fatalf("Invalid limit: %s", args[2])
		}
		params["limit"] = limit
	}

	return params
}
