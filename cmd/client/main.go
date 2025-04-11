package main

import (
	"encoding/json"
	"fmt"
	"io"
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

	toolName := os.Args[1]

	// Prepare the request parameters based on the method
	var params interface{}

	switch toolName {
	case "tempo_query":
		// Create parameters with tool parameters
		toolParams := parseTempoQueryParams(os.Args[2:])

		// Create the full parameters object for tools/call method
		params = map[string]interface{}{
			"name":      toolName,
			"arguments": toolParams,
		}
	default:
		log.Fatalf("Unknown tool: %s", toolName)
	}

	// Create the request with tools/call method
	request := MCPRequest{
		ID:      "test-1",
		Method:  "tools/call",
		Params:  params,
		JsonRPC: "2.0",
	}

	// Check if we're in response mode via environment variable
	clientMode := os.Getenv("MCP_CLIENT_MODE")
	if clientMode == "RESPONSE" || os.Args[1] == "TEST_RESPONSE_MODE" {
		// We're in response processing mode
		log.Println("Processing response mode")

		// Read response from stdin
		responseBytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("Error reading response: %v", err)
		}

		// If we have no response data, exit
		if len(responseBytes) == 0 {
			log.Println("No response data received")
			return
		}

		// Print first 200 chars of the raw response for debugging
		responseStr := string(responseBytes)
		debugLen := 200
		if len(responseStr) < debugLen {
			debugLen = len(responseStr)
		}
		log.Printf("Raw response (first %d chars): %s", debugLen, responseStr[:debugLen])

		// Decode the response
		var response MCPResponse
		if err := json.Unmarshal(responseBytes, &response); err != nil {
			log.Fatalf("Error decoding response: %v", err)
		}

		// Check for errors
		if response.Error != nil {
			log.Fatalf("MCP error: [%d] %s", response.Error.Code, response.Error.Message)
		}

		// Pretty print the result if available
		if len(response.Result) > 0 {
			var result interface{}
			if err := json.Unmarshal(response.Result, &result); err != nil {
				log.Fatalf("Error parsing result: %v", err)
			}

			prettyResult, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				log.Fatalf("Error formatting result: %v", err)
			}

			fmt.Println(string(prettyResult))
		} else {
			log.Println("Response contains no result data")
		}
		return
	}

	// If we reach here, we're in request generation mode
	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(request); err != nil {
		log.Fatalf("Error encoding request: %v", err)
	}
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

	// Optional start time - default is 15 minutes ago
	if len(args) >= 1 && args[0] != "" {
		// We could use the string directly, but for timestamps we want to be careful
		// to avoid overflow on the server side
		if strings.HasPrefix(args[0], "-") {
			// Keep the relative time format
			params["start"] = args[0]
		} else if args[0] == "now" {
			params["start"] = args[0]
		} else {
			// Get recent small timestamp value
			params["start"] = "-5m"
		}
	} else {
		// Default to 5 minutes ago - a small relative time
		params["start"] = "-5m"
	}

	// Optional end time - default is now
	if len(args) >= 2 && args[1] != "" {
		if strings.HasPrefix(args[1], "-") {
			// Keep the relative time format
			params["end"] = args[1]
		} else if args[1] == "now" {
			params["end"] = args[1]
		} else {
			// Default to now
			params["end"] = "now"
		}
	} else {
		// Default to now
		params["end"] = "now"
	}

	// Optional limit
	if len(args) >= 3 {
		var limit float64
		_, err := fmt.Sscanf(args[2], "%f", &limit)
		if err != nil {
			log.Fatalf("Invalid limit: %s", args[2])
		}
		params["limit"] = limit
	} else {
		// Default limit
		params["limit"] = 20.0
	}

	return params
}
