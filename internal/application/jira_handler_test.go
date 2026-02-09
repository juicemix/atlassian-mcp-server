package application

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"atlassian-mcp-server/internal/domain"
	"atlassian-mcp-server/internal/infrastructure"
)

// mockResponseMapper is a simple mock implementation of ResponseMapper for testing
type mockResponseMapper struct{}

func (m *mockResponseMapper) MapToToolResponse(apiResponse interface{}) (*domain.ToolResponse, error) {
	jsonBytes, _ := json.MarshalIndent(apiResponse, "", "  ")
	return &domain.ToolResponse{
		Content: []domain.ContentBlock{
			{
				Type: "text",
				Text: string(jsonBytes),
			},
		},
	}, nil
}

func (m *mockResponseMapper) MapError(err error) *domain.Error {
	return &domain.Error{
		Code:    domain.APIError,
		Message: err.Error(),
	}
}

// setupMockJiraServer creates a mock Jira server for testing
func setupMockJiraServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		// Get issue
		case r.Method == "GET" && r.URL.Path == "/rest/api/2/issue/TEST-123":
			json.NewEncoder(w).Encode(domain.JiraIssue{
				ID:  "10001",
				Key: "TEST-123",
				Fields: domain.JiraFields{
					Summary:     "Test issue",
					Description: "Test description",
					IssueType: domain.IssueType{
						ID:   "1",
						Name: "Bug",
					},
					Project: domain.Project{
						ID:   "10000",
						Key:  "TEST",
						Name: "Test Project",
					},
					Status: domain.Status{
						ID:   "1",
						Name: "Open",
					},
				},
			})

		// Create issue
		case r.Method == "POST" && r.URL.Path == "/rest/api/2/issue":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(domain.JiraIssue{
				ID:  "10002",
				Key: "TEST-124",
				Fields: domain.JiraFields{
					Summary: "New test issue",
					IssueType: domain.IssueType{
						ID:   "1",
						Name: "Bug",
					},
					Project: domain.Project{
						ID:   "10000",
						Key:  "TEST",
						Name: "Test Project",
					},
				},
			})

		// Update issue
		case r.Method == "PUT" && r.URL.Path == "/rest/api/2/issue/TEST-123":
			w.WriteHeader(http.StatusNoContent)

		// Delete issue
		case r.Method == "DELETE" && r.URL.Path == "/rest/api/2/issue/TEST-123":
			w.WriteHeader(http.StatusNoContent)

		// Search JQL
		case r.Method == "GET" && r.URL.Path == "/rest/api/2/search":
			json.NewEncoder(w).Encode(domain.SearchResults{
				Issues: []domain.JiraIssue{
					{
						ID:  "10001",
						Key: "TEST-123",
						Fields: domain.JiraFields{
							Summary: "Test issue",
						},
					},
				},
				Total:      1,
				StartAt:    0,
				MaxResults: 50,
			})

		// Transition issue
		case r.Method == "POST" && r.URL.Path == "/rest/api/2/issue/TEST-123/transitions":
			w.WriteHeader(http.StatusNoContent)

		// Add comment
		case r.Method == "POST" && r.URL.Path == "/rest/api/2/issue/TEST-123/comment":
			w.WriteHeader(http.StatusCreated)

		// List projects
		case r.Method == "GET" && r.URL.Path == "/rest/api/2/project":
			json.NewEncoder(w).Encode([]domain.Project{
				{
					ID:   "10000",
					Key:  "TEST",
					Name: "Test Project",
				},
			})

		default:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Not found",
			})
		}
	}))
}

func TestJiraHandler_ToolName(t *testing.T) {
	handler := NewJiraHandler(nil, nil, nil, "")
	if handler.ToolName() != "jira" {
		t.Errorf("expected tool name 'jira', got '%s'", handler.ToolName())
	}
}

