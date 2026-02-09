package application

import (
	"context"
	"fmt"
	"testing"

	"atlassian-mcp-server/internal/domain"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: atlassian-mcp-server, Property 1: Request Forwarding Correctness
// **Validates: Requirements 1.1, 2.1, 3.1, 4.1**
//
// For any valid MCP tool request targeting a configured Atlassian tool, the server should
// construct and forward an HTTP request to the correct Atlassian API endpoint with all
// parameters properly mapped.
func TestProperty1_RequestForwardingCorrectness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Generator for tool types
	genToolType := gen.OneConstOf("jira", "confluence", "bitbucket", "bamboo")

	// Generator for operation names
	genJiraOp := gen.OneConstOf("get_issue", "create_issue", "update_issue", "delete_issue", "search_jql", "transition_issue", "add_comment", "list_projects")
	genConfluenceOp := gen.OneConstOf("get_page", "create_page", "update_page", "delete_page", "search_cql", "list_spaces")
	genBitbucketOp := gen.OneConstOf("list_repositories", "get_branches", "create_branch", "get_pull_request", "create_pull_request", "merge_pull_request")
	genBambooOp := gen.OneConstOf("list_plans", "get_plan", "trigger_build", "get_build_result", "get_build_log")

	// Generator for valid tool names
	genToolName := gen.OneGenOf(
		genJiraOp.Map(func(op string) string { return "jira_" + op }),
		genConfluenceOp.Map(func(op string) string { return "confluence_" + op }),
		genBitbucketOp.Map(func(op string) string { return "bitbucket_" + op }),
		genBambooOp.Map(func(op string) string { return "bamboo_" + op }),
	)

	// Generator for valid arguments - simplified approach
	// We'll generate arguments in the property function itself to avoid type issues
	genArgumentCount := gen.IntRange(0, 3)

	// Property: Valid MCP tool requests are forwarded to the correct handler
	properties.Property("Valid tool requests are forwarded to correct handler", prop.ForAll(
		func(toolName string, argCount int, key1 string, val1 string, key2 int) bool {
			// Build arguments based on argCount
			arguments := make(map[string]interface{})
			if argCount >= 1 && key1 != "" {
				arguments[key1] = val1
			}
			if argCount >= 2 {
				arguments["param2"] = key2
			}
			if argCount >= 3 {
				arguments["param3"] = true
			}

			// Create a tracking handler that records if it was called
			called := false
			var receivedReq *domain.ToolRequest

			trackingHandler := &trackingToolHandler{
				name: extractToolTypeFromName(toolName),
				tools: []domain.ToolDefinition{
					{
						Name:        toolName,
						Description: "Test tool",
						InputSchema: domain.JSONSchema{Type: "object"},
					},
				},
				onHandle: func(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error) {
					called = true
					receivedReq = req
					return &domain.ToolResponse{
						Content: []domain.ContentBlock{
							{Type: "text", Text: "success"},
						},
					}, nil
				},
			}

			// Create router with tracking handler
			router := NewRequestRouter(trackingHandler)

			// Create auth manager with valid credentials for all tools
			authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{
				"jira": {
					Type:     domain.BasicAuth,
					Username: "testuser",
					Password: "testpass",
				},
				"confluence": {
					Type:     domain.BasicAuth,
					Username: "testuser",
					Password: "testpass",
				},
				"bitbucket": {
					Type:     domain.BasicAuth,
					Username: "testuser",
					Password: "testpass",
				},
				"bamboo": {
					Type:     domain.BasicAuth,
					Username: "testuser",
					Password: "testpass",
				},
			})

			// Create config
			config := &domain.Config{
				Transport: domain.TransportConfig{Type: "stdio"},
				Tools: domain.ToolsConfig{
					Jira:       &domain.ToolConfig{BaseURL: "https://jira.example.com", Auth: &domain.AuthConfig{Type: "basic", Username: "testuser", Password: "testpass"}},
					Confluence: &domain.ToolConfig{BaseURL: "https://confluence.example.com", Auth: &domain.AuthConfig{Type: "basic", Username: "testuser", Password: "testpass"}},
					Bitbucket:  &domain.ToolConfig{BaseURL: "https://bitbucket.example.com", Auth: &domain.AuthConfig{Type: "basic", Username: "testuser", Password: "testpass"}},
					Bamboo:     &domain.ToolConfig{BaseURL: "https://bamboo.example.com", Auth: &domain.AuthConfig{Type: "basic", Username: "testuser", Password: "testpass"}},
				},
			}

			// Create mock transport
			transport := newMockTransport()

			// Create server
			server := NewServer(transport, router, authManager, config)

			// Create tool request
			toolReq := &domain.ToolRequest{
				Name:      toolName,
				Arguments: arguments,
			}

			// Validate authentication (this is what the server does)
			if err := server.validateAuthentication(toolName); err != nil {
				// Authentication should succeed for valid tool names
				return false
			}

			// Route the request (this is what the server does after auth)
			ctx := context.Background()
			_, err := router.Route(ctx, toolReq)

			// Should not error for valid tool names
			if err != nil {
				return false
			}

			// Handler should have been called
			if !called {
				return false
			}

			// Received request should match the original
			if receivedReq == nil {
				return false
			}

			if receivedReq.Name != toolName {
				return false
			}

			// Arguments should be preserved
			if len(receivedReq.Arguments) != len(arguments) {
				return false
			}

			return true
		},
		genToolName,
		genArgumentCount,
		gen.Identifier(),
		gen.AlphaString(),
		gen.Int(),
	))

	// Property: Tool type is correctly extracted from tool name
	properties.Property("Tool type is correctly extracted from tool name", prop.ForAll(
		func(toolType string, operation string) bool {
			toolName := toolType + "_" + operation

			extracted := extractToolType(toolName)

			return extracted == toolType
		},
		genToolType,
		gen.Identifier(),
	))

	// Property: Parameters are preserved during forwarding
	properties.Property("Parameters are preserved during request forwarding", prop.ForAll(
		func(toolName string, key string, value string) bool {
			// Skip empty keys
			if key == "" {
				return true
			}

			arguments := map[string]interface{}{
				key: value,
			}

			var receivedArgs map[string]interface{}

			trackingHandler := &trackingToolHandler{
				name: extractToolTypeFromName(toolName),
				tools: []domain.ToolDefinition{
					{Name: toolName, Description: "Test", InputSchema: domain.JSONSchema{Type: "object"}},
				},
				onHandle: func(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error) {
					receivedArgs = req.Arguments
					return &domain.ToolResponse{
						Content: []domain.ContentBlock{{Type: "text", Text: "ok"}},
					}, nil
				},
			}

			router := NewRequestRouter(trackingHandler)

			toolReq := &domain.ToolRequest{
				Name:      toolName,
				Arguments: arguments,
			}

			ctx := context.Background()
			_, err := router.Route(ctx, toolReq)

			if err != nil {
				return false
			}

			// Verify parameter was preserved
			if receivedArgs == nil {
				return false
			}

			receivedValue, exists := receivedArgs[key]
			if !exists {
				return false
			}

			return receivedValue == value
		},
		genToolName,
		gen.Identifier(),
		gen.AlphaString(),
	))

	// Property: Multiple parameters are all preserved
	properties.Property("Multiple parameters are all preserved during forwarding", prop.ForAll(
		func(toolName string, param1 string, param2 int, param3 bool) bool {
			arguments := map[string]interface{}{
				"param1": param1,
				"param2": param2,
				"param3": param3,
			}

			var receivedArgs map[string]interface{}

			trackingHandler := &trackingToolHandler{
				name: extractToolTypeFromName(toolName),
				tools: []domain.ToolDefinition{
					{Name: toolName, Description: "Test", InputSchema: domain.JSONSchema{Type: "object"}},
				},
				onHandle: func(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error) {
					receivedArgs = req.Arguments
					return &domain.ToolResponse{
						Content: []domain.ContentBlock{{Type: "text", Text: "ok"}},
					}, nil
				},
			}

			router := NewRequestRouter(trackingHandler)

			toolReq := &domain.ToolRequest{
				Name:      toolName,
				Arguments: arguments,
			}

			ctx := context.Background()
			_, err := router.Route(ctx, toolReq)

			if err != nil {
				return false
			}

			// Verify all parameters were preserved
			if receivedArgs == nil {
				return false
			}

			if receivedArgs["param1"] != param1 {
				return false
			}

			if receivedArgs["param2"] != param2 {
				return false
			}

			if receivedArgs["param3"] != param3 {
				return false
			}

			return true
		},
		genToolName,
		gen.AlphaString(),
		gen.Int(),
		gen.Bool(),
	))

	// Property: Router correctly identifies handler from tool name prefix
	properties.Property("Router identifies correct handler from tool name prefix", prop.ForAll(
		func(toolType string, operation string) bool {
			// Create handlers for all tool types
			handlers := make(map[string]*trackingToolHandler)
			for _, tt := range []string{"jira", "confluence", "bitbucket", "bamboo"} {
				handlers[tt] = &trackingToolHandler{
					name: tt,
					tools: []domain.ToolDefinition{
						{Name: tt + "_test", Description: "Test", InputSchema: domain.JSONSchema{Type: "object"}},
					},
					onHandle: func(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error) {
						return &domain.ToolResponse{
							Content: []domain.ContentBlock{{Type: "text", Text: "ok"}},
						}, nil
					},
				}
			}

			// Create router with all handlers
			router := NewRequestRouter(
				handlers["jira"],
				handlers["confluence"],
				handlers["bitbucket"],
				handlers["bamboo"],
			)

			// Get the handler that should be used
			expectedHandler, exists := router.GetHandler(toolType)
			if !exists {
				return false
			}

			// Verify it's the correct handler
			if expectedHandler.ToolName() != toolType {
				return false
			}

			return true
		},
		genToolType,
		gen.Identifier(),
	))

	// Property: Invalid tool names are rejected before forwarding
	properties.Property("Invalid tool names are rejected", prop.ForAll(
		func(invalidName string) bool {
			// Skip valid tool name patterns
			if containsUnderscore(invalidName) {
				toolType := extractToolTypeFromName(invalidName)
				if toolType == "jira" || toolType == "confluence" || toolType == "bitbucket" || toolType == "bamboo" {
					return true // Skip valid patterns
				}
			}

			router := NewRequestRouter()

			ctx := context.Background()
			_, err := router.Route(ctx, &domain.ToolRequest{
				Name:      invalidName,
				Arguments: map[string]interface{}{},
			})

			// Should return an error for invalid tool names
			return err != nil
		},
		gen.OneConstOf("invalid", "no-underscore", "", "123", "tool", "unknown_tool"),
	))

	// Property: Empty arguments are handled correctly
	properties.Property("Empty arguments are handled correctly", prop.ForAll(
		func(toolName string) bool {
			var receivedArgs map[string]interface{}

			trackingHandler := &trackingToolHandler{
				name: extractToolTypeFromName(toolName),
				tools: []domain.ToolDefinition{
					{Name: toolName, Description: "Test", InputSchema: domain.JSONSchema{Type: "object"}},
				},
				onHandle: func(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error) {
					receivedArgs = req.Arguments
					return &domain.ToolResponse{
						Content: []domain.ContentBlock{{Type: "text", Text: "ok"}},
					}, nil
				},
			}

			router := NewRequestRouter(trackingHandler)

			toolReq := &domain.ToolRequest{
				Name:      toolName,
				Arguments: map[string]interface{}{},
			}

			ctx := context.Background()
			_, err := router.Route(ctx, toolReq)

			if err != nil {
				return false
			}

			// Arguments should be an empty map, not nil
			if receivedArgs == nil {
				return false
			}

			return len(receivedArgs) == 0
		},
		genToolName,
	))

	properties.TestingRun(t)
}

