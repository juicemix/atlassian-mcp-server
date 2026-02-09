package domain

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadConfig_ValidYAML tests loading a valid YAML configuration file.
func TestLoadConfig_ValidYAML(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	validConfig := `
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

	if err := os.WriteFile(configPath, []byte(validConfig), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Load the configuration
	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil", err)
	}

	// Verify the configuration was loaded correctly
	if config.Transport.Type != "stdio" {
		t.Errorf("Transport.Type = %s, want stdio", config.Transport.Type)
	}

	if config.Tools.Jira == nil {
		t.Fatal("Tools.Jira is nil, want non-nil")
	}

	if config.Tools.Jira.BaseURL != "https://jira.example.com" {
		t.Errorf("Jira.BaseURL = %s, want https://jira.example.com", config.Tools.Jira.BaseURL)
	}

	if config.Tools.Jira.Auth.Type != "basic" {
		t.Errorf("Jira.Auth.Type = %s, want basic", config.Tools.Jira.Auth.Type)
	}

	if config.Tools.Jira.Auth.Username != "testuser" {
		t.Errorf("Jira.Auth.Username = %s, want testuser", config.Tools.Jira.Auth.Username)
	}
}

// TestLoadConfig_MissingFile tests error handling when configuration file is missing.
func TestLoadConfig_MissingFile(t *testing.T) {
	config, err := LoadConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("LoadConfig() error = nil, want error for missing file")
	}

	if config != nil {
		t.Errorf("LoadConfig() config = %v, want nil", config)
	}

	// Check that error message mentions the file not being found
	if !contains(err.Error(), "not found") {
		t.Errorf("Error message should mention 'not found', got: %s", err.Error())
	}
}

// TestLoadConfig_InvalidYAMLSyntax tests error handling for invalid YAML syntax.
func TestLoadConfig_InvalidYAMLSyntax(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	invalidYAML := `
transport:
  type: stdio
  invalid yaml syntax here: [unclosed bracket
`

	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	config, err := LoadConfig(configPath)
	if err == nil {
		t.Fatal("LoadConfig() error = nil, want error for invalid YAML")
	}

	if config != nil {
		t.Errorf("LoadConfig() config = %v, want nil", config)
	}

	// Check that error message mentions invalid YAML
	if !contains(err.Error(), "invalid YAML") {
		t.Errorf("Error message should mention 'invalid YAML', got: %s", err.Error())
	}
}

// TestLoadConfig_HTTPTransport tests loading configuration with HTTP transport.
func TestLoadConfig_HTTPTransport(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	httpConfig := `
transport:
  type: http
  http:
    host: localhost
    port: 8080

tools:
  confluence:
    base_url: https://confluence.example.com
    auth:
      type: token
      token: secret-token-123
`

	if err := os.WriteFile(configPath, []byte(httpConfig), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil", err)
	}

	if config.Transport.Type != "http" {
		t.Errorf("Transport.Type = %s, want http", config.Transport.Type)
	}

	if config.Transport.HTTP.Host != "localhost" {
		t.Errorf("Transport.HTTP.Host = %s, want localhost", config.Transport.HTTP.Host)
	}

	if config.Transport.HTTP.Port != 8080 {
		t.Errorf("Transport.HTTP.Port = %d, want 8080", config.Transport.HTTP.Port)
	}

	if config.Tools.Confluence == nil {
		t.Fatal("Tools.Confluence is nil, want non-nil")
	}

	if config.Tools.Confluence.Auth.Type != "token" {
		t.Errorf("Confluence.Auth.Type = %s, want token", config.Tools.Confluence.Auth.Type)
	}

	if config.Tools.Confluence.Auth.Token != "secret-token-123" {
		t.Errorf("Confluence.Auth.Token = %s, want secret-token-123", config.Tools.Confluence.Auth.Token)
	}
}

// TestValidate_MissingTransportType tests validation error for missing transport type.
func TestValidate_MissingTransportType(t *testing.T) {
	config := &Config{
		Transport: TransportConfig{
			Type: "", // Missing
		},
		Tools: ToolsConfig{
			Jira: &ToolConfig{
				BaseURL: "https://jira.example.com",
				Auth: &AuthConfig{
					Type:     "basic",
					Username: "user",
					Password: "pass",
				},
			},
		},
	}

	err := config.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for missing transport type")
	}

	if !contains(err.Error(), "transport type is required") {
		t.Errorf("Error should mention 'transport type is required', got: %s", err.Error())
	}
}

// TestValidate_InvalidTransportType tests validation error for invalid transport type.
func TestValidate_InvalidTransportType(t *testing.T) {
	config := &Config{
		Transport: TransportConfig{
			Type: "websocket", // Invalid
		},
		Tools: ToolsConfig{
			Jira: &ToolConfig{
				BaseURL: "https://jira.example.com",
				Auth: &AuthConfig{
					Type:     "basic",
					Username: "user",
					Password: "pass",
				},
			},
		},
	}

	err := config.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for invalid transport type")
	}

	if !contains(err.Error(), "invalid transport type") {
		t.Errorf("Error should mention 'invalid transport type', got: %s", err.Error())
	}
}

// TestValidate_HTTPTransportMissingHost tests validation error for HTTP transport without host.
func TestValidate_HTTPTransportMissingHost(t *testing.T) {
	config := &Config{
		Transport: TransportConfig{
			Type: "http",
			HTTP: HTTPConfig{
				Host: "", // Missing
				Port: 8080,
			},
		},
		Tools: ToolsConfig{
			Jira: &ToolConfig{
				BaseURL: "https://jira.example.com",
				Auth: &AuthConfig{
					Type:     "basic",
					Username: "user",
					Password: "pass",
				},
			},
		},
	}

	err := config.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for missing HTTP host")
	}

	if !contains(err.Error(), "HTTP host is required") {
		t.Errorf("Error should mention 'HTTP host is required', got: %s", err.Error())
	}
}

// TestValidate_HTTPTransportInvalidPort tests validation error for invalid HTTP port.
func TestValidate_HTTPTransportInvalidPort(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"zero port", 0},
		{"negative port", -1},
		{"port too large", 70000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Transport: TransportConfig{
					Type: "http",
					HTTP: HTTPConfig{
						Host: "localhost",
						Port: tt.port,
					},
				},
				Tools: ToolsConfig{
					Jira: &ToolConfig{
						BaseURL: "https://jira.example.com",
						Auth: &AuthConfig{
							Type:     "basic",
							Username: "user",
							Password: "pass",
						},
					},
				},
			}

			err := config.Validate()
			if err == nil {
				t.Fatalf("Validate() error = nil, want error for invalid port %d", tt.port)
			}

			if !contains(err.Error(), "invalid HTTP port") {
				t.Errorf("Error should mention 'invalid HTTP port', got: %s", err.Error())
			}
		})
	}
}

// TestValidate_NoToolsConfigured tests validation error when no tools are configured.
func TestValidate_NoToolsConfigured(t *testing.T) {
	config := &Config{
		Transport: TransportConfig{
			Type: "stdio",
		},
		Tools: ToolsConfig{
			// All tools are nil
		},
	}

	err := config.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for no tools configured")
	}

	if !contains(err.Error(), "at least one Atlassian tool must be configured") {
		t.Errorf("Error should mention 'at least one Atlassian tool must be configured', got: %s", err.Error())
	}
}

// TestValidate_MissingBaseURL tests validation error for missing base URL.
func TestValidate_MissingBaseURL(t *testing.T) {
	config := &Config{
		Transport: TransportConfig{
			Type: "stdio",
		},
		Tools: ToolsConfig{
			Jira: &ToolConfig{
				BaseURL: "", // Missing
				Auth: &AuthConfig{
					Type:     "basic",
					Username: "user",
					Password: "pass",
				},
			},
		},
	}

	err := config.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for missing base URL")
	}

	if !contains(err.Error(), "base_url is required") {
		t.Errorf("Error should mention 'base_url is required', got: %s", err.Error())
	}
}

// TestValidate_InvalidBaseURL tests validation error for invalid base URL.
func TestValidate_InvalidBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
	}{
		{"invalid URL format", "not-a-valid-url"},
		{"ftp scheme", "ftp://jira.example.com"},
		{"no scheme", "jira.example.com"},
		{"scheme without host", "http://"},
		{"https without host", "https://"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Transport: TransportConfig{
					Type: "stdio",
				},
				Tools: ToolsConfig{
					Jira: &ToolConfig{
						BaseURL: tt.baseURL,
						Auth: &AuthConfig{
							Type:     "basic",
							Username: "user",
							Password: "pass",
						},
					},
				},
			}

			err := config.Validate()
			if err == nil {
				t.Fatalf("Validate() error = nil, want error for invalid base URL: %s", tt.baseURL)
			}

			if !contains(err.Error(), "base_url") {
				t.Errorf("Error should mention 'base_url', got: %s", err.Error())
			}
		})
	}
}

// TestValidate_MissingAuthType tests validation error for missing auth type.
func TestValidate_MissingAuthType(t *testing.T) {
	config := &Config{
		Transport: TransportConfig{
			Type: "stdio",
		},
		Tools: ToolsConfig{
			Jira: &ToolConfig{
				BaseURL: "https://jira.example.com",
				Auth: &AuthConfig{
					Type: "", // Missing
				},
			},
		},
	}

	err := config.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for missing auth type")
	}

	if !contains(err.Error(), "auth type is required") {
		t.Errorf("Error should mention 'auth type is required', got: %s", err.Error())
	}
}

// TestValidate_InvalidAuthType tests validation error for invalid auth type.
func TestValidate_InvalidAuthType(t *testing.T) {
	config := &Config{
		Transport: TransportConfig{
			Type: "stdio",
		},
		Tools: ToolsConfig{
			Jira: &ToolConfig{
				BaseURL: "https://jira.example.com",
				Auth: &AuthConfig{
					Type: "oauth", // Invalid
				},
			},
		},
	}

	err := config.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for invalid auth type")
	}

	if !contains(err.Error(), "auth type") && !contains(err.Error(), "invalid") {
		t.Errorf("Error should mention invalid auth type, got: %s", err.Error())
	}
}

// TestValidate_BasicAuthMissingUsername tests validation error for basic auth without username.
func TestValidate_BasicAuthMissingUsername(t *testing.T) {
	config := &Config{
		Transport: TransportConfig{
			Type: "stdio",
		},
		Tools: ToolsConfig{
			Jira: &ToolConfig{
				BaseURL: "https://jira.example.com",
				Auth: &AuthConfig{
					Type:     "basic",
					Username: "", // Missing
					Password: "pass",
				},
			},
		},
	}

	err := config.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for missing username")
	}

	if !contains(err.Error(), "username is required") {
		t.Errorf("Error should mention 'username is required', got: %s", err.Error())
	}
}

// TestValidate_BasicAuthMissingPassword tests validation error for basic auth without password.
func TestValidate_BasicAuthMissingPassword(t *testing.T) {
	config := &Config{
		Transport: TransportConfig{
			Type: "stdio",
		},
		Tools: ToolsConfig{
			Jira: &ToolConfig{
				BaseURL: "https://jira.example.com",
				Auth: &AuthConfig{
					Type:     "basic",
					Username: "user",
					Password: "", // Missing
				},
			},
		},
	}

	err := config.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for missing password")
	}

	if !contains(err.Error(), "password is required") {
		t.Errorf("Error should mention 'password is required', got: %s", err.Error())
	}
}

// TestValidate_TokenAuthMissingToken tests validation error for token auth without token.
func TestValidate_TokenAuthMissingToken(t *testing.T) {
	config := &Config{
		Transport: TransportConfig{
			Type: "stdio",
		},
		Tools: ToolsConfig{
			Jira: &ToolConfig{
				BaseURL: "https://jira.example.com",
				Auth: &AuthConfig{
					Type:  "token",
					Token: "", // Missing
				},
			},
		},
	}

	err := config.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for missing token")
	}

	if !contains(err.Error(), "token is required") {
		t.Errorf("Error should mention 'token is required', got: %s", err.Error())
	}
}

// TestValidate_MultipleTools tests validation with multiple tools configured.
func TestValidate_MultipleTools(t *testing.T) {
	config := &Config{
		Transport: TransportConfig{
			Type: "stdio",
		},
		Tools: ToolsConfig{
			Jira: &ToolConfig{
				BaseURL: "https://jira.example.com",
				Auth: &AuthConfig{
					Type:     "basic",
					Username: "jirauser",
					Password: "jirapass",
				},
			},
			Confluence: &ToolConfig{
				BaseURL: "https://confluence.example.com",
				Auth: &AuthConfig{
					Type:  "token",
					Token: "confluence-token",
				},
			},
			Bitbucket: &ToolConfig{
				BaseURL: "https://bitbucket.example.com",
				Auth: &AuthConfig{
					Type:     "basic",
					Username: "bitbucketuser",
					Password: "bitbucketpass",
				},
			},
			Bamboo: &ToolConfig{
				BaseURL: "https://bamboo.example.com",
				Auth: &AuthConfig{
					Type:  "token",
					Token: "bamboo-token",
				},
			},
		},
	}

	err := config.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil for valid config with multiple tools", err)
	}
}

// TestValidate_MultipleErrors tests that validation reports multiple errors.
func TestValidate_MultipleErrors(t *testing.T) {
	config := &Config{
		Transport: TransportConfig{
			Type: "", // Missing
		},
		Tools: ToolsConfig{
			Jira: &ToolConfig{
				BaseURL: "", // Missing
				Auth: &AuthConfig{
					Type: "", // Missing
				},
			},
		},
	}

	err := config.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for multiple validation failures")
	}

	// Check that multiple errors are reported
	errMsg := err.Error()
	if !contains(errMsg, "transport type is required") {
		t.Errorf("Error should mention 'transport type is required', got: %s", errMsg)
	}
	if !contains(errMsg, "base_url is required") {
		t.Errorf("Error should mention 'base_url is required', got: %s", errMsg)
	}
	if !contains(errMsg, "auth type is required") {
		t.Errorf("Error should mention 'auth type is required', got: %s", errMsg)
	}
}

// contains is a helper function to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
