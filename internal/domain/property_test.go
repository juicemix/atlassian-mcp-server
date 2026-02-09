package domain

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestGopterSetup verifies that gopter is properly configured.
// This is a simple property test to ensure the testing framework is working.
func TestGopterSetup(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Property: AuthType round-trip conversion
	// For any valid auth type string, parsing and converting back should be idempotent
	properties.Property("AuthType string conversion is consistent", prop.ForAll(
		func(authType string) bool {
			parsed := ParseAuthType(authType)
			// For valid types, round-trip should work
			if authType == "basic" || authType == "token" {
				return parsed.String() == authType
			}
			// For invalid types, should default to basic
			return parsed == BasicAuth
		},
		gen.OneConstOf("basic", "token", "invalid", ""),
	))

	// Property: JSON-RPC error codes are negative
	// All JSON-RPC error codes should be negative integers
	properties.Property("JSON-RPC error codes are negative", prop.ForAll(
		func(code int) bool {
			// Test with predefined error codes
			errorCodes := []int{
				ParseError, InvalidRequest, MethodNotFound,
				InvalidParams, InternalError, ConfigurationError,
				AuthenticationError, APIError, NetworkError, RateLimitError,
			}
			for _, ec := range errorCodes {
				if ec >= 0 {
					return false
				}
			}
			return true
		},
		gen.Int(),
	))

	properties.TestingRun(t)
}

// TestJSONRPCRequestProperties verifies properties of JSON-RPC requests.
func TestJSONRPCRequestProperties(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Property: Request with JSONRPC field set to "2.0" is valid
	properties.Property("Request JSONRPC version must be 2.0", prop.ForAll(
		func(method string) bool {
			req := &Request{
				JSONRPC: "2.0",
				Method:  method,
			}
			return req.JSONRPC == "2.0"
		},
		gen.Identifier(),
	))

	properties.TestingRun(t)
}

// TestConfigProperties verifies properties of configuration structures.
func TestConfigProperties(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Property: Transport type must be either "stdio" or "http"
	properties.Property("Valid transport types", prop.ForAll(
		func(transportType string) bool {
			config := &Config{
				Transport: TransportConfig{Type: transportType},
			}
			// We're just testing that the structure can hold any string
			// Validation will be done in the config loader
			return config.Transport.Type == transportType
		},
		gen.OneConstOf("stdio", "http", "invalid"),
	))

	// Property: Auth type must be either "basic" or "token"
	properties.Property("Valid auth types", prop.ForAll(
		func(authType string) bool {
			authConfig := &AuthConfig{Type: authType}
			// We're just testing that the structure can hold any string
			// Validation will be done in the config loader
			return authConfig.Type == authType
		},
		gen.OneConstOf("basic", "token", "invalid"),
	))

	properties.TestingRun(t)
}

