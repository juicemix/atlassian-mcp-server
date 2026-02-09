package domain

import (
	"testing"
)

// TestConfig_NoCredentials tests that config validation works without credentials
func TestConfig_NoCredentials(t *testing.T) {
	t.Run("valid config without credentials", func(t *testing.T) {
		config := &Config{
			Transport: TransportConfig{
				Type: "stdio",
			},
			Tools: ToolsConfig{
				Jira: &ToolConfig{
					BaseURL: "https://jira.example.com",
					Auth:    nil, // No credentials
				},
			},
		}

		err := config.Validate()
		if err != nil {
			t.Errorf("expected no error for config without credentials, got: %v", err)
		}
	})

	t.Run("valid config with mixed credentials", func(t *testing.T) {
		config := &Config{
			Transport: TransportConfig{
				Type: "stdio",
			},
			Tools: ToolsConfig{
				Jira: &ToolConfig{
					BaseURL: "https://jira.example.com",
					Auth:    nil, // No credentials
				},
				Confluence: &ToolConfig{
					BaseURL: "https://confluence.example.com",
					Auth: &AuthConfig{
						Type:     "basic",
						Username: "user",
						Password: "pass",
					},
				},
			},
		}

		err := config.Validate()
		if err != nil {
			t.Errorf("expected no error for mixed config, got: %v", err)
		}
	})

	t.Run("invalid config with incomplete credentials", func(t *testing.T) {
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
						// Missing password
					},
				},
			},
		}

		err := config.Validate()
		if err == nil {
			t.Error("expected error for incomplete credentials, got nil")
		}
	})
}

// TestAuthenticationManager_NoDefaultCredentials tests auth manager with no default credentials
func TestAuthenticationManager_NoDefaultCredentials(t *testing.T) {
	t.Run("GetAuthenticatedClient fails when no credentials", func(t *testing.T) {
		authManager := NewAuthenticationManager(map[string]*Credentials{})

		_, err := authManager.GetAuthenticatedClient("jira")
		if err == nil {
			t.Error("expected error when getting client with no credentials, got nil")
		}
	})

	t.Run("GetAuthenticatedClientWithCredentials works", func(t *testing.T) {
		authManager := NewAuthenticationManager(map[string]*Credentials{})

		creds := &Credentials{
			Type:     BasicAuth,
			Username: "test-user",
			Password: "test-pass",
		}

		client, err := authManager.GetAuthenticatedClientWithCredentials(creds)
		if err != nil {
			t.Errorf("expected no error with valid credentials, got: %v", err)
		}

		if client == nil {
			t.Error("expected client, got nil")
		}
	})
}

// TestNewAuthenticationManagerFromConfig_NoCredentials tests creating auth manager from config without credentials
func TestNewAuthenticationManagerFromConfig_NoCredentials(t *testing.T) {
	t.Run("creates manager with no default credentials", func(t *testing.T) {
		config := &Config{
			Transport: TransportConfig{
				Type: "stdio",
			},
			Tools: ToolsConfig{
				Jira: &ToolConfig{
					BaseURL: "https://jira.example.com",
					Auth:    nil, // No credentials
				},
				Confluence: &ToolConfig{
					BaseURL: "https://confluence.example.com",
					Auth:    nil, // No credentials
				},
			},
		}

		authManager := NewAuthenticationManagerFromConfig(config)
		if authManager == nil {
			t.Fatal("expected auth manager, got nil")
		}

		// Should fail to get client for jira (no credentials)
		_, err := authManager.GetAuthenticatedClient("jira")
		if err == nil {
			t.Error("expected error getting jira client with no credentials, got nil")
		}

		// Should fail to get client for confluence (no credentials)
		_, err = authManager.GetAuthenticatedClient("confluence")
		if err == nil {
			t.Error("expected error getting confluence client with no credentials, got nil")
		}
	})

	t.Run("creates manager with mixed credentials", func(t *testing.T) {
		config := &Config{
			Transport: TransportConfig{
				Type: "stdio",
			},
			Tools: ToolsConfig{
				Jira: &ToolConfig{
					BaseURL: "https://jira.example.com",
					Auth:    nil, // No credentials
				},
				Confluence: &ToolConfig{
					BaseURL: "https://confluence.example.com",
					Auth: &AuthConfig{
						Type:     "basic",
						Username: "user",
						Password: "pass",
					},
				},
			},
		}

		authManager := NewAuthenticationManagerFromConfig(config)
		if authManager == nil {
			t.Fatal("expected auth manager, got nil")
		}

		// Should fail for jira (no credentials)
		_, err := authManager.GetAuthenticatedClient("jira")
		if err == nil {
			t.Error("expected error getting jira client with no credentials, got nil")
		}

		// Should succeed for confluence (has credentials)
		client, err := authManager.GetAuthenticatedClient("confluence")
		if err != nil {
			t.Errorf("expected no error getting confluence client, got: %v", err)
		}
		if client == nil {
			t.Error("expected confluence client, got nil")
		}
	})
}