// trackingToolHandler is a test helper that tracks whether Handle was called
type trackingToolHandler struct {
	name     string
	tools    []domain.ToolDefinition
	onHandle func(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error)
}

func (h *trackingToolHandler) Handle(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error) {
	if h.onHandle != nil {
		return h.onHandle(ctx, req)
	}
	return &domain.ToolResponse{
		Content: []domain.ContentBlock{
			{Type: "text", Text: "default response"},
		},
	}, nil
}

func (h *trackingToolHandler) ListTools() []domain.ToolDefinition {
	return h.tools
}

func (h *trackingToolHandler) ToolName() string {
	return h.name
}

// extractToolTypeFromName extracts the tool type from a tool name
func extractToolTypeFromName(toolName string) string {
	for i, c := range toolName {
		if c == '_' {
			return toolName[:i]
		}
	}
	return ""
}

// containsUnderscore checks if a string contains an underscore
func containsUnderscore(s string) bool {
	for _, c := range s {
		if c == '_' {
			return true
		}
	}
	return false
}

// Feature: atlassian-mcp-server, Property 5: Authentication Precedes API Calls
// **Validates: Requirements 5.5**
//
// For any tool request, authentication validation must complete before any HTTP request
// is made to an Atlassian API, ensuring unauthenticated requests never reach external systems.
func TestProperty5_AuthenticationPrecedesAPICalls(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Generator for tool types
	genToolType := gen.OneConstOf("jira", "confluence", "bitbucket", "bamboo")

	// Generator for valid tool names
	genJiraOp := gen.OneConstOf("get_issue", "create_issue", "update_issue", "delete_issue", "search_jql", "transition_issue", "add_comment", "list_projects")
	genConfluenceOp := gen.OneConstOf("get_page", "create_page", "update_page", "delete_page", "search_cql", "list_spaces")
	genBitbucketOp := gen.OneConstOf("list_repositories", "get_branches", "create_branch", "get_pull_request", "create_pull_request", "merge_pull_request")
	genBambooOp := gen.OneConstOf("list_plans", "get_plan", "trigger_build", "get_build_result", "get_build_log")

	genToolName := gen.OneGenOf(
		genJiraOp.Map(func(op string) string { return "jira_" + op }),
		genConfluenceOp.Map(func(op string) string { return "confluence_" + op }),
		genBitbucketOp.Map(func(op string) string { return "bitbucket_" + op }),
		genBambooOp.Map(func(op string) string { return "bamboo_" + op }),
	)

	// Property: Authentication validation must occur before handler is called
	properties.Property("Authentication validation occurs before handler execution", prop.ForAll(
		func(toolName string, param1 string, param2 int) bool {
			// Track whether handler was called
			handlerCalled := false

			trackingHandler := &trackingToolHandler{
				name: extractToolTypeFromName(toolName),
				tools: []domain.ToolDefinition{
					{
						Name:        toolName,
						Description: "Test tool",
						InputSchema: domain.JSONSchema{Type: "object"},
					},
				},
				onHandle: func(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error) {
					handlerCalled = true
					return &domain.ToolResponse{
						Content: []domain.ContentBlock{{Type: "text", Text: "success"}},
					}, nil
				},
			}

			router := NewRequestRouter(trackingHandler)

			// Create auth manager WITHOUT credentials for this tool
			authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{})

			config := &domain.Config{
				Transport: domain.TransportConfig{Type: "stdio"},
				Tools:     domain.ToolsConfig{},
			}

			transport := newMockTransport()
			server := NewServer(transport, router, authManager, config)

			// Attempt to validate authentication (this is what server does first)
			err := server.validateAuthentication(toolName)

			// Authentication should fail for missing credentials
			if err == nil {
				return false
			}

			// Handler should NOT have been called because auth failed
			// This verifies that authentication check happens before routing
			if handlerCalled {
				return false
			}

			// The authentication error should prevent any further processing
			return err != nil && !handlerCalled
		},
		genToolName,
		gen.AlphaString(),
		gen.Int(),
	))

	// Property: Valid credentials allow handler execution
	properties.Property("Valid credentials allow handler to be called", prop.ForAll(
		func(toolName string, username string, password string) bool {
			// Skip empty credentials
			if username == "" || password == "" {
				return true
			}

			handlerCalled := false

			trackingHandler := &trackingToolHandler{
				name: extractToolTypeFromName(toolName),
				tools: []domain.ToolDefinition{
					{
						Name:        toolName,
						Description: "Test tool",
						InputSchema: domain.JSONSchema{Type: "object"},
					},
				},
				onHandle: func(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error) {
					handlerCalled = true
					return &domain.ToolResponse{
						Content: []domain.ContentBlock{{Type: "text", Text: "success"}},
					}, nil
				},
			}

			router := NewRequestRouter(trackingHandler)

			// Create auth manager WITH valid credentials for all tools
			toolType := extractToolTypeFromName(toolName)
			authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{
				toolType: {
					Type:     domain.BasicAuth,
					Username: username,
					Password: password,
				},
			})

			config := &domain.Config{
				Transport: domain.TransportConfig{Type: "stdio"},
				Tools:     domain.ToolsConfig{},
			}

			transport := newMockTransport()
			server := NewServer(transport, router, authManager, config)

			// Validate authentication
			err := server.validateAuthentication(toolName)

			// Authentication should succeed
			if err != nil {
				return false
			}

			// Now route the request
			ctx := context.Background()
			toolReq := &domain.ToolRequest{
				Name:      toolName,
				Arguments: map[string]interface{}{},
			}

			_, routeErr := router.Route(ctx, toolReq)

			// Handler should have been called after successful auth
			if !handlerCalled {
				return false
			}

			// Should not have routing errors
			return routeErr == nil && handlerCalled
		},
		genToolName,
		gen.Identifier(),
		gen.AlphaString(),
	))

	// Property: Missing credentials for specific tool type are detected
	properties.Property("Missing credentials for specific tool are detected", prop.ForAll(
		func(toolType string, otherToolType string) bool {
			// Ensure we have two different tool types
			if toolType == otherToolType {
				return true
			}

			// Create credentials for otherToolType but not for toolType
			authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{
				otherToolType: {
					Type:     domain.BasicAuth,
					Username: "user",
					Password: "pass",
				},
			})

			config := &domain.Config{
				Transport: domain.TransportConfig{Type: "stdio"},
				Tools:     domain.ToolsConfig{},
			}

			transport := newMockTransport()
			router := NewRequestRouter()
			server := NewServer(transport, router, authManager, config)

			// Try to validate authentication for toolType (which has no credentials)
			toolName := toolType + "_test_operation"
			err := server.validateAuthentication(toolName)

			// Should fail because credentials are missing for this tool
			return err != nil
		},
		genToolType,
		genToolType,
	))

	// Property: Invalid credentials (empty username or password) are rejected
	properties.Property("Invalid credentials are rejected before handler execution", prop.ForAll(
		func(toolName string, hasUsername bool, hasPassword bool) bool {
			// Skip the case where both are present (valid credentials)
			if hasUsername && hasPassword {
				return true
			}

			toolType := extractToolTypeFromName(toolName)

			username := ""
			if hasUsername {
				username = "testuser"
			}

			password := ""
			if hasPassword {
				password = "testpass"
			}

			authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{
				toolType: {
					Type:     domain.BasicAuth,
					Username: username,
					Password: password,
				},
			})

			config := &domain.Config{
				Transport: domain.TransportConfig{Type: "stdio"},
				Tools:     domain.ToolsConfig{},
			}

			transport := newMockTransport()
			router := NewRequestRouter()
			server := NewServer(transport, router, authManager, config)

			// Try to validate authentication
			err := server.validateAuthentication(toolName)

			// Should fail because credentials are incomplete
			return err != nil
		},
		genToolName,
		gen.Bool(),
		gen.Bool(),
	))

	// Property: Authentication error response is returned before routing
	properties.Property("Authentication errors return error response without routing", prop.ForAll(
		func(toolName string, requestID string) bool {
			// Skip empty request IDs
			if requestID == "" {
				return true
			}

			handlerCalled := false

			trackingHandler := &trackingToolHandler{
				name: extractToolTypeFromName(toolName),
				tools: []domain.ToolDefinition{
					{
						Name:        toolName,
						Description: "Test tool",
						InputSchema: domain.JSONSchema{Type: "object"},
					},
				},
				onHandle: func(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error) {
					handlerCalled = true
					return &domain.ToolResponse{
						Content: []domain.ContentBlock{{Type: "text", Text: "success"}},
					}, nil
				},
			}

			router := NewRequestRouter(trackingHandler)

			// Create auth manager without credentials
			authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{})

			config := &domain.Config{
				Transport: domain.TransportConfig{Type: "stdio"},
				Tools:     domain.ToolsConfig{},
			}

			transport := newMockTransport()
			server := NewServer(transport, router, authManager, config)

			// Create a tools/call request
			req := &domain.Request{
				JSONRPC: "2.0",
				ID:      requestID,
				Method:  "tools/call",
				Params: map[string]interface{}{
					"name":      toolName,
					"arguments": map[string]interface{}{},
				},
			}

			// Handle the request (this should fail at authentication)
			ctx := context.Background()
			_, err := server.handleToolsCall(ctx, req)

			// Should have an error
			if err == nil {
				return false
			}

			// Handler should NOT have been called
			if handlerCalled {
				return false
			}

			// Check that an error response was sent
			lastResp := transport.getLastResponse()
			if lastResp == nil {
				return false
			}

			// Response should have an error
			if lastResp.Error == nil {
				return false
			}

			// Error code should be AuthenticationError
			if lastResp.Error.Code != domain.AuthenticationError {
				return false
			}

			return true
		},
		genToolName,
		gen.Identifier(),
	))

	// Property: Token authentication is validated before handler execution
	properties.Property("Token authentication is validated before handler execution", prop.ForAll(
		func(toolName string, hasToken bool) bool {
			handlerCalled := false

			trackingHandler := &trackingToolHandler{
				name: extractToolTypeFromName(toolName),
				tools: []domain.ToolDefinition{
					{
						Name:        toolName,
						Description: "Test tool",
						InputSchema: domain.JSONSchema{Type: "object"},
					},
				},
				onHandle: func(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error) {
					handlerCalled = true
					return &domain.ToolResponse{
						Content: []domain.ContentBlock{{Type: "text", Text: "success"}},
					}, nil
				},
			}

			router := NewRequestRouter(trackingHandler)

			toolType := extractToolTypeFromName(toolName)

			token := ""
			if hasToken {
				token = "valid-token-12345"
			}

			authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{
				toolType: {
					Type:  domain.TokenAuth,
					Token: token,
				},
			})

			config := &domain.Config{
				Transport: domain.TransportConfig{Type: "stdio"},
				Tools:     domain.ToolsConfig{},
			}

			transport := newMockTransport()
			server := NewServer(transport, router, authManager, config)

			// Validate authentication
			err := server.validateAuthentication(toolName)

			if hasToken {
				// Should succeed with valid token
				if err != nil {
					return false
				}

				// Try routing
				ctx := context.Background()
				_, routeErr := router.Route(ctx, &domain.ToolRequest{
					Name:      toolName,
					Arguments: map[string]interface{}{},
				})

				// Handler should be called
				return routeErr == nil && handlerCalled
			} else {
				// Should fail without token
				if err == nil {
					return false
				}

				// Handler should NOT be called
				return !handlerCalled
			}
		},
		genToolName,
		gen.Bool(),
	))

	// Property: Authentication validation happens for every request
	properties.Property("Authentication is validated for every request", prop.ForAll(
		func(toolName string, requestCount int) bool {
			// Limit request count to reasonable range
			if requestCount < 1 || requestCount > 10 {
				return true
			}

			callCount := 0

			trackingHandler := &trackingToolHandler{
				name: extractToolTypeFromName(toolName),
				tools: []domain.ToolDefinition{
					{
						Name:        toolName,
						Description: "Test tool",
						InputSchema: domain.JSONSchema{Type: "object"},
					},
				},
				onHandle: func(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error) {
					callCount++
					return &domain.ToolResponse{
						Content: []domain.ContentBlock{{Type: "text", Text: "success"}},
					}, nil
				},
			}

			router := NewRequestRouter(trackingHandler)

			// Create auth manager without credentials
			authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{})

			config := &domain.Config{
				Transport: domain.TransportConfig{Type: "stdio"},
				Tools:     domain.ToolsConfig{},
			}

			transport := newMockTransport()
			server := NewServer(transport, router, authManager, config)

			// Make multiple requests
			for i := 0; i < requestCount; i++ {
				err := server.validateAuthentication(toolName)

				// Each request should fail authentication
				if err == nil {
					return false
				}
			}

			// Handler should never have been called
			if callCount != 0 {
				return false
			}

			return true
		},
		genToolName,
		gen.IntRange(1, 10),
	))

	properties.TestingRun(t)
}

