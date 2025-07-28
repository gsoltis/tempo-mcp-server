package handlers

import (
	"context"
	"fmt"

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
		)...
	)
}

func HandleTempoTrace(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	traceID := request.Params.Arguments["trace_id"].(string)
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

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(body),
			},
		},
	}, nil
}

func buildTempoTraceURL(tempoURL, traceID string) string {
	return fmt.Sprintf("%s/api/traces/%s", tempoURL, traceID)
}