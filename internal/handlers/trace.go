package handlers

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/scottlepp/tempo-mcp-server/internal/common"
)

func NewTempoTraceTool() mcp.Tool {
	return mcp.NewTool("tempo_trace",
		append(
			common.ConnectionParams(),
			mcp.WithDescription("Run a trace against Grafana Tempo"),
			mcp.WithString("trace_id",
				mcp.Required(),
				mcp.Description("Tempo trace ID"),
			),
			mcp.WithString("filename",
				mcp.Description("Filename to save the JSON trace data to"),
			),
		)...
	)
}

func HandleTempoTrace(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	traceID := request.Params.Arguments["trace_id"].(string)
	var filename string
	argFilename, ok := request.Params.Arguments["filename"].(string)
	if ok {
		filename = argFilename
	}

	logger.Printf("Received Tempo trace request: %s", traceID)

	// Extract parameters
	traceID, success := request.Params.Arguments["trace_id"].(string)
	if !success {
		return nil, fmt.Errorf("trace_id is required")
	}
	logger.Printf("Received Tempo trace request: %s", traceID)

	body, err := common.MakeTempoRequest(ctx, logger, request, func(tempoURL string) (string, error) {
		return buildTempoTraceURL(tempoURL, traceID), nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to make Tempo request: %v", err)
	}

	var responseText string
	if filename != "" {
		err = os.WriteFile(filename, body, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to save trace to file: %v", err)
		}
		responseText = fmt.Sprintf("Trace saved to %s", filename)
	} else {
		responseText = string(body)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: responseText,
			},
		},
	}, nil
}

func buildTempoTraceURL(tempoURL, traceID string) string {
	return fmt.Sprintf("%s/api/traces/%s", tempoURL, traceID)
}