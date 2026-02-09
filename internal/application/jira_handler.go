package application

import (
	"context"
	"fmt"

	"atlassian-mcp-server/internal/domain"
	"atlassian-mcp-server/internal/infrastructure"
)

// JiraHandler implements ToolHandler for Jira operations.
// It routes MCP tool calls to the appropriate JiraClient methods and
// transforms responses using the ResponseMapper.
type JiraHandler struct {
	client      *infrastructure.JiraClient
	mapper      domain.ResponseMapper
	authManager *domain.AuthenticationManager
	baseURL     string
}

// NewJiraHandler creates a new JiraHandler instance.
func NewJiraHandler(client *infrastructure.JiraClient, mapper domain.ResponseMapper, authManager *domain.AuthenticationManager, baseURL string) *JiraHandler {
	return &JiraHandler{
		client:      client,
		mapper:      mapper,
		authManager: authManager,
		baseURL:     baseURL,
	}
}

// Tool name constants for Jira operations
const (
	ToolJiraGetIssue     = "jira_get_issue"
	ToolJiraCreateIssue  = "jira_create_issue"
	ToolJiraUpdateIssue  = "jira_update_issue"
	ToolJiraDeleteIssue  = "jira_delete_issue"
	ToolJiraSearchJQL    = "jira_search_jql"
	ToolJiraTransition   = "jira_transition_issue"
	ToolJiraAddComment   = "jira_add_comment"
	ToolJiraListProjects = "jira_list_projects"
)

// ToolName returns the identifier for this handler.
func (h *JiraHandler) ToolName() string {
	return "jira"
}

// getAuthSchema returns the schema for optional authentication parameters.
// This can be included in any tool's input schema to allow client-provided credentials.
func getAuthSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"description": "Optional authentication credentials (if not provided, uses server config)",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":        "string",
				"description": "Authentication type: 'basic' or 'token'",
				"enum":        []string{"basic", "token"},
			},
			"username": map[string]interface{}{
				"type":        "string",
				"description": "Username for basic authentication",
			},
			"password": map[string]interface{}{
				"type":        "string",
				"description": "Password for basic authentication",
			},
			"token": map[string]interface{}{
				"type":        "string",
				"description": "Token for token authentication",
			},
		},
	}
}

