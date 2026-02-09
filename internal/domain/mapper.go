package domain

// ResponseMapper converts API responses to MCP tool responses.
// This interface is responsible for transforming Atlassian API responses
// into MCP-compliant format that can be consumed by MCP clients.
type ResponseMapper interface {
	// MapToToolResponse converts an API response to MCP format.
	// The apiResponse parameter should be the deserialized JSON response
	// from an Atlassian API. Returns an error if transformation fails.
	MapToToolResponse(apiResponse interface{}) (*ToolResponse, error)

	// MapError converts an API error to MCP error format.
	// This method maps HTTP status codes and error responses from
	// Atlassian APIs to appropriate JSON-RPC error codes and messages.
	MapError(err error) *Error
}