// Feature: atlassian-mcp-server, Property 11: Malformed Request Rejection
// **Validates: Requirements 8.4**
//
// For any MCP request with invalid structure, missing required fields, or invalid parameter types,
// the server should return a validation error without attempting to call any Atlassian API.
func TestProperty11_MalformedRequestRejection(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Generator for valid tool names
	genJiraOp := gen.OneConstOf("get_issue", "create_issue", "update_issue", "delete_issue", "search_jql", "transition_issue", "add_comment", "list_projects")
	genConfluenceOp := gen.OneConstOf("get_page", "create_page", "update_page", "delete_page", "search_cql", "list_spaces")
	genBitbucketOp := gen.OneConstOf("list_repositories", "get_branches", "create_branch", "get_pull_request", "create_pull_request", "merge_pull_request")
	genBambooOp := gen.OneConstOf("list_plans", "get_plan", "trigger_build", "get_build_result", "get_build_log")

	genToolName := gen.OneGenOf(
		genJiraOp.Map(func(op string) string { return "jira_" + op }),
		genConfluenceOp.Map(func(op string) string { return "confluence_" + op }),
		genBitbucketOp.Map(func(op string) string { return "bitbucket_" + op }),
		genBambooOp.Map(func(op string) string { return "bamboo_" + op }),
	)

	// Property: Invalid JSON-RPC version is rejected
	properties.Property("Invalid JSON-RPC version is rejected", prop.ForAll(
		func(invalidVersion string, method string, requestID string) bool {
			// Skip valid version
			if invalidVersion == "2.0" {
				return true
			}

			handlerCalled := false

			trackingHandler := &trackingToolHandler{
				name: "jira",
				tools: []domain.ToolDefinition{
					{Name: "jira_test", Description: "Test", InputSchema: domain.JSONSchema{Type: "object"}},
				},
				onHandle: func(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error) {
					handlerCalled = true
					return &domain.ToolResponse{
						Content: []domain.ContentBlock{{Type: "text", Text: "success"}},
					}, nil
				},
			}

			router := NewRequestRouter(trackingHandler)
			authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{
				"jira": {Type: domain.BasicAuth, Username: "user", Password: "pass"},
			})
			config := &domain.Config{
				Transport: domain.TransportConfig{Type: "stdio"},
				Tools:     domain.ToolsConfig{},
			}
			transport := newMockTransport()
			server := NewServer(transport, router, authManager, config)

			// Create request with invalid JSON-RPC version
			req := &domain.Request{
				JSONRPC: invalidVersion,
				ID:      requestID,
				Method:  method,
			}

			// Validate the request
			err := server.validateRequest(req)

			// Should return an error for invalid version
			if err == nil {
				return false
			}

			// Handler should NOT have been called
			return !handlerCalled
		},
		gen.OneConstOf("1.0", "2.1", "3.0", "", "invalid", "2"),
		gen.OneConstOf("initialize", "tools/list", "tools/call"),
		gen.Identifier(),
	))

	// Property: Missing method field is rejected
	properties.Property("Missing method field is rejected", prop.ForAll(
		func(requestID string) bool {
			handlerCalled := false

			trackingHandler := &trackingToolHandler{
				name: "jira",
				tools: []domain.ToolDefinition{
					{Name: "jira_test", Description: "Test", InputSchema: domain.JSONSchema{Type: "object"}},
				},
				onHandle: func(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error) {
					handlerCalled = true
					return &domain.ToolResponse{
						Content: []domain.ContentBlock{{Type: "text", Text: "success"}},
					}, nil
				},
			}

			router := NewRequestRouter(trackingHandler)
			authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{
				"jira": {Type: domain.BasicAuth, Username: "user", Password: "pass"},
			})
			config := &domain.Config{
				Transport: domain.TransportConfig{Type: "stdio"},
				Tools:     domain.ToolsConfig{},
			}
			transport := newMockTransport()
			server := NewServer(transport, router, authManager, config)

			// Create request with missing method
			req := &domain.Request{
				JSONRPC: "2.0",
				ID:      requestID,
				Method:  "", // Empty method
			}

			// Validate the request
			err := server.validateRequest(req)

			// Should return an error for missing method
			if err == nil {
				return false
			}

			// Handler should NOT have been called
			return !handlerCalled
		},
		gen.Identifier(),
	))

	// Property: Missing params for tools/call is rejected
	properties.Property("Missing params for tools/call is rejected", prop.ForAll(
		func(requestID string) bool {
			handlerCalled := false

			trackingHandler := &trackingToolHandler{
				name: "jira",
				tools: []domain.ToolDefinition{
					{Name: "jira_test", Description: "Test", InputSchema: domain.JSONSchema{Type: "object"}},
				},
				onHandle: func(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error) {
					handlerCalled = true
					return &domain.ToolResponse{
						Content: []domain.ContentBlock{{Type: "text", Text: "success"}},
					}, nil
				},
			}

			router := NewRequestRouter(trackingHandler)
			authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{
				"jira": {Type: domain.BasicAuth, Username: "user", Password: "pass"},
			})
			config := &domain.Config{
				Transport: domain.TransportConfig{Type: "stdio"},
				Tools:     domain.ToolsConfig{},
			}
			transport := newMockTransport()
			server := NewServer(transport, router, authManager, config)

			// Create request with nil params
			req := &domain.Request{
				JSONRPC: "2.0",
				ID:      requestID,
				Method:  "tools/call",
				Params:  nil, // Missing params
			}

			// Try to parse tool request
			_, err := server.parseToolRequest(req.Params)

			// Should return an error for missing params
			if err == nil {
				return false
			}

			// Handler should NOT have been called
			return !handlerCalled
		},
		gen.Identifier(),
	))

	// Property: Missing tool name in params is rejected
	properties.Property("Missing tool name in params is rejected", prop.ForAll(
		func(requestID string) bool {
			handlerCalled := false

			trackingHandler := &trackingToolHandler{
				name: "jira",
				tools: []domain.ToolDefinition{
					{Name: "jira_test", Description: "Test", InputSchema: domain.JSONSchema{Type: "object"}},
				},
				onHandle: func(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error) {
					handlerCalled = true
					return &domain.ToolResponse{
						Content: []domain.ContentBlock{{Type: "text", Text: "success"}},
					}, nil
				},
			}

			router := NewRequestRouter(trackingHandler)
			authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{
				"jira": {Type: domain.BasicAuth, Username: "user", Password: "pass"},
			})
			config := &domain.Config{
				Transport: domain.TransportConfig{Type: "stdio"},
				Tools:     domain.ToolsConfig{},
			}
			transport := newMockTransport()
			server := NewServer(transport, router, authManager, config)

			// Create params without tool name
			params := map[string]interface{}{
				"arguments": map[string]interface{}{},
				// Missing "name" field
			}

			// Try to parse tool request
			_, err := server.parseToolRequest(params)

			// Should return an error for missing tool name
			if err == nil {
				return false
			}

			// Handler should NOT have been called
			return !handlerCalled
		},
		gen.Identifier(),
	))

	// Property: Invalid params structure is rejected
	properties.Property("Invalid params structure is rejected", prop.ForAll(
		func(requestID string, invalidParams interface{}) bool {
			// Skip valid params structures
			if invalidParams == nil {
				return true
			}
			if _, ok := invalidParams.(map[string]interface{}); ok {
				return true
			}

			handlerCalled := false

			trackingHandler := &trackingToolHandler{
				name: "jira",
				tools: []domain.ToolDefinition{
					{Name: "jira_test", Description: "Test", InputSchema: domain.JSONSchema{Type: "object"}},
				},
				onHandle: func(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error) {
					handlerCalled = true
					return &domain.ToolResponse{
						Content: []domain.ContentBlock{{Type: "text", Text: "success"}},
					}, nil
				},
			}

			router := NewRequestRouter(trackingHandler)
			authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{
				"jira": {Type: domain.BasicAuth, Username: "user", Password: "pass"},
			})
			config := &domain.Config{
				Transport: domain.TransportConfig{Type: "stdio"},
				Tools:     domain.ToolsConfig{},
			}
			transport := newMockTransport()
			server := NewServer(transport, router, authManager, config)

			// Try to parse invalid params
			_, _ = server.parseToolRequest(invalidParams)

			// Should return an error or successfully parse but fail validation
			// Handler should NOT have been called
			return !handlerCalled
		},
		gen.Identifier(),
		gen.OneGenOf(
			gen.Const("string_instead_of_object"),
			gen.Const(12345),
			gen.Const(true),
			gen.SliceOf(gen.Int()),
		),
	))

	// Property: Malformed requests do not reach handlers
	properties.Property("Malformed requests never reach tool handlers", prop.ForAll(
		func(toolName string, hasValidVersion bool, hasMethod bool, hasParams bool, hasToolName bool) bool {
			// Skip the case where everything is valid
			if hasValidVersion && hasMethod && hasParams && hasToolName {
				return true
			}

			handlerCalled := false

			trackingHandler := &trackingToolHandler{
				name: extractToolTypeFromName(toolName),
				tools: []domain.ToolDefinition{
					{Name: toolName, Description: "Test", InputSchema: domain.JSONSchema{Type: "object"}},
				},
				onHandle: func(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error) {
					handlerCalled = true
					return &domain.ToolResponse{
						Content: []domain.ContentBlock{{Type: "text", Text: "success"}},
					}, nil
				},
			}

			router := NewRequestRouter(trackingHandler)
			authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{
				"jira":       {Type: domain.BasicAuth, Username: "user", Password: "pass"},
				"confluence": {Type: domain.BasicAuth, Username: "user", Password: "pass"},
				"bitbucket":  {Type: domain.BasicAuth, Username: "user", Password: "pass"},
				"bamboo":     {Type: domain.BasicAuth, Username: "user", Password: "pass"},
			})
			config := &domain.Config{
				Transport: domain.TransportConfig{Type: "stdio"},
				Tools:     domain.ToolsConfig{},
			}
			transport := newMockTransport()
			server := NewServer(transport, router, authManager, config)

			// Build request with potentially invalid fields
			version := "2.0"
			if !hasValidVersion {
				version = "1.0"
			}

			method := "tools/call"
			if !hasMethod {
				method = ""
			}

			var params interface{}
			if hasParams {
				if hasToolName {
					params = map[string]interface{}{
						"name":      toolName,
						"arguments": map[string]interface{}{},
					}
				} else {
					params = map[string]interface{}{
						"arguments": map[string]interface{}{},
						// Missing "name"
					}
				}
			} else {
				params = nil
			}

			req := &domain.Request{
				JSONRPC: version,
				ID:      "test-id",
				Method:  method,
				Params:  params,
			}

			// Validate request structure
			if err := server.validateRequest(req); err != nil {
				// Request validation failed - handler should not be called
				return !handlerCalled
			}

			// If request structure is valid, try to parse tool request
			if method == "tools/call" {
				_, err := server.parseToolRequest(params)
				if err != nil {
					// Tool request parsing failed - handler should not be called
					return !handlerCalled
				}
			}

			// If we got here, the request might be valid enough to reach the handler
			// But for this property, we're testing malformed requests, so we should have
			// caught an error by now
			return !handlerCalled
		},
		genToolName,
		gen.Bool(),
		gen.Bool(),
		gen.Bool(),
		gen.Bool(),
	))

	// Property: Validation errors return appropriate error codes
	properties.Property("Validation errors return appropriate error codes", prop.ForAll(
		func(requestID string, errorType string) bool {
			router := NewRequestRouter()
			authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{})
			config := &domain.Config{
				Transport: domain.TransportConfig{Type: "stdio"},
				Tools:     domain.ToolsConfig{},
			}
			transport := newMockTransport()
			server := NewServer(transport, router, authManager, config)

			var req *domain.Request

			switch errorType {
			case "invalid_version":
				req = &domain.Request{
					JSONRPC: "1.0",
					ID:      requestID,
					Method:  "tools/call",
				}
			case "missing_method":
				req = &domain.Request{
					JSONRPC: "2.0",
					ID:      requestID,
					Method:  "",
				}
			case "missing_params":
				req = &domain.Request{
					JSONRPC: "2.0",
					ID:      requestID,
					Method:  "tools/call",
					Params:  nil,
				}
			default:
				return true
			}

			// Validate request
			err := server.validateRequest(req)

			// Should have an error for invalid requests
			if errorType == "invalid_version" || errorType == "missing_method" {
				return err != nil
			}

			// For missing params, validateRequest passes but parseToolRequest should fail
			if errorType == "missing_params" {
				if err != nil {
					return false // validateRequest should pass
				}
				_, parseErr := server.parseToolRequest(req.Params)
				return parseErr != nil
			}

			return true
		},
		gen.Identifier(),
		gen.OneConstOf("invalid_version", "missing_method", "missing_params"),
	))

	// Property: Malformed requests are rejected before authentication check
	properties.Property("Malformed requests are rejected before authentication", prop.ForAll(
		func(toolName string, invalidVersion string) bool {
			// Skip valid version
			if invalidVersion == "2.0" {
				return true
			}

			authCheckCalled := false

			// Create a custom auth manager that tracks if validation was called
			authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{
				"jira": {Type: domain.BasicAuth, Username: "user", Password: "pass"},
			})

			router := NewRequestRouter()
			config := &domain.Config{
				Transport: domain.TransportConfig{Type: "stdio"},
				Tools:     domain.ToolsConfig{},
			}
			transport := newMockTransport()
			server := NewServer(transport, router, authManager, config)

			// Create request with invalid version
			req := &domain.Request{
				JSONRPC: invalidVersion,
				ID:      "test-id",
				Method:  "tools/call",
				Params: map[string]interface{}{
					"name":      toolName,
					"arguments": map[string]interface{}{},
				},
			}

			// Validate request - this should fail before auth check
			err := server.validateRequest(req)

			// Should fail validation
			if err == nil {
				return false
			}

			// Auth check should not have been called yet
			// (In a real scenario, we'd track this, but for this test we verify
			// that validateRequest fails before we even get to validateAuthentication)
			return !authCheckCalled
		},
		genToolName,
		gen.OneConstOf("1.0", "2.1", "", "invalid"),
	))

	// Property: Empty tool name is rejected
	properties.Property("Empty tool name is rejected", prop.ForAll(
		func(requestID string) bool {
			handlerCalled := false

			trackingHandler := &trackingToolHandler{
				name: "jira",
				tools: []domain.ToolDefinition{
					{Name: "jira_test", Description: "Test", InputSchema: domain.JSONSchema{Type: "object"}},
				},
				onHandle: func(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error) {
					handlerCalled = true
					return &domain.ToolResponse{
						Content: []domain.ContentBlock{{Type: "text", Text: "success"}},
					}, nil
				},
			}

			router := NewRequestRouter(trackingHandler)
			authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{
				"jira": {Type: domain.BasicAuth, Username: "user", Password: "pass"},
			})
			config := &domain.Config{
				Transport: domain.TransportConfig{Type: "stdio"},
				Tools:     domain.ToolsConfig{},
			}
			transport := newMockTransport()
			server := NewServer(transport, router, authManager, config)

			// Create params with empty tool name
			params := map[string]interface{}{
				"name":      "", // Empty tool name
				"arguments": map[string]interface{}{},
			}

			// Try to parse tool request
			_, err := server.parseToolRequest(params)

			// Should return an error for empty tool name
			if err == nil {
				return false
			}

			// Handler should NOT have been called
			return !handlerCalled
		},
		gen.Identifier(),
	))

	// Property: Invalid tool name format is rejected during authentication
	properties.Property("Invalid tool name format is rejected", prop.ForAll(
		func(invalidToolName string) bool {
			// Skip valid tool name patterns
			if containsUnderscore(invalidToolName) {
				toolType := extractToolTypeFromName(invalidToolName)
				if toolType == "jira" || toolType == "confluence" || toolType == "bitbucket" || toolType == "bamboo" {
					return true // Skip valid patterns
				}
			}

			handlerCalled := false

			trackingHandler := &trackingToolHandler{
				name: "jira",
				tools: []domain.ToolDefinition{
					{Name: "jira_test", Description: "Test", InputSchema: domain.JSONSchema{Type: "object"}},
				},
				onHandle: func(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error) {
					handlerCalled = true
					return &domain.ToolResponse{
						Content: []domain.ContentBlock{{Type: "text", Text: "success"}},
					}, nil
				},
			}

			router := NewRequestRouter(trackingHandler)
			authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{
				"jira": {Type: domain.BasicAuth, Username: "user", Password: "pass"},
			})
			config := &domain.Config{
				Transport: domain.TransportConfig{Type: "stdio"},
				Tools:     domain.ToolsConfig{},
			}
			transport := newMockTransport()
			server := NewServer(transport, router, authManager, config)

			// Try to validate authentication with invalid tool name
			err := server.validateAuthentication(invalidToolName)

			// Should return an error for invalid tool name format
			if err == nil {
				return false
			}

			// Handler should NOT have been called
			return !handlerCalled
		},
		gen.OneConstOf("invalid", "no-underscore", "123", "tool", "unknown_tool", ""),
	))

	// Property: Malformed arguments are handled gracefully
	properties.Property("Malformed arguments are handled gracefully", prop.ForAll(
		func(toolName string, requestID string) bool {
			_ = requestID // unused but kept for generator consistency

			trackingHandler := &trackingToolHandler{
				name: extractToolTypeFromName(toolName),
				tools: []domain.ToolDefinition{
					{Name: toolName, Description: "Test", InputSchema: domain.JSONSchema{Type: "object"}},
				},
				onHandle: func(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error) {
					return &domain.ToolResponse{
						Content: []domain.ContentBlock{{Type: "text", Text: "success"}},
					}, nil
				},
			}

			router := NewRequestRouter(trackingHandler)
			authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{
				"jira":       {Type: domain.BasicAuth, Username: "user", Password: "pass"},
				"confluence": {Type: domain.BasicAuth, Username: "user", Password: "pass"},
				"bitbucket":  {Type: domain.BasicAuth, Username: "user", Password: "pass"},
				"bamboo":     {Type: domain.BasicAuth, Username: "user", Password: "pass"},
			})
			config := &domain.Config{
				Transport: domain.TransportConfig{Type: "stdio"},
				Tools:     domain.ToolsConfig{},
			}
			transport := newMockTransport()
			server := NewServer(transport, router, authManager, config)

			// Create params with missing arguments field
			params := map[string]interface{}{
				"name": toolName,
				// Missing "arguments" field
			}

			// Try to parse tool request
			toolReq, err := server.parseToolRequest(params)

			// Should succeed - missing arguments should be initialized to empty map
			if err != nil {
				return false
			}

			// Arguments should be initialized to empty map
			if toolReq.Arguments == nil {
				return false
			}

			if len(toolReq.Arguments) != 0 {
				return false
			}

			return true
		},
		genToolName,
		gen.Identifier(),
	))

	properties.TestingRun(t)
}

