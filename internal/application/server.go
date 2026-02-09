package application

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"atlassian-mcp-server/internal/domain"
)

// Server is the main MCP server implementation.
// It orchestrates the transport layer, request routing, authentication,
// and implements the MCP protocol methods.
type Server struct {
	transport   domain.Transport
	router      *RequestRouter
	authManager *domain.AuthenticationManager
	config      *domain.Config
	logger      *StructuredLogger
}

// NewServer creates a new MCP server instance.
// It requires a transport, router, authentication manager, and configuration.
func NewServer(
	transport domain.Transport,
	router *RequestRouter,
	authManager *domain.AuthenticationManager,
	config *domain.Config,
) *Server {
	return &Server{
		transport:   transport,
		router:      router,
		authManager: authManager,
		config:      config,
		logger:      NewStructuredLogger(),
	}
}

// Start begins the server operation.
// It starts the transport layer and begins processing incoming requests.
func (s *Server) Start(ctx context.Context) error {
	// Start the transport layer
	if err := s.transport.Start(ctx); err != nil {
		s.logger.LogError("failed to start transport", err, map[string]interface{}{
			"transport_type": s.config.Transport.Type,
		})
		return fmt.Errorf("failed to start transport: %w", err)
	}

	s.logger.LogInfo("server started", map[string]interface{}{
		"transport_type": s.config.Transport.Type,
	})

	// Start processing requests
	go s.processRequests(ctx)

	return nil
}

// processRequests continuously processes incoming JSON-RPC requests.
func (s *Server) processRequests(ctx context.Context) {
	reqChan := s.transport.Receive()

	for {
		select {
		case <-ctx.Done():
			s.logger.LogInfo("server shutting down", nil)
			return
		case req, ok := <-reqChan:
			if !ok {
				// Channel closed, transport is shutting down
				return
			}

			// Process the request
			s.handleRequest(ctx, req)
		}
	}
}

// handleRequest processes a single JSON-RPC request.
func (s *Server) handleRequest(ctx context.Context, req *domain.Request) {
	// Log the incoming request
	s.logger.LogInfo("received request", map[string]interface{}{
		"method":     req.Method,
		"request_id": req.ID,
	})

	// Validate request structure
	if err := s.validateRequest(req); err != nil {
		s.sendErrorResponse(req.ID, domain.InvalidRequest, "Invalid Request", err.Error())
		return
	}

	// Route to appropriate handler based on method
	var response *domain.Response
	var err error

	switch req.Method {
	case "initialize":
		response, err = s.handleInitialize(req)
	case "tools/list":
		response, err = s.handleToolsList(req)
	case "tools/call":
		response, err = s.handleToolsCall(ctx, req)
	default:
		s.sendErrorResponse(req.ID, domain.MethodNotFound, "Method not found", fmt.Sprintf("unknown method: %s", req.Method))
		return
	}

	if err != nil {
		s.logger.LogError("request processing failed", err, map[string]interface{}{
			"method":     req.Method,
			"request_id": req.ID,
		})
		// Error response already sent by handler
		return
	}

	// Send the response
	if err := s.transport.Send(response); err != nil {
		s.logger.LogError("failed to send response", err, map[string]interface{}{
			"request_id": req.ID,
		})
	}
}

// validateRequest validates the basic structure of a JSON-RPC request.
func (s *Server) validateRequest(req *domain.Request) error {
	if req.JSONRPC != "2.0" {
		return fmt.Errorf("invalid jsonrpc version: %s", req.JSONRPC)
	}

	if req.Method == "" {
		return fmt.Errorf("method is required")
	}

	return nil
}

// handleInitialize handles the MCP initialize method.
// This is the initial handshake between client and server.
func (s *Server) handleInitialize(req *domain.Request) (*domain.Response, error) {
	// Parse initialize params (if any)
	// For now, we accept any params and return server capabilities

	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "atlassian-mcp-server",
			"version": "1.0.0",
		},
	}

	return &domain.Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}, nil
}

// handleToolsList handles the MCP tools/list method.
// Returns all available tools from registered handlers.
func (s *Server) handleToolsList(req *domain.Request) (*domain.Response, error) {
	// Get all tools from the router
	tools := s.router.ListAllTools()

	result := map[string]interface{}{
		"tools": tools,
	}

	return &domain.Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}, nil
}

// handleToolsCall handles the MCP tools/call method.
// Executes a tool call by routing it to the appropriate handler.
func (s *Server) handleToolsCall(ctx context.Context, req *domain.Request) (*domain.Response, error) {
	// Parse the tool request from params
	toolReq, err := s.parseToolRequest(req.Params)
	if err != nil {
		s.sendErrorResponse(req.ID, domain.InvalidParams, "Invalid params", err.Error())
		return nil, err
	}

	// Route the request to the appropriate handler
	// Authentication is now handled at the handler level
	toolResp, err := s.router.Route(ctx, toolReq)
	if err != nil {
		s.logger.LogError("tool execution failed", err, map[string]interface{}{
			"tool":       toolReq.Name,
			"request_id": req.ID,
		})

		// Map the error to an appropriate JSON-RPC error
		s.sendMappedError(req.ID, err)
		return nil, err
	}

	// Return successful response
	return &domain.Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  toolResp,
	}, nil
}

