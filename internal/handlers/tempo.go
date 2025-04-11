package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// TempoResult represents the structure of Tempo query results
type TempoResult struct {
	Traces      []TempoTrace `json:"traces"`
	Metrics     interface{}  `json:"metrics,omitempty"`
	ErrorStatus string       `json:"error,omitempty"`
}

// TempoTrace represents a single trace in the result
type TempoTrace struct {
	TraceID           string            `json:"traceID"`
	RootServiceName   string            `json:"rootServiceName"`
	RootTraceName     string            `json:"rootTraceName"`
	StartTimeUnixNano string            `json:"startTimeUnixNano"`
	DurationMs        int64             `json:"durationMs"`
	SpanSet           interface{}       `json:"spanSet,omitempty"`
	Attributes        map[string]string `json:"attributes,omitempty"`
}

// Environment variable name for Tempo URL
const EnvTempoURL = "TEMPO_URL"

// Default Tempo URL when environment variable is not set
const DefaultTempoURL = "http://localhost:3200"

// NewTempoQueryTool creates and returns a tool for querying Grafana Tempo
func NewTempoQueryTool() mcp.Tool {
	// Get Tempo URL from environment variable or use default
	tempoURL := os.Getenv(EnvTempoURL)
	if tempoURL == "" {
		tempoURL = DefaultTempoURL
	}

	return mcp.NewTool("tempo_query",
		mcp.WithDescription("Run a query against Grafana Tempo"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Tempo query string"),
		),
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
		mcp.WithString("start",
			mcp.Description("Start time for the query (default: 1h ago)"),
		),
		mcp.WithString("end",
			mcp.Description("End time for the query (default: now)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of traces to return (default: 20)"),
		),
	)
}

// HandleTempoQuery handles Tempo query tool requests
func HandleTempoQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters
	queryString := request.Params.Arguments["query"].(string)

	// Get Tempo URL from request arguments, if not present check environment
	var tempoURL string
	if urlArg, ok := request.Params.Arguments["url"].(string); ok && urlArg != "" {
		tempoURL = urlArg
	} else {
		// Fallback to environment variable
		tempoURL = os.Getenv(EnvTempoURL)
		if tempoURL == "" {
			tempoURL = DefaultTempoURL
		}
	}

	// Extract authentication parameters
	var username, password, token string
	if usernameArg, ok := request.Params.Arguments["username"].(string); ok {
		username = usernameArg
	}
	if passwordArg, ok := request.Params.Arguments["password"].(string); ok {
		password = passwordArg
	}
	if tokenArg, ok := request.Params.Arguments["token"].(string); ok {
		token = tokenArg
	}

	// Set defaults for optional parameters
	start := time.Now().Add(-1 * time.Hour).Unix()
	end := time.Now().Unix()
	limit := 20

	// Override defaults if parameters are provided
	if startStr, ok := request.Params.Arguments["start"].(string); ok && startStr != "" {
		startTime, err := parseTime(startStr)
		if err != nil {
			return nil, fmt.Errorf("invalid start time: %v", err)
		}
		start = startTime.Unix()
	}

	if endStr, ok := request.Params.Arguments["end"].(string); ok && endStr != "" {
		endTime, err := parseTime(endStr)
		if err != nil {
			return nil, fmt.Errorf("invalid end time: %v", err)
		}
		end = endTime.Unix()
	}

	if limitVal, ok := request.Params.Arguments["limit"].(float64); ok {
		limit = int(limitVal)
	}

	// Build query URL
	queryURL, err := buildTempoQueryURL(tempoURL, queryString, start, end, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to build query URL: %v", err)
	}

	// Execute query with authentication
	result, err := executeTempoQuery(ctx, queryURL, username, password, token)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %v", err)
	}

	// Format results
	formattedResult, err := formatTempoResults(result)
	if err != nil {
		return nil, fmt.Errorf("failed to format results: %v", err)
	}

	return mcp.NewToolResultText(formattedResult), nil
}

// parseTime converts a time string to a time.Time
func parseTime(timeStr string) (time.Time, error) {
	// Handle "now" keyword
	if timeStr == "now" {
		return time.Now(), nil
	}

	// Handle relative time strings like "-1h", "-30m"
	if len(timeStr) > 0 && timeStr[0] == '-' {
		duration, err := time.ParseDuration(timeStr)
		if err == nil {
			return time.Now().Add(duration), nil
		}
	}

	// Try parsing as RFC3339
	t, err := time.Parse(time.RFC3339, timeStr)
	if err == nil {
		return t, nil
	}

	// Try other common formats
	formats := []string{
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		t, err := time.Parse(format, timeStr)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported time format: %s", timeStr)
}

// buildTempoQueryURL constructs the Tempo query URL
func buildTempoQueryURL(baseURL, query string, start, end int64, limit int) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	// Path for Tempo search API
	if !strings.Contains(u.Path, "/api/search") {
		if u.Path == "" || u.Path == "/" {
			u.Path = "/api/search"
		} else {
			u.Path = fmt.Sprintf("%s/api/search", u.Path)
		}
	}

	// Add query parameters
	q := u.Query()
	q.Set("q", query)
	q.Set("start", fmt.Sprintf("%d", start*1000000000)) // Convert to nanoseconds
	q.Set("end", fmt.Sprintf("%d", end*1000000000))     // Convert to nanoseconds
	q.Set("limit", fmt.Sprintf("%d", limit))
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// executeTempoQuery sends the HTTP request to Tempo
func executeTempoQuery(ctx context.Context, queryURL, username, password, token string) (*TempoResult, error) {
	// Create HTTP request
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

	// Parse JSON response
	var result TempoResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	// Check for Tempo errors
	if result.ErrorStatus != "" {
		return nil, fmt.Errorf("Tempo error: %s", result.ErrorStatus)
	}

	return &result, nil
}

// formatTempoResults formats the Tempo query results into a readable string
func formatTempoResults(result *TempoResult) (string, error) {
	if len(result.Traces) == 0 {
		return "No traces found matching the query", nil
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Found %d traces:\n\n", len(result.Traces)))

	for i, trace := range result.Traces {
		// Format trace information
		output.WriteString(fmt.Sprintf("Trace %d:\n", i+1))
		output.WriteString(fmt.Sprintf("  TraceID: %s\n", trace.TraceID))
		output.WriteString(fmt.Sprintf("  Service: %s\n", trace.RootServiceName))
		output.WriteString(fmt.Sprintf("  Name: %s\n", trace.RootTraceName))

		// Parse timestamp if available
		if trace.StartTimeUnixNano != "" {
			ts, err := strconv.ParseInt(trace.StartTimeUnixNano, 10, 64)
			if err == nil {
				timestamp := time.Unix(0, ts)
				output.WriteString(fmt.Sprintf("  Start Time: %s\n", timestamp.Format(time.RFC3339)))
			}
		}

		output.WriteString(fmt.Sprintf("  Duration: %d ms\n", trace.DurationMs))

		// Add attributes if available
		if len(trace.Attributes) > 0 {
			output.WriteString("  Attributes:\n")
			for k, v := range trace.Attributes {
				output.WriteString(fmt.Sprintf("    %s: %s\n", k, v))
			}
		}

		output.WriteString("\n")
	}

	return output.String(), nil
}
