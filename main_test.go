package main

import (
	"os"
	"testing"

	"atlassian-mcp-server/internal/domain"
)

// TestConfigurationLoading tests that configuration can be loaded successfully
func TestConfigurationLoading(t *testing.T) {
	// Create a temporary config file
	configContent := `
transport:
  type: stdio

tools:
  jira:
    base_url: https://jira.example.com
    auth:
      type: basic
      username: testuser
      password: testpass
`

	// Write to temporary file
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	// Load configuration
	config, err := domain.LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// Verify configuration
	if config.Transport.Type != "stdio" {
		t.Errorf("Expected transport type 'stdio', got '%s'", config.Transport.Type)
	}

	if config.Tools.Jira == nil {
		t.Fatal("Expected Jira to be configured")
	}

	if config.Tools.Jira.BaseURL != "https://jira.example.com" {
		t.Errorf("Expected Jira base URL 'https://jira.example.com', got '%s'", config.Tools.Jira.BaseURL)
	}

	if config.Tools.Jira.Auth.Type != "basic" {
		t.Errorf("Expected auth type 'basic', got '%s'", config.Tools.Jira.Auth.Type)
	}
}

// TestAuthenticationManagerCreation tests that authentication manager can be created from config
func TestAuthenticationManagerCreation(t *testing.T) {
	// Create a test configuration
	config := &domain.Config{
		Transport: domain.TransportConfig{
			Type: "stdio",
		},
		Tools: domain.ToolsConfig{
			Jira: &domain.ToolConfig{
				BaseURL: "https://jira.example.com",
				Auth: domain.AuthConfig{
					Type:     "basic",
					Username: "testuser",
					Password: "testpass",
				},
			},
		},
	}

	// Create authentication manager
	authManager := domain.NewAuthenticationManagerFromConfig(config)
	if authManager == nil {
		t.Fatal("Failed to create authentication manager")
	}

	// Validate credentials
	err := authManager.ValidateCredentials("jira")
	if err != nil {
		t.Errorf("Failed to validate Jira credentials: %v", err)
	}

	// Test invalid tool
	err = authManager.ValidateCredentials("invalid")
	if err == nil {
		t.Error("Expected error for invalid tool, got nil")
	}
}

// TestMultipleToolsConfiguration tests configuration with multiple tools
func TestMultipleToolsConfiguration(t *testing.T) {
	configContent := `
transport:
  type: http
  http:
    host: localhost
    port: 8080

tools:
  jira:
    base_url: https://jira.example.com
    auth:
      type: basic
      username: jirauser
      password: jirapass
  
  confluence:
    base_url: https://confluence.example.com
    auth:
      type: token
      token: confluence-token
  
  bitbucket:
    base_url: https://bitbucket.example.com
    auth:
      type: basic
      username: bitbucketuser
      password: bitbucketpass
`

	// Write to temporary file
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	// Load configuration
	config, err := domain.LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// Verify HTTP transport
	if config.Transport.Type != "http" {
		t.Errorf("Expected transport type 'http', got '%s'", config.Transport.Type)
	}
	if config.Transport.HTTP.Host != "localhost" {
		t.Errorf("Expected HTTP host 'localhost', got '%s'", config.Transport.HTTP.Host)
	}
	if config.Transport.HTTP.Port != 8080 {
		t.Errorf("Expected HTTP port 8080, got %d", config.Transport.HTTP.Port)
	}

	// Verify all tools are configured
	if config.Tools.Jira == nil {
		t.Error("Expected Jira to be configured")
	}
	if config.Tools.Confluence == nil {
		t.Error("Expected Confluence to be configured")
	}
	if config.Tools.Bitbucket == nil {
		t.Error("Expected Bitbucket to be configured")
	}

	// Verify authentication types
	if config.Tools.Jira.Auth.Type != "basic" {
		t.Errorf("Expected Jira auth type 'basic', got '%s'", config.Tools.Jira.Auth.Type)
	}
	if config.Tools.Confluence.Auth.Type != "token" {
		t.Errorf("Expected Confluence auth type 'token', got '%s'", config.Tools.Confluence.Auth.Type)
	}

	// Create authentication manager and validate all tools
	authManager := domain.NewAuthenticationManagerFromConfig(config)

	if err := authManager.ValidateCredentials("jira"); err != nil {
		t.Errorf("Failed to validate Jira credentials: %v", err)
	}
	if err := authManager.ValidateCredentials("confluence"); err != nil {
		t.Errorf("Failed to validate Confluence credentials: %v", err)
	}
	if err := authManager.ValidateCredentials("bitbucket"); err != nil {
		t.Errorf("Failed to validate Bitbucket credentials: %v", err)
	}
}

// TestInvalidConfiguration tests that invalid configurations are rejected
func TestInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name          string
		configContent string
		expectError   bool
	}{
		{
			name: "Missing transport type",
			configContent: `
tools:
  jira:
    base_url: https://jira.example.com
    auth:
      type: basic
      username: user
      password: pass
`,
			expectError: true,
		},
		{
			name: "Invalid transport type",
			configContent: `
transport:
  type: invalid

tools:
  jira:
    base_url: https://jira.example.com
    auth:
      type: basic
      username: user
      password: pass
`,
			expectError: true,
		},
		{
			name: "HTTP transport without host",
			configContent: `
transport:
  type: http
  http:
    port: 8080

tools:
  jira:
    base_url: https://jira.example.com
    auth:
      type: basic
      username: user
      password: pass
`,
			expectError: true,
		},
		{
			name: "No tools configured",
			configContent: `
transport:
  type: stdio
`,
			expectError: true,
		},
		{
			name: "Missing base URL",
			configContent: `
transport:
  type: stdio

tools:
  jira:
    auth:
      type: basic
      username: user
      password: pass
`,
			expectError: true,
		},
		{
			name: "Invalid auth type",
			configContent: `
transport:
  type: stdio

tools:
  jira:
    base_url: https://jira.example.com
    auth:
      type: invalid
      username: user
      password: pass
`,
			expectError: true,
		},
		{
			name: "Basic auth without username",
			configContent: `
transport:
  type: stdio

tools:
  jira:
    base_url: https://jira.example.com
    auth:
      type: basic
      password: pass
`,
			expectError: true,
		},
		{
			name: "Token auth without token",
			configContent: `
transport:
  type: stdio

tools:
  jira:
    base_url: https://jira.example.com
    auth:
      type: token
`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write to temporary file
			tmpFile, err := os.CreateTemp("", "config-*.yaml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(tt.configContent); err != nil {
				t.Fatalf("Failed to write config: %v", err)
			}
			tmpFile.Close()

			// Try to load configuration
			_, err = domain.LoadConfig(tmpFile.Name())

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}
