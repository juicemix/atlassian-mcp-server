package domain

import (
	"context"
)

// ToolHandler processes requests for a specific Atlassian tool.
// Each Atlassian tool (Jira, Confluence, Bitbucket, Bamboo) has its own handler
// that implements this interface.
type ToolHandler interface {
	// Handle processes an MCP tool call request.
	// Returns the tool response or an error if processing fails.
	Handle(ctx context.Context, req *ToolRequest) (*ToolResponse, error)

	// ListTools returns available tools for this handler.
	// Each tool represents a specific operation (e.g., create_issue, get_page).
	ListTools() []ToolDefinition

	// ToolName returns the identifier for this handler.
	// This is used for routing requests to the appropriate handler.
	ToolName() string
}