func TestJiraHandler_ListTools(t *testing.T) {
	handler := NewJiraHandler(nil, nil, nil, "")
	tools := handler.ListTools()

	expectedTools := []string{
		ToolJiraGetIssue,
		ToolJiraCreateIssue,
		ToolJiraUpdateIssue,
		ToolJiraDeleteIssue,
		ToolJiraSearchJQL,
		ToolJiraTransition,
		ToolJiraAddComment,
		ToolJiraListProjects,
	}

	if len(tools) != len(expectedTools) {
		t.Fatalf("expected %d tools, got %d", len(expectedTools), len(tools))
	}

	// Check that all expected tools are present
	toolMap := make(map[string]bool)
	for _, tool := range tools {
		toolMap[tool.Name] = true
	}

	for _, expectedTool := range expectedTools {
		if !toolMap[expectedTool] {
			t.Errorf("expected tool '%s' not found", expectedTool)
		}
	}

	// Verify that each tool has a description and input schema
	for _, tool := range tools {
		if tool.Description == "" {
			t.Errorf("tool '%s' has no description", tool.Name)
		}
		if tool.InputSchema.Type == "" {
			t.Errorf("tool '%s' has no input schema", tool.Name)
		}
	}
}

func TestJiraHandler_HandleGetIssue(t *testing.T) {
	server := setupMockJiraServer()
	defer server.Close()

	client := infrastructure.NewJiraClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewJiraHandler(client, mapper, nil, "")

	req := &domain.ToolRequest{
		Name: ToolJiraGetIssue,
		Arguments: map[string]interface{}{
			"issueKey": "TEST-123",
		},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	if len(resp.Content) == 0 {
		t.Fatal("expected content in response")
	}

	// Verify the response contains the issue key
	if resp.Content[0].Type != "text" {
		t.Errorf("expected content type 'text', got '%s'", resp.Content[0].Type)
	}
}

func TestJiraHandler_HandleGetIssue_MissingParameter(t *testing.T) {
	server := setupMockJiraServer()
	defer server.Close()

	client := infrastructure.NewJiraClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewJiraHandler(client, mapper, nil, "")

	req := &domain.ToolRequest{
		Name:      ToolJiraGetIssue,
		Arguments: map[string]interface{}{},
	}

	_, err := handler.Handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing parameter, got nil")
	}

	domainErr, ok := err.(*domain.Error)
	if !ok {
		t.Fatalf("expected domain.Error, got %T", err)
	}

	if domainErr.Code != domain.InvalidParams {
		t.Errorf("expected error code %d, got %d", domain.InvalidParams, domainErr.Code)
	}
}