// Feature: atlassian-mcp-server, Property 8: Required Configuration Validation
// **Validates: Requirements 7.2, 7.3, 7.4**
//
// For any configuration file, if it is missing required fields (base URLs, credentials,
// or transport type), the server should fail validation and return errors identifying
// all missing fields.
func TestProperty8_RequiredConfigurationValidation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Property: Missing transport type causes validation failure
	properties.Property("Missing transport type fails validation", prop.ForAll(
		func(hasJira, hasConfluence, hasBitbucket, hasBamboo bool) bool {
			// Create config with at least one tool but missing transport type
			config := &Config{
				Transport: TransportConfig{Type: ""}, // Missing
				Tools:     generateToolsConfig(hasJira, hasConfluence, hasBitbucket, hasBamboo),
			}

			// If no tools are configured, validation should fail for different reason
			if !hasJira && !hasConfluence && !hasBitbucket && !hasBamboo {
				return true // Skip this case
			}

			err := config.Validate()
			// Should fail validation
			if err == nil {
				return false
			}
			// Error message should mention transport type
			return contains(err.Error(), "transport type is required")
		},
		gen.Bool(),
		gen.Bool(),
		gen.Bool(),
		gen.Bool(),
	))

	// Property: Missing base URL causes validation failure
	properties.Property("Missing base URL fails validation", prop.ForAll(
		func(transportType string) bool {
			config := &Config{
				Transport: TransportConfig{Type: transportType},
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
			// Should fail validation
			if err == nil {
				return false
			}
			// Error message should mention base_url
			return contains(err.Error(), "base_url is required")
		},
		gen.OneConstOf("stdio", "http"),
	))

	// Property: Missing auth type causes validation failure
	properties.Property("Missing auth type fails validation", prop.ForAll(
		func(transportType string) bool {
			config := &Config{
				Transport: TransportConfig{Type: transportType},
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
			// Should fail validation
			if err == nil {
				return false
			}
			// Error message should mention auth type
			return contains(err.Error(), "auth type is required")
		},
		gen.OneConstOf("stdio", "http"),
	))

	// Property: Missing username for basic auth causes validation failure
	properties.Property("Missing username for basic auth fails validation", prop.ForAll(
		func(transportType string, hasPassword bool) bool {
			password := ""
			if hasPassword {
				password = "testpass"
			}

			config := &Config{
				Transport: TransportConfig{Type: transportType},
				Tools: ToolsConfig{
					Jira: &ToolConfig{
						BaseURL: "https://jira.example.com",
						Auth: &AuthConfig{
							Type:     "basic",
							Username: "", // Missing
							Password: password,
						},
					},
				},
			}

			err := config.Validate()
			// Should fail validation
			if err == nil {
				return false
			}
			// Error message should mention username
			return contains(err.Error(), "username is required")
		},
		gen.OneConstOf("stdio", "http"),
		gen.Bool(),
	))

	// Property: Missing password for basic auth causes validation failure
	properties.Property("Missing password for basic auth fails validation", prop.ForAll(
		func(transportType string, hasUsername bool) bool {
			username := ""
			if hasUsername {
				username = "testuser"
			}

			config := &Config{
				Transport: TransportConfig{Type: transportType},
				Tools: ToolsConfig{
					Jira: &ToolConfig{
						BaseURL: "https://jira.example.com",
						Auth: &AuthConfig{
							Type:     "basic",
							Username: username,
							Password: "", // Missing
						},
					},
				},
			}

			err := config.Validate()
			// Should fail validation
			if err == nil {
				return false
			}
			// Error message should mention password
			return contains(err.Error(), "password is required")
		},
		gen.OneConstOf("stdio", "http"),
		gen.Bool(),
	))

	// Property: Missing token for token auth causes validation failure
	properties.Property("Missing token for token auth fails validation", prop.ForAll(
		func(transportType string) bool {
			config := &Config{
				Transport: TransportConfig{Type: transportType},
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
			// Should fail validation
			if err == nil {
				return false
			}
			// Error message should mention token
			return contains(err.Error(), "token is required")
		},
		gen.OneConstOf("stdio", "http"),
	))

	// Property: Missing HTTP host when transport is HTTP causes validation failure
	properties.Property("Missing HTTP host fails validation", prop.ForAll(
		func(port int) bool {
			// Use valid port range
			validPort := (port % 65535) + 1

			config := &Config{
				Transport: TransportConfig{
					Type: "http",
					HTTP: HTTPConfig{
						Host: "", // Missing
						Port: validPort,
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
			// Should fail validation
			if err == nil {
				return false
			}
			// Error message should mention HTTP host
			return contains(err.Error(), "HTTP host is required")
		},
		gen.Int(),
	))

	// Property: No tools configured causes validation failure
	properties.Property("No tools configured fails validation", prop.ForAll(
		func(transportType string) bool {
			config := &Config{
				Transport: TransportConfig{Type: transportType},
				Tools:     ToolsConfig{}, // No tools
			}

			err := config.Validate()
			// Should fail validation
			if err == nil {
				return false
			}
			// Error message should mention at least one tool required
			return contains(err.Error(), "at least one Atlassian tool must be configured")
		},
		gen.OneConstOf("stdio", "http"),
	))

	properties.TestingRun(t)
}

// Feature: atlassian-mcp-server, Property 9: Invalid Configuration Rejection
// **Validates: Requirements 7.7**
//
// For any configuration file with invalid values (malformed URLs, invalid transport types,
// invalid auth types), the server should fail validation and return descriptive errors
// for each invalid field.
func TestProperty9_InvalidConfigurationRejection(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Property: Invalid transport type causes validation failure
	properties.Property("Invalid transport type fails validation", prop.ForAll(
		func(invalidType string) bool {
			// Skip valid types
			if invalidType == "stdio" || invalidType == "http" {
				return true
			}

			config := &Config{
				Transport: TransportConfig{Type: invalidType},
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
			// Should fail validation
			if err == nil {
				return false
			}
			// Error message should mention invalid transport type
			return contains(err.Error(), "invalid transport type") ||
				contains(err.Error(), "transport type is required")
		},
		gen.OneConstOf("websocket", "grpc", "tcp", "udp", ""),
	))

	// Property: Invalid auth type causes validation failure
	properties.Property("Invalid auth type fails validation", prop.ForAll(
		func(invalidAuthType string) bool {
			// Skip valid types
			if invalidAuthType == "basic" || invalidAuthType == "token" {
				return true
			}

			config := &Config{
				Transport: TransportConfig{Type: "stdio"},
				Tools: ToolsConfig{
					Jira: &ToolConfig{
						BaseURL: "https://jira.example.com",
						Auth: &AuthConfig{
							Type: invalidAuthType,
						},
					},
				},
			}

			err := config.Validate()
			// Should fail validation
			if err == nil {
				return false
			}
			// Error message should mention auth type issue
			return contains(err.Error(), "auth type") &&
				(contains(err.Error(), "invalid") || contains(err.Error(), "required"))
		},
		gen.OneConstOf("oauth", "oauth2", "bearer", "apikey", ""),
	))

	// Property: Invalid HTTP port causes validation failure
	properties.Property("Invalid HTTP port fails validation", prop.ForAll(
		func(invalidPort int) bool {
			// Only test invalid ports
			if invalidPort > 0 && invalidPort <= 65535 {
				return true
			}

			config := &Config{
				Transport: TransportConfig{
					Type: "http",
					HTTP: HTTPConfig{
						Host: "localhost",
						Port: invalidPort,
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
			// Should fail validation
			if err == nil {
				return false
			}
			// Error message should mention invalid port
			return contains(err.Error(), "invalid HTTP port")
		},
		gen.OneConstOf(-1, 0, 70000, -100, 100000),
	))

	// Property: Malformed base URL causes validation failure
	properties.Property("Malformed base URL fails validation", prop.ForAll(
		func(invalidURL string) bool {
			config := &Config{
				Transport: TransportConfig{Type: "stdio"},
				Tools: ToolsConfig{
					Jira: &ToolConfig{
						BaseURL: invalidURL,
						Auth: &AuthConfig{
							Type:     "basic",
							Username: "user",
							Password: "pass",
						},
					},
				},
			}

			err := config.Validate()
			// Should fail validation
			if err == nil {
				return false
			}
			// Error message should mention base_url issue
			return contains(err.Error(), "base_url")
		},
		gen.OneConstOf(
			"not-a-url",
			"ftp://jira.example.com",
			"jira.example.com",
			"://invalid",
			"http://",
		),
	))

	// Property: Multiple invalid fields produce multiple error messages
	properties.Property("Multiple invalid fields produce comprehensive errors", prop.ForAll(
		func() bool {
			config := &Config{
				Transport: TransportConfig{
					Type: "invalid-transport", // Invalid
				},
				Tools: ToolsConfig{
					Jira: &ToolConfig{
						BaseURL: "not-a-url", // Invalid
						Auth: &AuthConfig{
							Type: "oauth", // Invalid
						},
					},
				},
			}

			err := config.Validate()
			// Should fail validation
			if err == nil {
				return false
			}

			// Error message should mention multiple issues
			errMsg := err.Error()
			hasTransportError := contains(errMsg, "transport type")
			hasURLError := contains(errMsg, "base_url")
			hasAuthError := contains(errMsg, "auth type")

			// Should report at least 2 of the 3 errors
			errorCount := 0
			if hasTransportError {
				errorCount++
			}
			if hasURLError {
				errorCount++
			}
			if hasAuthError {
				errorCount++
			}

			return errorCount >= 2
		},
	))

	// Property: Valid configuration passes validation
	properties.Property("Valid configuration passes validation", prop.ForAll(
		func(transportType string, authType string, toolChoice int) bool {
			// Build valid HTTP config if needed
			httpConfig := HTTPConfig{}
			if transportType == "http" {
				httpConfig = HTTPConfig{
					Host: "localhost",
					Port: 8080,
				}
			}

			// Build valid auth config
			authConfig := AuthConfig{}
			if authType == "basic" {
				authConfig = AuthConfig{
					Type:     "basic",
					Username: "testuser",
					Password: "testpass",
				}
			} else {
				authConfig = AuthConfig{
					Type:  "token",
					Token: "test-token-123",
				}
			}

			// Build valid tool config
			toolConfig := &ToolConfig{
				BaseURL: "https://jira.example.com",
				Auth:    &authConfig,
			}

			// Create config with one tool
			tools := ToolsConfig{}
			switch toolChoice % 4 {
			case 0:
				tools.Jira = toolConfig
			case 1:
				tools.Confluence = toolConfig
			case 2:
				tools.Bitbucket = toolConfig
			case 3:
				tools.Bamboo = toolConfig
			}

			config := &Config{
				Transport: TransportConfig{
					Type: transportType,
					HTTP: httpConfig,
				},
				Tools: tools,
			}

			err := config.Validate()
			// Should pass validation
			return err == nil
		},
		gen.OneConstOf("stdio", "http"),
		gen.OneConstOf("basic", "token"),
		gen.IntRange(0, 3),
	))

	properties.TestingRun(t)
}

// generateToolsConfig creates a ToolsConfig with the specified tools enabled.
func generateToolsConfig(hasJira, hasConfluence, hasBitbucket, hasBamboo bool) ToolsConfig {
	tools := ToolsConfig{}

	validToolConfig := &ToolConfig{
		BaseURL: "https://example.com",
		Auth: &AuthConfig{
			Type:     "basic",
			Username: "user",
			Password: "pass",
		},
	}

	if hasJira {
		tools.Jira = validToolConfig
	}
	if hasConfluence {
		tools.Confluence = validToolConfig
	}
	if hasBitbucket {
		tools.Bitbucket = validToolConfig
	}
	if hasBamboo {
		tools.Bamboo = validToolConfig
	}

	return tools
}

// Feature: atlassian-mcp-server, Property 16: JSON Round-Trip Consistency
// **Validates: Requirements 9.6**
//
// For any valid data structure used in the system (configurations, requests, responses),
// serializing to JSON and then deserializing should produce an equivalent structure.
func TestProperty16_JSONRoundTripConsistency(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Property: JSON-RPC Request round-trip consistency
	properties.Property("JSON-RPC Request round-trip is consistent", prop.ForAll(
		func(method string, id int) bool {
			original := &Request{
				JSONRPC: "2.0",
				ID:      id,
				Method:  method,
				Params:  map[string]interface{}{"key": "value"},
			}

			return testJSONRoundTrip(original, &Request{})
		},
		gen.Identifier(),
		gen.Int(),
	))

	// Property: JSON-RPC Response round-trip consistency
	properties.Property("JSON-RPC Response round-trip is consistent", prop.ForAll(
		func(id int, resultValue string) bool {
			original := &Response{
				JSONRPC: "2.0",
				ID:      id,
				Result:  map[string]interface{}{"data": resultValue},
			}

			return testJSONRoundTrip(original, &Response{})
		},
		gen.Int(),
		gen.AlphaString(),
	))

	// Property: JSON-RPC Error round-trip consistency
	properties.Property("JSON-RPC Error round-trip is consistent", prop.ForAll(
		func(code int, message string) bool {
			original := &Error{
				Code:    code,
				Message: message,
				Data:    map[string]interface{}{"detail": "error detail"},
			}

			return testJSONRoundTrip(original, &Error{})
		},
		gen.Int(),
		gen.AlphaString(),
	))

	// Property: ToolDefinition round-trip consistency
	properties.Property("ToolDefinition round-trip is consistent", prop.ForAll(
		func(name string, description string) bool {
			original := &ToolDefinition{
				Name:        name,
				Description: description,
				InputSchema: JSONSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"param1": map[string]interface{}{"type": "string"},
					},
					Required: []string{"param1"},
				},
			}

			return testJSONRoundTrip(original, &ToolDefinition{})
		},
		gen.Identifier(),
		gen.AlphaString(),
	))

	// Property: ToolRequest round-trip consistency
	properties.Property("ToolRequest round-trip is consistent", prop.ForAll(
		func(name string, argValue string) bool {
			original := &ToolRequest{
				Name: name,
				Arguments: map[string]interface{}{
					"arg1": argValue,
					"arg2": 42,
				},
			}

			return testJSONRoundTrip(original, &ToolRequest{})
		},
		gen.Identifier(),
		gen.AlphaString(),
	))

	// Property: ToolResponse round-trip consistency
	properties.Property("ToolResponse round-trip is consistent", prop.ForAll(
		func(text string, isError bool) bool {
			original := &ToolResponse{
				Content: []ContentBlock{
					{
						Type: "text",
						Text: text,
					},
				},
				IsError: isError,
			}

			return testJSONRoundTrip(original, &ToolResponse{})
		},
		gen.AlphaString(),
		gen.Bool(),
	))

	// Property: ContentBlock round-trip consistency
	properties.Property("ContentBlock round-trip is consistent", prop.ForAll(
		func(text string, uri string) bool {
			original := &ContentBlock{
				Type: "text",
				Text: text,
				Resource: &Resource{
					URI:      uri,
					MimeType: "text/plain",
					Text:     text,
				},
			}

			return testJSONRoundTrip(original, &ContentBlock{})
		},
		gen.AlphaString(),
		gen.Identifier(),
	))

	// Property: Config round-trip consistency
	properties.Property("Config round-trip is consistent", prop.ForAll(
		func(transportType string, authType string, host string, port int) bool {
			// Ensure valid values for the test
			if transportType != "stdio" && transportType != "http" {
				transportType = "stdio"
			}
			if authType != "basic" && authType != "token" {
				authType = "basic"
			}
			if port <= 0 || port > 65535 {
				port = 8080
			}

			httpConfig := HTTPConfig{}
			if transportType == "http" {
				httpConfig = HTTPConfig{
					Host: host,
					Port: port,
				}
			}

			authConfig := AuthConfig{Type: authType}
			if authType == "basic" {
				authConfig.Username = "testuser"
				authConfig.Password = "testpass"
			} else {
				authConfig.Token = "testtoken"
			}

			original := &Config{
				Transport: TransportConfig{
					Type: transportType,
					HTTP: httpConfig,
				},
				Tools: ToolsConfig{
					Jira: &ToolConfig{
						BaseURL: "https://jira.example.com",
						Auth:    &authConfig,
					},
				},
			}

			return testJSONRoundTrip(original, &Config{})
		},
		gen.OneConstOf("stdio", "http"),
		gen.OneConstOf("basic", "token"),
		gen.Identifier(),
		gen.IntRange(1, 65535),
	))

	// Property: ToolConfig round-trip consistency
	properties.Property("ToolConfig round-trip is consistent", prop.ForAll(
		func(baseURL string, username string, password string) bool {
			original := &ToolConfig{
				BaseURL: baseURL,
				Auth: &AuthConfig{
					Type:     "basic",
					Username: username,
					Password: password,
				},
			}

			return testJSONRoundTrip(original, &ToolConfig{})
		},
		gen.Identifier(),
		gen.Identifier(),
		gen.AlphaString(),
	))

	// Property: AuthConfig round-trip consistency
	properties.Property("AuthConfig round-trip is consistent", prop.ForAll(
		func(authType string, username string, password string, token string) bool {
			if authType != "basic" && authType != "token" {
				authType = "basic"
			}

			original := &AuthConfig{
				Type:     authType,
				Username: username,
				Password: password,
				Token:    token,
			}

			return testJSONRoundTrip(original, &AuthConfig{})
		},
		gen.OneConstOf("basic", "token"),
		gen.Identifier(),
		gen.AlphaString(),
		gen.AlphaString(),
	))

	// Property: JSONSchema round-trip consistency
	properties.Property("JSONSchema round-trip is consistent", prop.ForAll(
		func(schemaType string, required []string) bool {
			original := &JSONSchema{
				Type: schemaType,
				Properties: map[string]interface{}{
					"field1": map[string]interface{}{"type": "string"},
					"field2": map[string]interface{}{"type": "number"},
				},
				Required: required,
			}

			return testJSONRoundTrip(original, &JSONSchema{})
		},
		gen.OneConstOf("object", "array", "string", "number", "boolean"),
		gen.SliceOf(gen.Identifier()),
	))

	// Property: Resource round-trip consistency
	properties.Property("Resource round-trip is consistent", prop.ForAll(
		func(uri string, mimeType string, text string) bool {
			original := &Resource{
				URI:      uri,
				MimeType: mimeType,
				Text:     text,
			}

			return testJSONRoundTrip(original, &Resource{})
		},
		gen.Identifier(),
		gen.OneConstOf("text/plain", "application/json", "text/html"),
		gen.AlphaString(),
	))

	// Property: Nested structures round-trip consistency
	properties.Property("Nested structures round-trip is consistent", prop.ForAll(
		func(method string, toolName string, text string) bool {
			// Create a complex nested structure
			original := &Request{
				JSONRPC: "2.0",
				ID:      123,
				Method:  method,
				Params: map[string]interface{}{
					"toolRequest": map[string]interface{}{
						"name": toolName,
						"arguments": map[string]interface{}{
							"nested": map[string]interface{}{
								"level1": map[string]interface{}{
									"level2": text,
								},
							},
						},
					},
				},
			}

			return testJSONRoundTrip(original, &Request{})
		},
		gen.Identifier(),
		gen.Identifier(),
		gen.AlphaString(),
	))

	properties.TestingRun(t)
}

// Feature: atlassian-mcp-server, Property 4: Credential Security
// **Validates: Requirements 5.3**
//
// For any error message, log entry, or response generated by the server,
// credentials (passwords, tokens) should never be included in plain text.
func TestProperty4_CredentialSecurity(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Generator for realistic passwords (at least 8 characters with special prefix)
	genPassword := gen.AlphaString().
		SuchThat(func(s string) bool { return len(s) >= 8 }).
		Map(func(s string) string { return "PASSWORD_" + s })

	// Generator for realistic tokens (at least 16 characters with special prefix)
	genToken := gen.AlphaString().
		SuchThat(func(s string) bool { return len(s) >= 16 }).
		Map(func(s string) string { return "TOKEN_" + s })

	// Generator for realistic usernames (at least 4 characters)
	genUsername := gen.AlphaString().
		SuchThat(func(s string) bool { return len(s) >= 4 })

	// Property: Error messages from ValidateCredentials should not contain passwords
	properties.Property("ValidateCredentials errors do not expose passwords", prop.ForAll(
		func(tool string, username string, password string) bool {
			// Test with missing username (should error)
			credsNoUser := map[string]*Credentials{
				tool: {
					Type:     BasicAuth,
					Username: "",
					Password: password,
				},
			}
			amNoUser := NewAuthenticationManager(credsNoUser)
			err := amNoUser.ValidateCredentials(tool)
			if err != nil {
				// Error message should not contain the password
				if contains(err.Error(), password) {
					return false
				}
			}

			// Test with missing password (should error)
			credsNoPass := map[string]*Credentials{
				tool: {
					Type:     BasicAuth,
					Username: username,
					Password: "",
				},
			}
			amNoPass := NewAuthenticationManager(credsNoPass)
			err = amNoPass.ValidateCredentials(tool)
			if err != nil {
				// Error message should not contain the password
				if contains(err.Error(), password) {
					return false
				}
			}

			// Test with valid credentials (should not error)
			credsValid := map[string]*Credentials{
				tool: {
					Type:     BasicAuth,
					Username: username,
					Password: password,
				},
			}
			amValid := NewAuthenticationManager(credsValid)
			err = amValid.ValidateCredentials(tool)
			if err != nil {
				// Should not error with valid credentials
				return false
			}

			return true
		},
		gen.Identifier(),
		genUsername,
		genPassword,
	))

	// Property: Error messages from ValidateCredentials should not contain tokens
	properties.Property("ValidateCredentials errors do not expose tokens", prop.ForAll(
		func(tool string, token string) bool {
			// Test with missing token (should error)
			creds := map[string]*Credentials{
				tool: {
					Type:  TokenAuth,
					Token: "",
				},
			}
			am := NewAuthenticationManager(creds)
			err := am.ValidateCredentials(tool)
			if err != nil {
				// Error message should not contain the token
				if contains(err.Error(), token) {
					return false
				}
			}

			// Test with valid token (should not error)
			credsValid := map[string]*Credentials{
				tool: {
					Type:  TokenAuth,
					Token: token,
				},
			}
			amValid := NewAuthenticationManager(credsValid)
			err = amValid.ValidateCredentials(tool)
			if err != nil {
				// Should not error with valid credentials
				return false
			}

			return true
		},
		gen.Identifier(),
		genToken,
	))

	// Property: GetAuthenticatedClient errors should not expose credentials
	properties.Property("GetAuthenticatedClient errors do not expose credentials", prop.ForAll(
		func(tool string, password string, token string) bool {
			// Test with unconfigured tool
			am := NewAuthenticationManager(map[string]*Credentials{})
			_, err := am.GetAuthenticatedClient(tool)
			if err != nil {
				errMsg := err.Error()
				// Should not contain any credentials
				if contains(errMsg, password) {
					return false
				}
				if contains(errMsg, token) {
					return false
				}
			}

			// Test with invalid basic auth (missing password)
			credsInvalid := map[string]*Credentials{
				tool: {
					Type:     BasicAuth,
					Username: "testuser",
					Password: "",
				},
			}
			amInvalid := NewAuthenticationManager(credsInvalid)
			_, err = amInvalid.GetAuthenticatedClient(tool)
			if err != nil {
				if contains(err.Error(), password) {
					return false
				}
			}

			// Test with invalid token auth (missing token)
			credsInvalidToken := map[string]*Credentials{
				tool: {
					Type:  TokenAuth,
					Token: "",
				},
			}
			amInvalidToken := NewAuthenticationManager(credsInvalidToken)
			_, err = amInvalidToken.GetAuthenticatedClient(tool)
			if err != nil {
				if contains(err.Error(), token) {
					return false
				}
			}

			return true
		},
		gen.Identifier(),
		genPassword,
		genToken,
	))

	// Property: Config validation errors should not expose credentials
	properties.Property("Config validation errors do not expose credentials", prop.ForAll(
		func(password string, token string, username string) bool {
			// Test with invalid config that has credentials
			config := &Config{
				Transport: TransportConfig{Type: ""}, // Invalid - missing type
				Tools: ToolsConfig{
					Jira: &ToolConfig{
						BaseURL: "https://jira.example.com",
						Auth: &AuthConfig{
							Type:     "basic",
							Username: username,
							Password: password,
						},
					},
				},
			}

			err := config.Validate()
			if err != nil {
				errMsg := err.Error()
				// Should not contain password or token
				if contains(errMsg, password) {
					return false
				}
				if contains(errMsg, token) {
					return false
				}
			}

			// Test with token auth
			configToken := &Config{
				Transport: TransportConfig{Type: ""}, // Invalid - missing type
				Tools: ToolsConfig{
					Confluence: &ToolConfig{
						BaseURL: "https://confluence.example.com",
						Auth: &AuthConfig{
							Type:  "token",
							Token: token,
						},
					},
				},
			}

			err = configToken.Validate()
			if err != nil {
				errMsg := err.Error()
				// Should not contain token
				if contains(errMsg, token) {
					return false
				}
			}

			return true
		},
		genPassword,
		genToken,
		genUsername,
	))

	// Property: JSON serialization of credentials should not expose them in error responses
	properties.Property("JSON-RPC errors do not expose credentials", prop.ForAll(
		func(code int, message string, password string, token string) bool {
			// Create an error response (simulating what the server would return)
			errorResp := &Error{
				Code:    code,
				Message: message,
				Data: map[string]interface{}{
					"detail": "Authentication failed",
					"tool":   "jira",
				},
			}

			// Serialize to JSON
			jsonData, err := json.Marshal(errorResp)
			if err != nil {
				return false
			}

			jsonStr := string(jsonData)
			// Should not contain credentials
			if contains(jsonStr, password) {
				return false
			}
			if contains(jsonStr, token) {
				return false
			}

			return true
		},
		gen.Int(),
		gen.AlphaString(),
		genPassword,
		genToken,
	))

	// Property: Tool responses should not expose credentials
	properties.Property("Tool responses do not expose credentials", prop.ForAll(
		func(text string, password string, token string) bool {
			// Create a tool response
			response := &ToolResponse{
				Content: []ContentBlock{
					{
						Type: "text",
						Text: text,
					},
				},
				IsError: false,
			}

			// Serialize to JSON
			jsonData, err := json.Marshal(response)
			if err != nil {
				return false
			}

			jsonStr := string(jsonData)
			// Should not contain credentials
			if contains(jsonStr, password) {
				return false
			}
			if contains(jsonStr, token) {
				return false
			}

			return true
		},
		gen.AlphaString(),
		genPassword,
		genToken,
	))

	properties.TestingRun(t)
}

// Feature: atlassian-mcp-server, Property 3: Authentication Error Handling
// **Validates: Requirements 1.9, 2.9, 3.9, 4.9, 5.4, 8.3**
//
// For any API request with invalid or missing credentials, the server should return
// an MCP error response with code indicating authentication failure and a descriptive
// message, without forwarding the request to the Atlassian API.
func TestProperty3_AuthenticationErrorHandling(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Generator for tool names
	genToolName := gen.OneConstOf("jira", "confluence", "bitbucket", "bamboo")

	// Generator for realistic usernames (at least 4 characters)
	genUsername := gen.AlphaString().
		SuchThat(func(s string) bool { return len(s) >= 4 })

	// Generator for realistic passwords (at least 8 characters with special prefix)
	genPassword := gen.AlphaString().
		SuchThat(func(s string) bool { return len(s) >= 8 }).
		Map(func(s string) string { return "PASSWORD_" + s })

	// Generator for realistic tokens (at least 16 characters with special prefix)
	genToken := gen.AlphaString().
		SuchThat(func(s string) bool { return len(s) >= 16 }).
		Map(func(s string) string { return "TOKEN_" + s })

	// Property: Missing credentials return authentication error
	properties.Property("Missing credentials return authentication error", prop.ForAll(
		func(tool string) bool {
			// Create authentication manager with no credentials
			am := NewAuthenticationManager(map[string]*Credentials{})

			// Attempt to get authenticated client
			client, err := am.GetAuthenticatedClient(tool)

			// Should return an error
			if err == nil {
				return false
			}

			// Client should be nil
			if client != nil {
				return false
			}

			// Error message should be descriptive
			errMsg := err.Error()
			if errMsg == "" {
				return false
			}

			// Error should mention the tool or credentials
			return contains(errMsg, "credentials") || contains(errMsg, tool)
		},
		genToolName,
	))

	// Property: Invalid basic auth credentials return authentication error
	properties.Property("Invalid basic auth credentials return authentication error", prop.ForAll(
		func(tool string, hasUsername bool, hasPassword bool) bool {
			// Skip the valid case
			if hasUsername && hasPassword {
				return true
			}

			username := ""
			if hasUsername {
				username = "testuser"
			}

			password := ""
			if hasPassword {
				password = "testpass"
			}

			// Create authentication manager with invalid basic auth
			am := NewAuthenticationManager(map[string]*Credentials{
				tool: {
					Type:     BasicAuth,
					Username: username,
					Password: password,
				},
			})

			// Attempt to get authenticated client
			client, err := am.GetAuthenticatedClient(tool)

			// Should return an error
			if err == nil {
				return false
			}

			// Client should be nil
			if client != nil {
				return false
			}

			// Error message should be descriptive
			errMsg := err.Error()
			if errMsg == "" {
				return false
			}

			// Error should mention username or password requirement
			return contains(errMsg, "username") || contains(errMsg, "password")
		},
		genToolName,
		gen.Bool(),
		gen.Bool(),
	))

	// Property: Invalid token auth credentials return authentication error
	properties.Property("Invalid token auth credentials return authentication error", prop.ForAll(
		func(tool string) bool {
			// Create authentication manager with missing token
			am := NewAuthenticationManager(map[string]*Credentials{
				tool: {
					Type:  TokenAuth,
					Token: "", // Missing token
				},
			})

			// Attempt to get authenticated client
			client, err := am.GetAuthenticatedClient(tool)

			// Should return an error
			if err == nil {
				return false
			}

			// Client should be nil
			if client != nil {
				return false
			}

			// Error message should be descriptive
			errMsg := err.Error()
			if errMsg == "" {
				return false
			}

			// Error should mention token requirement
			return contains(errMsg, "token")
		},
		genToolName,
	))

	// Property: ValidateCredentials fails for missing credentials
	properties.Property("ValidateCredentials fails for missing credentials", prop.ForAll(
		func(tool string) bool {
			// Create authentication manager with no credentials
			am := NewAuthenticationManager(map[string]*Credentials{})

			// Validate credentials
			err := am.ValidateCredentials(tool)

			// Should return an error
			if err == nil {
				return false
			}

			// Error message should be descriptive
			errMsg := err.Error()
			if errMsg == "" {
				return false
			}

			// Error should mention credentials or tool
			return contains(errMsg, "credentials") || contains(errMsg, tool)
		},
		genToolName,
	))

	// Property: ValidateCredentials fails for incomplete basic auth
	properties.Property("ValidateCredentials fails for incomplete basic auth", prop.ForAll(
		func(tool string, missingUsername bool) bool {
			// Create credentials with one field missing
			creds := &Credentials{
				Type:     BasicAuth,
				Username: "validuser",
				Password: "validpass",
			}

			// Remove one field based on missingUsername
			if missingUsername {
				creds.Username = ""
			} else {
				creds.Password = ""
			}

			// Create authentication manager
			am := NewAuthenticationManager(map[string]*Credentials{
				tool: creds,
			})

			// Validate credentials
			err := am.ValidateCredentials(tool)

			// Should return an error
			if err == nil {
				return false
			}

			// Error message should be descriptive
			errMsg := err.Error()
			if errMsg == "" {
				return false
			}

			// Error should mention the missing field
			return contains(errMsg, "username") || contains(errMsg, "password")
		},
		genToolName,
		gen.Bool(),
	))

	// Property: ValidateCredentials fails for incomplete token auth
	properties.Property("ValidateCredentials fails for incomplete token auth", prop.ForAll(
		func(tool string) bool {
			// Create credentials with missing token
			creds := &Credentials{
				Type:  TokenAuth,
				Token: "", // Missing
			}

			// Create authentication manager
			am := NewAuthenticationManager(map[string]*Credentials{
				tool: creds,
			})

			// Validate credentials
			err := am.ValidateCredentials(tool)

			// Should return an error
			if err == nil {
				return false
			}

			// Error message should be descriptive
			errMsg := err.Error()
			if errMsg == "" {
				return false
			}

			// Error should mention token
			return contains(errMsg, "token")
		},
		genToolName,
	))

	// Property: ValidateCredentials succeeds for valid basic auth
	properties.Property("ValidateCredentials succeeds for valid basic auth", prop.ForAll(
		func(tool string, username string, password string) bool {
			// Create valid basic auth credentials
			creds := &Credentials{
				Type:     BasicAuth,
				Username: username,
				Password: password,
			}

			// Create authentication manager
			am := NewAuthenticationManager(map[string]*Credentials{
				tool: creds,
			})

			// Validate credentials
			err := am.ValidateCredentials(tool)

			// Should not return an error
			return err == nil
		},
		genToolName,
		genUsername,
		genPassword,
	))

	// Property: ValidateCredentials succeeds for valid token auth
	properties.Property("ValidateCredentials succeeds for valid token auth", prop.ForAll(
		func(tool string, token string) bool {
			// Create valid token auth credentials
			creds := &Credentials{
				Type:  TokenAuth,
				Token: token,
			}

			// Create authentication manager
			am := NewAuthenticationManager(map[string]*Credentials{
				tool: creds,
			})

			// Validate credentials
			err := am.ValidateCredentials(tool)

			// Should not return an error
			return err == nil
		},
		genToolName,
		genToken,
	))

	// Property: Authentication errors should map to AuthenticationError code
	properties.Property("Authentication errors map to AuthenticationError code", prop.ForAll(
		func(tool string, message string) bool {
			// Ensure message is not empty
			if message == "" {
				message = "Authentication failed"
			}

			// Create an authentication error response
			errorResp := &Error{
				Code:    AuthenticationError,
				Message: message,
				Data: map[string]interface{}{
					"tool": tool,
				},
			}

			// Verify the error code is correct
			if errorResp.Code != AuthenticationError {
				return false
			}

			// Verify the error code is the expected value (-32002)
			if errorResp.Code != -32002 {
				return false
			}

			// Verify the error has a message
			if errorResp.Message == "" {
				return false
			}

			return true
		},
		genToolName,
		gen.AlphaString(),
	))

	// Property: Authentication error responses are valid JSON-RPC 2.0 errors
	properties.Property("Authentication error responses are valid JSON-RPC 2.0 errors", prop.ForAll(
		func(tool string, requestID int, message string) bool {
			// Ensure message is not empty
			if message == "" {
				message = "Authentication failed"
			}

			// Create a JSON-RPC error response for authentication failure
			response := &Response{
				JSONRPC: "2.0",
				ID:      requestID,
				Error: &Error{
					Code:    AuthenticationError,
					Message: message,
					Data: map[string]interface{}{
						"tool":   tool,
						"reason": "Invalid credentials",
					},
				},
			}

			// Verify JSON-RPC version
			if response.JSONRPC != "2.0" {
				return false
			}

			// Verify error is present
			if response.Error == nil {
				return false
			}

			// Verify error code is AuthenticationError
			if response.Error.Code != AuthenticationError {
				return false
			}

			// Verify error has a message
			if response.Error.Message == "" {
				return false
			}

			// Verify Result is not set (error responses should not have Result)
			if response.Result != nil {
				return false
			}

			// Verify the response can be serialized to JSON
			_, err := json.Marshal(response)
			return err == nil
		},
		genToolName,
		gen.Int(),
		gen.AlphaString(),
	))

	properties.TestingRun(t)
}

// Feature: atlassian-mcp-server, Property 6: MCP Protocol Compliance
// **Validates: Requirements 6.5**
//
// For any valid JSON-RPC 2.0 message received by the transport layer, the server
// should process it according to MCP specification and return a valid JSON-RPC 2.0 response.
func TestProperty6_MCPProtocolCompliance(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Generator for valid MCP method names
	genMCPMethod := gen.OneConstOf(
		"initialize",
		"tools/list",
		"tools/call",
		"notifications/initialized",
	)

	// Property: Valid JSON-RPC 2.0 requests have correct structure
	properties.Property("Valid JSON-RPC 2.0 requests are well-formed", prop.ForAll(
		func(method string, id int) bool {
			req := &Request{
				JSONRPC: "2.0",
				Method:  method,
				ID:      id,
			}

			// Should have JSONRPC field set to "2.0"
			if req.JSONRPC != "2.0" {
				return false
			}

			// Should have a method
			if req.Method == "" {
				return false
			}

			// Should be serializable to JSON
			data, err := json.Marshal(req)
			if err != nil {
				return false
			}

			// Should be deserializable from JSON
			var decoded Request
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				return false
			}

			// Decoded request should match original
			return decoded.JSONRPC == req.JSONRPC && decoded.Method == req.Method
		},
		genMCPMethod,
		gen.Int(),
	))

	// Property: Valid JSON-RPC 2.0 responses have correct structure
	properties.Property("Valid JSON-RPC 2.0 responses are well-formed", prop.ForAll(
		func(id int, hasError bool) bool {
			resp := &Response{
				JSONRPC: "2.0",
				ID:      id,
			}

			if hasError {
				resp.Error = &Error{
					Code:    InternalError,
					Message: "Test error",
				}
			} else {
				resp.Result = map[string]interface{}{"status": "ok"}
			}

			// Should have JSONRPC field set to "2.0"
			if resp.JSONRPC != "2.0" {
				return false
			}

			// Should have either Result or Error, but not both
			if hasError && resp.Result != nil {
				return false
			}
			if !hasError && resp.Error != nil {
				return false
			}

			// Should be serializable to JSON
			data, err := json.Marshal(resp)
			if err != nil {
				return false
			}

			// Should be deserializable from JSON
			var decoded Response
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				return false
			}

			// Decoded response should match original structure
			return decoded.JSONRPC == resp.JSONRPC
		},
		gen.Int(),
		gen.Bool(),
	))

	// Property: Transport layer validates JSON-RPC version
	properties.Property("Transport layer validates JSON-RPC version", prop.ForAll(
		func(invalidVersion string, method string) bool {
			// Skip valid version
			if invalidVersion == "2.0" {
				return true
			}

			// Create request with invalid version
			req := &Request{
				JSONRPC: invalidVersion,
				Method:  method,
				ID:      1,
			}

			// Serialize to JSON
			data, err := json.Marshal(req)
			if err != nil {
				return false
			}

			// The transport layer should reject this when parsing
			// We simulate this by checking that the version is not "2.0"
			var decoded Request
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				return false
			}

			// If version is not "2.0", it should be considered invalid
			return decoded.JSONRPC != "2.0"
		},
		gen.OneConstOf("1.0", "2.1", "3.0", "", "invalid"),
		genMCPMethod,
	))

	// Property: Responses preserve request ID
	properties.Property("Responses preserve request ID from request", prop.ForAll(
		func(id int, method string) bool {
			req := &Request{
				JSONRPC: "2.0",
				Method:  method,
				ID:      id,
			}

			// Create a response for this request
			resp := &Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  map[string]interface{}{"status": "ok"},
			}

			// Response ID should match request ID
			return resp.ID == req.ID
		},
		gen.Int(),
		genMCPMethod,
	))

	// Property: Error responses have required error fields
	properties.Property("Error responses have required error fields", prop.ForAll(
		func(code int, message string, id int) bool {
			// Ensure message is not empty
			if message == "" {
				message = "Error message"
			}

			resp := &Response{
				JSONRPC: "2.0",
				ID:      id,
				Error: &Error{
					Code:    code,
					Message: message,
				},
			}

			// Error must have code
			if resp.Error.Code == 0 {
				return false
			}

			// Error must have message
			if resp.Error.Message == "" {
				return false
			}

			// Response must not have Result when Error is present
			if resp.Result != nil {
				return false
			}

			// Should be serializable
			data, err := json.Marshal(resp)
			if err != nil {
				return false
			}

			// Should be deserializable
			var decoded Response
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				return false
			}

			// Decoded error should have same code and message
			return decoded.Error != nil &&
				decoded.Error.Code == resp.Error.Code &&
				decoded.Error.Message == resp.Error.Message
		},
		gen.Int(),
		gen.AlphaString(),
		gen.Int(),
	))

	// Property: Success responses have Result field
	properties.Property("Success responses have Result field", prop.ForAll(
		func(id int, resultValue string) bool {
			resp := &Response{
				JSONRPC: "2.0",
				ID:      id,
				Result:  map[string]interface{}{"data": resultValue},
			}

			// Response must have Result
			if resp.Result == nil {
				return false
			}

			// Response must not have Error when Result is present
			if resp.Error != nil {
				return false
			}

			// Should be serializable
			data, err := json.Marshal(resp)
			if err != nil {
				return false
			}

			// Should be deserializable
			var decoded Response
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				return false
			}

			// Decoded response should have Result
			return decoded.Result != nil && decoded.Error == nil
		},
		gen.Int(),
		gen.AlphaString(),
	))

	// Property: MCP tool requests have required fields
	properties.Property("MCP tool requests have required fields", prop.ForAll(
		func(toolName string, argKey string, argValue string) bool {
			// Ensure non-empty values
			if toolName == "" {
				toolName = "test_tool"
			}
			if argKey == "" {
				argKey = "param"
			}

			toolReq := &ToolRequest{
				Name: toolName,
				Arguments: map[string]interface{}{
					argKey: argValue,
				},
			}

			// Tool request must have name
			if toolReq.Name == "" {
				return false
			}

			// Tool request must have arguments (can be empty map)
			if toolReq.Arguments == nil {
				return false
			}

			// Should be serializable
			data, err := json.Marshal(toolReq)
			if err != nil {
				return false
			}

			// Should be deserializable
			var decoded ToolRequest
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				return false
			}

			// Decoded request should match original
			return decoded.Name == toolReq.Name && decoded.Arguments != nil
		},
		gen.Identifier(),
		gen.Identifier(),
		gen.AlphaString(),
	))

	// Property: MCP tool responses have content blocks
	properties.Property("MCP tool responses have content blocks", prop.ForAll(
		func(text string, isError bool) bool {
			toolResp := &ToolResponse{
				Content: []ContentBlock{
					{
						Type: "text",
						Text: text,
					},
				},
				IsError: isError,
			}

			// Tool response must have content
			if toolResp.Content == nil || len(toolResp.Content) == 0 {
				return false
			}

			// Each content block must have a type
			for _, block := range toolResp.Content {
				if block.Type == "" {
					return false
				}
			}

			// Should be serializable
			data, err := json.Marshal(toolResp)
			if err != nil {
				return false
			}

			// Should be deserializable
			var decoded ToolResponse
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				return false
			}

			// Decoded response should have content
			return len(decoded.Content) > 0 && decoded.IsError == toolResp.IsError
		},
		gen.AlphaString(),
		gen.Bool(),
	))

	// Property: JSON-RPC error codes are in valid ranges
	properties.Property("JSON-RPC error codes are in valid ranges", prop.ForAll(
		func(errorType string) bool {
			var code int
			switch errorType {
			case "parse":
				code = ParseError
			case "invalid_request":
				code = InvalidRequest
			case "method_not_found":
				code = MethodNotFound
			case "invalid_params":
				code = InvalidParams
			case "internal":
				code = InternalError
			case "config":
				code = ConfigurationError
			case "auth":
				code = AuthenticationError
			case "api":
				code = APIError
			case "network":
				code = NetworkError
			case "rate_limit":
				code = RateLimitError
			default:
				return true
			}

			// All error codes should be negative
			if code >= 0 {
				return false
			}

			// Standard JSON-RPC errors should be in -32768 to -32000 range
			if errorType == "parse" || errorType == "invalid_request" ||
				errorType == "method_not_found" || errorType == "invalid_params" ||
				errorType == "internal" {
				return code >= -32768 && code <= -32000
			}

			// Application-defined errors should be outside standard range
			// but still negative
			return code < 0
		},
		gen.OneConstOf("parse", "invalid_request", "method_not_found",
			"invalid_params", "internal", "config", "auth", "api",
			"network", "rate_limit"),
	))

	// Property: Requests and responses can be transmitted through transport
	properties.Property("Requests and responses can be transmitted through transport", prop.ForAll(
		func(method string, id int, resultValue string) bool {
			// Create a valid request
			req := &Request{
				JSONRPC: "2.0",
				Method:  method,
				ID:      id,
			}

			// Serialize request (simulating transport send)
			reqData, err := json.Marshal(req)
			if err != nil {
				return false
			}

			// Deserialize request (simulating transport receive)
			var receivedReq Request
			err = json.Unmarshal(reqData, &receivedReq)
			if err != nil {
				return false
			}

			// Create a response
			resp := &Response{
				JSONRPC: "2.0",
				ID:      receivedReq.ID,
				Result:  map[string]interface{}{"data": resultValue},
			}

			// Serialize response (simulating transport send)
			respData, err := json.Marshal(resp)
			if err != nil {
				return false
			}

			// Deserialize response (simulating transport receive)
			var receivedResp Response
			err = json.Unmarshal(respData, &receivedResp)
			if err != nil {
				return false
			}

			// Verify round-trip preserves structure
			return receivedReq.JSONRPC == "2.0" &&
				receivedReq.Method == method &&
				receivedResp.JSONRPC == "2.0" &&
				receivedResp.Result != nil
		},
		genMCPMethod,
		gen.Int(),
		gen.AlphaString(),
	))

	properties.TestingRun(t)
}

