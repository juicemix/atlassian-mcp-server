package application

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"atlassian-mcp-server/internal/domain"
	"atlassian-mcp-server/internal/infrastructure"
)

// TestRouterWithRealHandlers tests the router with actual handler implementations
func TestRouterWithRealHandlers(t *testing.T) {
	// Create mock Atlassian servers
	jiraServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"key":"TEST-123","fields":{"summary":"Test Issue"}}`))
	}))
	defer jiraServer.Close()

	confluenceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"123","title":"Test Page"}`))
	}))
	defer confluenceServer.Close()

	bitbucketServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"values":[{"slug":"test-repo"}]}`))
	}))
	defer bitbucketServer.Close()

	bambooServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"plans":{"plan":[{"key":"TEST-PLAN"}]}}`))
	}))
	defer bambooServer.Close()

	// Create clients
	jiraClient := infrastructure.NewJiraClient(jiraServer.URL, &http.Client{})
	confluenceClient := infrastructure.NewConfluenceClient(confluenceServer.URL, &http.Client{})
	bitbucketClient := infrastructure.NewBitbucketClient(bitbucketServer.URL, &http.Client{})
	bambooClient := infrastructure.NewBambooClient(bambooServer.URL, &http.Client{})

	// Create mapper
	mapper := domain.NewResponseMapper()

	// Create handlers
	jiraHandler := NewJiraHandler(jiraClient, mapper, nil, "")
	confluenceHandler := NewConfluenceHandler(confluenceClient, mapper)
	bitbucketHandler := NewBitbucketHandler(bitbucketClient, mapper)
	bambooHandler := NewBambooHandler(bambooClient, mapper)

	// Create router with all handlers
	router := NewRequestRouter(jiraHandler, confluenceHandler, bitbucketHandler, bambooHandler)

	ctx := context.Background()

	// Test routing to each handler
	testCases := []struct {
		name     string
		toolName string
		args     map[string]interface{}
	}{
		{
			name:     "Jira get issue",
			toolName: "jira_get_issue",
			args:     map[string]interface{}{"issueKey": "TEST-123"},
		},
		{
			name:     "Confluence get page",
			toolName: "confluence_get_page",
			args:     map[string]interface{}{"pageId": "123"},
		},
		{
			name:     "Bitbucket get repositories",
			toolName: "bitbucket_get_repositories",
			args:     map[string]interface{}{"project": "PROJ"},
		},
		{
			name:     "Bamboo get plans",
			toolName: "bamboo_get_plans",
			args:     map[string]interface{}{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &domain.ToolRequest{
				Name:      tc.toolName,
				Arguments: tc.args,
			}

			resp, err := router.Route(ctx, req)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if resp == nil {
				t.Fatal("Expected response, got nil")
			}

			if len(resp.Content) == 0 {
				t.Fatal("Expected content in response, got empty")
			}

			// Verify response has content
			if resp.Content[0].Type != "text" {
				t.Errorf("Expected content type 'text', got '%s'", resp.Content[0].Type)
			}

			if resp.Content[0].Text == "" {
				t.Error("Expected non-empty text content")
			}
		})
	}
}

// TestRouterToolDiscovery tests that the router aggregates tools from all handlers
func TestRouterToolDiscovery(t *testing.T) {
	// Create minimal clients (won't be used for tool discovery)
	jiraClient := infrastructure.NewJiraClient("http://localhost", &http.Client{})
	confluenceClient := infrastructure.NewConfluenceClient("http://localhost", &http.Client{})
	bitbucketClient := infrastructure.NewBitbucketClient("http://localhost", &http.Client{})
	bambooClient := infrastructure.NewBambooClient("http://localhost", &http.Client{})

	// Create mapper
	mapper := domain.NewResponseMapper()

	// Create handlers
	jiraHandler := NewJiraHandler(jiraClient, mapper, nil, "")
	confluenceHandler := NewConfluenceHandler(confluenceClient, mapper)
	bitbucketHandler := NewBitbucketHandler(bitbucketClient, mapper)
	bambooHandler := NewBambooHandler(bambooClient, mapper)

	// Create router with all handlers
	router := NewRequestRouter(jiraHandler, confluenceHandler, bitbucketHandler, bambooHandler)

	// Get all tools
	allTools := router.ListAllTools()

	// Verify we have tools from all handlers
	if len(allTools) == 0 {
		t.Fatal("Expected tools to be returned, got empty list")
	}

	// Count tools by prefix
	toolCounts := make(map[string]int)
	for _, tool := range allTools {
		prefix := router.extractHandlerName(tool.Name)
		toolCounts[prefix]++
	}

	// Verify we have tools from all four handlers
	expectedHandlers := []string{"jira", "confluence", "bitbucket", "bamboo"}
	for _, handler := range expectedHandlers {
		if count, exists := toolCounts[handler]; !exists || count == 0 {
			t.Errorf("Expected tools from '%s' handler, got %d", handler, count)
		}
	}

	// Verify each tool has required fields
	for _, tool := range allTools {
		if tool.Name == "" {
			t.Error("Tool has empty name")
		}
		if tool.Description == "" {
			t.Errorf("Tool '%s' has empty description", tool.Name)
		}
		if tool.InputSchema.Type == "" {
			t.Errorf("Tool '%s' has empty input schema type", tool.Name)
		}
	}
}

// TestRouterErrorHandling tests error handling for various scenarios
func TestRouterErrorHandling(t *testing.T) {
	// Create a handler that will be registered
	jiraClient := infrastructure.NewJiraClient("http://localhost", &http.Client{})
	mapper := domain.NewResponseMapper()
	jiraHandler := NewJiraHandler(jiraClient, mapper, nil, "")

	router := NewRequestRouter(jiraHandler)
	ctx := context.Background()

	testCases := []struct {
		name          string
		toolName      string
		expectedError string
	}{
		{
			name:          "Unknown handler",
			toolName:      "unknown_tool",
			expectedError: "unknown tool: unknown_tool (no handler registered for 'unknown')",
		},
		{
			name:          "Invalid format - no underscore",
			toolName:      "invalidtool",
			expectedError: "invalid tool name format: invalidtool (expected format: <handler>_<operation>)",
		},
		{
			name:          "Empty tool name",
			toolName:      "",
			expectedError: "invalid tool name format:  (expected format: <handler>_<operation>)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &domain.ToolRequest{
				Name:      tc.toolName,
				Arguments: map[string]interface{}{},
			}

			resp, err := router.Route(ctx, req)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}

			if resp != nil {
				t.Errorf("Expected nil response, got: %v", resp)
			}

			if err.Error() != tc.expectedError {
				t.Errorf("Expected error '%s', got '%s'", tc.expectedError, err.Error())
			}
		})
	}
}