// ListTools returns available tools for Jira operations.
func (h *JiraHandler) ListTools() []domain.ToolDefinition {
	return []domain.ToolDefinition{
		{
			Name:        ToolJiraGetIssue,
			Description: "Retrieve a Jira issue by its key (e.g., TEST-123)",
			InputSchema: domain.JSONSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"issueKey": map[string]interface{}{
						"type":        "string",
						"description": "The issue key (e.g., TEST-123)",
					},
					"auth": getAuthSchema(),
				},
				Required: []string{"issueKey"},
			},
		},
		{
			Name:        ToolJiraCreateIssue,
			Description: "Create a new Jira issue",
			InputSchema: domain.JSONSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"projectKey": map[string]interface{}{
						"type":        "string",
						"description": "The project key (e.g., TEST)",
					},
					"summary": map[string]interface{}{
						"type":        "string",
						"description": "The issue summary/title",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "The issue description (optional)",
					},
					"issueType": map[string]interface{}{
						"type":        "string",
						"description": "The issue type name (e.g., Bug, Story, Task)",
					},
					"assignee": map[string]interface{}{
						"type":        "string",
						"description": "The assignee username (optional)",
					},
				},
				Required: []string{"projectKey", "summary", "issueType"},
			},
		},
		{
			Name:        ToolJiraUpdateIssue,
			Description: "Update an existing Jira issue",
			InputSchema: domain.JSONSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"issueKey": map[string]interface{}{
						"type":        "string",
						"description": "The issue key (e.g., TEST-123)",
					},
					"summary": map[string]interface{}{
						"type":        "string",
						"description": "The new summary/title (optional)",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "The new description (optional)",
					},
					"assignee": map[string]interface{}{
						"type":        "string",
						"description": "The new assignee username (optional)",
					},
				},
				Required: []string{"issueKey"},
			},
		},
		{
			Name:        ToolJiraDeleteIssue,
			Description: "Delete a Jira issue",
			InputSchema: domain.JSONSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"issueKey": map[string]interface{}{
						"type":        "string",
						"description": "The issue key (e.g., TEST-123)",
					},
				},
				Required: []string{"issueKey"},
			},
		},
		{
			Name:        ToolJiraSearchJQL,
			Description: "Search for Jira issues using JQL (Jira Query Language)",
			InputSchema: domain.JSONSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"jql": map[string]interface{}{
						"type":        "string",
						"description": "The JQL query string",
					},
					"startAt": map[string]interface{}{
						"type":        "integer",
						"description": "The index of the first issue to return (0-based, optional)",
					},
					"maxResults": map[string]interface{}{
						"type":        "integer",
						"description": "The maximum number of issues to return (optional)",
					},
				},
				Required: []string{"jql"},
			},
		},
		{
			Name:        ToolJiraTransition,
			Description: "Transition a Jira issue to a new status",
			InputSchema: domain.JSONSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"issueKey": map[string]interface{}{
						"type":        "string",
						"description": "The issue key (e.g., TEST-123)",
					},
					"transitionId": map[string]interface{}{
						"type":        "string",
						"description": "The transition ID (optional if transitionName is provided)",
					},
					"transitionName": map[string]interface{}{
						"type":        "string",
						"description": "The transition name (optional if transitionId is provided)",
					},
				},
				Required: []string{"issueKey"},
			},
		},
		{
			Name:        ToolJiraAddComment,
			Description: "Add a comment to a Jira issue",
			InputSchema: domain.JSONSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"issueKey": map[string]interface{}{
						"type":        "string",
						"description": "The issue key (e.g., TEST-123)",
					},
					"body": map[string]interface{}{
						"type":        "string",
						"description": "The comment text",
					},
				},
				Required: []string{"issueKey", "body"},
			},
		},
		{
			Name:        ToolJiraListProjects,
			Description: "List all Jira projects accessible to the authenticated user",
			InputSchema: domain.JSONSchema{
				Type:       "object",
				Properties: map[string]interface{}{},
				Required:   []string{},
			},
		},
	}
}

// getClientForRequest returns the appropriate Jira client based on provided credentials.
// If credentials are provided in args, creates a new client with those credentials.
// Otherwise, returns the default client configured from the config file.
// Returns an error if no credentials are provided and no default client is configured.
func (h *JiraHandler) getClientForRequest(args map[string]interface{}) (*infrastructure.JiraClient, error) {
	// Try to extract credentials from arguments
	creds, err := domain.ExtractCredentialsFromArguments(args)
	if err != nil {
		return nil, &domain.Error{
			Code:    domain.InvalidParams,
			Message: fmt.Sprintf("invalid credentials: %v", err),
		}
	}

	// If credentials provided, create a new client with those credentials
	if creds != nil {
		httpClient, err := h.authManager.GetAuthenticatedClientWithCredentials(creds)
		if err != nil {
			return nil, &domain.Error{
				Code:    domain.AuthenticationError,
				Message: fmt.Sprintf("failed to create authenticated client: %v", err),
			}
		}
		return infrastructure.NewJiraClient(h.baseURL, httpClient), nil
	}

	// No credentials provided - check if we have a default client
	if h.client == nil {
		return nil, &domain.Error{
			Code:    domain.AuthenticationError,
			Message: "authentication required: no credentials provided and no default credentials configured",
		}
	}

	// Use the default client
	return h.client, nil
}

// Handle processes an MCP tool call request for Jira operations.
func (h *JiraHandler) Handle(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error) {
	// Validate that we have arguments
	if req.Arguments == nil {
		req.Arguments = make(map[string]interface{})
	}

	// Route to the appropriate handler based on tool name
	switch req.Name {
	case ToolJiraGetIssue:
		return h.handleGetIssue(ctx, req.Arguments)
	case ToolJiraCreateIssue:
		return h.handleCreateIssue(ctx, req.Arguments)
	case ToolJiraUpdateIssue:
		return h.handleUpdateIssue(ctx, req.Arguments)
	case ToolJiraDeleteIssue:
		return h.handleDeleteIssue(ctx, req.Arguments)
	case ToolJiraSearchJQL:
		return h.handleSearchJQL(ctx, req.Arguments)
	case ToolJiraTransition:
		return h.handleTransition(ctx, req.Arguments)
	case ToolJiraAddComment:
		return h.handleAddComment(ctx, req.Arguments)
	case ToolJiraListProjects:
		return h.handleListProjects(ctx, req.Arguments)
	default:
		return nil, &domain.Error{
			Code:    domain.MethodNotFound,
			Message: fmt.Sprintf("unknown Jira tool: %s", req.Name),
		}
	}
}