func TestJiraHandler_HandleCreateIssue(t *testing.T) {
	server := setupMockJiraServer()
	defer server.Close()

	client := infrastructure.NewJiraClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewJiraHandler(client, mapper, nil, "")

	req := &domain.ToolRequest{
		Name: ToolJiraCreateIssue,
		Arguments: map[string]interface{}{
			"projectKey":  "TEST",
			"summary":     "New test issue",
			"issueType":   "Bug",
			"description": "Test description",
		},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

func TestJiraHandler_HandleCreateIssue_WithAssignee(t *testing.T) {
	server := setupMockJiraServer()
	defer server.Close()

	client := infrastructure.NewJiraClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewJiraHandler(client, mapper, nil, "")

	req := &domain.ToolRequest{
		Name: ToolJiraCreateIssue,
		Arguments: map[string]interface{}{
			"projectKey": "TEST",
			"summary":    "New test issue",
			"issueType":  "Bug",
			"assignee":   "testuser",
		},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

func TestJiraHandler_HandleUpdateIssue(t *testing.T) {
	server := setupMockJiraServer()
	defer server.Close()

	client := infrastructure.NewJiraClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewJiraHandler(client, mapper, nil, "")

	req := &domain.ToolRequest{
		Name: ToolJiraUpdateIssue,
		Arguments: map[string]interface{}{
			"issueKey": "TEST-123",
			"summary":  "Updated summary",
			"assignee": "newuser",
		},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

func TestJiraHandler_HandleDeleteIssue(t *testing.T) {
	server := setupMockJiraServer()
	defer server.Close()

	client := infrastructure.NewJiraClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewJiraHandler(client, mapper, nil, "")

	req := &domain.ToolRequest{
		Name: ToolJiraDeleteIssue,
		Arguments: map[string]interface{}{
			"issueKey": "TEST-123",
		},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

func TestJiraHandler_HandleSearchJQL(t *testing.T) {
	server := setupMockJiraServer()
	defer server.Close()

	client := infrastructure.NewJiraClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewJiraHandler(client, mapper, nil, "")

	req := &domain.ToolRequest{
		Name: ToolJiraSearchJQL,
		Arguments: map[string]interface{}{
			"jql":        "project = TEST",
			"startAt":    float64(0),
			"maxResults": float64(50),
		},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

func TestJiraHandler_HandleTransition(t *testing.T) {
	server := setupMockJiraServer()
	defer server.Close()

	client := infrastructure.NewJiraClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewJiraHandler(client, mapper, nil, "")

	req := &domain.ToolRequest{
		Name: ToolJiraTransition,
		Arguments: map[string]interface{}{
			"issueKey":     "TEST-123",
			"transitionId": "21",
		},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

func TestJiraHandler_HandleTransition_WithName(t *testing.T) {
	server := setupMockJiraServer()
	defer server.Close()

	client := infrastructure.NewJiraClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewJiraHandler(client, mapper, nil, "")

	req := &domain.ToolRequest{
		Name: ToolJiraTransition,
		Arguments: map[string]interface{}{
			"issueKey":       "TEST-123",
			"transitionName": "Done",
		},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

func TestJiraHandler_HandleTransition_MissingBothIdAndName(t *testing.T) {
	server := setupMockJiraServer()
	defer server.Close()

	client := infrastructure.NewJiraClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewJiraHandler(client, mapper, nil, "")

	req := &domain.ToolRequest{
		Name: ToolJiraTransition,
		Arguments: map[string]interface{}{
			"issueKey": "TEST-123",
		},
	}

	_, err := handler.Handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing transition ID and name, got nil")
	}

	domainErr, ok := err.(*domain.Error)
	if !ok {
		t.Fatalf("expected domain.Error, got %T", err)
	}

	if domainErr.Code != domain.InvalidParams {
		t.Errorf("expected error code %d, got %d", domain.InvalidParams, domainErr.Code)
	}
}

func TestJiraHandler_HandleAddComment(t *testing.T) {
	server := setupMockJiraServer()
	defer server.Close()

	client := infrastructure.NewJiraClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewJiraHandler(client, mapper, nil, "")

	req := &domain.ToolRequest{
		Name: ToolJiraAddComment,
		Arguments: map[string]interface{}{
			"issueKey": "TEST-123",
			"body":     "This is a test comment",
		},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

func TestJiraHandler_HandleListProjects(t *testing.T) {
	server := setupMockJiraServer()
	defer server.Close()

	client := infrastructure.NewJiraClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewJiraHandler(client, mapper, nil, "")

	req := &domain.ToolRequest{
		Name:      ToolJiraListProjects,
		Arguments: map[string]interface{}{},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

func TestJiraHandler_HandleUnknownTool(t *testing.T) {
	server := setupMockJiraServer()
	defer server.Close()

	client := infrastructure.NewJiraClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewJiraHandler(client, mapper, nil, "")

	req := &domain.ToolRequest{
		Name:      "jira_unknown_tool",
		Arguments: map[string]interface{}{},
	}

	_, err := handler.Handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for unknown tool, got nil")
	}

	domainErr, ok := err.(*domain.Error)
	if !ok {
		t.Fatalf("expected domain.Error, got %T", err)
	}

	if domainErr.Code != domain.MethodNotFound {
		t.Errorf("expected error code %d, got %d", domain.MethodNotFound, domainErr.Code)
	}
}

func TestJiraHandler_ParameterValidation_InvalidType(t *testing.T) {
	server := setupMockJiraServer()
	defer server.Close()

	client := infrastructure.NewJiraClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewJiraHandler(client, mapper, nil, "")

	req := &domain.ToolRequest{
		Name: ToolJiraGetIssue,
		Arguments: map[string]interface{}{
			"issueKey": 123, // Should be string
		},
	}

	_, err := handler.Handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for invalid parameter type, got nil")
	}

	domainErr, ok := err.(*domain.Error)
	if !ok {
		t.Fatalf("expected domain.Error, got %T", err)
	}

	if domainErr.Code != domain.InvalidParams {
		t.Errorf("expected error code %d, got %d", domain.InvalidParams, domainErr.Code)
	}
}

func TestJiraHandler_IntegerParameterValidation(t *testing.T) {
	server := setupMockJiraServer()
	defer server.Close()

	client := infrastructure.NewJiraClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewJiraHandler(client, mapper, nil, "")

	// Test with float64 (from JSON)
	req := &domain.ToolRequest{
		Name: ToolJiraSearchJQL,
		Arguments: map[string]interface{}{
			"jql":        "project = TEST",
			"maxResults": float64(10),
		},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error with float64: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	// Test with int
	req2 := &domain.ToolRequest{
		Name: ToolJiraSearchJQL,
		Arguments: map[string]interface{}{
			"jql":        "project = TEST",
			"maxResults": 10,
		},
	}

	resp2, err := handler.Handle(context.Background(), req2)
	if err != nil {
		t.Fatalf("unexpected error with int: %v", err)
	}

	if resp2 == nil {
		t.Fatal("expected response, got nil")
	}

	// Test with invalid type
	req3 := &domain.ToolRequest{
		Name: ToolJiraSearchJQL,
		Arguments: map[string]interface{}{
			"jql":        "project = TEST",
			"maxResults": "invalid",
		},
	}

	_, err = handler.Handle(context.Background(), req3)
	if err == nil {
		t.Fatal("expected error for invalid integer type, got nil")
	}

	domainErr, ok := err.(*domain.Error)
	if !ok {
		t.Fatalf("expected domain.Error, got %T", err)
	}

	if domainErr.Code != domain.InvalidParams {
		t.Errorf("expected error code %d, got %d", domain.InvalidParams, domainErr.Code)
	}
}

func TestJiraHandler_NilArguments(t *testing.T) {
	server := setupMockJiraServer()
	defer server.Close()

	client := infrastructure.NewJiraClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewJiraHandler(client, mapper, nil, "")

	req := &domain.ToolRequest{
		Name:      ToolJiraListProjects,
		Arguments: nil,
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

// TestJiraHandler_CreateIssue_MissingRequiredParameters tests validation of all required parameters
func TestJiraHandler_CreateIssue_MissingRequiredParameters(t *testing.T) {
	server := setupMockJiraServer()
	defer server.Close()

	client := infrastructure.NewJiraClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewJiraHandler(client, mapper, nil, "")

	testCases := []struct {
		name      string
		arguments map[string]interface{}
		missing   string
	}{
		{
			name:      "missing projectKey",
			arguments: map[string]interface{}{"summary": "Test", "issueType": "Bug"},
			missing:   "projectKey",
		},
		{
			name:      "missing summary",
			arguments: map[string]interface{}{"projectKey": "TEST", "issueType": "Bug"},
			missing:   "summary",
		},
		{
			name:      "missing issueType",
			arguments: map[string]interface{}{"projectKey": "TEST", "summary": "Test"},
			missing:   "issueType",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &domain.ToolRequest{
				Name:      ToolJiraCreateIssue,
				Arguments: tc.arguments,
			}

			_, err := handler.Handle(context.Background(), req)
			if err == nil {
				t.Fatalf("expected error for missing %s, got nil", tc.missing)
			}

			domainErr, ok := err.(*domain.Error)
			if !ok {
				t.Fatalf("expected domain.Error, got %T", err)
			}

			if domainErr.Code != domain.InvalidParams {
				t.Errorf("expected error code %d, got %d", domain.InvalidParams, domainErr.Code)
			}
		})
	}
}

// TestJiraHandler_UpdateIssue_MissingIssueKey tests validation for update operation
func TestJiraHandler_UpdateIssue_MissingIssueKey(t *testing.T) {
	server := setupMockJiraServer()
	defer server.Close()

	client := infrastructure.NewJiraClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewJiraHandler(client, mapper, nil, "")

	req := &domain.ToolRequest{
		Name: ToolJiraUpdateIssue,
		Arguments: map[string]interface{}{
			"summary": "Updated summary",
		},
	}

	_, err := handler.Handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing issueKey, got nil")
	}

	domainErr, ok := err.(*domain.Error)
	if !ok {
		t.Fatalf("expected domain.Error, got %T", err)
	}

	if domainErr.Code != domain.InvalidParams {
		t.Errorf("expected error code %d, got %d", domain.InvalidParams, domainErr.Code)
	}
}

// TestJiraHandler_SearchJQL_MissingJQL tests validation for search operation
func TestJiraHandler_SearchJQL_MissingJQL(t *testing.T) {
	server := setupMockJiraServer()
	defer server.Close()

	client := infrastructure.NewJiraClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewJiraHandler(client, mapper, nil, "")

	req := &domain.ToolRequest{
		Name:      ToolJiraSearchJQL,
		Arguments: map[string]interface{}{},
	}

	_, err := handler.Handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing jql, got nil")
	}

	domainErr, ok := err.(*domain.Error)
	if !ok {
		t.Fatalf("expected domain.Error, got %T", err)
	}

	if domainErr.Code != domain.InvalidParams {
		t.Errorf("expected error code %d, got %d", domain.InvalidParams, domainErr.Code)
	}
}

// TestJiraHandler_AddComment_MissingParameters tests validation for comment operation
func TestJiraHandler_AddComment_MissingParameters(t *testing.T) {
	server := setupMockJiraServer()
	defer server.Close()

	client := infrastructure.NewJiraClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewJiraHandler(client, mapper, nil, "")

	testCases := []struct {
		name      string
		arguments map[string]interface{}
		missing   string
	}{
		{
			name:      "missing issueKey",
			arguments: map[string]interface{}{"body": "Test comment"},
			missing:   "issueKey",
		},
		{
			name:      "missing body",
			arguments: map[string]interface{}{"issueKey": "TEST-123"},
			missing:   "body",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &domain.ToolRequest{
				Name:      ToolJiraAddComment,
				Arguments: tc.arguments,
			}

			_, err := handler.Handle(context.Background(), req)
			if err == nil {
				t.Fatalf("expected error for missing %s, got nil", tc.missing)
			}

			domainErr, ok := err.(*domain.Error)
			if !ok {
				t.Fatalf("expected domain.Error, got %T", err)
			}

			if domainErr.Code != domain.InvalidParams {
				t.Errorf("expected error code %d, got %d", domain.InvalidParams, domainErr.Code)
			}
		})
	}
}

// TestJiraHandler_ParameterTypeValidation tests type validation for various parameters
func TestJiraHandler_ParameterTypeValidation(t *testing.T) {
	server := setupMockJiraServer()
	defer server.Close()

	client := infrastructure.NewJiraClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewJiraHandler(client, mapper, nil, "")

	testCases := []struct {
		name      string
		toolName  string
		arguments map[string]interface{}
		paramName string
	}{
		{
			name:      "createIssue with non-string projectKey",
			toolName:  ToolJiraCreateIssue,
			arguments: map[string]interface{}{"projectKey": 123, "summary": "Test", "issueType": "Bug"},
			paramName: "projectKey",
		},
		{
			name:      "createIssue with non-string summary",
			toolName:  ToolJiraCreateIssue,
			arguments: map[string]interface{}{"projectKey": "TEST", "summary": 123, "issueType": "Bug"},
			paramName: "summary",
		},
		{
			name:      "createIssue with non-string issueType",
			toolName:  ToolJiraCreateIssue,
			arguments: map[string]interface{}{"projectKey": "TEST", "summary": "Test", "issueType": 123},
			paramName: "issueType",
		},
		{
			name:      "updateIssue with non-string issueKey",
			toolName:  ToolJiraUpdateIssue,
			arguments: map[string]interface{}{"issueKey": 123, "summary": "Test"},
			paramName: "issueKey",
		},
		{
			name:      "addComment with non-string body",
			toolName:  ToolJiraAddComment,
			arguments: map[string]interface{}{"issueKey": "TEST-123", "body": 123},
			paramName: "body",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &domain.ToolRequest{
				Name:      tc.toolName,
				Arguments: tc.arguments,
			}

			_, err := handler.Handle(context.Background(), req)
			if err == nil {
				t.Fatalf("expected error for invalid type of %s, got nil", tc.paramName)
			}

			domainErr, ok := err.(*domain.Error)
			if !ok {
				t.Fatalf("expected domain.Error, got %T", err)
			}

			if domainErr.Code != domain.InvalidParams {
				t.Errorf("expected error code %d, got %d", domain.InvalidParams, domainErr.Code)
			}
		})
	}
}

// TestJiraHandler_EdgeCases tests edge cases like empty strings and special characters
func TestJiraHandler_EdgeCases(t *testing.T) {
	server := setupMockJiraServer()
	defer server.Close()

	client := infrastructure.NewJiraClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewJiraHandler(client, mapper, nil, "")

	t.Run("empty string parameters", func(t *testing.T) {
		// Empty strings should be accepted (they're still strings)
		req := &domain.ToolRequest{
			Name: ToolJiraUpdateIssue,
			Arguments: map[string]interface{}{
				"issueKey":    "TEST-123",
				"summary":     "",
				"description": "",
			},
		}

		// This should not fail validation (empty strings are valid)
		_, err := handler.Handle(context.Background(), req)
		// The error here would be from the API, not from validation
		// We're just checking that validation doesn't reject empty strings
		if err != nil {
			domainErr, ok := err.(*domain.Error)
			if ok && domainErr.Code == domain.InvalidParams {
				t.Error("empty strings should not fail parameter validation")
			}
		}
	})

	t.Run("special characters in parameters", func(t *testing.T) {
		req := &domain.ToolRequest{
			Name: ToolJiraSearchJQL,
			Arguments: map[string]interface{}{
				"jql": "project = TEST AND summary ~ \"special chars: !@#$%^&*()\"",
			},
		}

		// Should not fail validation
		_, err := handler.Handle(context.Background(), req)
		if err != nil {
			domainErr, ok := err.(*domain.Error)
			if ok && domainErr.Code == domain.InvalidParams {
				t.Error("special characters should not fail parameter validation")
			}
		}
	})
}

// TestJiraHandler_OptionalParameterCombinations tests various combinations of optional parameters
func TestJiraHandler_OptionalParameterCombinations(t *testing.T) {
	server := setupMockJiraServer()
	defer server.Close()

	client := infrastructure.NewJiraClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewJiraHandler(client, mapper, nil, "")

	t.Run("createIssue with all optional parameters", func(t *testing.T) {
		req := &domain.ToolRequest{
			Name: ToolJiraCreateIssue,
			Arguments: map[string]interface{}{
				"projectKey":  "TEST",
				"summary":     "Test issue",
				"issueType":   "Bug",
				"description": "Test description",
				"assignee":    "testuser",
			},
		}

		resp, err := handler.Handle(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected response, got nil")
		}
	})

	t.Run("createIssue with no optional parameters", func(t *testing.T) {
		req := &domain.ToolRequest{
			Name: ToolJiraCreateIssue,
			Arguments: map[string]interface{}{
				"projectKey": "TEST",
				"summary":    "Test issue",
				"issueType":  "Bug",
			},
		}

		resp, err := handler.Handle(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected response, got nil")
		}
	})

	t.Run("updateIssue with only summary", func(t *testing.T) {
		req := &domain.ToolRequest{
			Name: ToolJiraUpdateIssue,
			Arguments: map[string]interface{}{
				"issueKey": "TEST-123",
				"summary":  "Updated summary",
			},
		}

		resp, err := handler.Handle(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected response, got nil")
		}
	})

	t.Run("updateIssue with only description", func(t *testing.T) {
		req := &domain.ToolRequest{
			Name: ToolJiraUpdateIssue,
			Arguments: map[string]interface{}{
				"issueKey":    "TEST-123",
				"description": "Updated description",
			},
		}

		resp, err := handler.Handle(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected response, got nil")
		}
	})

	t.Run("searchJQL with pagination parameters", func(t *testing.T) {
		req := &domain.ToolRequest{
			Name: ToolJiraSearchJQL,
			Arguments: map[string]interface{}{
				"jql":        "project = TEST",
				"startAt":    float64(10),
				"maxResults": float64(25),
			},
		}

		resp, err := handler.Handle(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected response, got nil")
		}
	})

	t.Run("searchJQL without pagination parameters", func(t *testing.T) {
		req := &domain.ToolRequest{
			Name: ToolJiraSearchJQL,
			Arguments: map[string]interface{}{
				"jql": "project = TEST",
			},
		}

		resp, err := handler.Handle(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected response, got nil")
		}
	})
}

// TestJiraHandler_HTTPErrorHandling tests that various HTTP errors are properly mapped
func TestJiraHandler_HTTPErrorHandling(t *testing.T) {
	testCases := []struct {
		name           string
		statusCode     int
		expectedErrMsg string
	}{
		{
			name:           "404 Not Found",
			statusCode:     http.StatusNotFound,
			expectedErrMsg: "404",
		},
		{
			name:           "401 Unauthorized",
			statusCode:     http.StatusUnauthorized,
			expectedErrMsg: "401",
		},
		{
			name:           "403 Forbidden",
			statusCode:     http.StatusForbidden,
			expectedErrMsg: "403",
		},
		{
			name:           "500 Internal Server Error",
			statusCode:     http.StatusInternalServerError,
			expectedErrMsg: "500",
		},
		{
			name:           "503 Service Unavailable",
			statusCode:     http.StatusServiceUnavailable,
			expectedErrMsg: "503",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock server that returns the specific error code
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "Test error",
				})
			}))
			defer server.Close()

			client := infrastructure.NewJiraClient(server.URL, server.Client())
			mapper := &mockResponseMapper{}
			handler := NewJiraHandler(client, mapper, nil, "")

			req := &domain.ToolRequest{
				Name: ToolJiraGetIssue,
				Arguments: map[string]interface{}{
					"issueKey": "TEST-123",
				},
			}

			_, err := handler.Handle(context.Background(), req)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			// Verify the error contains the status code
			if !contains(err.Error(), tc.expectedErrMsg) {
				t.Errorf("expected error to contain '%s', got: %v", tc.expectedErrMsg, err)
			}
		})
	}
}

// TestJiraHandler_ClientErrorPropagation tests that client errors are properly propagated
func TestJiraHandler_ClientErrorPropagation(t *testing.T) {
	// Create a server that closes immediately to simulate network error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Close the connection immediately
		hj, ok := w.(http.Hijacker)
		if ok {
			conn, _, _ := hj.Hijack()
			conn.Close()
		}
	}))
	server.Close() // Close the server to ensure connection failures

	client := infrastructure.NewJiraClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewJiraHandler(client, mapper, nil, "")

	req := &domain.ToolRequest{
		Name: ToolJiraGetIssue,
		Arguments: map[string]interface{}{
			"issueKey": "TEST-123",
		},
	}

	_, err := handler.Handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for closed connection, got nil")
	}
}

// TestJiraHandler_ResponseMapperIntegration tests integration with ResponseMapper
func TestJiraHandler_ResponseMapperIntegration(t *testing.T) {
	server := setupMockJiraServer()
	defer server.Close()

	client := infrastructure.NewJiraClient(server.URL, server.Client())

	// Use the real response mapper
	mapper := domain.NewResponseMapper()
	handler := NewJiraHandler(client, mapper, nil, "")

	t.Run("successful response mapping", func(t *testing.T) {
		req := &domain.ToolRequest{
			Name: ToolJiraGetIssue,
			Arguments: map[string]interface{}{
				"issueKey": "TEST-123",
			},
		}

		resp, err := handler.Handle(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp == nil {
			t.Fatal("expected response, got nil")
		}

		if len(resp.Content) == 0 {
			t.Fatal("expected content in response")
		}

		// Verify the content is valid JSON
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(resp.Content[0].Text), &jsonData); err != nil {
			t.Fatalf("response is not valid JSON: %v", err)
		}
	})

	t.Run("error response mapping", func(t *testing.T) {
		// Create a server that returns 404
		errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Issue not found",
			})
		}))
		defer errorServer.Close()

		errorClient := infrastructure.NewJiraClient(errorServer.URL, errorServer.Client())
		errorHandler := NewJiraHandler(errorClient, mapper, nil, "")

		req := &domain.ToolRequest{
			Name: ToolJiraGetIssue,
			Arguments: map[string]interface{}{
				"issueKey": "NOTFOUND-999",
			},
		}

		_, err := errorHandler.Handle(context.Background(), req)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		// Verify it's a domain error with proper structure
		domainErr, ok := err.(*domain.Error)
		if !ok {
			t.Fatalf("expected domain.Error, got %T", err)
		}

		if domainErr.Code == 0 {
			t.Error("expected non-zero error code")
		}

		if domainErr.Message == "" {
			t.Error("expected error message")
		}
	})
}