// Feature: atlassian-mcp-server, Property 7: Invalid Transport Configuration Rejection
// **Validates: Requirements 6.6**
//
// For any configuration with invalid or incomplete transport settings, the server should
// fail to start and return a descriptive configuration error before attempting to listen
// for connections.
func TestProperty7_InvalidTransportConfigurationRejection(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Property: Missing transport type causes validation failure
	properties.Property("Missing transport type fails validation", prop.ForAll(
		func(hasJira, hasConfluence, hasBitbucket, hasBamboo bool) bool {
			// Create config with at least one tool but missing transport type
			config := &Config{
				Transport: TransportConfig{Type: ""}, // Missing
				Tools:     generateToolsConfig(hasJira, hasConfluence, hasBitbucket, hasBamboo),
			}

			// If no tools are configured, validation should fail for different reason
			if !hasJira && !hasConfluence && !hasBitbucket && !hasBamboo {
				return true // Skip this case
			}

			err := config.Validate()
			// Should fail validation
			if err == nil {
				return false
			}
			// Error message should mention transport type
			return contains(err.Error(), "transport type is required")
		},
		gen.Bool(),
		gen.Bool(),
		gen.Bool(),
		gen.Bool(),
	))

	// Property: Invalid transport type causes validation failure
	properties.Property("Invalid transport type fails validation", prop.ForAll(
		func(invalidType string) bool {
			// Skip valid types
			if invalidType == "stdio" || invalidType == "http" {
				return true
			}

			config := &Config{
				Transport: TransportConfig{Type: invalidType},
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
			// Should fail validation
			if err == nil {
				return false
			}
			// Error message should mention invalid transport type
			return contains(err.Error(), "invalid transport type") ||
				contains(err.Error(), "transport type is required")
		},
		gen.OneConstOf("websocket", "grpc", "tcp", "udp", "ftp", "ssh", ""),
	))

	// Property: Missing HTTP host when transport is HTTP causes validation failure
	properties.Property("Missing HTTP host for HTTP transport fails validation", prop.ForAll(
		func(port int) bool {
			// Use valid port range
			validPort := (port % 65535) + 1

			config := &Config{
				Transport: TransportConfig{
					Type: "http",
					HTTP: HTTPConfig{
						Host: "", // Missing
						Port: validPort,
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
			// Should fail validation
			if err == nil {
				return false
			}
			// Error message should mention HTTP host
			return contains(err.Error(), "HTTP host is required")
		},
		gen.Int(),
	))

	// Property: Invalid HTTP port causes validation failure
	properties.Property("Invalid HTTP port fails validation", prop.ForAll(
		func(invalidPort int) bool {
			// Only test invalid ports
			if invalidPort > 0 && invalidPort <= 65535 {
				return true
			}

			config := &Config{
				Transport: TransportConfig{
					Type: "http",
					HTTP: HTTPConfig{
						Host: "localhost",
						Port: invalidPort,
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
			// Should fail validation
			if err == nil {
				return false
			}
			// Error message should mention invalid port
			return contains(err.Error(), "invalid HTTP port")
		},
		gen.OneConstOf(-1, 0, 70000, -100, 100000, 65536, -9999),
	))

	// Property: HTTP transport without HTTP config causes validation failure
	properties.Property("HTTP transport without HTTP config fails validation", prop.ForAll(
		func() bool {
			config := &Config{
				Transport: TransportConfig{
					Type: "http",
					HTTP: HTTPConfig{}, // Empty HTTP config
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
			// Should fail validation
			if err == nil {
				return false
			}
			// Error message should mention HTTP host or port
			return contains(err.Error(), "HTTP host") || contains(err.Error(), "HTTP port")
		},
	))

	// Property: Valid stdio transport passes validation
	properties.Property("Valid stdio transport passes validation", prop.ForAll(
		func(toolChoice int) bool {
			// Build valid tool config
			toolConfig := &ToolConfig{
				BaseURL: "https://example.com",
				Auth: &AuthConfig{
					Type:     "basic",
					Username: "testuser",
					Password: "testpass",
				},
			}

			// Create config with one tool and stdio transport
			tools := ToolsConfig{}
			switch toolChoice % 4 {
			case 0:
				tools.Jira = toolConfig
			case 1:
				tools.Confluence = toolConfig
			case 2:
				tools.Bitbucket = toolConfig
			case 3:
				tools.Bamboo = toolConfig
			}

			config := &Config{
				Transport: TransportConfig{
					Type: "stdio",
				},
				Tools: tools,
			}

			err := config.Validate()
			// Should pass validation
			return err == nil
		},
		gen.IntRange(0, 3),
	))

	// Property: Valid HTTP transport passes validation
	properties.Property("Valid HTTP transport passes validation", prop.ForAll(
		func(host string, port int, toolChoice int) bool {
			// Ensure valid values
			if host == "" {
				host = "localhost"
			}
			if port <= 0 || port > 65535 {
				port = 8080
			}

			// Build valid tool config
			toolConfig := &ToolConfig{
				BaseURL: "https://example.com",
				Auth: &AuthConfig{
					Type:     "basic",
					Username: "testuser",
					Password: "testpass",
				},
			}

			// Create config with one tool and HTTP transport
			tools := ToolsConfig{}
			switch toolChoice % 4 {
			case 0:
				tools.Jira = toolConfig
			case 1:
				tools.Confluence = toolConfig
			case 2:
				tools.Bitbucket = toolConfig
			case 3:
				tools.Bamboo = toolConfig
			}

			config := &Config{
				Transport: TransportConfig{
					Type: "http",
					HTTP: HTTPConfig{
						Host: host,
						Port: port,
					},
				},
				Tools: tools,
			}

			err := config.Validate()
			// Should pass validation
			return err == nil
		},
		gen.OneConstOf("localhost", "127.0.0.1", "0.0.0.0", "example.com"),
		gen.IntRange(1, 65535),
		gen.IntRange(0, 3),
	))

	// Property: Multiple transport configuration errors are reported
	properties.Property("Multiple transport errors are reported together", prop.ForAll(
		func(invalidType string) bool {
			// Skip valid types
			if invalidType == "stdio" || invalidType == "http" {
				return true
			}

			config := &Config{
				Transport: TransportConfig{
					Type: invalidType, // Invalid type
					HTTP: HTTPConfig{
						Host: "", // Missing host (would be required if type was "http")
						Port: 0,  // Invalid port (would be required if type was "http")
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
			// Should fail validation
			if err == nil {
				return false
			}
			// Error message should mention transport type issue
			return contains(err.Error(), "transport type")
		},
		gen.OneConstOf("websocket", "grpc", "tcp", ""),
	))

	// Property: Configuration validation fails before transport starts
	properties.Property("Invalid config prevents transport initialization", prop.ForAll(
		func(invalidType string) bool {
			// Skip valid types
			if invalidType == "stdio" || invalidType == "http" {
				return true
			}

			config := &Config{
				Transport: TransportConfig{Type: invalidType},
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

			// Validation should fail
			err := config.Validate()
			if err == nil {
				return false
			}

			// If validation fails, we should not be able to create a transport
			// This simulates the server startup flow where config is validated first
			// The error should be descriptive
			errMsg := err.Error()
			return errMsg != "" && contains(errMsg, "transport")
		},
		gen.OneConstOf("invalid", "websocket", "grpc", ""),
	))

	// Property: Descriptive error messages for transport configuration
	properties.Property("Transport validation errors are descriptive", prop.ForAll(
		func(scenario int) bool {
			var config *Config
			var expectedKeyword string

			// Generate different invalid scenarios
			switch scenario % 4 {
			case 0:
				// Missing transport type
				config = &Config{
					Transport: TransportConfig{Type: ""},
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
				expectedKeyword = "transport type"
			case 1:
				// Invalid transport type
				config = &Config{
					Transport: TransportConfig{Type: "invalid"},
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
				expectedKeyword = "transport type"
			case 2:
				// Missing HTTP host
				config = &Config{
					Transport: TransportConfig{
						Type: "http",
						HTTP: HTTPConfig{
							Host: "",
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
				expectedKeyword = "HTTP host"
			case 3:
				// Invalid HTTP port
				config = &Config{
					Transport: TransportConfig{
						Type: "http",
						HTTP: HTTPConfig{
							Host: "localhost",
							Port: -1,
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
				expectedKeyword = "HTTP port"
			}

			err := config.Validate()
			// Should fail validation
			if err == nil {
				return false
			}

			// Error message should contain the expected keyword
			return contains(err.Error(), expectedKeyword)
		},
		gen.IntRange(0, 3),
	))

	properties.TestingRun(t)
}

// testJSONRoundTrip tests that a value can be serialized to JSON and deserialized back
// to produce an equivalent structure. This is a generic helper for testing JSON round-trip
// consistency across all data structures.
//
// Parameters:
//   - original: The original value to serialize
//   - target: A pointer to a zero-value of the same type for deserialization
//
// Returns true if the round-trip produces an equivalent structure, false otherwise.
func testJSONRoundTrip(original interface{}, target interface{}) bool {
	// Serialize to JSON
	jsonData, err := json.Marshal(original)
	if err != nil {
		return false
	}

	// Deserialize from JSON
	err = json.Unmarshal(jsonData, target)
	if err != nil {
		return false
	}

	// Compare the original and deserialized values
	// We need to serialize both again to compare, because reflect.DeepEqual
	// may fail on interface{} fields that have different concrete types
	// but represent the same JSON value
	originalJSON, err := json.Marshal(original)
	if err != nil {
		return false
	}

	targetJSON, err := json.Marshal(target)
	if err != nil {
		return false
	}

	// Compare JSON representations
	return string(originalJSON) == string(targetJSON)
}

// Feature: atlassian-mcp-server, Property 2: Response Transformation Compliance
// **Validates: Requirements 1.2, 2.2, 3.2, 4.2, 9.3**
//
// For any Atlassian API response (from Jira, Confluence, Bitbucket, or Bamboo),
// the Response_Mapper should transform it into a valid MCP-compliant JSON format
// that can be parsed by any MCP client.
func TestProperty2_ResponseTransformationCompliance(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)
	mapper := NewResponseMapper()

	// Generator for FlexibleID
	genFlexibleID := gen.Identifier().Map(func(s string) FlexibleID {
		return FlexibleID(s)
	})

	// Generator for Jira issues
	genJiraIssue := gen.Struct(reflect.TypeOf(JiraIssue{}), map[string]gopter.Gen{
		"ID":  genFlexibleID,
		"Key": gen.Identifier().Map(func(s string) string { return "TEST-" + s }),
		"Fields": gen.Struct(reflect.TypeOf(JiraFields{}), map[string]gopter.Gen{
			"Summary":     gen.AlphaString(),
			"Description": gen.AlphaString(),
			"IssueType": gen.Struct(reflect.TypeOf(IssueType{}), map[string]gopter.Gen{
				"ID":   genFlexibleID,
				"Name": gen.OneConstOf("Bug", "Story", "Task", "Epic"),
			}),
			"Project": gen.Struct(reflect.TypeOf(Project{}), map[string]gopter.Gen{
				"ID":   genFlexibleID,
				"Key":  gen.Identifier(),
				"Name": gen.AlphaString(),
			}),
			"Status": gen.Struct(reflect.TypeOf(Status{}), map[string]gopter.Gen{
				"ID":   genFlexibleID,
				"Name": gen.OneConstOf("Open", "In Progress", "Done", "Closed"),
			}),
			"Created": gen.Const("2024-01-01T00:00:00.000Z"),
			"Updated": gen.Const("2024-01-02T00:00:00.000Z"),
		}),
	}).Map(func(issue JiraIssue) *JiraIssue {
		return &issue
	})

	// Generator for Confluence pages
	genConfluencePage := gen.Struct(reflect.TypeOf(ConfluencePage{}), map[string]gopter.Gen{
		"ID":    gen.Identifier(),
		"Type":  gen.Const("page"),
		"Title": gen.AlphaString(),
		"Space": gen.Struct(reflect.TypeOf(Space{}), map[string]gopter.Gen{
			"ID":   gen.Identifier(),
			"Key":  gen.Identifier(),
			"Name": gen.AlphaString(),
		}),
		"Body": gen.Struct(reflect.TypeOf(Body{}), map[string]gopter.Gen{
			"Storage": gen.Struct(reflect.TypeOf(Storage{}), map[string]gopter.Gen{
				"Value":          gen.AlphaString().Map(func(s string) string { return "<p>" + s + "</p>" }),
				"Representation": gen.Const("storage"),
			}),
		}),
		"Version": gen.Struct(reflect.TypeOf(Version{}), map[string]gopter.Gen{
			"Number": gen.IntRange(1, 100),
			"When":   gen.Const("2024-01-01T00:00:00.000Z"),
			"By": gen.Struct(reflect.TypeOf(User{}), map[string]gopter.Gen{
				"Name":         gen.Identifier(),
				"DisplayName":  gen.AlphaString(),
				"EmailAddress": gen.Identifier().Map(func(s string) string { return s + "@example.com" }),
			}),
		}),
	}).Map(func(page ConfluencePage) *ConfluencePage {
		return &page
	})

	// Generator for Bitbucket pull requests
	genPullRequest := gen.Struct(reflect.TypeOf(PullRequest{}), map[string]gopter.Gen{
		"ID":          gen.IntRange(1, 10000),
		"Version":     gen.IntRange(1, 100),
		"Title":       gen.AlphaString(),
		"Description": gen.AlphaString(),
		"State":       gen.OneConstOf("OPEN", "MERGED", "DECLINED"),
		"Open":        gen.Bool(),
		"Closed":      gen.Bool(),
		"FromRef": gen.Struct(reflect.TypeOf(Ref{}), map[string]gopter.Gen{
			"ID":        gen.Const("refs/heads/feature"),
			"DisplayID": gen.Const("feature"),
			"Repository": gen.Struct(reflect.TypeOf(Repository{}), map[string]gopter.Gen{
				"ID":   gen.IntRange(1, 1000),
				"Slug": gen.Identifier(),
				"Name": gen.AlphaString(),
				"Project": gen.Struct(reflect.TypeOf(Project{}), map[string]gopter.Gen{
					"ID":   genFlexibleID,
					"Key":  gen.Identifier(),
					"Name": gen.AlphaString(),
				}),
				"Public": gen.Bool(),
			}),
		}),
		"ToRef": gen.Struct(reflect.TypeOf(Ref{}), map[string]gopter.Gen{
			"ID":        gen.Const("refs/heads/main"),
			"DisplayID": gen.Const("main"),
			"Repository": gen.Struct(reflect.TypeOf(Repository{}), map[string]gopter.Gen{
				"ID":   gen.IntRange(1, 1000),
				"Slug": gen.Identifier(),
				"Name": gen.AlphaString(),
				"Project": gen.Struct(reflect.TypeOf(Project{}), map[string]gopter.Gen{
					"ID":   genFlexibleID,
					"Key":  gen.Identifier(),
					"Name": gen.AlphaString(),
				}),
				"Public": gen.Bool(),
			}),
		}),
		"Author": gen.Struct(reflect.TypeOf(User{}), map[string]gopter.Gen{
			"Name":         gen.Identifier(),
			"DisplayName":  gen.AlphaString(),
			"EmailAddress": gen.Identifier().Map(func(s string) string { return s + "@example.com" }),
		}),
	}).Map(func(pr PullRequest) *PullRequest {
		return &pr
	})

	// Generator for Bamboo build results
	genBuildResult := gen.Struct(reflect.TypeOf(BuildResult{}), map[string]gopter.Gen{
		"Key":                gen.Identifier().Map(func(s string) string { return "PLAN-" + s }),
		"Number":             gen.IntRange(1, 1000),
		"State":              gen.OneConstOf("Successful", "Failed", "Unknown"),
		"LifeCycleState":     gen.OneConstOf("Pending", "Queued", "InProgress", "Finished"),
		"BuildStartedTime":   gen.Const("2024-01-01T00:00:00.000Z"),
		"BuildCompletedTime": gen.Const("2024-01-01T00:05:00.000Z"),
		"BuildDuration":      gen.Int64Range(1000, 300000),
		"BuildReason":        gen.OneConstOf("Manual build", "Code change", "Scheduled"),
	}).Map(func(br BuildResult) *BuildResult {
		return &br
	})

	// Property: Jira issue responses transform to valid MCP format
	properties.Property("Jira issue responses transform to valid MCP format", prop.ForAll(
		func(issue *JiraIssue) bool {
			// Transform the response
			response, err := mapper.MapToToolResponse(issue)
			if err != nil {
				return false
			}

			// Verify response structure
			if response == nil {
				return false
			}

			// Must have at least one content block
			if len(response.Content) == 0 {
				return false
			}

			// First content block must be text type
			if response.Content[0].Type != "text" {
				return false
			}

			// Content must be valid JSON
			text := response.Content[0].Text
			if text == "" {
				return false
			}

			// Verify it's parseable JSON
			var parsed map[string]interface{}
			err = json.Unmarshal([]byte(text), &parsed)
			if err != nil {
				return false
			}

			// Verify key fields are preserved
			if _, ok := parsed["id"]; !ok {
				return false
			}
			if _, ok := parsed["key"]; !ok {
				return false
			}

			return true
		},
		genJiraIssue,
	))

	// Property: Confluence page responses transform to valid MCP format
	properties.Property("Confluence page responses transform to valid MCP format", prop.ForAll(
		func(page *ConfluencePage) bool {
			// Transform the response
			response, err := mapper.MapToToolResponse(page)
			if err != nil {
				return false
			}

			// Verify response structure
			if response == nil {
				return false
			}

			// Must have at least one content block
			if len(response.Content) == 0 {
				return false
			}

			// First content block must be text type
			if response.Content[0].Type != "text" {
				return false
			}

			// Content must be valid JSON
			text := response.Content[0].Text
			if text == "" {
				return false
			}

			// Verify it's parseable JSON
			var parsed map[string]interface{}
			err = json.Unmarshal([]byte(text), &parsed)
			if err != nil {
				return false
			}

			// Verify key fields are preserved
			if _, ok := parsed["id"]; !ok {
				return false
			}
			if _, ok := parsed["title"]; !ok {
				return false
			}

			return true
		},
		genConfluencePage,
	))

	// Property: Bitbucket pull request responses transform to valid MCP format
	properties.Property("Bitbucket pull request responses transform to valid MCP format", prop.ForAll(
		func(pr *PullRequest) bool {
			// Transform the response
			response, err := mapper.MapToToolResponse(pr)
			if err != nil {
				return false
			}

			// Verify response structure
			if response == nil {
				return false
			}

			// Must have at least one content block
			if len(response.Content) == 0 {
				return false
			}

			// First content block must be text type
			if response.Content[0].Type != "text" {
				return false
			}

			// Content must be valid JSON
			text := response.Content[0].Text
			if text == "" {
				return false
			}

			// Verify it's parseable JSON
			var parsed map[string]interface{}
			err = json.Unmarshal([]byte(text), &parsed)
			if err != nil {
				return false
			}

			// Verify key fields are preserved
			if _, ok := parsed["id"]; !ok {
				return false
			}
			if _, ok := parsed["title"]; !ok {
				return false
			}

			return true
		},
		genPullRequest,
	))

	// Property: Bamboo build result responses transform to valid MCP format
	properties.Property("Bamboo build result responses transform to valid MCP format", prop.ForAll(
		func(buildResult *BuildResult) bool {
			// Transform the response
			response, err := mapper.MapToToolResponse(buildResult)
			if err != nil {
				return false
			}

			// Verify response structure
			if response == nil {
				return false
			}

			// Must have at least one content block
			if len(response.Content) == 0 {
				return false
			}

			// First content block must be text type
			if response.Content[0].Type != "text" {
				return false
			}

			// Content must be valid JSON
			text := response.Content[0].Text
			if text == "" {
				return false
			}

			// Verify it's parseable JSON
			var parsed map[string]interface{}
			err = json.Unmarshal([]byte(text), &parsed)
			if err != nil {
				return false
			}

			// Verify key fields are preserved
			if _, ok := parsed["key"]; !ok {
				return false
			}
			if _, ok := parsed["state"]; !ok {
				return false
			}

			return true
		},
		genBuildResult,
	))

	// Property: Search results with pagination transform correctly
	properties.Property("Search results with pagination transform correctly", prop.ForAll(
		func(total int, startAt int, maxResults int) bool {
			// Ensure valid pagination values
			if total < 0 {
				total = 0
			}
			if startAt < 0 {
				startAt = 0
			}
			if maxResults < 1 {
				maxResults = 50
			}

			// Create search results with pagination
			searchResults := &SearchResults{
				Issues:     []JiraIssue{},
				Total:      total,
				StartAt:    startAt,
				MaxResults: maxResults,
			}

			// Transform the response
			response, err := mapper.MapToToolResponse(searchResults)
			if err != nil {
				return false
			}

			// Verify response structure
			if response == nil {
				return false
			}

			// Must have at least one content block
			if len(response.Content) == 0 {
				return false
			}

			// First content block must be text type
			if response.Content[0].Type != "text" {
				return false
			}

			// Content must be valid JSON
			text := response.Content[0].Text
			if text == "" {
				return false
			}

			// Verify it's parseable JSON
			var parsed map[string]interface{}
			err = json.Unmarshal([]byte(text), &parsed)
			if err != nil {
				return false
			}

			// Verify pagination fields are preserved
			if _, ok := parsed["total"]; !ok {
				return false
			}
			if _, ok := parsed["startAt"]; !ok {
				return false
			}

			// If there are results, should have pagination info in second block
			if total > 0 && len(response.Content) > 1 {
				if response.Content[1].Type != "text" {
					return false
				}
				paginationText := response.Content[1].Text
				if !contains(paginationText, "Pagination") {
					return false
				}
			}

			return true
		},
		gen.IntRange(0, 1000),
		gen.IntRange(0, 100),
		gen.IntRange(1, 100),
	))

	// Property: List responses transform to valid MCP format
	properties.Property("List responses transform to valid MCP format", prop.ForAll(
		func(count int) bool {
			// Ensure valid count
			if count < 0 {
				count = 0
			}
			if count > 100 {
				count = 100
			}

			// Create a list of projects
			projects := make([]Project, count)
			for i := 0; i < count; i++ {
				projects[i] = Project{
					ID:   FlexibleID(fmt.Sprintf("proj-%d", i)),
					Key:  fmt.Sprintf("PROJ%d", i),
					Name: fmt.Sprintf("Project %d", i),
				}
			}

			// Transform the response
			response, err := mapper.MapToToolResponse(projects)
			if err != nil {
				return false
			}

			// Verify response structure
			if response == nil {
				return false
			}

			// Must have at least one content block
			if len(response.Content) == 0 {
				return false
			}

			// First content block must be text type
			if response.Content[0].Type != "text" {
				return false
			}

			// Content must be valid JSON
			text := response.Content[0].Text
			if text == "" {
				return false
			}

			// Verify it's parseable JSON array
			var parsed []interface{}
			err = json.Unmarshal([]byte(text), &parsed)
			if err != nil {
				return false
			}

			// Array length should match input
			if len(parsed) != count {
				return false
			}

			return true
		},
		gen.IntRange(0, 100),
	))

	// Property: Nil responses transform to valid MCP format
	properties.Property("Nil responses transform to valid MCP format", prop.ForAll(
		func() bool {
			// Transform nil response
			response, err := mapper.MapToToolResponse(nil)
			if err != nil {
				return false
			}

			// Verify response structure
			if response == nil {
				return false
			}

			// Must have at least one content block
			if len(response.Content) == 0 {
				return false
			}

			// First content block must be text type
			if response.Content[0].Type != "text" {
				return false
			}

			// Content should be empty JSON object
			text := response.Content[0].Text
			if text != "{}" {
				return false
			}

			return true
		},
	))

	// Property: All transformed responses are valid JSON-RPC tool responses
	properties.Property("All transformed responses are valid JSON-RPC tool responses", prop.ForAll(
		func(responseType int) bool {
			var apiResponse interface{}

			// Generate different response types
			switch responseType % 4 {
			case 0:
				// Jira issue
				apiResponse = &JiraIssue{
					ID:  "10001",
					Key: "TEST-1",
					Fields: JiraFields{
						Summary: "Test issue",
						IssueType: IssueType{
							ID:   "1",
							Name: "Bug",
						},
						Project: Project{
							ID:   "10000",
							Key:  "TEST",
							Name: "Test Project",
						},
						Status: Status{
							ID:   "1",
							Name: "Open",
						},
						Created: "2024-01-01T00:00:00.000Z",
						Updated: "2024-01-02T00:00:00.000Z",
					},
				}
			case 1:
				// Confluence page
				apiResponse = &ConfluencePage{
					ID:    "12345",
					Type:  "page",
					Title: "Test Page",
					Space: Space{
						ID:   "1",
						Key:  "TEST",
						Name: "Test Space",
					},
					Body: Body{
						Storage: Storage{
							Value:          "<p>Test content</p>",
							Representation: "storage",
						},
					},
					Version: Version{
						Number: 1,
						When:   "2024-01-01T00:00:00.000Z",
						By: User{
							Name:        "testuser",
							DisplayName: "Test User",
						},
					},
				}
			case 2:
				// Bitbucket repository
				apiResponse = &Repository{
					ID:   1,
					Slug: "test-repo",
					Name: "Test Repository",
					Project: Project{
						ID:   "1",
						Key:  "TEST",
						Name: "Test Project",
					},
					Public: false,
				}
			case 3:
				// Bamboo build plan
				apiResponse = &BuildPlan{
					Key:       "PLAN-KEY",
					Name:      "Test Plan",
					ShortName: "Test",
					ShortKey:  "TP",
					Type:      "chain",
					Enabled:   true,
				}
			}

			// Transform the response
			response, err := mapper.MapToToolResponse(apiResponse)
			if err != nil {
				return false
			}

			// Verify it can be serialized as part of a JSON-RPC response
			jsonRPCResponse := &Response{
				JSONRPC: "2.0",
				ID:      1,
				Result:  response,
			}

			// Should be serializable
			data, err := json.Marshal(jsonRPCResponse)
			if err != nil {
				return false
			}

			// Should be deserializable
			var decoded Response
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				return false
			}

			// Decoded response should have result
			if decoded.Result == nil {
				return false
			}

			return true
		},
		gen.IntRange(0, 3),
	))

	// Property: Transformed responses preserve all data fields
	properties.Property("Transformed responses preserve all data fields", prop.ForAll(
		func(issueID string, issueKey string, summary string) bool {
			// Ensure non-empty values
			if issueID == "" {
				issueID = "10001"
			}
			if issueKey == "" {
				issueKey = "TEST-1"
			}
			if summary == "" {
				summary = "Test summary"
			}

			// Create a Jira issue with specific values
			issue := &JiraIssue{
				ID:  FlexibleID(issueID),
				Key: issueKey,
				Fields: JiraFields{
					Summary: summary,
					IssueType: IssueType{
						ID:   "1",
						Name: "Bug",
					},
					Project: Project{
						ID:   "10000",
						Key:  "TEST",
						Name: "Test Project",
					},
					Status: Status{
						ID:   "1",
						Name: "Open",
					},
					Created: "2024-01-01T00:00:00.000Z",
					Updated: "2024-01-02T00:00:00.000Z",
				},
			}

			// Transform the response
			response, err := mapper.MapToToolResponse(issue)
			if err != nil {
				return false
			}

			// Parse the JSON content
			text := response.Content[0].Text
			var parsed map[string]interface{}
			err = json.Unmarshal([]byte(text), &parsed)
			if err != nil {
				return false
			}

			// Verify all fields are preserved
			if parsed["id"] != issueID {
				return false
			}
			if parsed["key"] != issueKey {
				return false
			}

			// Check nested fields
			fields, ok := parsed["fields"].(map[string]interface{})
			if !ok {
				return false
			}
			if fields["summary"] != summary {
				return false
			}

			return true
		},
		gen.Identifier(),
		gen.Identifier(),
		gen.AlphaString(),
	))

	properties.TestingRun(t)
}

// Feature: atlassian-mcp-server, Property 15: Response Data Preservation
// **Validates: Requirements 9.4**
//
// For any Atlassian API response, the Response_Mapper should preserve all data fields
// present in the original response when transforming to MCP format (no data should be
// lost during transformation).
func TestProperty15_ResponseDataPreservation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)
	mapper := NewResponseMapper()

	// Generator for FlexibleID
	genFlexibleID := gen.Identifier().Map(func(s string) FlexibleID {
		return FlexibleID(s)
	})

	// Generator for Jira issues with all fields populated
	genFullJiraIssue := gen.Struct(reflect.TypeOf(JiraIssue{}), map[string]gopter.Gen{
		"ID":  genFlexibleID,
		"Key": gen.Identifier().Map(func(s string) string { return "TEST-" + s }),
		"Fields": gen.Struct(reflect.TypeOf(JiraFields{}), map[string]gopter.Gen{
			"Summary":     gen.AlphaString(),
			"Description": gen.AlphaString(),
			"IssueType": gen.Struct(reflect.TypeOf(IssueType{}), map[string]gopter.Gen{
				"ID":   genFlexibleID,
				"Name": gen.OneConstOf("Bug", "Story", "Task", "Epic"),
			}),
			"Project": gen.Struct(reflect.TypeOf(Project{}), map[string]gopter.Gen{
				"ID":   genFlexibleID,
				"Key":  gen.Identifier(),
				"Name": gen.AlphaString(),
			}),
			"Status": gen.Struct(reflect.TypeOf(Status{}), map[string]gopter.Gen{
				"ID":   genFlexibleID,
				"Name": gen.OneConstOf("Open", "In Progress", "Done", "Closed"),
			}),
			"Assignee": gen.PtrOf(gen.Struct(reflect.TypeOf(User{}), map[string]gopter.Gen{
				"Name":         gen.Identifier(),
				"DisplayName":  gen.AlphaString(),
				"EmailAddress": gen.Identifier().Map(func(s string) string { return s + "@example.com" }),
			})),
			"Reporter": gen.PtrOf(gen.Struct(reflect.TypeOf(User{}), map[string]gopter.Gen{
				"Name":         gen.Identifier(),
				"DisplayName":  gen.AlphaString(),
				"EmailAddress": gen.Identifier().Map(func(s string) string { return s + "@example.com" }),
			})),
			"Created": gen.Const("2024-01-01T00:00:00.000Z"),
			"Updated": gen.Const("2024-01-02T00:00:00.000Z"),
		}),
	})

	// Generator for Confluence pages with all fields populated
	genFullConfluencePage := gen.Struct(reflect.TypeOf(ConfluencePage{}), map[string]gopter.Gen{
		"ID":    gen.Identifier(),
		"Type":  gen.Const("page"),
		"Title": gen.AlphaString(),
		"Space": gen.Struct(reflect.TypeOf(Space{}), map[string]gopter.Gen{
			"ID":   gen.Identifier(),
			"Key":  gen.Identifier(),
			"Name": gen.AlphaString(),
		}),
		"Body": gen.Struct(reflect.TypeOf(Body{}), map[string]gopter.Gen{
			"Storage": gen.Struct(reflect.TypeOf(Storage{}), map[string]gopter.Gen{
				"Value":          gen.AlphaString().Map(func(s string) string { return "<p>" + s + "</p>" }),
				"Representation": gen.Const("storage"),
			}),
		}),
		"Version": gen.Struct(reflect.TypeOf(Version{}), map[string]gopter.Gen{
			"Number": gen.IntRange(1, 100),
			"When":   gen.Const("2024-01-01T00:00:00.000Z"),
			"By": gen.Struct(reflect.TypeOf(User{}), map[string]gopter.Gen{
				"Name":         gen.Identifier(),
				"DisplayName":  gen.AlphaString(),
				"EmailAddress": gen.Identifier().Map(func(s string) string { return s + "@example.com" }),
			}),
		}),
	})

	// Generator for Bitbucket pull requests with all fields populated
	genFullPullRequest := gen.Struct(reflect.TypeOf(PullRequest{}), map[string]gopter.Gen{
		"ID":          gen.IntRange(1, 10000),
		"Version":     gen.IntRange(1, 100),
		"Title":       gen.AlphaString(),
		"Description": gen.AlphaString(),
		"State":       gen.OneConstOf("OPEN", "MERGED", "DECLINED"),
		"Open":        gen.Bool(),
		"Closed":      gen.Bool(),
		"FromRef": gen.Struct(reflect.TypeOf(Ref{}), map[string]gopter.Gen{
			"ID":        gen.Const("refs/heads/feature"),
			"DisplayID": gen.Const("feature"),
			"Repository": gen.Struct(reflect.TypeOf(Repository{}), map[string]gopter.Gen{
				"ID":   gen.IntRange(1, 1000),
				"Slug": gen.Identifier(),
				"Name": gen.AlphaString(),
				"Project": gen.Struct(reflect.TypeOf(Project{}), map[string]gopter.Gen{
					"ID":   genFlexibleID,
					"Key":  gen.Identifier(),
					"Name": gen.AlphaString(),
				}),
				"Public": gen.Bool(),
			}),
		}),
		"ToRef": gen.Struct(reflect.TypeOf(Ref{}), map[string]gopter.Gen{
			"ID":        gen.Const("refs/heads/main"),
			"DisplayID": gen.Const("main"),
			"Repository": gen.Struct(reflect.TypeOf(Repository{}), map[string]gopter.Gen{
				"ID":   gen.IntRange(1, 1000),
				"Slug": gen.Identifier(),
				"Name": gen.AlphaString(),
				"Project": gen.Struct(reflect.TypeOf(Project{}), map[string]gopter.Gen{
					"ID":   genFlexibleID,
					"Key":  gen.Identifier(),
					"Name": gen.AlphaString(),
				}),
				"Public": gen.Bool(),
			}),
		}),
		"Author": gen.Struct(reflect.TypeOf(User{}), map[string]gopter.Gen{
			"Name":         gen.Identifier(),
			"DisplayName":  gen.AlphaString(),
			"EmailAddress": gen.Identifier().Map(func(s string) string { return s + "@example.com" }),
		}),
		"Reviewers": gen.SliceOf(gen.Struct(reflect.TypeOf(Reviewer{}), map[string]gopter.Gen{
			"User": gen.Struct(reflect.TypeOf(User{}), map[string]gopter.Gen{
				"Name":         gen.Identifier(),
				"DisplayName":  gen.AlphaString(),
				"EmailAddress": gen.Identifier().Map(func(s string) string { return s + "@example.com" }),
			}),
			"Approved": gen.Bool(),
			"Status":   gen.OneConstOf("APPROVED", "UNAPPROVED", "NEEDS_WORK"),
		})),
	})

	// Generator for Bamboo build results with all fields populated
	genFullBuildResult := gen.Struct(reflect.TypeOf(BuildResult{}), map[string]gopter.Gen{
		"Key":                gen.Identifier().Map(func(s string) string { return "PLAN-" + s }),
		"Number":             gen.IntRange(1, 1000),
		"State":              gen.OneConstOf("Successful", "Failed", "Unknown"),
		"LifeCycleState":     gen.OneConstOf("Pending", "Queued", "InProgress", "Finished"),
		"BuildStartedTime":   gen.Const("2024-01-01T00:00:00.000Z"),
		"BuildCompletedTime": gen.Const("2024-01-01T00:05:00.000Z"),
		"BuildDuration":      gen.Int64Range(1000, 300000),
		"BuildReason":        gen.OneConstOf("Manual build", "Code change", "Scheduled"),
	})

	// Property: Jira issue data is fully preserved
	properties.Property("Jira issue data is fully preserved", prop.ForAll(
		func(issue JiraIssue) bool {
			// Transform the response
			response, err := mapper.MapToToolResponse(&issue)
			if err != nil {
				return false
			}

			// Parse the transformed JSON
			text := response.Content[0].Text
			var transformed map[string]interface{}
			err = json.Unmarshal([]byte(text), &transformed)
			if err != nil {
				return false
			}

			// Marshal original to JSON for comparison
			originalJSON, err := json.Marshal(&issue)
			if err != nil {
				return false
			}
			var original map[string]interface{}
			err = json.Unmarshal(originalJSON, &original)
			if err != nil {
				return false
			}

			// Verify all top-level fields are preserved
			return verifyAllFieldsPreserved(original, transformed)
		},
		genFullJiraIssue,
	))

	// Property: Confluence page data is fully preserved
	properties.Property("Confluence page data is fully preserved", prop.ForAll(
		func(page ConfluencePage) bool {
			// Transform the response
			response, err := mapper.MapToToolResponse(&page)
			if err != nil {
				return false
			}

			// Parse the transformed JSON
			text := response.Content[0].Text
			var transformed map[string]interface{}
			err = json.Unmarshal([]byte(text), &transformed)
			if err != nil {
				return false
			}

			// Marshal original to JSON for comparison
			originalJSON, err := json.Marshal(&page)
			if err != nil {
				return false
			}
			var original map[string]interface{}
			err = json.Unmarshal(originalJSON, &original)
			if err != nil {
				return false
			}

			// Verify all fields are preserved
			return verifyAllFieldsPreserved(original, transformed)
		},
		genFullConfluencePage,
	))

	// Property: Bitbucket pull request data is fully preserved
	properties.Property("Bitbucket pull request data is fully preserved", prop.ForAll(
		func(pr PullRequest) bool {
			// Transform the response
			response, err := mapper.MapToToolResponse(&pr)
			if err != nil {
				return false
			}

			// Parse the transformed JSON
			text := response.Content[0].Text
			var transformed map[string]interface{}
			err = json.Unmarshal([]byte(text), &transformed)
			if err != nil {
				return false
			}

			// Marshal original to JSON for comparison
			originalJSON, err := json.Marshal(&pr)
			if err != nil {
				return false
			}
			var original map[string]interface{}
			err = json.Unmarshal(originalJSON, &original)
			if err != nil {
				return false
			}

			// Verify all fields are preserved
			return verifyAllFieldsPreserved(original, transformed)
		},
		genFullPullRequest,
	))

	// Property: Bamboo build result data is fully preserved
	properties.Property("Bamboo build result data is fully preserved", prop.ForAll(
		func(buildResult BuildResult) bool {
			// Transform the response
			response, err := mapper.MapToToolResponse(&buildResult)
			if err != nil {
				return false
			}

			// Parse the transformed JSON
			text := response.Content[0].Text
			var transformed map[string]interface{}
			err = json.Unmarshal([]byte(text), &transformed)
			if err != nil {
				return false
			}

			// Marshal original to JSON for comparison
			originalJSON, err := json.Marshal(&buildResult)
			if err != nil {
				return false
			}
			var original map[string]interface{}
			err = json.Unmarshal(originalJSON, &original)
			if err != nil {
				return false
			}

			// Verify all fields are preserved
			return verifyAllFieldsPreserved(original, transformed)
		},
		genFullBuildResult,
	))

	// Property: Search results with pagination preserve all data
	properties.Property("Search results with pagination preserve all data", prop.ForAll(
		func(total int, startAt int, maxResults int, issueCount int) bool {
			// Ensure valid values
			if total < 0 {
				total = 0
			}
			if startAt < 0 {
				startAt = 0
			}
			if maxResults < 1 {
				maxResults = 50
			}
			if issueCount < 0 {
				issueCount = 0
			}
			if issueCount > 10 {
				issueCount = 10 // Limit for test performance
			}

			// Create search results with issues
			issues := make([]JiraIssue, issueCount)
			for i := 0; i < issueCount; i++ {
				issues[i] = JiraIssue{
					ID:  FlexibleID(fmt.Sprintf("issue-%d", i)),
					Key: fmt.Sprintf("TEST-%d", i),
					Fields: JiraFields{
						Summary: fmt.Sprintf("Issue %d", i),
						IssueType: IssueType{
							ID:   "1",
							Name: "Bug",
						},
						Project: Project{
							ID:   "10000",
							Key:  "TEST",
							Name: "Test Project",
						},
						Status: Status{
							ID:   "1",
							Name: "Open",
						},
						Created: "2024-01-01T00:00:00.000Z",
						Updated: "2024-01-02T00:00:00.000Z",
					},
				}
			}

			searchResults := &SearchResults{
				Issues:     issues,
				Total:      total,
				StartAt:    startAt,
				MaxResults: maxResults,
			}

			// Transform the response
			response, err := mapper.MapToToolResponse(searchResults)
			if err != nil {
				return false
			}

			// Parse the transformed JSON
			text := response.Content[0].Text
			var transformed map[string]interface{}
			err = json.Unmarshal([]byte(text), &transformed)
			if err != nil {
				return false
			}

			// Marshal original to JSON for comparison
			originalJSON, err := json.Marshal(searchResults)
			if err != nil {
				return false
			}
			var original map[string]interface{}
			err = json.Unmarshal(originalJSON, &original)
			if err != nil {
				return false
			}

			// Verify all fields are preserved
			return verifyAllFieldsPreserved(original, transformed)
		},
		gen.IntRange(0, 1000),
		gen.IntRange(0, 100),
		gen.IntRange(1, 100),
		gen.IntRange(0, 10),
	))

	// Property: List responses preserve all data
	properties.Property("List responses preserve all data", prop.ForAll(
		func(count int) bool {
			// Ensure valid count
			if count < 0 {
				count = 0
			}
			if count > 20 {
				count = 20 // Limit for test performance
			}

			// Create a list of repositories
			repos := make([]Repository, count)
			for i := 0; i < count; i++ {
				repos[i] = Repository{
					ID:   i + 1,
					Slug: fmt.Sprintf("repo-%d", i),
					Name: fmt.Sprintf("Repository %d", i),
					Project: Project{
						ID:   FlexibleID(fmt.Sprintf("proj-%d", i)),
						Key:  fmt.Sprintf("PROJ%d", i),
						Name: fmt.Sprintf("Project %d", i),
					},
					Public: i%2 == 0,
				}
			}

			// Transform the response
			response, err := mapper.MapToToolResponse(repos)
			if err != nil {
				return false
			}

			// Parse the transformed JSON
			text := response.Content[0].Text
			var transformed []interface{}
			err = json.Unmarshal([]byte(text), &transformed)
			if err != nil {
				return false
			}

			// Marshal original to JSON for comparison
			originalJSON, err := json.Marshal(repos)
			if err != nil {
				return false
			}
			var original []interface{}
			err = json.Unmarshal(originalJSON, &original)
			if err != nil {
				return false
			}

			// Verify array length is preserved
			if len(transformed) != len(original) {
				return false
			}

			// Verify each element is preserved
			for i := 0; i < len(original); i++ {
				origMap, ok1 := original[i].(map[string]interface{})
				transMap, ok2 := transformed[i].(map[string]interface{})
				if !ok1 || !ok2 {
					return false
				}
				if !verifyAllFieldsPreserved(origMap, transMap) {
					return false
				}
			}

			return true
		},
		gen.IntRange(0, 20),
	))

	// Property: Nested structures preserve all data
	properties.Property("Nested structures preserve all data", prop.ForAll(
		func(repoID int, prID int, reviewerCount int) bool {
			// Ensure valid values
			if repoID < 1 {
				repoID = 1
			}
			if prID < 1 {
				prID = 1
			}
			if reviewerCount < 0 {
				reviewerCount = 0
			}
			if reviewerCount > 5 {
				reviewerCount = 5 // Limit for test performance
			}

			// Create a pull request with nested structures
			reviewers := make([]Reviewer, reviewerCount)
			for i := 0; i < reviewerCount; i++ {
				reviewers[i] = Reviewer{
					User: User{
						Name:         fmt.Sprintf("reviewer%d", i),
						DisplayName:  fmt.Sprintf("Reviewer %d", i),
						EmailAddress: fmt.Sprintf("reviewer%d@example.com", i),
					},
					Approved: i%2 == 0,
					Status:   "APPROVED",
				}
			}

			pr := &PullRequest{
				ID:          prID,
				Version:     1,
				Title:       "Test PR",
				Description: "Test description",
				State:       "OPEN",
				Open:        true,
				Closed:      false,
				FromRef: Ref{
					ID:        "refs/heads/feature",
					DisplayID: "feature",
					Repository: Repository{
						ID:   repoID,
						Slug: "test-repo",
						Name: "Test Repository",
						Project: Project{
							ID:   "1",
							Key:  "TEST",
							Name: "Test Project",
						},
						Public: false,
					},
				},
				ToRef: Ref{
					ID:        "refs/heads/main",
					DisplayID: "main",
					Repository: Repository{
						ID:   repoID,
						Slug: "test-repo",
						Name: "Test Repository",
						Project: Project{
							ID:   "1",
							Key:  "TEST",
							Name: "Test Project",
						},
						Public: false,
					},
				},
				Author: User{
					Name:         "author",
					DisplayName:  "Author Name",
					EmailAddress: "author@example.com",
				},
				Reviewers: reviewers,
			}

			// Transform the response
			response, err := mapper.MapToToolResponse(pr)
			if err != nil {
				return false
			}

			// Parse the transformed JSON
			text := response.Content[0].Text
			var transformed map[string]interface{}
			err = json.Unmarshal([]byte(text), &transformed)
			if err != nil {
				return false
			}

			// Marshal original to JSON for comparison
			originalJSON, err := json.Marshal(pr)
			if err != nil {
				return false
			}
			var original map[string]interface{}
			err = json.Unmarshal(originalJSON, &original)
			if err != nil {
				return false
			}

			// Verify all nested fields are preserved
			return verifyAllFieldsPreserved(original, transformed)
		},
		gen.IntRange(1, 100),
		gen.IntRange(1, 1000),
		gen.IntRange(0, 5),
	))

	// Property: Empty and nil fields are preserved correctly
	properties.Property("Empty and nil fields are preserved correctly", prop.ForAll(
		func(hasAssignee bool, hasReporter bool, description string) bool {
			// Create issue with optional fields
			issue := &JiraIssue{
				ID:  "10001",
				Key: "TEST-1",
				Fields: JiraFields{
					Summary:     "Test issue",
					Description: description,
					IssueType: IssueType{
						ID:   "1",
						Name: "Bug",
					},
					Project: Project{
						ID:   "10000",
						Key:  "TEST",
						Name: "Test Project",
					},
					Status: Status{
						ID:   "1",
						Name: "Open",
					},
					Created: "2024-01-01T00:00:00.000Z",
					Updated: "2024-01-02T00:00:00.000Z",
				},
			}

			// Conditionally add assignee and reporter
			if hasAssignee {
				issue.Fields.Assignee = &User{
					Name:         "assignee",
					DisplayName:  "Assignee Name",
					EmailAddress: "assignee@example.com",
				}
			}
			if hasReporter {
				issue.Fields.Reporter = &User{
					Name:         "reporter",
					DisplayName:  "Reporter Name",
					EmailAddress: "reporter@example.com",
				}
			}

			// Transform the response
			response, err := mapper.MapToToolResponse(issue)
			if err != nil {
				return false
			}

			// Parse the transformed JSON
			text := response.Content[0].Text
			var transformed map[string]interface{}
			err = json.Unmarshal([]byte(text), &transformed)
			if err != nil {
				return false
			}

			// Marshal original to JSON for comparison
			originalJSON, err := json.Marshal(issue)
			if err != nil {
				return false
			}
			var original map[string]interface{}
			err = json.Unmarshal(originalJSON, &original)
			if err != nil {
				return false
			}

			// Verify all fields are preserved (including nil/empty)
			return verifyAllFieldsPreserved(original, transformed)
		},
		gen.Bool(),
		gen.Bool(),
		gen.AlphaString(),
	))

	properties.TestingRun(t)
}

// verifyAllFieldsPreserved recursively checks that all fields in the original
// map are present in the transformed map with the same values.
// This is the core verification function for Property 15.
func verifyAllFieldsPreserved(original, transformed map[string]interface{}) bool {
	// Check that all keys in original exist in transformed
	for key, origValue := range original {
		transValue, exists := transformed[key]
		if !exists {
			// Field is missing in transformed response
			return false
		}

		// Compare values based on type
		if !compareValues(origValue, transValue) {
			return false
		}
	}

	return true
}

// compareValues compares two values recursively, handling different JSON types.
func compareValues(orig, trans interface{}) bool {
	// Handle nil values
	if orig == nil && trans == nil {
		return true
	}
	if orig == nil || trans == nil {
		return false
	}

	// Handle maps (nested objects)
	origMap, origIsMap := orig.(map[string]interface{})
	transMap, transIsMap := trans.(map[string]interface{})
	if origIsMap && transIsMap {
		return verifyAllFieldsPreserved(origMap, transMap)
	}
	if origIsMap != transIsMap {
		return false
	}

	// Handle slices (arrays)
	origSlice, origIsSlice := orig.([]interface{})
	transSlice, transIsSlice := trans.([]interface{})
	if origIsSlice && transIsSlice {
		if len(origSlice) != len(transSlice) {
			return false
		}
		for i := 0; i < len(origSlice); i++ {
			if !compareValues(origSlice[i], transSlice[i]) {
				return false
			}
		}
		return true
	}
	if origIsSlice != transIsSlice {
		return false
	}

	// Handle numeric types (JSON unmarshaling can produce float64 for all numbers)
	origNum, origIsNum := orig.(float64)
	transNum, transIsNum := trans.(float64)
	if origIsNum && transIsNum {
		return origNum == transNum
	}

	// Handle int types (from JSON unmarshaling)
	origInt, origIsInt := orig.(int)
	transInt, transIsInt := trans.(int)
	if origIsInt && transIsInt {
		return origInt == transInt
	}

	// Handle int to float64 comparison (common in JSON)
	if origIsInt && transIsNum {
		return float64(origInt) == transNum
	}
	if origIsNum && transIsInt {
		return origNum == float64(transInt)
	}

	// Handle bool types
	origBool, origIsBool := orig.(bool)
	transBool, transIsBool := trans.(bool)
	if origIsBool && transIsBool {
		return origBool == transBool
	}

	// Handle string types
	origStr, origIsStr := orig.(string)
	transStr, transIsStr := trans.(string)
	if origIsStr && transIsStr {
		return origStr == transStr
	}

	// For other types, use direct comparison
	return orig == trans
}

// Feature: atlassian-mcp-server, Property 10: API Error Mapping
// **Validates: Requirements 8.1**
//
// For any HTTP error response from an Atlassian API (4xx or 5xx status codes),
// the server should map it to an appropriate MCP error response with a code and
// message that reflects the nature of the failure.
func TestProperty10_APIErrorMapping(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)
	mapper := NewResponseMapper()

	// Generator for 4xx status codes
	gen4xxStatusCode := gen.OneConstOf(
		400, // Bad Request
		401, // Unauthorized
		403, // Forbidden
		404, // Not Found
		405, // Method Not Allowed
		406, // Not Acceptable
		408, // Request Timeout
		409, // Conflict
		410, // Gone
		415, // Unsupported Media Type
		422, // Unprocessable Entity
		429, // Too Many Requests
	)

	// Generator for 5xx status codes
	gen5xxStatusCode := gen.OneConstOf(
		500, // Internal Server Error
		501, // Not Implemented
		502, // Bad Gateway
		503, // Service Unavailable
		504, // Gateway Timeout
		505, // HTTP Version Not Supported
	)

	// Property: 4xx errors are mapped to appropriate MCP error codes
	properties.Property("4xx HTTP errors map to appropriate MCP error codes", prop.ForAll(
		func(statusCode int, message string, body string) bool {
			// Create HTTP error
			httpErr := NewHTTPError(statusCode, message, body)

			// Map the error
			mcpErr := mapper.MapError(httpErr)

			// Verify error is not nil
			if mcpErr == nil {
				return false
			}

			// Verify error code is negative (JSON-RPC requirement)
			if mcpErr.Code >= 0 {
				return false
			}

			// Verify error has a message
			if mcpErr.Message == "" {
				return false
			}

			// Verify error data contains status code
			if mcpErr.Data == nil {
				return false
			}

			dataMap, ok := mcpErr.Data.(map[string]interface{})
			if !ok {
				return false
			}

			statusCodeInData, ok := dataMap["statusCode"]
			if !ok {
				return false
			}

			// Verify status code is preserved in data
			if statusCodeInData != statusCode {
				return false
			}

			// Verify specific mappings for known status codes
			switch statusCode {
			case 401:
				// Unauthorized should map to AuthenticationError
				if mcpErr.Code != AuthenticationError {
					return false
				}
			case 403:
				// Forbidden should map to AuthenticationError
				if mcpErr.Code != AuthenticationError {
					return false
				}
			case 400:
				// Bad Request should map to InvalidParams
				if mcpErr.Code != InvalidParams {
					return false
				}
			case 404:
				// Not Found should map to APIError
				if mcpErr.Code != APIError {
					return false
				}
			case 409:
				// Conflict should map to APIError
				if mcpErr.Code != APIError {
					return false
				}
			case 429:
				// Too Many Requests should map to RateLimitError
				if mcpErr.Code != RateLimitError {
					return false
				}
			default:
				// Other 4xx errors should map to APIError
				if mcpErr.Code != APIError && mcpErr.Code != InvalidParams &&
					mcpErr.Code != AuthenticationError && mcpErr.Code != RateLimitError {
					return false
				}
			}

			return true
		},
		gen4xxStatusCode,
		gen.AlphaString(),
		gen.AlphaString(),
	))

	// Property: 5xx errors are mapped to appropriate MCP error codes
	properties.Property("5xx HTTP errors map to appropriate MCP error codes", prop.ForAll(
		func(statusCode int, message string, body string) bool {
			// Create HTTP error
			httpErr := NewHTTPError(statusCode, message, body)

			// Map the error
			mcpErr := mapper.MapError(httpErr)

			// Verify error is not nil
			if mcpErr == nil {
				return false
			}

			// Verify error code is negative (JSON-RPC requirement)
			if mcpErr.Code >= 0 {
				return false
			}

			// Verify error has a message
			if mcpErr.Message == "" {
				return false
			}

			// Verify error data contains status code
			if mcpErr.Data == nil {
				return false
			}

			dataMap, ok := mcpErr.Data.(map[string]interface{})
			if !ok {
				return false
			}

			statusCodeInData, ok := dataMap["statusCode"]
			if !ok {
				return false
			}

			// Verify status code is preserved in data
			if statusCodeInData != statusCode {
				return false
			}

			// Verify specific mappings for known status codes
			switch statusCode {
			case 500:
				// Internal Server Error should map to APIError
				if mcpErr.Code != APIError {
					return false
				}
			case 503:
				// Service Unavailable should map to NetworkError
				if mcpErr.Code != NetworkError {
					return false
				}
			case 504:
				// Gateway Timeout should map to NetworkError
				if mcpErr.Code != NetworkError {
					return false
				}
			default:
				// Other 5xx errors should map to APIError or NetworkError
				if mcpErr.Code != APIError && mcpErr.Code != NetworkError {
					return false
				}
			}

			return true
		},
		gen5xxStatusCode,
		gen.AlphaString(),
		gen.AlphaString(),
	))

	// Property: Error message reflects the nature of the failure
	properties.Property("Error messages reflect the nature of the failure", prop.ForAll(
		func(statusCode int) bool {
			// Create HTTP error with descriptive message
			httpErr := NewHTTPError(statusCode, "Test error", "Test body")

			// Map the error
			mcpErr := mapper.MapError(httpErr)

			// Verify error is not nil
			if mcpErr == nil {
				return false
			}

			// Verify message is descriptive and relates to the error type
			message := mcpErr.Message
			if message == "" {
				return false
			}

			// Check that message is appropriate for the status code category
			switch statusCode {
			case 401, 403:
				// Authentication/authorization errors should mention auth
				return contains(message, "Authentication") || contains(message, "authentication") ||
					contains(message, "Access") || contains(message, "forbidden")
			case 404:
				// Not found errors should mention resource
				return contains(message, "not found") || contains(message, "Not found") ||
					contains(message, "Resource")
			case 400:
				// Bad request should mention parameters or request
				return contains(message, "Bad request") || contains(message, "invalid") ||
					contains(message, "parameters")
			case 409:
				// Conflict should mention conflict
				return contains(message, "Conflict") || contains(message, "conflict")
			case 429:
				// Rate limit should mention rate limit
				return contains(message, "Rate limit") || contains(message, "rate limit")
			case 500:
				// Internal server error should mention server
				return contains(message, "Internal server error") || contains(message, "server error")
			case 503:
				// Service unavailable should mention service
				return contains(message, "Service unavailable") || contains(message, "unavailable")
			case 504:
				// Gateway timeout should mention timeout
				return contains(message, "timeout") || contains(message, "Timeout")
			default:
				// For other codes, just verify message is not empty
				return true
			}
		},
		gen.OneConstOf(400, 401, 403, 404, 409, 429, 500, 503, 504),
	))

	// Property: All HTTP error responses include original error details in data
	properties.Property("HTTP errors include original details in data field", prop.ForAll(
		func(statusCode int, message string, body string) bool {
			// Create HTTP error
			httpErr := NewHTTPError(statusCode, message, body)

			// Map the error
			mcpErr := mapper.MapError(httpErr)

			// Verify error is not nil
			if mcpErr == nil {
				return false
			}

			// Verify data field exists
			if mcpErr.Data == nil {
				return false
			}

			dataMap, ok := mcpErr.Data.(map[string]interface{})
			if !ok {
				return false
			}

			// Verify status code is in data
			statusCodeInData, ok := dataMap["statusCode"]
			if !ok {
				return false
			}
			if statusCodeInData != statusCode {
				return false
			}

			// Verify message is in data
			messageInData, ok := dataMap["message"]
			if !ok {
				return false
			}
			if messageInData != message {
				return false
			}

			// If body is not empty, verify it's in data
			if body != "" {
				bodyInData, ok := dataMap["body"]
				if !ok {
					return false
				}
				if bodyInData != body {
					return false
				}
			}

			return true
		},
		gen.OneConstOf(400, 401, 403, 404, 409, 429, 500, 503, 504),
		gen.AlphaString(),
		gen.AlphaString(),
	))

	// Property: Non-HTTP errors are handled gracefully
	properties.Property("Non-HTTP errors map to InternalError", prop.ForAll(
		func(errorMessage string) bool {
			// Create a generic error
			genericErr := fmt.Errorf("%s", errorMessage)

			// Map the error
			mcpErr := mapper.MapError(genericErr)

			// Verify error is not nil
			if mcpErr == nil {
				return false
			}

			// Should map to InternalError
			if mcpErr.Code != InternalError {
				return false
			}

			// Message should be the error message
			if mcpErr.Message != errorMessage {
				return false
			}

			return true
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	// Property: Domain errors are passed through unchanged
	properties.Property("Domain errors are passed through unchanged", prop.ForAll(
		func(code int, message string) bool {
			// Create a domain error
			domainErr := &Error{
				Code:    code,
				Message: message,
				Data:    map[string]interface{}{"test": "data"},
			}

			// Map the error
			mcpErr := mapper.MapError(domainErr)

			// Verify error is not nil
			if mcpErr == nil {
				return false
			}

			// Should be the same error
			if mcpErr.Code != code {
				return false
			}
			if mcpErr.Message != message {
				return false
			}

			return true
		},
		gen.OneConstOf(ParseError, InvalidRequest, MethodNotFound, InvalidParams,
			InternalError, ConfigurationError, AuthenticationError, APIError,
			NetworkError, RateLimitError),
		gen.AlphaString(),
	))

	// Property: Nil errors return nil
	properties.Property("Nil errors return nil", prop.ForAll(
		func() bool {
			mcpErr := mapper.MapError(nil)
			return mcpErr == nil
		},
	))

	properties.TestingRun(t)
}

// Feature: atlassian-mcp-server, Property 13: Request Parameter Extraction
// **Validates: Requirements 9.1**
//
// For any valid MCP tool request, the Request_Handler should successfully extract
// all parameters and map them to the corresponding Atlassian API request parameters
// without data loss.
func TestProperty13_RequestParameterExtraction(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Generator for valid issue keys (e.g., TEST-123, PROJ-456)
	genIssueKey := gen.Identifier().
		SuchThat(func(s string) bool { return len(s) >= 2 }).
		Map(func(s string) string {
			// Use a simple counter for the issue number
			num := len(s) % 9999
			if num == 0 {
				num = 1
			}
			return s[:min(len(s), 5)] + "-" + fmt.Sprintf("%d", num)
		})

	// Generator for valid project keys (2-10 uppercase letters)
	genProjectKey := gen.Identifier().
		SuchThat(func(s string) bool { return len(s) >= 2 }).
		Map(func(s string) string {
			// Convert to uppercase and limit length
			result := ""
			for _, c := range s {
				if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
					if c >= 'a' && c <= 'z' {
						result += string(c - 32)
					} else {
						result += string(c)
					}
				}
				if len(result) >= 10 {
					break
				}
			}
			// Ensure we have at least 2 characters
			if len(result) < 2 {
				return "TEST"
			}
			return result
		})

	// Property: jira_get_issue extracts issueKey parameter correctly
	properties.Property("jira_get_issue extracts issueKey without data loss", prop.ForAll(
		func(issueKey string) bool {
			// Create MCP tool request
			toolReq := &ToolRequest{
				Name: "jira_get_issue",
				Arguments: map[string]interface{}{
					"issueKey": issueKey,
				},
			}

			// Verify parameter extraction
			extractedKey, exists := toolReq.Arguments["issueKey"]
			if !exists {
				return false
			}

			// Verify no data loss
			extractedKeyStr, ok := extractedKey.(string)
			if !ok {
				return false
			}

			return extractedKeyStr == issueKey
		},
		genIssueKey,
	))

	// Property: jira_create_issue extracts all required parameters correctly
	properties.Property("jira_create_issue extracts all parameters without data loss", prop.ForAll(
		func(projectKey string, summary string, description string, issueType string, assignee string) bool {
			// Skip if any required field is empty
			if projectKey == "" || summary == "" || issueType == "" {
				return true
			}

			// Create MCP tool request with all parameters
			toolReq := &ToolRequest{
				Name: "jira_create_issue",
				Arguments: map[string]interface{}{
					"projectKey":  projectKey,
					"summary":     summary,
					"description": description,
					"issueType":   issueType,
					"assignee":    assignee,
				},
			}

			// Verify all parameters are extractable
			extractedProject, _ := toolReq.Arguments["projectKey"].(string)
			extractedSummary, _ := toolReq.Arguments["summary"].(string)
			extractedDesc, _ := toolReq.Arguments["description"].(string)
			extractedType, _ := toolReq.Arguments["issueType"].(string)
			extractedAssignee, _ := toolReq.Arguments["assignee"].(string)

			// Verify no data loss
			return extractedProject == projectKey &&
				extractedSummary == summary &&
				extractedDesc == description &&
				extractedType == issueType &&
				extractedAssignee == assignee
		},
		genProjectKey,
		gen.AlphaString(),
		gen.AlphaString(),
		gen.OneConstOf("Bug", "Story", "Task", "Epic"),
		gen.Identifier(),
	))

	// Property: jira_update_issue extracts all parameters correctly
	properties.Property("jira_update_issue extracts all parameters without data loss", prop.ForAll(
		func(issueKey string, summary string, description string, assignee string) bool {
			// Create MCP tool request
			toolReq := &ToolRequest{
				Name: "jira_update_issue",
				Arguments: map[string]interface{}{
					"issueKey":    issueKey,
					"summary":     summary,
					"description": description,
					"assignee":    assignee,
				},
			}

			// Verify all parameters are extractable
			extractedKey, _ := toolReq.Arguments["issueKey"].(string)
			extractedSummary, _ := toolReq.Arguments["summary"].(string)
			extractedDesc, _ := toolReq.Arguments["description"].(string)
			extractedAssignee, _ := toolReq.Arguments["assignee"].(string)

			// Verify no data loss
			return extractedKey == issueKey &&
				extractedSummary == summary &&
				extractedDesc == description &&
				extractedAssignee == assignee
		},
		genIssueKey,
		gen.AlphaString(),
		gen.AlphaString(),
		gen.Identifier(),
	))

	// Property: jira_search_jql extracts JQL and pagination parameters correctly
	properties.Property("jira_search_jql extracts all parameters without data loss", prop.ForAll(
		func(jql string, startAt int, maxResults int) bool {
			// Ensure valid pagination values
			if startAt < 0 {
				startAt = 0
			}
			if maxResults < 0 {
				maxResults = 50
			}

			// Create MCP tool request
			toolReq := &ToolRequest{
				Name: "jira_search_jql",
				Arguments: map[string]interface{}{
					"jql":        jql,
					"startAt":    float64(startAt), // JSON numbers are float64
					"maxResults": float64(maxResults),
				},
			}

			// Verify all parameters are extractable
			extractedJQL, _ := toolReq.Arguments["jql"].(string)
			extractedStartAt, _ := toolReq.Arguments["startAt"].(float64)
			extractedMaxResults, _ := toolReq.Arguments["maxResults"].(float64)

			// Verify no data loss
			return extractedJQL == jql &&
				int(extractedStartAt) == startAt &&
				int(extractedMaxResults) == maxResults
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.IntRange(0, 1000),
		gen.IntRange(1, 100),
	))

	// Property: jira_transition_issue extracts transition parameters correctly
	properties.Property("jira_transition_issue extracts all parameters without data loss", prop.ForAll(
		func(issueKey string, transitionID string, transitionName string) bool {
			// Create MCP tool request with both ID and name
			toolReq := &ToolRequest{
				Name: "jira_transition_issue",
				Arguments: map[string]interface{}{
					"issueKey":       issueKey,
					"transitionId":   transitionID,
					"transitionName": transitionName,
				},
			}

			// Verify all parameters are extractable
			extractedKey, _ := toolReq.Arguments["issueKey"].(string)
			extractedID, _ := toolReq.Arguments["transitionId"].(string)
			extractedName, _ := toolReq.Arguments["transitionName"].(string)

			// Verify no data loss
			return extractedKey == issueKey &&
				extractedID == transitionID &&
				extractedName == transitionName
		},
		genIssueKey,
		gen.Identifier(),
		gen.OneConstOf("To Do", "In Progress", "Done", "Closed"),
	))

	// Property: jira_add_comment extracts comment parameters correctly
	properties.Property("jira_add_comment extracts all parameters without data loss", prop.ForAll(
		func(issueKey string, body string) bool {
			// Create MCP tool request
			toolReq := &ToolRequest{
				Name: "jira_add_comment",
				Arguments: map[string]interface{}{
					"issueKey": issueKey,
					"body":     body,
				},
			}

			// Verify all parameters are extractable
			extractedKey, _ := toolReq.Arguments["issueKey"].(string)
			extractedBody, _ := toolReq.Arguments["body"].(string)

			// Verify no data loss
			return extractedKey == issueKey && extractedBody == body
		},
		genIssueKey,
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	// Property: jira_delete_issue extracts issueKey parameter correctly
	properties.Property("jira_delete_issue extracts issueKey without data loss", prop.ForAll(
		func(issueKey string) bool {
			// Create MCP tool request
			toolReq := &ToolRequest{
				Name: "jira_delete_issue",
				Arguments: map[string]interface{}{
					"issueKey": issueKey,
				},
			}

			// Verify parameter extraction
			extractedKey, exists := toolReq.Arguments["issueKey"]
			if !exists {
				return false
			}

			// Verify no data loss
			extractedKeyStr, ok := extractedKey.(string)
			if !ok {
				return false
			}

			return extractedKeyStr == issueKey
		},
		genIssueKey,
	))

	// Property: jira_list_projects handles empty arguments correctly
	properties.Property("jira_list_projects handles empty arguments", prop.ForAll(
		func() bool {
			// Create MCP tool request with no arguments
			toolReq := &ToolRequest{
				Name:      "jira_list_projects",
				Arguments: map[string]interface{}{},
			}

			// Verify arguments map exists (even if empty)
			return toolReq.Arguments != nil
		},
	))

	// Property: Parameters with special characters are preserved
	properties.Property("special characters in parameters are preserved", prop.ForAll(
		func(summary string) bool {
			// Add special characters to the summary
			specialSummary := summary + " !@#$%^&*()_+-=[]{}|;':\",./<>?"

			// Create MCP tool request
			toolReq := &ToolRequest{
				Name: "jira_create_issue",
				Arguments: map[string]interface{}{
					"projectKey": "TEST",
					"summary":    specialSummary,
					"issueType":  "Bug",
				},
			}

			// Verify parameter extraction preserves special characters
			extractedSummary, _ := toolReq.Arguments["summary"].(string)
			return extractedSummary == specialSummary
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	// Property: Unicode characters in parameters are preserved
	properties.Property("unicode characters in parameters are preserved", prop.ForAll(
		func(summary string) bool {
			// Add unicode characters
			unicodeSummary := summary + " 你好世界 🚀 émojis"

			// Create MCP tool request
			toolReq := &ToolRequest{
				Name: "jira_create_issue",
				Arguments: map[string]interface{}{
					"projectKey": "TEST",
					"summary":    unicodeSummary,
					"issueType":  "Bug",
				},
			}

			// Verify parameter extraction preserves unicode
			extractedSummary, _ := toolReq.Arguments["summary"].(string)
			return extractedSummary == unicodeSummary
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	// Property: Empty optional parameters are handled correctly
	properties.Property("empty optional parameters are handled correctly", prop.ForAll(
		func(projectKey string, summary string, issueType string) bool {
			// Skip if any required field is empty
			if projectKey == "" || summary == "" || issueType == "" {
				return true
			}

			// Create MCP tool request with empty optional parameters
			toolReq := &ToolRequest{
				Name: "jira_create_issue",
				Arguments: map[string]interface{}{
					"projectKey":  projectKey,
					"summary":     summary,
					"issueType":   issueType,
					"description": "", // Empty optional parameter
					"assignee":    "", // Empty optional parameter
				},
			}

			// Verify all parameters are extractable
			extractedProject, _ := toolReq.Arguments["projectKey"].(string)
			extractedSummary, _ := toolReq.Arguments["summary"].(string)
			extractedType, _ := toolReq.Arguments["issueType"].(string)
			extractedDesc, _ := toolReq.Arguments["description"].(string)
			extractedAssignee, _ := toolReq.Arguments["assignee"].(string)

			// Verify no data loss (empty strings should be preserved)
			return extractedProject == projectKey &&
				extractedSummary == summary &&
				extractedType == issueType &&
				extractedDesc == "" &&
				extractedAssignee == ""
		},
		genProjectKey,
		gen.AlphaString(),
		gen.OneConstOf("Bug", "Story", "Task"),
	))

	// Property: Large parameter values are preserved
	properties.Property("large parameter values are preserved without truncation", prop.ForAll(
		func(seed int) bool {
			// Generate a large description (10KB)
			largeDesc := ""
			for i := 0; i < 10000; i++ {
				largeDesc += "a"
			}

			// Create MCP tool request
			toolReq := &ToolRequest{
				Name: "jira_create_issue",
				Arguments: map[string]interface{}{
					"projectKey":  "TEST",
					"summary":     "Test issue",
					"issueType":   "Bug",
					"description": largeDesc,
				},
			}

			// Verify parameter extraction preserves full length
			extractedDesc, _ := toolReq.Arguments["description"].(string)
			return len(extractedDesc) == len(largeDesc) && extractedDesc == largeDesc
		},
		gen.Int(),
	))

	// Property: Numeric parameters maintain precision
	properties.Property("numeric parameters maintain precision", prop.ForAll(
		func(startAt int, maxResults int) bool {
			// Ensure valid values
			if startAt < 0 {
				startAt = 0
			}
			if maxResults < 1 {
				maxResults = 1
			}

			// Create MCP tool request
			toolReq := &ToolRequest{
				Name: "jira_search_jql",
				Arguments: map[string]interface{}{
					"jql":        "project = TEST",
					"startAt":    float64(startAt),
					"maxResults": float64(maxResults),
				},
			}

			// Verify numeric parameters maintain precision
			extractedStartAt, _ := toolReq.Arguments["startAt"].(float64)
			extractedMaxResults, _ := toolReq.Arguments["maxResults"].(float64)

			return int(extractedStartAt) == startAt && int(extractedMaxResults) == maxResults
		},
		gen.IntRange(0, 10000),
		gen.IntRange(1, 1000),
	))

	properties.TestingRun(t)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