// handleGetIssue handles the jira_get_issue tool call.
func (h *JiraHandler) handleGetIssue(ctx context.Context, args map[string]interface{}) (*domain.ToolResponse, error) {
	// Get the appropriate client (with custom credentials if provided)
	client, err := h.getClientForRequest(args)
	if err != nil {
		return nil, err
	}

	// Validate required parameters
	issueKey, err := getStringParam(args, "issueKey", true)
	if err != nil {
		return nil, err
	}

	// Call the Jira client
	issue, err := client.GetIssue(issueKey)
	if err != nil {
		return nil, h.mapper.MapError(err)
	}

	// Transform the response
	return h.mapper.MapToToolResponse(issue)
}

// handleCreateIssue handles the jira_create_issue tool call.
func (h *JiraHandler) handleCreateIssue(ctx context.Context, args map[string]interface{}) (*domain.ToolResponse, error) {
	// Get the appropriate client (with custom credentials if provided)
	client, err := h.getClientForRequest(args)
	if err != nil {
		return nil, err
	}

	// Validate required parameters
	projectKey, err := getStringParam(args, "projectKey", true)
	if err != nil {
		return nil, err
	}
	summary, err := getStringParam(args, "summary", true)
	if err != nil {
		return nil, err
	}
	issueType, err := getStringParam(args, "issueType", true)
	if err != nil {
		return nil, err
	}

	// Optional parameters
	description, _ := getStringParam(args, "description", false)
	assignee, _ := getStringParam(args, "assignee", false)

	// Build the create request
	createReq := &domain.JiraIssueCreate{
		Fields: domain.JiraFieldsCreate{
			Summary:     summary,
			Description: description,
			IssueType: domain.IssueTypeRef{
				Name: issueType,
			},
			Project: domain.ProjectRef{
				Key: projectKey,
			},
		},
	}

	// Add assignee if provided
	if assignee != "" {
		createReq.Fields.Assignee = &domain.UserRef{
			Name: assignee,
		}
	}

	// Call the Jira client
	issue, err := client.CreateIssue(createReq)
	if err != nil {
		return nil, h.mapper.MapError(err)
	}

	// Transform the response
	return h.mapper.MapToToolResponse(issue)
}

// handleUpdateIssue handles the jira_update_issue tool call.
func (h *JiraHandler) handleUpdateIssue(ctx context.Context, args map[string]interface{}) (*domain.ToolResponse, error) {
	// Get the appropriate client (with custom credentials if provided)
	client, err := h.getClientForRequest(args)
	if err != nil {
		return nil, err
	}

	// Validate required parameters
	issueKey, err := getStringParam(args, "issueKey", true)
	if err != nil {
		return nil, err
	}

	// Optional parameters
	summary, _ := getStringParam(args, "summary", false)
	description, _ := getStringParam(args, "description", false)
	assignee, _ := getStringParam(args, "assignee", false)

	// Build the update request
	updateReq := &domain.JiraIssueUpdate{
		Fields: domain.JiraFieldsUpdate{
			Summary:     summary,
			Description: description,
		},
	}

	// Add assignee if provided
	if assignee != "" {
		updateReq.Fields.Assignee = &domain.UserRef{
			Name: assignee,
		}
	}

	// Call the Jira client
	err = client.UpdateIssue(issueKey, updateReq)
	if err != nil {
		return nil, h.mapper.MapError(err)
	}

	// Return success response
	return h.mapper.MapToToolResponse(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Issue %s updated successfully", issueKey),
	})
}