// TestJiraHandler_AllToolsHaveValidSchemas ensures all tools have properly defined schemas
func TestJiraHandler_AllToolsHaveValidSchemas(t *testing.T) {
	handler := NewJiraHandler(nil, nil, nil, "")
	tools := handler.ListTools()

	for _, tool := range tools {
		t.Run(tool.Name, func(t *testing.T) {
			// Verify basic schema structure
			if tool.InputSchema.Type != "object" {
				t.Errorf("expected schema type 'object', got '%s'", tool.InputSchema.Type)
			}

			// Verify required fields are in properties
			for _, requiredField := range tool.InputSchema.Required {
				if _, exists := tool.InputSchema.Properties[requiredField]; !exists {
					t.Errorf("required field '%s' not found in properties", requiredField)
				}
			}

			// Verify all properties have type and description
			for propName, propValue := range tool.InputSchema.Properties {
				propMap, ok := propValue.(map[string]interface{})
				if !ok {
					t.Errorf("property '%s' is not a map", propName)
					continue
				}

				if _, hasType := propMap["type"]; !hasType {
					t.Errorf("property '%s' missing type", propName)
				}

				if _, hasDesc := propMap["description"]; !hasDesc {
					t.Errorf("property '%s' missing description", propName)
				}
			}
		})
	}
}

// TestJiraHandler_ContextPropagation tests that context is properly propagated
func TestJiraHandler_ContextPropagation(t *testing.T) {
	server := setupMockJiraServer()
	defer server.Close()

	client := infrastructure.NewJiraClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewJiraHandler(client, mapper, nil, "")

	// Create a context with a value
	ctx := context.WithValue(context.Background(), "test-key", "test-value")

	req := &domain.ToolRequest{
		Name: ToolJiraGetIssue,
		Arguments: map[string]interface{}{
			"issueKey": "TEST-123",
		},
	}

	// This should not panic or error due to context
	resp, err := handler.Handle(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
