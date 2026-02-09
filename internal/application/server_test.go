package application

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"atlassian-mcp-server/internal/domain"
)

// mockTransport is a mock implementation of domain.Transport for testing.
type mockTransport struct {
	mu        sync.Mutex
	reqChan   chan *domain.Request
	responses []*domain.Response
	started   bool
	closed    bool
}

func newMockTransport() *mockTransport {
	return &mockTransport{
		reqChan:   make(chan *domain.Request, 10),
		responses: make([]*domain.Response, 0),
	}
}

func (m *mockTransport) Start(ctx context.Context) error {
	m.started = true
	return nil
}

func (m *mockTransport) Send(response *domain.Response) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses = append(m.responses, response)
	return nil
}

func (m *mockTransport) Receive() <-chan *domain.Request {
	return m.reqChan
}

func (m *mockTransport) Close() error {
	m.closed = true
	close(m.reqChan)
	return nil
}

func (m *mockTransport) sendRequest(req *domain.Request) {
	m.reqChan <- req
}

func (m *mockTransport) getLastResponse() *domain.Response {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.responses) == 0 {
		return nil
	}
	return m.responses[len(m.responses)-1]
}

func (m *mockTransport) getAllResponses() []*domain.Response {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Return a copy to avoid race conditions
	result := make([]*domain.Response, len(m.responses))
	copy(result, m.responses)
	return result
}

// mockToolHandler is a mock implementation of domain.ToolHandler for testing.
type mockToolHandler struct {
	name     string
	tools    []domain.ToolDefinition
	response *domain.ToolResponse
	err      error
}

func (m *mockToolHandler) Handle(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func (m *mockToolHandler) ListTools() []domain.ToolDefinition {
	return m.tools
}

func (m *mockToolHandler) ToolName() string {
	return m.name
}

// createTestServer creates a server with mock dependencies for testing.
func createTestServer() (*Server, *mockTransport) {
	transport := newMockTransport()

	// Create mock handlers
	jiraHandler := &mockToolHandler{
		name: "jira",
		tools: []domain.ToolDefinition{
			{
				Name:        "jira_get_issue",
				Description: "Get a Jira issue",
				InputSchema: domain.JSONSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"issueKey": map[string]interface{}{"type": "string"},
					},
					Required: []string{"issueKey"},
				},
			},
		},
		response: &domain.ToolResponse{
			Content: []domain.ContentBlock{
				{Type: "text", Text: "Issue retrieved"},
			},
		},
	}

	router := NewRequestRouter(jiraHandler)

	// Create auth manager with valid credentials
	authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{
		"jira": {
			Type:     domain.BasicAuth,
			Username: "testuser",
			Password: "testpass",
		},
	})

	// Create config
	config := &domain.Config{
		Transport: domain.TransportConfig{
			Type: "stdio",
		},
		Tools: domain.ToolsConfig{
			Jira: &domain.ToolConfig{
				BaseURL: "https://jira.example.com",
				Auth: &domain.AuthConfig{
					Type:     "basic",
					Username: "testuser",
					Password: "testpass",
				},
			},
		},
	}

	server := NewServer(transport, router, authManager, config)
	return server, transport
}

func TestNewServer(t *testing.T) {
	server, transport := createTestServer()

	if server == nil {
		t.Fatal("NewServer returned nil")
	}

	if server.transport != transport {
		t.Error("Server transport not set correctly")
	}

	if server.router == nil {
		t.Error("Server router is nil")
	}

	if server.authManager == nil {
		t.Error("Server authManager is nil")
	}

	if server.config == nil {
		t.Error("Server config is nil")
	}

	if server.logger == nil {
		t.Error("Server logger is nil")
	}
}

func TestServerStart(t *testing.T) {
	server, transport := createTestServer()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := server.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !transport.started {
		t.Error("Transport was not started")
	}
}

func TestHandleInitialize(t *testing.T) {
	server, transport := createTestServer()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the server
	if err := server.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Send initialize request
	req := &domain.Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  map[string]interface{}{},
	}

	transport.sendRequest(req)

	// Wait for response
	time.Sleep(100 * time.Millisecond)

	resp := transport.getLastResponse()
	if resp == nil {
		t.Fatal("No response received")
	}

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}

	// Verify response structure
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if result["protocolVersion"] == nil {
		t.Error("Missing protocolVersion in response")
	}

	if result["serverInfo"] == nil {
		t.Error("Missing serverInfo in response")
	}

	if result["capabilities"] == nil {
		t.Error("Missing capabilities in response")
	}
}