// parseToolRequest parses the params field into a ToolRequest.
func (s *Server) parseToolRequest(params interface{}) (*domain.ToolRequest, error) {
	if params == nil {
		return nil, fmt.Errorf("params is required for tools/call")
	}

	// Convert params to JSON and back to ToolRequest
	// This handles both map[string]interface{} and already-parsed structs
	jsonData, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}

	var toolReq domain.ToolRequest
	if err := json.Unmarshal(jsonData, &toolReq); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tool request: %w", err)
	}

	// Validate required fields
	if toolReq.Name == "" {
		return nil, fmt.Errorf("tool name is required")
	}

	if toolReq.Arguments == nil {
		toolReq.Arguments = make(map[string]interface{})
	}

	return &toolReq, nil
}

// validateAuthentication validates that authentication is configured for the tool.
// This ensures unauthenticated requests never reach external systems.
func (s *Server) validateAuthentication(toolName string) error {
	// Extract the tool type from the tool name (e.g., "jira_get_issue" -> "jira")
	toolType := extractToolType(toolName)
	if toolType == "" {
		return fmt.Errorf("invalid tool name format: %s", toolName)
	}

	// Validate credentials for the tool
	if err := s.authManager.ValidateCredentials(toolType); err != nil {
		return fmt.Errorf("authentication validation failed for %s: %w", toolType, err)
	}

	return nil
}

// extractToolType extracts the tool type from a tool name.
// Tool names follow the pattern: <tool>_<operation> (e.g., "jira_get_issue" -> "jira")
func extractToolType(toolName string) string {
	// Find the first underscore
	for i, c := range toolName {
		if c == '_' {
			return toolName[:i]
		}
	}
	return ""
}

// sendErrorResponse sends a JSON-RPC error response.
func (s *Server) sendErrorResponse(id interface{}, code int, message string, data interface{}) {
	response := &domain.Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &domain.Error{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}

	if err := s.transport.Send(response); err != nil {
		s.logger.LogError("failed to send error response", err, map[string]interface{}{
			"request_id":    id,
			"error_code":    code,
			"error_message": message,
		})
	}
}

// sendMappedError maps an error to an appropriate JSON-RPC error and sends it.
func (s *Server) sendMappedError(id interface{}, err error) {
	// Default to internal error
	code := domain.InternalError
	message := "Internal error"
	data := err.Error()

	// Try to map specific error types
	errorStr := err.Error()
	if containsSubstring(errorStr, "unknown tool") || containsSubstring(errorStr, "no handler registered") {
		code = domain.MethodNotFound
		message = "Tool not found"
	} else if containsSubstring(errorStr, "authentication") || containsSubstring(errorStr, "credentials") {
		code = domain.AuthenticationError
		message = "Authentication failed"
	} else if containsSubstring(errorStr, "invalid") || containsSubstring(errorStr, "required") {
		code = domain.InvalidParams
		message = "Invalid parameters"
	} else if containsSubstring(errorStr, "network") || containsSubstring(errorStr, "connection") {
		code = domain.NetworkError
		message = "Network error"
	} else if containsSubstring(errorStr, "rate limit") {
		code = domain.RateLimitError
		message = "Rate limit exceeded"
	}

	s.sendErrorResponse(id, code, message, data)
}

// containsSubstring checks if a string contains a substring.
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Close gracefully shuts down the server.
func (s *Server) Close() error {
	s.logger.LogInfo("closing server", nil)
	return s.transport.Close()
}

// StructuredLogger provides structured logging with context.
type StructuredLogger struct {
	logger *log.Logger
}

// NewStructuredLogger creates a new structured logger.
func NewStructuredLogger() *StructuredLogger {
	return &StructuredLogger{
		logger: log.Default(),
	}
}

// LogInfo logs an informational message with context.
func (l *StructuredLogger) LogInfo(message string, context map[string]interface{}) {
	entry := l.buildLogEntry("INFO", message, nil, context)
	l.logger.Println(entry)
}

// LogError logs an error message with context.
func (l *StructuredLogger) LogError(message string, err error, context map[string]interface{}) {
	entry := l.buildLogEntry("ERROR", message, err, context)
	l.logger.Println(entry)
}

// buildLogEntry constructs a structured log entry.
func (l *StructuredLogger) buildLogEntry(level, message string, err error, context map[string]interface{}) string {
	entry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"level":     level,
		"message":   message,
	}

	if err != nil {
		entry["error"] = err.Error()
	}

	for k, v := range context {
		entry[k] = v
	}

	// Convert to JSON
	jsonData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Sprintf(`{"level":"ERROR","message":"failed to marshal log entry","error":"%s"}`, err.Error())
	}

	return string(jsonData)
}