// handleDeleteIssue handles the jira_delete_issue tool call.
func (h *JiraHandler) handleDeleteIssue(ctx context.Context, args map[string]interface{}) (*domain.ToolResponse, error) {
	// Get the appropriate client (with custom credentials if provided)
	client, err := h.getClientForRequest(args)
	if err != nil {
		return nil, err
	}

	// Validate required parameters
	issueKey, err := getStringParam(args, "issueKey", true)
	if err != nil {
		return nil, err
	}

	// Call the Jira client
	err = client.DeleteIssue(issueKey)
	if err != nil {
		return nil, h.mapper.MapError(err)
	}

	// Return success response
	return h.mapper.MapToToolResponse(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Issue %s deleted successfully", issueKey),
	})
}

// handleSearchJQL handles the jira_search_jql tool call.
func (h *JiraHandler) handleSearchJQL(ctx context.Context, args map[string]interface{}) (*domain.ToolResponse, error) {
	// Get the appropriate client (with custom credentials if provided)
	client, err := h.getClientForRequest(args)
	if err != nil {
		return nil, err
	}

	// Validate required parameters
	jql, err := getStringParam(args, "jql", true)
	if err != nil {
		return nil, err
	}

	// Optional parameters
	startAt, err := getIntParam(args, "startAt", false)
	if err != nil {
		return nil, err
	}
	maxResults, err := getIntParam(args, "maxResults", false)
	if err != nil {
		return nil, err
	}

	// Build search options
	options := &infrastructure.SearchOptions{
		JQL:        jql,
		StartAt:    startAt,
		MaxResults: maxResults,
	}

	// Call the Jira client
	results, err := client.SearchJQL(jql, options)
	if err != nil {
		return nil, h.mapper.MapError(err)
	}

	// Transform the response
	return h.mapper.MapToToolResponse(results)
}

// handleTransition handles the jira_transition_issue tool call.
func (h *JiraHandler) handleTransition(ctx context.Context, args map[string]interface{}) (*domain.ToolResponse, error) {
	// Get the appropriate client (with custom credentials if provided)
	client, err := h.getClientForRequest(args)
	if err != nil {
		return nil, err
	}

	// Validate required parameters
	issueKey, err := getStringParam(args, "issueKey", true)
	if err != nil {
		return nil, err
	}

	// Get transition ID or name (at least one is required)
	transitionID, _ := getStringParam(args, "transitionId", false)
	transitionName, _ := getStringParam(args, "transitionName", false)

	if transitionID == "" && transitionName == "" {
		return nil, &domain.Error{
			Code:    domain.InvalidParams,
			Message: "either transitionId or transitionName must be provided",
		}
	}

	// Build the transition request
	transition := &domain.IssueTransition{
		Transition: domain.TransitionRef{
			ID:   transitionID,
			Name: transitionName,
		},
	}

	// Call the Jira client
	err = client.TransitionIssue(issueKey, transition)
	if err != nil {
		return nil, h.mapper.MapError(err)
	}

	// Return success response
	return h.mapper.MapToToolResponse(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Issue %s transitioned successfully", issueKey),
	})
}

// handleAddComment handles the jira_add_comment tool call.
func (h *JiraHandler) handleAddComment(ctx context.Context, args map[string]interface{}) (*domain.ToolResponse, error) {
	// Get the appropriate client (with custom credentials if provided)
	client, err := h.getClientForRequest(args)
	if err != nil {
		return nil, err
	}

	// Validate required parameters
	issueKey, err := getStringParam(args, "issueKey", true)
	if err != nil {
		return nil, err
	}
	body, err := getStringParam(args, "body", true)
	if err != nil {
		return nil, err
	}

	// Build the comment
	comment := &domain.Comment{
		Body: body,
	}

	// Call the Jira client
	err = client.AddComment(issueKey, comment)
	if err != nil {
		return nil, h.mapper.MapError(err)
	}

	// Return success response
	return h.mapper.MapToToolResponse(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Comment added to issue %s successfully", issueKey),
	})
}

// handleListProjects handles the jira_list_projects tool call.
func (h *JiraHandler) handleListProjects(ctx context.Context, args map[string]interface{}) (*domain.ToolResponse, error) {
	// Get the appropriate client (with custom credentials if provided)
	client, err := h.getClientForRequest(args)
	if err != nil {
		return nil, err
	}

	// Call the Jira client
	projects, err := client.GetProjects()
	if err != nil {
		return nil, h.mapper.MapError(err)
	}

	// Transform the response
	return h.mapper.MapToToolResponse(projects)
}
