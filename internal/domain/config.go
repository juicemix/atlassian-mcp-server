package domain

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the server configuration.
// This is the root configuration structure loaded from YAML files.
type Config struct {
	Transport TransportConfig `yaml:"transport"`
	Tools     ToolsConfig     `yaml:"tools"`
}

// TransportConfig defines transport settings.
// Specifies whether to use stdio or HTTP transport.
type TransportConfig struct {
	Type string     `yaml:"type"` // "stdio" or "http"
	HTTP HTTPConfig `yaml:"http,omitempty"`
}

// HTTPConfig defines HTTP transport settings.
// Only used when transport type is "http".
type HTTPConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// ToolsConfig defines Atlassian tool configurations.
// Each tool is optional - only configured tools will be available.
type ToolsConfig struct {
	Jira       *ToolConfig `yaml:"jira,omitempty"`
	Confluence *ToolConfig `yaml:"confluence,omitempty"`
	Bitbucket  *ToolConfig `yaml:"bitbucket,omitempty"`
	Bamboo     *ToolConfig `yaml:"bamboo,omitempty"`
}

// ToolConfig defines configuration for a single Atlassian tool.
type ToolConfig struct {
	BaseURL string      `yaml:"base_url"`
	Auth    *AuthConfig `yaml:"auth,omitempty"` // Optional - if not provided, client must provide credentials
}

// AuthConfig defines authentication settings.
// Supports both basic authentication and token-based authentication.
type AuthConfig struct {
	Type     string `yaml:"type"` // "basic" or "token"
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	Token    string `yaml:"token,omitempty"`
}

// AuthType defines supported authentication methods.
type AuthType int

const (
	// BasicAuth uses username and password authentication
	BasicAuth AuthType = iota
	// TokenAuth uses personal access token authentication
	TokenAuth
)

// String returns the string representation of AuthType.
func (a AuthType) String() string {
	switch a {
	case BasicAuth:
		return "basic"
	case TokenAuth:
		return "token"
	default:
		return "unknown"
	}
}

// ParseAuthType converts a string to AuthType.
func ParseAuthType(s string) AuthType {
	switch s {
	case "basic":
		return BasicAuth
	case "token":
		return TokenAuth
	default:
		return BasicAuth
	}
}

// LoadConfig reads and validates configuration from a YAML file.
// Returns an error if the file is missing, has invalid syntax, or fails validation.
func LoadConfig(path string) (*Config, error) {
	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("configuration file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("invalid YAML syntax in configuration file: %w", err)
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
}

// Validate checks the configuration for completeness and correctness.
// Returns an error describing all validation failures.
func (c *Config) Validate() error {
	var errors []string

	// Validate transport configuration
	if err := c.validateTransport(); err != nil {
		errors = append(errors, err.Error())
	}

	// Validate tools configuration
	if err := c.validateTools(); err != nil {
		errors = append(errors, err.Error())
	}

	// Check that at least one tool is configured
	if c.Tools.Jira == nil && c.Tools.Confluence == nil &&
		c.Tools.Bitbucket == nil && c.Tools.Bamboo == nil {
		errors = append(errors, "at least one Atlassian tool must be configured")
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// validateTransport validates the transport configuration.
func (c *Config) validateTransport() error {
	var errors []string

	// Check transport type is specified
	if c.Transport.Type == "" {
		errors = append(errors, "transport type is required")
	} else if c.Transport.Type != "stdio" && c.Transport.Type != "http" {
		errors = append(errors, fmt.Sprintf("invalid transport type '%s': must be 'stdio' or 'http'", c.Transport.Type))
	}

	// If HTTP transport, validate HTTP configuration
	if c.Transport.Type == "http" {
		if c.Transport.HTTP.Host == "" {
			errors = append(errors, "HTTP host is required when transport type is 'http'")
		}
		if c.Transport.HTTP.Port <= 0 || c.Transport.HTTP.Port > 65535 {
			errors = append(errors, fmt.Sprintf("invalid HTTP port %d: must be between 1 and 65535", c.Transport.HTTP.Port))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}

	return nil
}

// validateTools validates all configured Atlassian tools.
func (c *Config) validateTools() error {
	var errors []string

	if c.Tools.Jira != nil {
		if err := c.Tools.Jira.Validate("Jira"); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if c.Tools.Confluence != nil {
		if err := c.Tools.Confluence.Validate("Confluence"); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if c.Tools.Bitbucket != nil {
		if err := c.Tools.Bitbucket.Validate("Bitbucket"); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if c.Tools.Bamboo != nil {
		if err := c.Tools.Bamboo.Validate("Bamboo"); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}

	return nil
}

// Validate validates a single tool configuration.
func (tc *ToolConfig) Validate(toolName string) error {
	var errors []string

	// Check base URL is specified
	if tc.BaseURL == "" {
		errors = append(errors, fmt.Sprintf("%s base_url is required", toolName))
	} else {
		// Validate URL format
		parsedURL, err := url.Parse(tc.BaseURL)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s base_url is invalid: %v", toolName, err))
		} else if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			errors = append(errors, fmt.Sprintf("%s base_url must use http or https scheme", toolName))
		} else if parsedURL.Host == "" {
			errors = append(errors, fmt.Sprintf("%s base_url must include a host", toolName))
		}
	}

	// Validate authentication configuration (only if provided)
	if tc.Auth != nil {
		if err := tc.Auth.Validate(toolName); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}

	return nil
}

// Validate validates authentication configuration.
func (ac *AuthConfig) Validate(toolName string) error {
	var errors []string

	// Check auth type is specified
	if ac.Type == "" {
		errors = append(errors, fmt.Sprintf("%s auth type is required", toolName))
	} else if ac.Type != "basic" && ac.Type != "token" {
		errors = append(errors, fmt.Sprintf("%s auth type '%s' is invalid: must be 'basic' or 'token'", toolName, ac.Type))
	}

	// Validate credentials based on auth type
	if ac.Type == "basic" {
		if ac.Username == "" {
			errors = append(errors, fmt.Sprintf("%s username is required for basic auth", toolName))
		}
		if ac.Password == "" {
			errors = append(errors, fmt.Sprintf("%s password is required for basic auth", toolName))
		}
	} else if ac.Type == "token" {
		if ac.Token == "" {
			errors = append(errors, fmt.Sprintf("%s token is required for token auth", toolName))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}

	return nil
}
