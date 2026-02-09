package application

import (
	"context"
	"testing"

	"atlassian-mcp-server/internal/domain"
)

// mockHandler is a test implementation of ToolHandler
type mockHandler struct {
	name  string
	tools []domain.ToolDefinition
}

func (m *mockHandler) Handle(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error) {
	// Simple mock implementation that echoes the tool name
	return &domain.ToolResponse{
		Content: []domain.ContentBlock{
			{
				Type: "text",
				Text: "Handled by " + m.name + ": " + req.Name,
			},
		},
	}, nil
}

func (m *mockHandler) ListTools() []domain.ToolDefinition {
	return m.tools
}

func (m *mockHandler) ToolName() string {
	return m.name
}

// TestNewRequestRouter tests router creation with multiple handlers
func TestNewRequestRouter(t *testing.T) {
	jiraHandler := &mockHandler{
		name: "jira",
		tools: []domain.ToolDefinition{
			{Name: "jira_get_issue", Description: "Get Jira issue"},
		},
	}

	confluenceHandler := &mockHandler{
		name: "confluence",
		tools: []domain.ToolDefinition{
			{Name: "confluence_get_page", Description: "Get Confluence page"},
		},
	}

	router := NewRequestRouter(jiraHandler, confluenceHandler)

	if router == nil {
		t.Fatal("Expected router to be created, got nil")
	}

	if len(router.handlers) != 2 {
		t.Errorf("Expected 2 handlers, got %d", len(router.handlers))
	}

	// Verify handlers are registered correctly
	if handler, exists := router.GetHandler("jira"); !exists || handler != jiraHandler {
		t.Error("Jira handler not registered correctly")
	}

	if handler, exists := router.GetHandler("confluence"); !exists || handler != confluenceHandler {
		t.Error("Confluence handler not registered correctly")
	}
}

// TestRouteToJiraHandler tests routing to Jira handler
func TestRouteToJiraHandler(t *testing.T) {
	jiraHandler := &mockHandler{
		name: "jira",
		tools: []domain.ToolDefinition{
			{Name: "jira_get_issue", Description: "Get Jira issue"},
		},
	}

	router := NewRequestRouter(jiraHandler)
	ctx := context.Background()

	req := &domain.ToolRequest{
		Name: "jira_get_issue",
		Arguments: map[string]interface{}{
			"issueKey": "TEST-123",
		},
	}

	resp, err := router.Route(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if len(resp.Content) != 1 {
		t.Fatalf("Expected 1 content block, got %d", len(resp.Content))
	}

	expectedText := "Handled by jira: jira_get_issue"
	if resp.Content[0].Text != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, resp.Content[0].Text)
	}
}

// TestRouteToConfluenceHandler tests routing to Confluence handler
func TestRouteToConfluenceHandler(t *testing.T) {
	confluenceHandler := &mockHandler{
		name: "confluence",
		tools: []domain.ToolDefinition{
			{Name: "confluence_create_page", Description: "Create Confluence page"},
		},
	}

	router := NewRequestRouter(confluenceHandler)
	ctx := context.Background()

	req := &domain.ToolRequest{
		Name: "confluence_create_page",
		Arguments: map[string]interface{}{
			"spaceKey": "TEST",
			"title":    "Test Page",
		},
	}

	resp, err := router.Route(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	expectedText := "Handled by confluence: confluence_create_page"
	if resp.Content[0].Text != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, resp.Content[0].Text)
	}
}

// TestRouteToBitbucketHandler tests routing to Bitbucket handler
func TestRouteToBitbucketHandler(t *testing.T) {
	bitbucketHandler := &mockHandler{
		name: "bitbucket",
		tools: []domain.ToolDefinition{
			{Name: "bitbucket_get_repositories", Description: "Get Bitbucket repositories"},
		},
	}

	router := NewRequestRouter(bitbucketHandler)
	ctx := context.Background()

	req := &domain.ToolRequest{
		Name: "bitbucket_get_repositories",
		Arguments: map[string]interface{}{
			"project": "PROJ",
		},
	}

	resp, err := router.Route(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	expectedText := "Handled by bitbucket: bitbucket_get_repositories"
	if resp.Content[0].Text != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, resp.Content[0].Text)
	}
}

