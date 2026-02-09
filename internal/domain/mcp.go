package domain

// ToolDefinition represents an MCP tool definition.
// This describes a tool that can be called by MCP clients.
type ToolDefinition struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	InputSchema JSONSchema `json:"inputSchema"`
}

// ToolRequest represents an MCP tool call request.
// This is the request format when a client invokes a tool.
type ToolRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResponse represents an MCP tool call response.
// This is the response format returned to the client after tool execution.
type ToolResponse struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ContentBlock represents a piece of content in the response.
// MCP supports different content types (text, resource, etc.).
type ContentBlock struct {
	Type     string    `json:"type"` // "text", "resource", etc.
	Text     string    `json:"text,omitempty"`
	Resource *Resource `json:"resource,omitempty"`
}

// Resource represents a resource reference in MCP.
type Resource struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
}

// JSONSchema represents a JSON Schema for tool input validation.
// This is used to define the expected structure of tool arguments.
type JSONSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
}