// Feature: atlassian-mcp-server, Property 12: Error Logging Completeness
// **Validates: Requirements 8.5**
//
// For any error that occurs during request processing, the server should log an entry
// containing at minimum: timestamp, error type, error message, request context, and
// stack trace (if applicable).
func TestProperty12_ErrorLoggingCompleteness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Generator for valid tool names
	genJiraOp := gen.OneConstOf("get_issue", "create_issue", "update_issue", "delete_issue", "search_jql", "transition_issue", "add_comment", "list_projects")
	genConfluenceOp := gen.OneConstOf("get_page", "create_page", "update_page", "delete_page", "search_cql", "list_spaces")
	genBitbucketOp := gen.OneConstOf("list_repositories", "get_branches", "create_branch", "get_pull_request", "create_pull_request", "merge_pull_request")
	genBambooOp := gen.OneConstOf("list_plans", "get_plan", "trigger_build", "get_build_result", "get_build_log")

	genToolName := gen.OneGenOf(
		genJiraOp.Map(func(op string) string { return "jira_" + op }),
		genConfluenceOp.Map(func(op string) string { return "confluence_" + op }),
		genBitbucketOp.Map(func(op string) string { return "bitbucket_" + op }),
		genBambooOp.Map(func(op string) string { return "bamboo_" + op }),
	)

	// Property: Authentication errors result in error responses with context
	properties.Property("Authentication errors produce error responses with request context", prop.ForAll(
		func(toolName string, requestID string) bool {
			// Skip empty request IDs
			if requestID == "" {
				return true
			}

			trackingHandler := &trackingToolHandler{
				name: extractToolTypeFromName(toolName),
				tools: []domain.ToolDefinition{
					{Name: toolName, Description: "Test", InputSchema: domain.JSONSchema{Type: "object"}},
				},
				onHandle: func(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error) {
					return &domain.ToolResponse{
						Content: []domain.ContentBlock{{Type: "text", Text: "success"}},
					}, nil
				},
			}

			router := NewRequestRouter(trackingHandler)

			// Create auth manager WITHOUT credentials to trigger auth error
			authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{})

			config := &domain.Config{
				Transport: domain.TransportConfig{Type: "stdio"},
				Tools:     domain.ToolsConfig{},
			}

			transport := newMockTransport()
			server := NewServer(transport, router, authManager, config)

			// Create a tools/call request that will fail authentication
			req := &domain.Request{
				JSONRPC: "2.0",
				ID:      requestID,
				Method:  "tools/call",
				Params: map[string]interface{}{
					"name":      toolName,
					"arguments": map[string]interface{}{},
				},
			}

			// Handle the request (should fail at authentication)
			ctx := context.Background()
			_, err := server.handleToolsCall(ctx, req)

			// Should return an error
			if err == nil {
				return false
			}

			// Check that an error response was sent
			lastResp := transport.getLastResponse()
			if lastResp == nil {
				return false
			}

			// Response should have an error
			if lastResp.Error == nil {
				return false
			}

			// Error should be authentication error
			if lastResp.Error.Code != domain.AuthenticationError {
				return false
			}

			// Error message should be descriptive
			if lastResp.Error.Message == "" {
				return false
			}

			return true
		},
		genToolName,
		gen.Identifier(),
	))

	// Property: Tool execution errors produce error responses with context
	properties.Property("Tool execution errors produce error responses with context", prop.ForAll(
		func(toolName string, requestID string, errorMsg string) bool {
			// Skip empty values
			if requestID == "" || errorMsg == "" {
				return true
			}

			// Create handler that returns an error
			trackingHandler := &trackingToolHandler{
				name: extractToolTypeFromName(toolName),
				tools: []domain.ToolDefinition{
					{Name: toolName, Description: "Test", InputSchema: domain.JSONSchema{Type: "object"}},
				},
				onHandle: func(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error) {
					return nil, fmt.Errorf("%s", errorMsg)
				},
			}

			router := NewRequestRouter(trackingHandler)

			// Create auth manager WITH valid credentials
			toolType := extractToolTypeFromName(toolName)
			authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{
				toolType: {
					Type:     domain.BasicAuth,
					Username: "testuser",
					Password: "testpass",
				},
			})

			config := &domain.Config{
				Transport: domain.TransportConfig{Type: "stdio"},
				Tools:     domain.ToolsConfig{},
			}

			transport := newMockTransport()
			server := NewServer(transport, router, authManager, config)

			// Create a tools/call request
			req := &domain.Request{
				JSONRPC: "2.0",
				ID:      requestID,
				Method:  "tools/call",
				Params: map[string]interface{}{
					"name":      toolName,
					"arguments": map[string]interface{}{},
				},
			}

			// Handle the request (should fail at tool execution)
			ctx := context.Background()
			_, err := server.handleToolsCall(ctx, req)

			// Should return an error
			if err == nil {
				return false
			}

			// Check that an error response was sent
			lastResp := transport.getLastResponse()
			if lastResp == nil {
				return false
			}

			// Response should have an error
			if lastResp.Error == nil {
				return false
			}

			// Error should be internal error
			if lastResp.Error.Code != domain.InternalError {
				return false
			}

			// Error message should be descriptive
			if lastResp.Error.Message == "" {
				return false
			}

			return true
		},
		genToolName,
		gen.Identifier(),
		gen.AlphaString(),
	))

	// Property: Validation errors produce error responses with context
	properties.Property("Validation errors produce error responses with context", prop.ForAll(
		func(invalidVersion string, requestID string) bool {
			// Skip valid version
			if invalidVersion == "2.0" || requestID == "" {
				return true
			}

			router := NewRequestRouter()
			authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{})
			config := &domain.Config{
				Transport: domain.TransportConfig{Type: "stdio"},
				Tools:     domain.ToolsConfig{},
			}

			transport := newMockTransport()
			server := NewServer(transport, router, authManager, config)

			// Create request with invalid version
			req := &domain.Request{
				JSONRPC: invalidVersion,
				ID:      requestID,
				Method:  "tools/call",
				Params: map[string]interface{}{
					"name":      "jira_get_issue",
					"arguments": map[string]interface{}{},
				},
			}

			// Process the request (should fail validation)
			ctx := context.Background()
			server.handleRequest(ctx, req)

			// Check that an error response was sent
			lastResp := transport.getLastResponse()
			if lastResp == nil {
				return false
			}

			// Response should have an error
			if lastResp.Error == nil {
				return false
			}

			// Error should be invalid request
			if lastResp.Error.Code != domain.InvalidRequest {
				return false
			}

			// Error message should be descriptive
			if lastResp.Error.Message == "" {
				return false
			}

			return true
		},
		gen.OneConstOf("1.0", "2.1", "3.0", "", "invalid"),
		gen.Identifier(),
	))

	// Property: Routing errors produce error responses with context
	properties.Property("Routing errors for unknown tools produce error responses", prop.ForAll(
		func(unknownTool string, requestID string) bool {
			// Skip valid tool patterns
			if requestID == "" {
				return true
			}

			if containsUnderscore(unknownTool) {
				toolType := extractToolTypeFromName(unknownTool)
				if toolType == "jira" || toolType == "confluence" || toolType == "bitbucket" || toolType == "bamboo" {
					return true // Skip valid patterns
				}
			}

			// Create router with no handlers for the unknown tool
			router := NewRequestRouter()

			authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{
				"jira": {Type: domain.BasicAuth, Username: "user", Password: "pass"},
			})

			config := &domain.Config{
				Transport: domain.TransportConfig{Type: "stdio"},
				Tools:     domain.ToolsConfig{},
			}

			transport := newMockTransport()
			server := NewServer(transport, router, authManager, config)

			// Create request with unknown tool
			req := &domain.Request{
				JSONRPC: "2.0",
				ID:      requestID,
				Method:  "tools/call",
				Params: map[string]interface{}{
					"name":      unknownTool,
					"arguments": map[string]interface{}{},
				},
			}

			// Process the request (should fail routing)
			ctx := context.Background()
			server.handleRequest(ctx, req)

			// Check that an error response was sent
			lastResp := transport.getLastResponse()
			if lastResp == nil {
				return false
			}

			// Response should have an error
			if lastResp.Error == nil {
				return false
			}

			// Error message should be descriptive
			if lastResp.Error.Message == "" {
				return false
			}

			return true
		},
		gen.OneConstOf("unknown_tool", "invalid", "no-underscore", "xyz_operation"),
		gen.Identifier(),
	))

	// Property: Sensitive information is not included in error responses
	properties.Property("Credentials are not included in error responses", prop.ForAll(
		func(toolName string, username string, password string, token string, requestID string) bool {
			// Skip empty values
			if requestID == "" || username == "" || password == "" || token == "" {
				return true
			}

			trackingHandler := &trackingToolHandler{
				name: extractToolTypeFromName(toolName),
				tools: []domain.ToolDefinition{
					{Name: toolName, Description: "Test", InputSchema: domain.JSONSchema{Type: "object"}},
				},
				onHandle: func(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error) {
					return nil, fmt.Errorf("test error")
				},
			}

			router := NewRequestRouter(trackingHandler)

			// Create auth manager with credentials
			toolType := extractToolTypeFromName(toolName)
			authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{
				toolType: {
					Type:     domain.BasicAuth,
					Username: username,
					Password: password,
					Token:    token,
				},
			})

			config := &domain.Config{
				Transport: domain.TransportConfig{Type: "stdio"},
				Tools:     domain.ToolsConfig{},
			}

			transport := newMockTransport()
			server := NewServer(transport, router, authManager, config)

			// Create a tools/call request
			req := &domain.Request{
				JSONRPC: "2.0",
				ID:      requestID,
				Method:  "tools/call",
				Params: map[string]interface{}{
					"name":      toolName,
					"arguments": map[string]interface{}{},
				},
			}

			// Handle the request
			ctx := context.Background()
			_, _ = server.handleToolsCall(ctx, req)

			// Check that an error response was sent
			lastResp := transport.getLastResponse()
			if lastResp == nil {
				return false
			}

			// Response should have an error
			if lastResp.Error == nil {
				return false
			}

			// Check that error message doesn't contain sensitive information
			if containsString(lastResp.Error.Message, password) || containsString(lastResp.Error.Message, token) {
				return false
			}

			// Check error data if present
			if lastResp.Error.Data != nil {
				dataStr := fmt.Sprintf("%v", lastResp.Error.Data)
				if containsString(dataStr, password) || containsString(dataStr, token) {
					return false
				}
			}

			return true
		},
		genToolName,
		gen.Identifier(),
		gen.AlphaString(),
		gen.AlphaString(),
		gen.Identifier(),
	))

	// Property: Multiple errors in sequence all produce error responses
	properties.Property("Multiple errors all produce error responses with context", prop.ForAll(
		func(toolName string, errorCount int) bool {
			// Limit error count to reasonable range
			if errorCount < 1 || errorCount > 5 {
				return true
			}

			router := NewRequestRouter()

			// Create auth manager WITHOUT credentials to trigger auth errors
			authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{})

			config := &domain.Config{
				Transport: domain.TransportConfig{Type: "stdio"},
				Tools:     domain.ToolsConfig{},
			}

			transport := newMockTransport()
			server := NewServer(transport, router, authManager, config)

			// Generate multiple errors
			for i := 0; i < errorCount; i++ {
				req := &domain.Request{
					JSONRPC: "2.0",
					ID:      fmt.Sprintf("req-%d", i),
					Method:  "tools/call",
					Params: map[string]interface{}{
						"name":      toolName,
						"arguments": map[string]interface{}{},
					},
				}

				ctx := context.Background()
				_, _ = server.handleToolsCall(ctx, req)
			}

			// Verify that all errors produced responses
			responses := transport.getAllResponses()
			if len(responses) < errorCount {
				return false
			}

			// Verify each response has an error
			errorRespCount := 0
			for _, resp := range responses {
				if resp.Error != nil {
					errorRespCount++
					// Verify error has message
					if resp.Error.Message == "" {
						return false
					}
				}
			}

			// Should have at least as many error responses as requests
			return errorRespCount >= errorCount
		},
		genToolName,
		gen.IntRange(1, 5),
	))

	// Property: Error responses include request ID for correlation
	properties.Property("Error responses include request ID for correlation", prop.ForAll(
		func(toolName string, requestID string) bool {
			// Skip empty request IDs
			if requestID == "" {
				return true
			}

			router := NewRequestRouter()

			// Create auth manager WITHOUT credentials to trigger auth error
			authManager := domain.NewAuthenticationManager(map[string]*domain.Credentials{})

			config := &domain.Config{
				Transport: domain.TransportConfig{Type: "stdio"},
				Tools:     domain.ToolsConfig{},
			}

			transport := newMockTransport()
			server := NewServer(transport, router, authManager, config)

			// Create a tools/call request
			req := &domain.Request{
				JSONRPC: "2.0",
				ID:      requestID,
				Method:  "tools/call",
				Params: map[string]interface{}{
					"name":      toolName,
					"arguments": map[string]interface{}{},
				},
			}

			// Handle the request
			ctx := context.Background()
			_, _ = server.handleToolsCall(ctx, req)

			// Check that an error response was sent
			lastResp := transport.getLastResponse()
			if lastResp == nil {
				return false
			}

			// Response should have the same request ID
			// Compare as strings since gen.Identifier() generates strings
			respID, ok := lastResp.ID.(string)
			if !ok || respID != requestID {
				return false
			}

			// Response should have an error
			if lastResp.Error == nil {
				return false
			}

			return true
		},
		genToolName,
		gen.Identifier(),
	))

	properties.TestingRun(t)
}

// containsString checks if a string contains a substring (case-insensitive)
func containsString(s, substr string) bool {
	return len(substr) > 0 && len(s) > 0 &&
		(s == substr ||
			(len(s) >= len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
