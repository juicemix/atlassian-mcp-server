package domain

import (
	"encoding/json"
	"testing"
)

// TestRequestJSONSerialization verifies Request struct JSON serialization.
func TestRequestJSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		request  *Request
		expected string
	}{
		{
			name: "request with all fields",
			request: &Request{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "tools/list",
				Params:  map[string]interface{}{"key": "value"},
			},
			expected: `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"key":"value"}}`,
		},
		{
			name: "request without ID",
			request: &Request{
				JSONRPC: "2.0",
				Method:  "initialize",
			},
			expected: `{"jsonrpc":"2.0","method":"initialize"}`,
		},
		{
			name: "request without params",
			request: &Request{
				JSONRPC: "2.0",
				ID:      "test-id",
				Method:  "tools/list",
			},
			expected: `{"jsonrpc":"2.0","id":"test-id","method":"tools/list"}`,
		},
		{
			name: "request with string ID",
			request: &Request{
				JSONRPC: "2.0",
				ID:      "abc-123",
				Method:  "tools/call",
				Params:  map[string]interface{}{"name": "test_tool"},
			},
			expected: `{"jsonrpc":"2.0","id":"abc-123","method":"tools/call","params":{"name":"test_tool"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			// Compare JSON strings
			if string(data) != tt.expected {
				t.Errorf("json.Marshal() = %s, want %s", string(data), tt.expected)
			}

			// Unmarshal back to verify round-trip
			var decoded Request
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			// Verify JSONRPC field
			if decoded.JSONRPC != tt.request.JSONRPC {
				t.Errorf("decoded.JSONRPC = %s, want %s", decoded.JSONRPC, tt.request.JSONRPC)
			}

			// Verify Method field
			if decoded.Method != tt.request.Method {
				t.Errorf("decoded.Method = %s, want %s", decoded.Method, tt.request.Method)
			}
		})
	}
}

// TestResponseJSONSerialization verifies Response struct JSON serialization.
func TestResponseJSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		response *Response
		expected string
	}{
		{
			name: "response with result",
			response: &Response{
				JSONRPC: "2.0",
				ID:      1,
				Result:  map[string]interface{}{"status": "ok"},
			},
			expected: `{"jsonrpc":"2.0","id":1,"result":{"status":"ok"}}`,
		},
		{
			name: "response with error",
			response: &Response{
				JSONRPC: "2.0",
				ID:      2,
				Error: &Error{
					Code:    InvalidRequest,
					Message: "Invalid request",
				},
			},
			expected: `{"jsonrpc":"2.0","id":2,"error":{"code":-32600,"message":"Invalid request"}}`,
		},
		{
			name: "response with error and data",
			response: &Response{
				JSONRPC: "2.0",
				ID:      "test-id",
				Error: &Error{
					Code:    AuthenticationError,
					Message: "Authentication failed",
					Data:    map[string]interface{}{"tool": "jira"},
				},
			},
			expected: `{"jsonrpc":"2.0","id":"test-id","error":{"code":-32002,"message":"Authentication failed","data":{"tool":"jira"}}}`,
		},
		{
			name: "response without ID (notification response)",
			response: &Response{
				JSONRPC: "2.0",
				Result:  "success",
			},
			expected: `{"jsonrpc":"2.0","result":"success"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.response)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			// Compare JSON strings
			if string(data) != tt.expected {
				t.Errorf("json.Marshal() = %s, want %s", string(data), tt.expected)
			}

			// Unmarshal back to verify round-trip
			var decoded Response
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			// Verify JSONRPC field
			if decoded.JSONRPC != tt.response.JSONRPC {
				t.Errorf("decoded.JSONRPC = %s, want %s", decoded.JSONRPC, tt.response.JSONRPC)
			}
		})
	}
}