// TestRouteToBambooHandler tests routing to Bamboo handler
func TestRouteToBambooHandler(t *testing.T) {
	bambooHandler := &mockHandler{
		name: "bamboo",
		tools: []domain.ToolDefinition{
			{Name: "bamboo_get_plans", Description: "Get Bamboo plans"},
		},
	}

	router := NewRequestRouter(bambooHandler)
	ctx := context.Background()

	req := &domain.ToolRequest{
		Name:      "bamboo_get_plans",
		Arguments: map[string]interface{}{},
	}

	resp, err := router.Route(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	expectedText := "Handled by bamboo: bamboo_get_plans"
	if resp.Content[0].Text != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, resp.Content[0].Text)
	}
}

// TestRouteUnknownTool tests error handling for unknown tool names
func TestRouteUnknownTool(t *testing.T) {
	jiraHandler := &mockHandler{
		name: "jira",
		tools: []domain.ToolDefinition{
			{Name: "jira_get_issue", Description: "Get Jira issue"},
		},
	}

	router := NewRequestRouter(jiraHandler)
	ctx := context.Background()

	req := &domain.ToolRequest{
		Name:      "unknown_tool",
		Arguments: map[string]interface{}{},
	}

	resp, err := router.Route(ctx, req)
	if err == nil {
		t.Fatal("Expected error for unknown tool, got nil")
	}

	if resp != nil {
		t.Errorf("Expected nil response for unknown tool, got: %v", resp)
	}

	expectedError := "unknown tool: unknown_tool (no handler registered for 'unknown')"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

// TestRouteInvalidToolNameFormat tests error handling for invalid tool name format
func TestRouteInvalidToolNameFormat(t *testing.T) {
	jiraHandler := &mockHandler{
		name: "jira",
		tools: []domain.ToolDefinition{
			{Name: "jira_get_issue", Description: "Get Jira issue"},
		},
	}

	router := NewRequestRouter(jiraHandler)
	ctx := context.Background()

	// Test tool name without underscore
	req := &domain.ToolRequest{
		Name:      "invalidtoolname",
		Arguments: map[string]interface{}{},
	}

	resp, err := router.Route(ctx, req)
	if err == nil {
		t.Fatal("Expected error for invalid tool name format, got nil")
	}

	if resp != nil {
		t.Errorf("Expected nil response for invalid tool name, got: %v", resp)
	}

	expectedError := "invalid tool name format: invalidtoolname (expected format: <handler>_<operation>)"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

// TestListAllTools tests tool discovery aggregation
func TestListAllTools(t *testing.T) {
	jiraHandler := &mockHandler{
		name: "jira",
		tools: []domain.ToolDefinition{
			{Name: "jira_get_issue", Description: "Get Jira issue"},
			{Name: "jira_create_issue", Description: "Create Jira issue"},
		},
	}

	confluenceHandler := &mockHandler{
		name: "confluence",
		tools: []domain.ToolDefinition{
			{Name: "confluence_get_page", Description: "Get Confluence page"},
			{Name: "confluence_create_page", Description: "Create Confluence page"},
		},
	}

	bitbucketHandler := &mockHandler{
		name: "bitbucket",
		tools: []domain.ToolDefinition{
			{Name: "bitbucket_get_repositories", Description: "Get Bitbucket repositories"},
		},
	}

	bambooHandler := &mockHandler{
		name: "bamboo",
		tools: []domain.ToolDefinition{
			{Name: "bamboo_get_plans", Description: "Get Bamboo plans"},
		},
	}

	router := NewRequestRouter(jiraHandler, confluenceHandler, bitbucketHandler, bambooHandler)

	allTools := router.ListAllTools()

	expectedCount := 6 // 2 + 2 + 1 + 1
	if len(allTools) != expectedCount {
		t.Errorf("Expected %d tools, got %d", expectedCount, len(allTools))
	}

	// Verify all tools are present
	toolNames := make(map[string]bool)
	for _, tool := range allTools {
		toolNames[tool.Name] = true
	}

	expectedTools := []string{
		"jira_get_issue",
		"jira_create_issue",
		"confluence_get_page",
		"confluence_create_page",
		"bitbucket_get_repositories",
		"bamboo_get_plans",
	}

	for _, expectedTool := range expectedTools {
		if !toolNames[expectedTool] {
			t.Errorf("Expected tool '%s' not found in aggregated tools", expectedTool)
		}
	}
}

// TestListAllToolsEmptyRouter tests tool discovery with no handlers
func TestListAllToolsEmptyRouter(t *testing.T) {
	router := NewRequestRouter()

	allTools := router.ListAllTools()

	if len(allTools) != 0 {
		t.Errorf("Expected 0 tools for empty router, got %d", len(allTools))
	}
}

// TestRouteWithAllFourHandlers tests routing with all four Atlassian handlers registered
func TestRouteWithAllFourHandlers(t *testing.T) {
	jiraHandler := &mockHandler{name: "jira"}
	confluenceHandler := &mockHandler{name: "confluence"}
	bitbucketHandler := &mockHandler{name: "bitbucket"}
	bambooHandler := &mockHandler{name: "bamboo"}

	router := NewRequestRouter(jiraHandler, confluenceHandler, bitbucketHandler, bambooHandler)
	ctx := context.Background()

	testCases := []struct {
		toolName        string
		expectedHandler string
	}{
		{"jira_get_issue", "jira"},
		{"confluence_get_page", "confluence"},
		{"bitbucket_get_repositories", "bitbucket"},
		{"bamboo_get_plans", "bamboo"},
	}

	for _, tc := range testCases {
		t.Run(tc.toolName, func(t *testing.T) {
			req := &domain.ToolRequest{
				Name:      tc.toolName,
				Arguments: map[string]interface{}{},
			}

			resp, err := router.Route(ctx, req)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if resp == nil {
				t.Fatal("Expected response, got nil")
			}

			expectedText := "Handled by " + tc.expectedHandler + ": " + tc.toolName
			if resp.Content[0].Text != expectedText {
				t.Errorf("Expected text '%s', got '%s'", expectedText, resp.Content[0].Text)
			}
		})
	}
}

// TestExtractHandlerName tests the handler name extraction logic
func TestExtractHandlerName(t *testing.T) {
	router := NewRequestRouter()

	testCases := []struct {
		toolName     string
		expectedName string
	}{
		{"jira_get_issue", "jira"},
		{"confluence_create_page", "confluence"},
		{"bitbucket_get_repositories", "bitbucket"},
		{"bamboo_trigger_build", "bamboo"},
		{"jira_search_jql", "jira"},
		{"confluence_search_cql", "confluence"},
		{"invalidname", ""}, // No underscore
		{"", ""},            // Empty string
	}

	for _, tc := range testCases {
		t.Run(tc.toolName, func(t *testing.T) {
			result := router.extractHandlerName(tc.toolName)
			if result != tc.expectedName {
				t.Errorf("For tool name '%s', expected handler '%s', got '%s'",
					tc.toolName, tc.expectedName, result)
			}
		})
	}
}

// TestGetHandler tests the GetHandler method
func TestGetHandler(t *testing.T) {
	jiraHandler := &mockHandler{name: "jira"}
	confluenceHandler := &mockHandler{name: "confluence"}

	router := NewRequestRouter(jiraHandler, confluenceHandler)

	// Test existing handler
	handler, exists := router.GetHandler("jira")
	if !exists {
		t.Error("Expected jira handler to exist")
	}
	if handler != jiraHandler {
		t.Error("Expected to get the same jira handler instance")
	}

	// Test non-existing handler
	handler, exists = router.GetHandler("nonexistent")
	if exists {
		t.Error("Expected nonexistent handler to not exist")
	}
	if handler != nil {
		t.Error("Expected nil handler for nonexistent handler")
	}
}
