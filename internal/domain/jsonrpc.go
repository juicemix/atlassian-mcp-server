package domain

// Request represents a JSON-RPC 2.0 request message.
type Request struct {
	JSONRPC string      `json:"jsonrpc"` // Must be "2.0"
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response message.
type Response struct {
	JSONRPC string      `json:"jsonrpc"` // Must be "2.0"
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

// Error represents a JSON-RPC 2.0 error object.
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Error implements the error interface for Error.
func (e *Error) Error() string {
	return e.Message
}

// JSON-RPC 2.0 error codes
const (
	// Standard JSON-RPC 2.0 error codes
	ParseError     = -32700 // Invalid JSON received
	InvalidRequest = -32600 // Invalid JSON-RPC request structure
	MethodNotFound = -32601 // Unknown MCP method
	InvalidParams  = -32602 // Invalid method parameters
	InternalError  = -32603 // Server internal error

	// Application-specific error codes
	ConfigurationError  = -32001 // Configuration validation failed
	AuthenticationError = -32002 // Authentication failed
	APIError            = -32003 // Atlassian API returned error
	NetworkError        = -32004 // Network connectivity issue
	RateLimitError      = -32005 // Rate limit exceeded
)