// TestErrorJSONSerialization verifies Error struct JSON serialization.
func TestErrorJSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		error    *Error
		expected string
	}{
		{
			name: "error without data",
			error: &Error{
				Code:    InternalError,
				Message: "Internal server error",
			},
			expected: `{"code":-32603,"message":"Internal server error"}`,
		},
		{
			name: "error with data",
			error: &Error{
				Code:    APIError,
				Message: "API request failed",
				Data: map[string]interface{}{
					"statusCode": 500,
					"tool":       "confluence",
				},
			},
			expected: `{"code":-32003,"message":"API request failed","data":{"statusCode":500,"tool":"confluence"}}`,
		},
		{
			name: "parse error",
			error: &Error{
				Code:    ParseError,
				Message: "Parse error",
			},
			expected: `{"code":-32700,"message":"Parse error"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.error)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			// Compare JSON strings (note: map order may vary, so we unmarshal and compare)
			var decoded Error
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			// Verify fields
			if decoded.Code != tt.error.Code {
				t.Errorf("decoded.Code = %d, want %d", decoded.Code, tt.error.Code)
			}
			if decoded.Message != tt.error.Message {
				t.Errorf("decoded.Message = %s, want %s", decoded.Message, tt.error.Message)
			}
		})
	}
}

// TestRequestDeserialization verifies Request struct JSON deserialization.
func TestRequestDeserialization(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected *Request
		wantErr  bool
	}{
		{
			name: "valid request with integer ID",
			json: `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
			expected: &Request{
				JSONRPC: "2.0",
				ID:      float64(1), // JSON numbers unmarshal to float64
				Method:  "tools/list",
			},
			wantErr: false,
		},
		{
			name: "valid request with string ID",
			json: `{"jsonrpc":"2.0","id":"test-123","method":"initialize"}`,
			expected: &Request{
				JSONRPC: "2.0",
				ID:      "test-123",
				Method:  "initialize",
			},
			wantErr: false,
		},
		{
			name: "valid request without ID",
			json: `{"jsonrpc":"2.0","method":"tools/call"}`,
			expected: &Request{
				JSONRPC: "2.0",
				Method:  "tools/call",
			},
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			json:    `{"jsonrpc":"2.0","method":}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req Request
			err := json.Unmarshal([]byte(tt.json), &req)

			if (err != nil) != tt.wantErr {
				t.Errorf("json.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if req.JSONRPC != tt.expected.JSONRPC {
					t.Errorf("req.JSONRPC = %s, want %s", req.JSONRPC, tt.expected.JSONRPC)
				}
				if req.Method != tt.expected.Method {
					t.Errorf("req.Method = %s, want %s", req.Method, tt.expected.Method)
				}
			}
		})
	}
}

// TestResponseDeserialization verifies Response struct JSON deserialization.
func TestResponseDeserialization(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected *Response
		wantErr  bool
	}{
		{
			name: "valid response with result",
			json: `{"jsonrpc":"2.0","id":1,"result":{"status":"ok"}}`,
			expected: &Response{
				JSONRPC: "2.0",
				ID:      float64(1),
				Result:  map[string]interface{}{"status": "ok"},
			},
			wantErr: false,
		},
		{
			name: "valid response with error",
			json: `{"jsonrpc":"2.0","id":2,"error":{"code":-32600,"message":"Invalid request"}}`,
			expected: &Response{
				JSONRPC: "2.0",
				ID:      float64(2),
				Error: &Error{
					Code:    InvalidRequest,
					Message: "Invalid request",
				},
			},
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			json:    `{"jsonrpc":"2.0","result":}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp Response
			err := json.Unmarshal([]byte(tt.json), &resp)

			if (err != nil) != tt.wantErr {
				t.Errorf("json.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if resp.JSONRPC != tt.expected.JSONRPC {
					t.Errorf("resp.JSONRPC = %s, want %s", resp.JSONRPC, tt.expected.JSONRPC)
				}
				if tt.expected.Error != nil {
					if resp.Error == nil {
						t.Error("resp.Error is nil, want non-nil")
					} else {
						if resp.Error.Code != tt.expected.Error.Code {
							t.Errorf("resp.Error.Code = %d, want %d", resp.Error.Code, tt.expected.Error.Code)
						}
						if resp.Error.Message != tt.expected.Error.Message {
							t.Errorf("resp.Error.Message = %s, want %s", resp.Error.Message, tt.expected.Error.Message)
						}
					}
				}
			}
		})
	}
}

// TestJSONRPCStructTags verifies that JSON struct tags are correctly defined.
func TestJSONRPCStructTags(t *testing.T) {
	// Test Request struct tags
	req := &Request{
		JSONRPC: "2.0",
		ID:      123,
		Method:  "test",
		Params:  map[string]string{"key": "value"},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal(Request) error = %v", err)
	}

	// Verify field names in JSON
	jsonStr := string(data)
	if !contains(jsonStr, `"jsonrpc"`) {
		t.Error("JSON should contain 'jsonrpc' field")
	}
	if !contains(jsonStr, `"id"`) {
		t.Error("JSON should contain 'id' field")
	}
	if !contains(jsonStr, `"method"`) {
		t.Error("JSON should contain 'method' field")
	}
	if !contains(jsonStr, `"params"`) {
		t.Error("JSON should contain 'params' field")
	}

	// Test Response struct tags
	resp := &Response{
		JSONRPC: "2.0",
		ID:      456,
		Result:  "success",
	}

	data, err = json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal(Response) error = %v", err)
	}

	jsonStr = string(data)
	if !contains(jsonStr, `"jsonrpc"`) {
		t.Error("JSON should contain 'jsonrpc' field")
	}
	if !contains(jsonStr, `"id"`) {
		t.Error("JSON should contain 'id' field")
	}
	if !contains(jsonStr, `"result"`) {
		t.Error("JSON should contain 'result' field")
	}

	// Test Error struct tags
	errObj := &Error{
		Code:    -32600,
		Message: "test error",
		Data:    map[string]string{"detail": "info"},
	}

	data, err = json.Marshal(errObj)
	if err != nil {
		t.Fatalf("json.Marshal(Error) error = %v", err)
	}

	jsonStr = string(data)
	if !contains(jsonStr, `"code"`) {
		t.Error("JSON should contain 'code' field")
	}
	if !contains(jsonStr, `"message"`) {
		t.Error("JSON should contain 'message' field")
	}
	if !contains(jsonStr, `"data"`) {
		t.Error("JSON should contain 'data' field")
	}
}

// TestOmitEmptyBehavior verifies that omitempty works correctly for optional fields.
func TestOmitEmptyBehavior(t *testing.T) {
	// Request without ID and Params should omit those fields
	req := &Request{
		JSONRPC: "2.0",
		Method:  "test",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	jsonStr := string(data)
	// ID and Params should not be present when omitted
	if contains(jsonStr, `"id"`) {
		t.Error("JSON should not contain 'id' field when omitted")
	}
	if contains(jsonStr, `"params"`) {
		t.Error("JSON should not contain 'params' field when omitted")
	}

	// Response without Error should omit error field
	resp := &Response{
		JSONRPC: "2.0",
		ID:      1,
		Result:  "ok",
	}

	data, err = json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	jsonStr = string(data)
	if contains(jsonStr, `"error"`) {
		t.Error("JSON should not contain 'error' field when omitted")
	}

	// Response without Result should omit result field
	resp2 := &Response{
		JSONRPC: "2.0",
		ID:      2,
		Error: &Error{
			Code:    -32600,
			Message: "error",
		},
	}

	data, err = json.Marshal(resp2)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	jsonStr = string(data)
	if contains(jsonStr, `"result"`) {
		t.Error("JSON should not contain 'result' field when omitted")
	}
}
