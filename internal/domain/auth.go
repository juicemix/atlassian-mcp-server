package domain

import (
	"encoding/base64"
	"fmt"
	"net/http"
)

// Credentials stores authentication information for an Atlassian tool.
// Supports both basic authentication (username/password) and token authentication.
type Credentials struct {
	Type     AuthType // BasicAuth or TokenAuth
	Username string   // Used for basic auth
	Password string   // Used for basic auth
	Token    string   // Used for token auth
}

// AuthenticationManager handles credentials for Atlassian tools.
// It stores credentials for each configured tool and provides authenticated
// HTTP clients for making API calls.
type AuthenticationManager struct {
	credentials map[string]*Credentials
}

// NewAuthenticationManager creates a new authentication manager.
// The credentials map should contain entries for each configured tool,
// keyed by tool name (e.g., "jira", "confluence", "bitbucket", "bamboo").
func NewAuthenticationManager(credentials map[string]*Credentials) *AuthenticationManager {
	return &AuthenticationManager{
		credentials: credentials,
	}
}

// NewAuthenticationManagerFromConfig creates an authentication manager from a configuration.
// It extracts credentials from the config for each configured tool.
// If a tool has no auth configured, it will not have default credentials (client must provide them).
func NewAuthenticationManagerFromConfig(config *Config) *AuthenticationManager {
	credentials := make(map[string]*Credentials)

	if config.Tools.Jira != nil && config.Tools.Jira.Auth != nil {
		credentials["jira"] = credentialsFromAuthConfig(config.Tools.Jira.Auth)
	}

	if config.Tools.Confluence != nil && config.Tools.Confluence.Auth != nil {
		credentials["confluence"] = credentialsFromAuthConfig(config.Tools.Confluence.Auth)
	}

	if config.Tools.Bitbucket != nil && config.Tools.Bitbucket.Auth != nil {
		credentials["bitbucket"] = credentialsFromAuthConfig(config.Tools.Bitbucket.Auth)
	}

	if config.Tools.Bamboo != nil && config.Tools.Bamboo.Auth != nil {
		credentials["bamboo"] = credentialsFromAuthConfig(config.Tools.Bamboo.Auth)
	}

	return NewAuthenticationManager(credentials)
}

// credentialsFromAuthConfig converts an AuthConfig to Credentials.
func credentialsFromAuthConfig(authConfig *AuthConfig) *Credentials {
	return &Credentials{
		Type:     ParseAuthType(authConfig.Type),
		Username: authConfig.Username,
		Password: authConfig.Password,
		Token:    authConfig.Token,
	}
}

// GetAuthenticatedClient returns an HTTP client with authentication headers configured.
// The client is pre-configured with the appropriate authentication method for the tool.
// Returns an error if the tool is not configured or credentials are invalid.
func (am *AuthenticationManager) GetAuthenticatedClient(tool string) (*http.Client, error) {
	// Validate credentials first
	if err := am.ValidateCredentials(tool); err != nil {
		return nil, err
	}

	// Get credentials for the tool
	creds := am.credentials[tool]

	// Create a custom transport that adds authentication headers
	transport := &authenticatedTransport{
		base:        http.DefaultTransport,
		credentials: creds,
	}

	// Return a client with the authenticated transport
	return &http.Client{
		Transport: transport,
	}, nil
}

// GetAuthenticatedClientWithCredentials returns an HTTP client with the provided credentials.
// This allows clients to provide their own credentials at runtime instead of using config file credentials.
// Returns an error if the provided credentials are invalid.
func (am *AuthenticationManager) GetAuthenticatedClientWithCredentials(creds *Credentials) (*http.Client, error) {
	// Validate the provided credentials
	if err := validateCredentials(creds); err != nil {
		return nil, err
	}

	// Create a custom transport that adds authentication headers
	transport := &authenticatedTransport{
		base:        http.DefaultTransport,
		credentials: creds,
	}

	// Return a client with the authenticated transport
	return &http.Client{
		Transport: transport,
	}, nil
}

// validateCredentials validates a Credentials object.
func validateCredentials(creds *Credentials) error {
	if creds == nil {
		return fmt.Errorf("credentials cannot be nil")
	}

	switch creds.Type {
	case BasicAuth:
		if creds.Username == "" {
			return fmt.Errorf("username is required for basic authentication")
		}
		if creds.Password == "" {
			return fmt.Errorf("password is required for basic authentication")
		}
	case TokenAuth:
		if creds.Token == "" {
			return fmt.Errorf("token is required for token authentication")
		}
	default:
		return fmt.Errorf("invalid authentication type: %v", creds.Type)
	}

	return nil
}

// ValidateCredentials checks if credentials are properly configured for a tool.
// Returns an error if the tool is not configured or if credentials are missing/invalid.
func (am *AuthenticationManager) ValidateCredentials(tool string) error {
	// Check if tool is configured
	creds, ok := am.credentials[tool]
	if !ok {
		return fmt.Errorf("no credentials configured for tool: %s", tool)
	}

	// Validate credentials based on auth type
	switch creds.Type {
	case BasicAuth:
		if creds.Username == "" {
			return fmt.Errorf("username is required for basic authentication: %s", tool)
		}
		if creds.Password == "" {
			return fmt.Errorf("password is required for basic authentication: %s", tool)
		}
	case TokenAuth:
		if creds.Token == "" {
			return fmt.Errorf("token is required for token authentication: %s", tool)
		}
	default:
		return fmt.Errorf("invalid authentication type for tool: %s", tool)
	}

	return nil
}

// authenticatedTransport is an http.RoundTripper that adds authentication headers.
type authenticatedTransport struct {
	base        http.RoundTripper
	credentials *Credentials
}

// RoundTrip implements http.RoundTripper by adding authentication headers to requests.
func (t *authenticatedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	clonedReq := req.Clone(req.Context())

	// Add authentication headers based on credentials type
	switch t.credentials.Type {
	case BasicAuth:
		// Basic authentication: encode username:password in base64
		auth := t.credentials.Username + ":" + t.credentials.Password
		encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
		clonedReq.Header.Set("Authorization", "Basic "+encodedAuth)
	case TokenAuth:
		// Token authentication: use Bearer token
		clonedReq.Header.Set("Authorization", "Bearer "+t.credentials.Token)
	}

	// Execute the request with the base transport
	return t.base.RoundTrip(clonedReq)
}

// ExtractCredentialsFromArguments extracts optional credentials from tool call arguments.
// Returns nil if no credentials are provided in the arguments.
// Supports both "auth" object and individual credential fields.
func ExtractCredentialsFromArguments(args map[string]interface{}) (*Credentials, error) {
	// Check if auth object is provided
	authObj, hasAuth := args["auth"]
	if !hasAuth {
		return nil, nil
	}

	// Convert auth object to map
	authMap, ok := authObj.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("auth must be an object")
	}

	// Extract auth type
	authTypeStr, _ := authMap["type"].(string)
	if authTypeStr == "" {
		authTypeStr = "basic" // Default to basic auth
	}
	authType := ParseAuthType(authTypeStr)

	// Build credentials based on type
	creds := &Credentials{
		Type: authType,
	}

	switch authType {
	case BasicAuth:
		username, _ := authMap["username"].(string)
		password, _ := authMap["password"].(string)
		creds.Username = username
		creds.Password = password
	case TokenAuth:
		token, _ := authMap["token"].(string)
		creds.Token = token
	}

	// Validate the extracted credentials
	if err := validateCredentials(creds); err != nil {
		return nil, fmt.Errorf("invalid credentials provided: %w", err)
	}

	return creds, nil
}
