package common

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// Environment variable name for Tempo URL
const EnvTempoURL = "TEMPO_URL"

// Default Tempo URL when environment variable is not set
const DefaultTempoURL = "http://localhost:3200"

func ConnectionParams() []mcp.ToolOption {
	tempoURL := os.Getenv(EnvTempoURL)
	if tempoURL == "" {
		tempoURL = DefaultTempoURL
	}
	return []mcp.ToolOption{
	mcp.WithString("url",
		mcp.Description(fmt.Sprintf("Tempo server URL (default: %s from %s env var)", tempoURL, EnvTempoURL)),
		mcp.DefaultString(tempoURL),
	),
	mcp.WithString("username",
		mcp.Description("Username for basic authentication"),
	),
	mcp.WithString("password",
		mcp.Description("Password for basic authentication"),
	),
	mcp.WithString("token",
			mcp.Description("Bearer token for authentication"),
		),
	}
}

func MakeTempoRequest(ctx context.Context, logger *log.Logger, toolRequest mcp.CallToolRequest, makeQueryURL func(string) (string, error)) ([]byte, error) {
	// Get Tempo URL from request arguments, if not present check environment
	var tempoURL string
	if urlArg, ok := toolRequest.Params.Arguments["url"].(string); ok && urlArg != "" {
		tempoURL = urlArg
	} else {
		// Fallback to environment variable
		tempoURL = os.Getenv(EnvTempoURL)
		if tempoURL == "" {
			tempoURL = DefaultTempoURL
		}
	}
	logger.Printf("Using Tempo URL: %s", tempoURL)

	// Extract authentication parameters
	var username, password, token string
	if usernameArg, ok := toolRequest.Params.Arguments["username"].(string); ok {
		username = usernameArg
	}
	if passwordArg, ok := toolRequest.Params.Arguments["password"].(string); ok {
		password = passwordArg
	}
	if tokenArg, ok := toolRequest.Params.Arguments["token"].(string); ok {
		token = tokenArg
	}

	// Create HTTP request
	proxyAddr := os.Getenv("HTTP_PROXY")
	var transport *http.Transport
	if proxyAddr != "" {
		logger.Printf("Using HTTP_PROXY: %s", proxyAddr)
		proxyURL, err := url.Parse("http://" + proxyAddr)
		if err != nil {
			logger.Printf("ERROR parsing HTTP_PROXY: %v", err)
		}
		transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				// Always connect to proxy
				dialer := &net.Dialer{Timeout: 10 * time.Second}
				return dialer.DialContext(ctx, "tcp", proxyAddr)
			},
		}
	}

	queryURL, err := makeQueryURL(tempoURL)
	if err != nil {
		return nil, err
	}

	logger.Printf("Executing Tempo query: %s", queryURL)
	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, err
	}

	// Add authentication if provided
	if token != "" {
		// Bearer token authentication
		req.Header.Add("Authorization", "Bearer "+token)
	} else if username != "" || password != "" {
		// Basic authentication
		req.SetBasicAuth(username, password)
	}

	// Execute request
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: transport,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d - %s", resp.StatusCode, string(body))
	}

	// Log to stderr instead of stdout
	logger.Printf("Tempo raw response length: %d bytes", len(body))
	return body, nil
}
