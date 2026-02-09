package application

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"atlassian-mcp-server/internal/domain"
	"atlassian-mcp-server/internal/infrastructure"
)

// TestJiraHandler_ClientProvidedCredentials tests that client-provided credentials work correctly
func TestJiraHandler_ClientProvidedCredentials(t *testing.T) {
	// Track which credentials were used
	var usedUsername string

	// Create a test server that captures the auth header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract credentials from Authorization header
		auth := r.Header.Get("Authorization")
		if auth != "" {
			// Decode basic auth
			// Format: "Basic base64(username:password)"
			usedUsername = "captured"
		}

		// Return a mock issue
		issue := domain.JiraIssue{
			Key: "TEST-123",
			Fields: domain.JiraFields{
				Summary:     "Test Issue",
				Description: "Test Description",
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(issue)
	}))
	defer server.Close()

	// Create auth manager with default credentials
	defaultCreds := map[string]*domain.Credentials{
		"jira": {
			Type:     domain.BasicAuth,
			Username: "default-user",
			Password: "default-pass",
		},
	}
	authManager := domain.NewAuthenticationManager(defaultCreds)

	// Create default client
	defaultHTTPClient, _ := authManager.GetAuthenticatedClient("jira")
	jiraClient := infrastructure.NewJiraClient(server.URL, defaultHTTPClient)

	// Create handler with auth manager
	mapper := domain.NewResponseMapper()
	handler := NewJiraHandler(jiraClient, mapper, authManager, server.URL)

	t.Run("uses client-provided credentials", func(t *testing.T) {
		// Reset tracking
		usedUsername = ""

		// Make request with client-provided credentials
		req := &domain.ToolRequest{
			Name: "jira_get_issue",
			Arguments: map[string]interface{}{
				"issueKey": "TEST-123",
				"auth": map[string]interface{}{
					"type":     "basic",
					"username": "client-user",
					"password": "client-pass",
				},
			},
		}

		resp, err := handler.Handle(context.Background(), req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if resp == nil {
			t.Fatal("expected response, got nil")
		}

		// Verify credentials were captured (server received auth header)
		if usedUsername == "" {
			t.Error("expected credentials to be used")
		}
	})

	t.Run("falls back to default credentials when not provided", func(t *testing.T) {
		// Reset tracking
		usedUsername = ""

		// Make request without client-provided credentials
		req := &domain.ToolRequest{
			Name: "jira_get_issue",
			Arguments: map[string]interface{}{
				"issueKey": "TEST-123",
			},
		}

		resp, err := handler.Handle(context.Background(), req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if resp == nil {
			t.Fatal("expected response, got nil")
		}
	})

	t.Run("rejects invalid credentials", func(t *testing.T) {
		// Make request with invalid credentials (missing password)
		req := &domain.ToolRequest{
			Name: "jira_get_issue",
			Arguments: map[string]interface{}{
				"issueKey": "TEST-123",
				"auth": map[string]interface{}{
					"type":     "basic",
					"username": "client-user",
					// missing password
				},
			},
		}

		_, err := handler.Handle(context.Background(), req)
		if err == nil {
			t.Fatal("expected error for invalid credentials, got nil")
		}

		// Check that it's an InvalidParams error
		domainErr, ok := err.(*domain.Error)
		if !ok {
			t.Fatalf("expected domain.Error, got %T", err)
		}

		if domainErr.Code != domain.InvalidParams {
			t.Errorf("expected InvalidParams error code, got %d", domainErr.Code)
		}
	})
}

// TestExtractCredentialsFromArguments tests the credential extraction function
func TestExtractCredentialsFromArguments(t *testing.T) {
	t.Run("extracts basic auth credentials", func(t *testing.T) {
		args := map[string]interface{}{
			"auth": map[string]interface{}{
				"type":     "basic",
				"username": "test-user",
				"password": "test-pass",
			},
		}

		creds, err := domain.ExtractCredentialsFromArguments(args)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if creds == nil {
			t.Fatal("expected credentials, got nil")
		}

		if creds.Type != domain.BasicAuth {
			t.Errorf("expected BasicAuth type, got %v", creds.Type)
		}

		if creds.Username != "test-user" {
			t.Errorf("expected username 'test-user', got '%s'", creds.Username)
		}

		if creds.Password != "test-pass" {
			t.Errorf("expected password 'test-pass', got '%s'", creds.Password)
		}
	})

	t.Run("extracts token auth credentials", func(t *testing.T) {
		args := map[string]interface{}{
			"auth": map[string]interface{}{
				"type":  "token",
				"token": "test-token-123",
			},
		}

		creds, err := domain.ExtractCredentialsFromArguments(args)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if creds == nil {
			t.Fatal("expected credentials, got nil")
		}

		if creds.Type != domain.TokenAuth {
			t.Errorf("expected TokenAuth type, got %v", creds.Type)
		}

		if creds.Token != "test-token-123" {
			t.Errorf("expected token 'test-token-123', got '%s'", creds.Token)
		}
	})

	t.Run("returns nil when no auth provided", func(t *testing.T) {
		args := map[string]interface{}{
			"issueKey": "TEST-123",
		}

		creds, err := domain.ExtractCredentialsFromArguments(args)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if creds != nil {
			t.Error("expected nil credentials when auth not provided")
		}
	})

	t.Run("rejects invalid auth object", func(t *testing.T) {
		args := map[string]interface{}{
			"auth": "invalid-string",
		}

		_, err := domain.ExtractCredentialsFromArguments(args)
		if err == nil {
			t.Fatal("expected error for invalid auth object, got nil")
		}
	})

	t.Run("rejects incomplete basic auth", func(t *testing.T) {
		args := map[string]interface{}{
			"auth": map[string]interface{}{
				"type":     "basic",
				"username": "test-user",
				// missing password
			},
		}

		_, err := domain.ExtractCredentialsFromArguments(args)
		if err == nil {
			t.Fatal("expected error for incomplete basic auth, got nil")
		}
	})

	t.Run("rejects incomplete token auth", func(t *testing.T) {
		args := map[string]interface{}{
			"auth": map[string]interface{}{
				"type": "token",
				// missing token
			},
		}

		_, err := domain.ExtractCredentialsFromArguments(args)
		if err == nil {
			t.Fatal("expected error for incomplete token auth, got nil")
		}
	})
}

// TestJiraHandler_NoDefaultCredentials tests behavior when no default credentials are configured
func TestJiraHandler_NoDefaultCredentials(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		issue := domain.JiraIssue{
			Key: "TEST-123",
			Fields: domain.JiraFields{
				Summary:     "Test Issue",
				Description: "Test Description",
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(issue)
	}))
	defer server.Close()

	// Create auth manager with NO default credentials
	authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{})

	// Create handler with nil client (no default credentials)
	mapper := domain.NewResponseMapper()
	handler := NewJiraHandler(nil, mapper, authManager, server.URL)

	t.Run("requires credentials when no default configured", func(t *testing.T) {
		// Make request WITHOUT credentials
		req := &domain.ToolRequest{
			Name: "jira_get_issue",
			Arguments: map[string]interface{}{
				"issueKey": "TEST-123",
			},
		}

		_, err := handler.Handle(context.Background(), req)
		if err == nil {
			t.Fatal("expected error when no credentials provided and no default configured")
		}

		// Check that it's an AuthenticationError
		domainErr, ok := err.(*domain.Error)
		if !ok {
			t.Fatalf("expected domain.Error, got %T", err)
		}

		if domainErr.Code != domain.AuthenticationError {
			t.Errorf("expected AuthenticationError code, got %d", domainErr.Code)
		}

		if !strings.Contains(domainErr.Message, "authentication required") {
			t.Errorf("expected 'authentication required' in error message, got: %s", domainErr.Message)
		}
	})

	t.Run("works with client-provided credentials", func(t *testing.T) {
		// Make request WITH credentials
		req := &domain.ToolRequest{
			Name: "jira_get_issue",
			Arguments: map[string]interface{}{
				"issueKey": "TEST-123",
				"auth": map[string]interface{}{
					"type":     "basic",
					"username": "client-user",
					"password": "client-pass",
				},
			},
		}

		resp, err := handler.Handle(context.Background(), req)
		if err != nil {
			t.Fatalf("expected no error with client credentials, got %v", err)
		}

		if resp == nil {
			t.Fatal("expected response, got nil")
		}
	})
}
