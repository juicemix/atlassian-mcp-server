package application

import (
	"context"
	"testing"
	"time"

	"atlassian-mcp-server/internal/domain"
)

// TestServerIntegration_FullFlow tests the complete server flow from request to response.
func TestServerIntegration_FullFlow(t *testing.T) {
	// Create transport
	transport := newMockTransport()

	// Create auth manager
	authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{
		"jira": {
			Type:     domain.BasicAuth,
			Username: "testuser",
			Password: "testpass",
		},
	})

	// Create handler with mock
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
				{
					Type: "text",
					Text: `{"id":"10001","key":"TEST-1","fields":{"summary":"Test issue"}}`,
				},
			},
		},
	}

	// Create router
	router := NewRequestRouter(jiraHandler)

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

	// Create server
	server := NewServer(transport, router, authManager, config)

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := server.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Test 1: Initialize
	t.Run("Initialize", func(t *testing.T) {
		req := &domain.Request{
			JSONRPC: "2.0",
			ID:      1,
			Method:  "initialize",
			Params:  map[string]interface{}{},
		}

		transport.sendRequest(req)
		time.Sleep(50 * time.Millisecond)

		resp := transport.getLastResponse()
		if resp == nil {
			t.Fatal("No response received")
		}

		if resp.Error != nil {
			t.Fatalf("Unexpected error: %v", resp.Error)
		}

		result, ok := resp.Result.(map[string]interface{})
		if !ok {
			t.Fatal("Result is not a map")
		}

		if result["protocolVersion"] == nil {
			t.Error("Missing protocolVersion")
		}
	})

	// Test 2: List tools
	t.Run("ListTools", func(t *testing.T) {
		req := &domain.Request{
			JSONRPC: "2.0",
			ID:      2,
			Method:  "tools/list",
		}

		transport.sendRequest(req)
		time.Sleep(50 * time.Millisecond)

		resp := transport.getLastResponse()
		if resp == nil {
			t.Fatal("No response received")
		}

		if resp.Error != nil {
			t.Fatalf("Unexpected error: %v", resp.Error)
		}

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
	})

	// Test 3: Call tool successfully
	t.Run("CallTool_Success", func(t *testing.T) {
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
		time.Sleep(50 * time.Millisecond)

		resp := transport.getLastResponse()
		if resp == nil {
			t.Fatal("No response received")
		}

		if resp.Error != nil {
			t.Fatalf("Unexpected error: %v", resp.Error)
		}

		if resp.Result == nil {
			t.Fatal("Result is nil")
		}
	})

	// Test 4: Authentication validation
	t.Run("AuthenticationValidation", func(t *testing.T) {
		// Create a server without credentials
		transportNoAuth := newMockTransport()
		authManagerNoAuth := domain.NewAuthenticationManager(map[string]*domain.Credentials{})
		serverNoAuth := NewServer(transportNoAuth, router, authManagerNoAuth, config)

		ctxNoAuth, cancelNoAuth := context.WithCancel(context.Background())
		defer cancelNoAuth()

		if err := serverNoAuth.Start(ctxNoAuth); err != nil {
			t.Fatalf("Failed to start server: %v", err)
		}

		req := &domain.Request{
			JSONRPC: "2.0",
			ID:      4,
			Method:  "tools/call",
			Params: map[string]interface{}{
				"name": "jira_get_issue",
				"arguments": map[string]interface{}{
					"issueKey": "TEST-1",
				},
			},
		}

		transportNoAuth.sendRequest(req)
		time.Sleep(50 * time.Millisecond)

		resp := transportNoAuth.getLastResponse()
		if resp == nil {
			t.Fatal("No response received")
		}

		if resp.Error == nil {
			t.Fatal("Expected authentication error")
		}

		if resp.Error.Code != domain.AuthenticationError {
			t.Errorf("Expected error code %d, got %d", domain.AuthenticationError, resp.Error.Code)
		}
	})

	// Test 5: Invalid request handling
	t.Run("InvalidRequest", func(t *testing.T) {
		req := &domain.Request{
			JSONRPC: "1.0", // Invalid version
			ID:      5,
			Method:  "initialize",
		}

		transport.sendRequest(req)
		time.Sleep(50 * time.Millisecond)

		resp := transport.getLastResponse()
		if resp == nil {
			t.Fatal("No response received")
		}

		if resp.Error == nil {
			t.Fatal("Expected error for invalid JSONRPC version")
		}

		if resp.Error.Code != domain.InvalidRequest {
			t.Errorf("Expected error code %d, got %d", domain.InvalidRequest, resp.Error.Code)
		}
	})

	// Clean up
	if err := server.Close(); err != nil {
		t.Errorf("Failed to close server: %v", err)
	}
}

// TestServerIntegration_ConcurrentRequests tests handling of concurrent requests.
func TestServerIntegration_ConcurrentRequests(t *testing.T) {
	transport := newMockTransport()

	jiraHandler := &mockToolHandler{
		name: "jira",
		tools: []domain.ToolDefinition{
			{Name: "jira_get_issue", Description: "Get issue"},
		},
		response: &domain.ToolResponse{
			Content: []domain.ContentBlock{{Type: "text", Text: "Success"}},
		},
	}

	router := NewRequestRouter(jiraHandler)

	authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{
		"jira": {
			Type:     domain.BasicAuth,
			Username: "test",
			Password: "test",
		},
	})

	config := &domain.Config{
		Transport: domain.TransportConfig{Type: "stdio"},
	}

	server := NewServer(transport, router, authManager, config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := server.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Send multiple requests concurrently
	numRequests := 10
	for i := 0; i < numRequests; i++ {
		req := &domain.Request{
			JSONRPC: "2.0",
			ID:      i,
			Method:  "tools/call",
			Params: map[string]interface{}{
				"name": "jira_get_issue",
				"arguments": map[string]interface{}{
					"issueKey": "TEST-1",
				},
			},
		}
		transport.sendRequest(req)
	}

	// Wait for all responses
	time.Sleep(200 * time.Millisecond)

	// Verify we got responses
	responses := transport.getAllResponses()
	if len(responses) < numRequests {
		t.Errorf("Expected %d responses, got %d", numRequests, len(responses))
	}
}