func TestHandleToolsList(t *testing.T) {
	server, transport := createTestServer()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the server
	if err := server.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Send tools/list request
	req := &domain.Request{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
	}

	transport.sendRequest(req)

	// Wait for response
	time.Sleep(100 * time.Millisecond)

	resp := transport.getLastResponse()
	if resp == nil {
		t.Fatal("No response received")
	}

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}

	// Verify response structure
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	tools, ok := result["tools"].([]domain.ToolDefinition)
	if !ok {
		t.Fatal("Tools is not a slice of ToolDefinition")
	}

	if len(tools) == 0 {
		t.Error("Expected at least one tool")
	}

	// Verify the tool definition
	if tools[0].Name != "jira_get_issue" {
		t.Errorf("Expected tool name 'jira_get_issue', got '%s'", tools[0].Name)
	}
}

func TestHandleToolsCall_Success(t *testing.T) {
	server, transport := createTestServer()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the server
	if err := server.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Send tools/call request
	req := &domain.Request{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "jira_get_issue",
			"arguments": map[string]interface{}{
				"issueKey": "TEST-1",
			},
		},
	}

	transport.sendRequest(req)

	// Wait for response
	time.Sleep(100 * time.Millisecond)

	resp := transport.getLastResponse()
	if resp == nil {
		t.Fatal("No response received")
	}

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}

	// Verify response contains tool response
	if resp.Result == nil {
		t.Fatal("Result is nil")
	}
}

func TestHandleToolsCall_MissingParams(t *testing.T) {
	server, transport := createTestServer()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the server
	if err := server.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Send tools/call request without params
	req := &domain.Request{
		JSONRPC: "2.0",
		ID:      4,
		Method:  "tools/call",
		Params:  nil,
	}

	transport.sendRequest(req)

	// Wait for response
	time.Sleep(100 * time.Millisecond)

	resp := transport.getLastResponse()
	if resp == nil {
		t.Fatal("No response received")
	}

	if resp.Error == nil {
		t.Fatal("Expected error response")
	}

	if resp.Error.Code != domain.InvalidParams {
		t.Errorf("Expected error code %d, got %d", domain.InvalidParams, resp.Error.Code)
	}
}

func TestHandleToolsCall_MissingToolName(t *testing.T) {
	server, transport := createTestServer()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the server
	if err := server.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Send tools/call request without tool name
	req := &domain.Request{
		JSONRPC: "2.0",
		ID:      5,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"arguments": map[string]interface{}{},
		},
	}

	transport.sendRequest(req)

	// Wait for response
	time.Sleep(100 * time.Millisecond)

	resp := transport.getLastResponse()
	if resp == nil {
		t.Fatal("No response received")
	}

	if resp.Error == nil {
		t.Fatal("Expected error response")
	}

	if resp.Error.Code != domain.InvalidParams {
		t.Errorf("Expected error code %d, got %d", domain.InvalidParams, resp.Error.Code)
	}
}

func TestHandleToolsCall_AuthenticationFailure(t *testing.T) {
	transport := newMockTransport()

	// Create handler
	jiraHandler := &mockToolHandler{
		name: "jira",
		tools: []domain.ToolDefinition{
			{Name: "jira_get_issue", Description: "Get issue"},
		},
	}

	router := NewRequestRouter(jiraHandler)

	// Create auth manager WITHOUT credentials
	authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{})

	config := &domain.Config{
		Transport: domain.TransportConfig{Type: "stdio"},
	}

	server := NewServer(transport, router, authManager, config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the server
	if err := server.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Send tools/call request
	req := &domain.Request{
		JSONRPC: "2.0",
		ID:      6,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "jira_get_issue",
			"arguments": map[string]interface{}{
				"issueKey": "TEST-1",
			},
		},
	}

	transport.sendRequest(req)

	// Wait for response
	time.Sleep(100 * time.Millisecond)

	resp := transport.getLastResponse()
	if resp == nil {
		t.Fatal("No response received")
	}

	if resp.Error == nil {
		t.Fatal("Expected error response for missing credentials")
	}

	if resp.Error.Code != domain.AuthenticationError {
		t.Errorf("Expected error code %d, got %d", domain.AuthenticationError, resp.Error.Code)
	}
}

func TestHandleToolsCall_UnknownTool(t *testing.T) {
	server, transport := createTestServer()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the server
	if err := server.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Send tools/call request for unknown tool
	// Note: This will fail authentication first since "unknown" tool has no credentials
	req := &domain.Request{
		JSONRPC: "2.0",
		ID:      7,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      "unknown_tool",
			"arguments": map[string]interface{}{},
		},
	}

	transport.sendRequest(req)

	// Wait for response
	time.Sleep(100 * time.Millisecond)

	resp := transport.getLastResponse()
	if resp == nil {
		t.Fatal("No response received")
	}

	if resp.Error == nil {
		t.Fatal("Expected error response for unknown tool")
	}

	// Should be AuthenticationError since authentication is validated before routing
	// This is correct behavior - unauthenticated requests should never reach the router
	if resp.Error.Code != domain.AuthenticationError {
		t.Errorf("Expected error code %d, got %d", domain.AuthenticationError, resp.Error.Code)
	}
}

