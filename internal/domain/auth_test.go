package domain

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestNewAuthenticationManager tests creating a new authentication manager.
func TestNewAuthenticationManager(t *testing.T) {
	credentials := map[string]*Credentials{
		"jira": {
			Type:     BasicAuth,
			Username: "user",
			Password: "pass",
		},
	}

	am := NewAuthenticationManager(credentials)

	if am == nil {
		t.Fatal("expected non-nil authentication manager")
	}

	if am.credentials == nil {
		t.Fatal("expected non-nil credentials map")
	}

	if len(am.credentials) != 1 {
		t.Errorf("expected 1 credential, got %d", len(am.credentials))
	}
}

// TestNewAuthenticationManagerFromConfig tests creating an authentication manager from config.
func TestNewAuthenticationManagerFromConfig(t *testing.T) {
	config := &Config{
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
		},
	}

	am := NewAuthenticationManagerFromConfig(config)

	if am == nil {
		t.Fatal("expected non-nil authentication manager")
	}

	// Check Jira credentials
	jiraCreds, ok := am.credentials["jira"]
	if !ok {
		t.Fatal("expected jira credentials")
	}
	if jiraCreds.Type != BasicAuth {
		t.Errorf("expected BasicAuth, got %v", jiraCreds.Type)
	}
	if jiraCreds.Username != "jirauser" {
		t.Errorf("expected username 'jirauser', got '%s'", jiraCreds.Username)
	}
	if jiraCreds.Password != "jirapass" {
		t.Errorf("expected password 'jirapass', got '%s'", jiraCreds.Password)
	}

	// Check Confluence credentials
	confluenceCreds, ok := am.credentials["confluence"]
	if !ok {
		t.Fatal("expected confluence credentials")
	}
	if confluenceCreds.Type != TokenAuth {
		t.Errorf("expected TokenAuth, got %v", confluenceCreds.Type)
	}
	if confluenceCreds.Token != "confluence-token" {
		t.Errorf("expected token 'confluence-token', got '%s'", confluenceCreds.Token)
	}
}

// TestValidateCredentials_BasicAuth tests validating basic auth credentials.
func TestValidateCredentials_BasicAuth(t *testing.T) {
	tests := []struct {
		name        string
		credentials *Credentials
		wantErr     bool
		errContains string
	}{
		{
			name: "valid basic auth",
			credentials: &Credentials{
				Type:     BasicAuth,
				Username: "user",
				Password: "pass",
			},
			wantErr: false,
		},
		{
			name: "missing username",
			credentials: &Credentials{
				Type:     BasicAuth,
				Username: "",
				Password: "pass",
			},
			wantErr:     true,
			errContains: "username is required",
		},
		{
			name: "missing password",
			credentials: &Credentials{
				Type:     BasicAuth,
				Username: "user",
				Password: "",
			},
			wantErr:     true,
			errContains: "password is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			am := NewAuthenticationManager(map[string]*Credentials{
				"test": tt.credentials,
			})

			err := am.ValidateCredentials("test")

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}

			if tt.wantErr && err != nil && tt.errContains != "" {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("expected error to contain '%s', got: %v", tt.errContains, err)
				}
			}
		})
	}
}

// TestValidateCredentials_TokenAuth tests validating token auth credentials.
func TestValidateCredentials_TokenAuth(t *testing.T) {
	tests := []struct {
		name        string
		credentials *Credentials
		wantErr     bool
		errContains string
	}{
		{
			name: "valid token auth",
			credentials: &Credentials{
				Type:  TokenAuth,
				Token: "my-token",
			},
			wantErr: false,
		},
		{
			name: "missing token",
			credentials: &Credentials{
				Type:  TokenAuth,
				Token: "",
			},
			wantErr:     true,
			errContains: "token is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			am := NewAuthenticationManager(map[string]*Credentials{
				"test": tt.credentials,
			})

			err := am.ValidateCredentials("test")

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}

			if tt.wantErr && err != nil && tt.errContains != "" {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("expected error to contain '%s', got: %v", tt.errContains, err)
				}
			}
		})
	}
}

