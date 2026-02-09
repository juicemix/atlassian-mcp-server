package domain

import (
	"testing"
)

// TestDomainTypesCompile verifies that all domain types compile correctly.
// This is a basic sanity check to ensure the interfaces and types are well-formed.
func TestDomainTypesCompile(t *testing.T) {
	// Test that we can create instances of the basic types
	var _ *Request = &Request{
		JSONRPC: "2.0",
		Method:  "test",
	}

	var _ *Response = &Response{
		JSONRPC: "2.0",
	}

	var _ *Error = &Error{
		Code:    InternalError,
		Message: "test error",
	}

	var _ *ToolDefinition = &ToolDefinition{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: JSONSchema{Type: "object"},
	}

	var _ *ToolRequest = &ToolRequest{
		Name:      "test_tool",
		Arguments: map[string]interface{}{},
	}

	var _ *ToolResponse = &ToolResponse{
		Content: []ContentBlock{},
	}

	var _ *Config = &Config{
		Transport: TransportConfig{Type: "stdio"},
		Tools:     ToolsConfig{},
	}

	// Test AuthType enum
	if BasicAuth.String() != "basic" {
		t.Errorf("BasicAuth.String() = %s, want basic", BasicAuth.String())
	}

	if TokenAuth.String() != "token" {
		t.Errorf("TokenAuth.String() = %s, want token", TokenAuth.String())
	}

	// Test ParseAuthType
	if ParseAuthType("basic") != BasicAuth {
		t.Error("ParseAuthType(basic) should return BasicAuth")
	}

	if ParseAuthType("token") != TokenAuth {
		t.Error("ParseAuthType(token) should return TokenAuth")
	}

	if ParseAuthType("invalid") != BasicAuth {
		t.Error("ParseAuthType(invalid) should return BasicAuth as default")
	}
}

// TestErrorCodes verifies that error codes are defined correctly.
func TestErrorCodes(t *testing.T) {
	tests := []struct {
		name string
		code int
		want int
	}{
		{"ParseError", ParseError, -32700},
		{"InvalidRequest", InvalidRequest, -32600},
		{"MethodNotFound", MethodNotFound, -32601},
		{"InvalidParams", InvalidParams, -32602},
		{"InternalError", InternalError, -32603},
		{"ConfigurationError", ConfigurationError, -32001},
		{"AuthenticationError", AuthenticationError, -32002},
		{"APIError", APIError, -32003},
		{"NetworkError", NetworkError, -32004},
		{"RateLimitError", RateLimitError, -32005},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.code, tt.want)
			}
		})
	}
}

// TestJSONRPCVersion verifies the JSON-RPC version constant.
func TestJSONRPCVersion(t *testing.T) {
	req := &Request{JSONRPC: "2.0"}
	if req.JSONRPC != "2.0" {
		t.Errorf("Request.JSONRPC = %s, want 2.0", req.JSONRPC)
	}

	resp := &Response{JSONRPC: "2.0"}
	if resp.JSONRPC != "2.0" {
		t.Errorf("Response.JSONRPC = %s, want 2.0", resp.JSONRPC)
	}
}
