package application

import (
	"context"
	"fmt"
	"strings"

	"atlassian-mcp-server/internal/domain"
)

// RequestRouter dispatches MCP tool requests to the appropriate ToolHandler.
// It maintains a registry of handlers for each Atlassian tool (Jira, Confluence, Bitbucket, Bamboo)
// and routes requests based on tool name prefixes.
type RequestRouter struct {
	handlers map[string]domain.ToolHandler
}

// NewRequestRouter creates a new RequestRouter with the provided handlers.
// Handlers are registered by their ToolName() identifier.
func NewRequestRouter(handlers ...domain.ToolHandler) *RequestRouter {
	router := &RequestRouter{
		handlers: make(map[string]domain.ToolHandler),
	}

	for _, handler := range handlers {
		router.handlers[handler.ToolName()] = handler
	}

	return router
}

// Route dispatches a tool request to the appropriate handler based on the tool name.
// Tool names follow the pattern: <handler>_<operation> (e.g., jira_get_issue, confluence_create_page).
// Returns an error if the tool name is unknown or if the handler fails to process the request.
func (r *RequestRouter) Route(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error) {
	// Extract handler name from tool name prefix
	handlerName := r.extractHandlerName(req.Name)
	if handlerName == "" {
		return nil, fmt.Errorf("invalid tool name format: %s (expected format: <handler>_<operation>)", req.Name)
	}

	// Find the appropriate handler
	handler, exists := r.handlers[handlerName]
	if !exists {
		return nil, fmt.Errorf("unknown tool: %s (no handler registered for '%s')", req.Name, handlerName)
	}

	// Delegate to the handler
	return handler.Handle(ctx, req)
}

// ListAllTools aggregates tool definitions from all registered handlers.
// This is used for MCP tool discovery (tools/list method).
func (r *RequestRouter) ListAllTools() []domain.ToolDefinition {
	var allTools []domain.ToolDefinition

	// Collect tools from all handlers
	for _, handler := range r.handlers {
		tools := handler.ListTools()
		allTools = append(allTools, tools...)
	}

	return allTools
}

// extractHandlerName extracts the handler identifier from a tool name.
// Tool names follow the pattern: <handler>_<operation>
// For example: "jira_get_issue" -> "jira", "confluence_create_page" -> "confluence"
func (r *RequestRouter) extractHandlerName(toolName string) string {
	// Find the first underscore
	idx := strings.Index(toolName, "_")
	if idx == -1 {
		return ""
	}

	return toolName[:idx]
}

// GetHandler returns the handler for a specific tool name.
// This is useful for testing and debugging.
func (r *RequestRouter) GetHandler(handlerName string) (domain.ToolHandler, bool) {
	handler, exists := r.handlers[handlerName]
	return handler, exists
}