func TestHandleUnknownMethod(t *testing.T) {
	server, transport := createTestServer()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the server
	if err := server.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Send request with unknown method
	req := &domain.Request{
		JSONRPC: "2.0",
		ID:      8,
		Method:  "unknown/method",
	}

	transport.sendRequest(req)

	// Wait for response
	time.Sleep(100 * time.Millisecond)

	resp := transport.getLastResponse()
	if resp == nil {
		t.Fatal("No response received")
	}

	if resp.Error == nil {
		t.Fatal("Expected error response")
	}

	if resp.Error.Code != domain.MethodNotFound {
		t.Errorf("Expected error code %d, got %d", domain.MethodNotFound, resp.Error.Code)
	}
}

func TestValidateRequest_InvalidJSONRPC(t *testing.T) {
	server, _ := createTestServer()

	req := &domain.Request{
		JSONRPC: "1.0",
		Method:  "test",
	}

	err := server.validateRequest(req)
	if err == nil {
		t.Fatal("Expected validation error for invalid JSONRPC version")
	}
}

func TestValidateRequest_MissingMethod(t *testing.T) {
	server, _ := createTestServer()

	req := &domain.Request{
		JSONRPC: "2.0",
		Method:  "",
	}

	err := server.validateRequest(req)
	if err == nil {
		t.Fatal("Expected validation error for missing method")
	}
}

func TestParseToolRequest_Valid(t *testing.T) {
	server, _ := createTestServer()

	params := map[string]interface{}{
		"name": "jira_get_issue",
		"arguments": map[string]interface{}{
			"issueKey": "TEST-1",
		},
	}

	toolReq, err := server.parseToolRequest(params)
	if err != nil {
		t.Fatalf("Failed to parse tool request: %v", err)
	}

	if toolReq.Name != "jira_get_issue" {
		t.Errorf("Expected name 'jira_get_issue', got '%s'", toolReq.Name)
	}

	if toolReq.Arguments["issueKey"] != "TEST-1" {
		t.Errorf("Expected issueKey 'TEST-1', got '%v'", toolReq.Arguments["issueKey"])
	}
}

func TestParseToolRequest_NilParams(t *testing.T) {
	server, _ := createTestServer()

	_, err := server.parseToolRequest(nil)
	if err == nil {
		t.Fatal("Expected error for nil params")
	}
}

func TestParseToolRequest_MissingName(t *testing.T) {
	server, _ := createTestServer()

	params := map[string]interface{}{
		"arguments": map[string]interface{}{},
	}

	_, err := server.parseToolRequest(params)
	if err == nil {
		t.Fatal("Expected error for missing tool name")
	}
}

func TestValidateAuthentication_Valid(t *testing.T) {
	server, _ := createTestServer()

	err := server.validateAuthentication("jira_get_issue")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestValidateAuthentication_MissingCredentials(t *testing.T) {
	transport := newMockTransport()
	router := NewRequestRouter()
	authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{})
	config := &domain.Config{}

	server := NewServer(transport, router, authManager, config)

	err := server.validateAuthentication("jira_get_issue")
	if err == nil {
		t.Fatal("Expected error for missing credentials")
	}
}

func TestExtractToolType(t *testing.T) {
	tests := []struct {
		toolName string
		expected string
	}{
		{"jira_get_issue", "jira"},
		{"confluence_create_page", "confluence"},
		{"bitbucket_list_repos", "bitbucket"},
		{"bamboo_trigger_build", "bamboo"},
		{"invalid", ""},
		{"", ""},
	}

	for _, tt := range tests {
		result := extractToolType(tt.toolName)
		if result != tt.expected {
			t.Errorf("extractToolType(%s) = %s, expected %s", tt.toolName, result, tt.expected)
		}
	}
}

func TestServerClose(t *testing.T) {
	server, transport := createTestServer()

	err := server.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	if !transport.closed {
		t.Error("Transport was not closed")
	}
}

func TestStructuredLogger(t *testing.T) {
	logger := NewStructuredLogger()

	if logger == nil {
		t.Fatal("NewStructuredLogger returned nil")
	}

	// Test LogInfo
	logger.LogInfo("test message", map[string]interface{}{
		"key": "value",
	})

	// Test LogError
	logger.LogError("error message", nil, map[string]interface{}{
		"context": "test",
	})
}

func TestStructuredLogger_BuildLogEntry(t *testing.T) {
	logger := NewStructuredLogger()

	entry := logger.buildLogEntry("INFO", "test", nil, map[string]interface{}{
		"key": "value",
	})

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(entry), &parsed); err != nil {
		t.Fatalf("Log entry is not valid JSON: %v", err)
	}

	if parsed["level"] != "INFO" {
		t.Errorf("Expected level 'INFO', got '%v'", parsed["level"])
	}

	if parsed["message"] != "test" {
		t.Errorf("Expected message 'test', got '%v'", parsed["message"])
	}

	if parsed["key"] != "value" {
		t.Errorf("Expected key 'value', got '%v'", parsed["key"])
	}
}
