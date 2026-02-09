package application

import (
	"context"
	"encoding/json"
	"testing"

	"atlassian-mcp-server/internal/domain"
	"atlassian-mcp-server/internal/infrastructure"
)

// TestJiraHandler_IntegrationWithRealResponseMapper tests the JiraHandler
// with the actual DefaultResponseMapper implementation to ensure proper integration.
func TestJiraHandler_IntegrationWithRealResponseMapper(t *testing.T) {
	server := setupMockJiraServer()
	defer server.Close()

	client := infrastructure.NewJiraClient(server.URL, server.Client())
	mapper := domain.NewResponseMapper()
	handler := NewJiraHandler(client, mapper, nil, "")

	t.Run("GetIssue with real mapper", func(t *testing.T) {
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

		// Verify the response is valid JSON
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(resp.Content[0].Text), &jsonData); err != nil {
			t.Fatalf("response is not valid JSON: %v", err)
		}

		// Verify key fields are present
		if _, ok := jsonData["key"]; !ok {
			t.Error("expected 'key' field in response")
		}
		if _, ok := jsonData["fields"]; !ok {
			t.Error("expected 'fields' field in response")
		}
	})

	t.Run("SearchJQL with pagination", func(t *testing.T) {
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

		// The real mapper should include pagination info as a second content block
		if len(resp.Content) < 2 {
			t.Fatal("expected pagination info in response")
		}

		// First block should be the search results
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(resp.Content[0].Text), &jsonData); err != nil {
			t.Fatalf("response is not valid JSON: %v", err)
		}

		// Verify search result fields
		if _, ok := jsonData["issues"]; !ok {
			t.Error("expected 'issues' field in response")
		}
		if _, ok := jsonData["total"]; !ok {
			t.Error("expected 'total' field in response")
		}

		// Second block should be pagination info
		paginationText := resp.Content[1].Text
		if paginationText == "" {
			t.Error("expected pagination text in second content block")
		}
	})

	t.Run("CreateIssue with real mapper", func(t *testing.T) {
		req := &domain.ToolRequest{
			Name: ToolJiraCreateIssue,
			Arguments: map[string]interface{}{
				"projectKey": "TEST",
				"summary":    "New test issue",
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

		// Verify the response contains the created issue
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(resp.Content[0].Text), &jsonData); err != nil {
			t.Fatalf("response is not valid JSON: %v", err)
		}

		if _, ok := jsonData["key"]; !ok {
			t.Error("expected 'key' field in created issue response")
		}
	})

	t.Run("Error mapping with real mapper", func(t *testing.T) {
		req := &domain.ToolRequest{
			Name: ToolJiraGetIssue,
			Arguments: map[string]interface{}{
				"issueKey": "NOTFOUND-999",
			},
		}

		_, err := handler.Handle(context.Background(), req)
		if err == nil {
			t.Fatal("expected error for non-existent issue, got nil")
		}

		// Verify it's a domain error
		domainErr, ok := err.(*domain.Error)
		if !ok {
			t.Fatalf("expected domain.Error, got %T", err)
		}

		// Verify error has proper structure
		if domainErr.Code == 0 {
			t.Error("expected non-zero error code")
		}
		if domainErr.Message == "" {
			t.Error("expected error message")
		}
	})
}

// TestJiraHandler_ToolSchemaValidation tests that tool schemas are properly defined
func TestJiraHandler_ToolSchemaValidation(t *testing.T) {
	handler := NewJiraHandler(nil, nil, nil, "")
	tools := handler.ListTools()

	for _, tool := range tools {
		t.Run(tool.Name, func(t *testing.T) {
			// Verify schema has type
			if tool.InputSchema.Type == "" {
				t.Error("tool schema missing type")
			}

			// Verify schema type is "object"
			if tool.InputSchema.Type != "object" {
				t.Errorf("expected schema type 'object', got '%s'", tool.InputSchema.Type)
			}

			// Verify properties exist (except for list_projects which has no params)
			if tool.Name != ToolJiraListProjects {
				if tool.InputSchema.Properties == nil {
					t.Error("tool schema missing properties")
				}
				if len(tool.InputSchema.Properties) == 0 {
					t.Error("tool schema has empty properties")
				}
			}

			// Verify required fields are defined for tools that need them
			switch tool.Name {
			case ToolJiraGetIssue:
				if len(tool.InputSchema.Required) != 1 || tool.InputSchema.Required[0] != "issueKey" {
					t.Error("expected required field 'issueKey'")
				}
			case ToolJiraCreateIssue:
				if len(tool.InputSchema.Required) != 3 {
					t.Error("expected 3 required fields for create issue")
				}
			case ToolJiraSearchJQL:
				if len(tool.InputSchema.Required) != 1 || tool.InputSchema.Required[0] != "jql" {
					t.Error("expected required field 'jql'")
				}
			case ToolJiraAddComment:
				if len(tool.InputSchema.Required) != 2 {
					t.Error("expected 2 required fields for add comment")
				}
			}

			// Verify each property has a description
			for propName, propValue := range tool.InputSchema.Properties {
				propMap, ok := propValue.(map[string]interface{})
				if !ok {
					t.Errorf("property '%s' is not a map", propName)
					continue
				}

				if _, hasDesc := propMap["description"]; !hasDesc {
					t.Errorf("property '%s' missing description", propName)
				}

				if _, hasType := propMap["type"]; !hasType {
					t.Errorf("property '%s' missing type", propName)
				}
			}
		})
	}
}

// TestJiraHandler_AllToolsCovered ensures all tool constants are included in ListTools
func TestJiraHandler_AllToolsCovered(t *testing.T) {
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

	toolMap := make(map[string]bool)
	for _, tool := range tools {
		toolMap[tool.Name] = true
	}

	for _, expectedTool := range expectedTools {
		if !toolMap[expectedTool] {
			t.Errorf("tool '%s' not found in ListTools output", expectedTool)
		}
	}

	// Verify no extra tools
	if len(tools) != len(expectedTools) {
		t.Errorf("expected %d tools, got %d", len(expectedTools), len(tools))
	}
}