// TestValidateCredentials_UnconfiguredTool tests validating credentials for unconfigured tool.
func TestValidateCredentials_UnconfiguredTool(t *testing.T) {
	am := NewAuthenticationManager(map[string]*Credentials{})

	err := am.ValidateCredentials("nonexistent")

	if err == nil {
		t.Fatal("expected error for unconfigured tool")
	}

	if !contains(err.Error(), "no credentials configured") {
		t.Errorf("expected error about unconfigured tool, got: %v", err)
	}
}

// TestGetAuthenticatedClient_BasicAuth tests getting an authenticated client with basic auth.
func TestGetAuthenticatedClient_BasicAuth(t *testing.T) {
	am := NewAuthenticationManager(map[string]*Credentials{
		"jira": {
			Type:     BasicAuth,
			Username: "testuser",
			Password: "testpass",
		},
	})

	client, err := am.GetAuthenticatedClient("jira")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client == nil {
		t.Fatal("expected non-nil client")
	}

	// Create a test server to verify authentication headers
	expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != expectedAuth {
			t.Errorf("expected Authorization header '%s', got '%s'", expectedAuth, auth)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Make a request using the authenticated client
	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

// TestGetAuthenticatedClient_TokenAuth tests getting an authenticated client with token auth.
func TestGetAuthenticatedClient_TokenAuth(t *testing.T) {
	am := NewAuthenticationManager(map[string]*Credentials{
		"confluence": {
			Type:  TokenAuth,
			Token: "my-secret-token",
		},
	})

	client, err := am.GetAuthenticatedClient("confluence")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client == nil {
		t.Fatal("expected non-nil client")
	}

	// Create a test server to verify authentication headers
	expectedAuth := "Bearer my-secret-token"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != expectedAuth {
			t.Errorf("expected Authorization header '%s', got '%s'", expectedAuth, auth)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Make a request using the authenticated client
	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

// TestGetAuthenticatedClient_InvalidCredentials tests getting a client with invalid credentials.
func TestGetAuthenticatedClient_InvalidCredentials(t *testing.T) {
	tests := []struct {
		name        string
		tool        string
		credentials map[string]*Credentials
		errContains string
	}{
		{
			name: "unconfigured tool",
			tool: "nonexistent",
			credentials: map[string]*Credentials{
				"jira": {Type: BasicAuth, Username: "user", Password: "pass"},
			},
			errContains: "no credentials configured",
		},
		{
			name: "missing username",
			tool: "jira",
			credentials: map[string]*Credentials{
				"jira": {Type: BasicAuth, Username: "", Password: "pass"},
			},
			errContains: "username is required",
		},
		{
			name: "missing password",
			tool: "jira",
			credentials: map[string]*Credentials{
				"jira": {Type: BasicAuth, Username: "user", Password: ""},
			},
			errContains: "password is required",
		},
		{
			name: "missing token",
			tool: "confluence",
			credentials: map[string]*Credentials{
				"confluence": {Type: TokenAuth, Token: ""},
			},
			errContains: "token is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			am := NewAuthenticationManager(tt.credentials)

			client, err := am.GetAuthenticatedClient(tt.tool)

			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if client != nil {
				t.Error("expected nil client on error")
			}

			if !contains(err.Error(), tt.errContains) {
				t.Errorf("expected error to contain '%s', got: %v", tt.errContains, err)
			}
		})
	}
}

// TestAuthenticatedTransport_PreservesOriginalRequest tests that the transport doesn't modify the original request.
func TestAuthenticatedTransport_PreservesOriginalRequest(t *testing.T) {
	creds := &Credentials{
		Type:     BasicAuth,
		Username: "user",
		Password: "pass",
	}

	transport := &authenticatedTransport{
		base:        http.DefaultTransport,
		credentials: creds,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a request with a custom header
	req, _ := http.NewRequest("GET", server.URL, nil)
	req.Header.Set("X-Custom-Header", "custom-value")

	// Store original header count
	originalHeaderCount := len(req.Header)

	// Execute the request
	_, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify original request wasn't modified
	if len(req.Header) != originalHeaderCount {
		t.Errorf("original request was modified: expected %d headers, got %d", originalHeaderCount, len(req.Header))
	}

	if req.Header.Get("Authorization") != "" {
		t.Error("original request should not have Authorization header")
	}

	if req.Header.Get("X-Custom-Header") != "custom-value" {
		t.Error("original request custom header was modified")
	}
}
