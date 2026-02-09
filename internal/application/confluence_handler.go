package application

import (
	"context"
	"fmt"

	"atlassian-mcp-server/internal/domain"
	"atlassian-mcp-server/internal/infrastructure"
)

// ConfluenceHandler implements ToolHandler for Confluence operations.
// It routes MCP tool calls to the appropriate ConfluenceClient methods and
// transforms responses using the ResponseMapper.
type ConfluenceHandler struct {
	client *infrastructure.ConfluenceClient
	mapper domain.ResponseMapper
}

// NewConfluenceHandler creates a new ConfluenceHandler instance.
func NewConfluenceHandler(client *infrastructure.ConfluenceClient, mapper domain.ResponseMapper) *ConfluenceHandler {
	return &ConfluenceHandler{
		client: client,
		mapper: mapper,
	}
}

// Tool name constants for Confluence operations
const (
	ToolConfluenceGetPage        = "confluence_get_page"
	ToolConfluenceCreatePage     = "confluence_create_page"
	ToolConfluenceUpdatePage     = "confluence_update_page"
	ToolConfluenceDeletePage     = "confluence_delete_page"
	ToolConfluenceSearchCQL      = "confluence_search_cql"
	ToolConfluenceGetSpaces      = "confluence_get_spaces"
	ToolConfluenceGetPageHistory = "confluence_get_page_history"
)

// ToolName returns the identifier for this handler.
func (h *ConfluenceHandler) ToolName() string {
	return "confluence"
}

// ListTools returns available tools for Confluence operations.
func (h *ConfluenceHandler) ListTools() []domain.ToolDefinition {
	return []domain.ToolDefinition{
		{
			Name:        ToolConfluenceGetPage,
			Description: "Retrieve a Confluence page by its ID",
			InputSchema: domain.JSONSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"pageId": map[string]interface{}{
						"type":        "string",
						"description": "The page ID",
					},
				},
				Required: []string{"pageId"},
			},
		},
		{
			Name:        ToolConfluenceCreatePage,
			Description: "Create a new Confluence page",
			InputSchema: domain.JSONSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"spaceKey": map[string]interface{}{
						"type":        "string",
						"description": "The space key where the page will be created",
					},
					"title": map[string]interface{}{
						"type":        "string",
						"description": "The page title",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "The page content in storage format (HTML)",
					},
					"representation": map[string]interface{}{
						"type":        "string",
						"description": "The content representation format (default: storage)",
					},
				},
				Required: []string{"spaceKey", "title", "content"},
			},
		},
		{
			Name:        ToolConfluenceUpdatePage,
			Description: "Update an existing Confluence page",
			InputSchema: domain.JSONSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"pageId": map[string]interface{}{
						"type":        "string",
						"description": "The page ID",
					},
					"version": map[string]interface{}{
						"type":        "integer",
						"description": "The current version number of the page (required for optimistic locking)",
					},
					"title": map[string]interface{}{
						"type":        "string",
						"description": "The new page title (optional)",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "The new page content in storage format (optional)",
					},
					"representation": map[string]interface{}{
						"type":        "string",
						"description": "The content representation format (default: storage)",
					},
				},
				Required: []string{"pageId", "version"},
			},
		},
		{
			Name:        ToolConfluenceDeletePage,
			Description: "Delete a Confluence page",
			InputSchema: domain.JSONSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"pageId": map[string]interface{}{
						"type":        "string",
						"description": "The page ID",
					},
				},
				Required: []string{"pageId"},
			},
		},
		{
			Name:        ToolConfluenceSearchCQL,
			Description: "Search for Confluence content using CQL (Confluence Query Language)",
			InputSchema: domain.JSONSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"cql": map[string]interface{}{
						"type":        "string",
						"description": "The CQL query string",
					},
					"start": map[string]interface{}{
						"type":        "integer",
						"description": "The index of the first result to return (0-based, optional)",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "The maximum number of results to return (optional)",
					},
					"expand": map[string]interface{}{
						"type":        "string",
						"description": "Comma-separated list of properties to expand (optional)",
					},
				},
				Required: []string{"cql"},
			},
		},
		{
			Name:        ToolConfluenceGetSpaces,
			Description: "List all Confluence spaces accessible to the authenticated user",
			InputSchema: domain.JSONSchema{
				Type:       "object",
				Properties: map[string]interface{}{},
				Required:   []string{},
			},
		},
		{
			Name:        ToolConfluenceGetPageHistory,
			Description: "Retrieve the history information for a Confluence page",
			InputSchema: domain.JSONSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"pageId": map[string]interface{}{
						"type":        "string",
						"description": "The page ID",
					},
				},
				Required: []string{"pageId"},
			},
		},
	}
}

// Handle processes an MCP tool call request for Confluence operations.
func (h *ConfluenceHandler) Handle(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error) {
	// Validate that we have arguments
	if req.Arguments == nil {
		req.Arguments = make(map[string]interface{})
	}

	// Route to the appropriate handler based on tool name
	switch req.Name {
	case ToolConfluenceGetPage:
		return h.handleGetPage(ctx, req.Arguments)
	case ToolConfluenceCreatePage:
		return h.handleCreatePage(ctx, req.Arguments)
	case ToolConfluenceUpdatePage:
		return h.handleUpdatePage(ctx, req.Arguments)
	case ToolConfluenceDeletePage:
		return h.handleDeletePage(ctx, req.Arguments)
	case ToolConfluenceSearchCQL:
		return h.handleSearchCQL(ctx, req.Arguments)
	case ToolConfluenceGetSpaces:
		return h.handleGetSpaces(ctx, req.Arguments)
	case ToolConfluenceGetPageHistory:
		return h.handleGetPageHistory(ctx, req.Arguments)
	default:
		return nil, &domain.Error{
			Code:    domain.MethodNotFound,
			Message: fmt.Sprintf("unknown Confluence tool: %s", req.Name),
		}
	}
}

// handleGetPage handles the confluence_get_page tool call.
func (h *ConfluenceHandler) handleGetPage(ctx context.Context, args map[string]interface{}) (*domain.ToolResponse, error) {
	// Validate required parameters
	pageID, err := getStringParam(args, "pageId", true)
	if err != nil {
		return nil, err
	}

	// Call the Confluence client
	page, err := h.client.GetPage(pageID)
	if err != nil {
		return nil, h.mapper.MapError(err)
	}

	// Transform the response
	return h.mapper.MapToToolResponse(page)
}

// handleCreatePage handles the confluence_create_page tool call.
func (h *ConfluenceHandler) handleCreatePage(ctx context.Context, args map[string]interface{}) (*domain.ToolResponse, error) {
	// Validate required parameters
	spaceKey, err := getStringParam(args, "spaceKey", true)
	if err != nil {
		return nil, err
	}
	title, err := getStringParam(args, "title", true)
	if err != nil {
		return nil, err
	}
	content, err := getStringParam(args, "content", true)
	if err != nil {
		return nil, err
	}

	// Optional parameters
	representation, _ := getStringParam(args, "representation", false)
	if representation == "" {
		representation = "storage"
	}

	// Build the create request
	createReq := &domain.PageCreate{
		Type:  "page",
		Title: title,
		Space: domain.SpaceRef{
			Key: spaceKey,
		},
		Body: domain.BodyCreate{
			Storage: domain.StorageCreate{
				Value:          content,
				Representation: representation,
			},
		},
	}

	// Call the Confluence client
	page, err := h.client.CreatePage(createReq)
	if err != nil {
		return nil, h.mapper.MapError(err)
	}

	// Transform the response
	return h.mapper.MapToToolResponse(page)
}

// handleUpdatePage handles the confluence_update_page tool call.
func (h *ConfluenceHandler) handleUpdatePage(ctx context.Context, args map[string]interface{}) (*domain.ToolResponse, error) {
	// Validate required parameters
	pageID, err := getStringParam(args, "pageId", true)
	if err != nil {
		return nil, err
	}
	version, err := getIntParam(args, "version", true)
	if err != nil {
		return nil, err
	}

	// Optional parameters
	title, _ := getStringParam(args, "title", false)
	content, _ := getStringParam(args, "content", false)
	representation, _ := getStringParam(args, "representation", false)
	if representation == "" {
		representation = "storage"
	}

	// Build the update request
	updateReq := &domain.PageUpdate{
		Version: domain.VersionUpdate{
			Number: version,
		},
		Type: "page",
	}

	// Add optional fields if provided
	if title != "" {
		updateReq.Title = title
	}
	if content != "" {
		updateReq.Body = &domain.BodyCreate{
			Storage: domain.StorageCreate{
				Value:          content,
				Representation: representation,
			},
		}
	}

	// Call the Confluence client
	page, err := h.client.UpdatePage(pageID, updateReq)
	if err != nil {
		return nil, h.mapper.MapError(err)
	}

	// Transform the response
	return h.mapper.MapToToolResponse(page)
}

// handleDeletePage handles the confluence_delete_page tool call.
func (h *ConfluenceHandler) handleDeletePage(ctx context.Context, args map[string]interface{}) (*domain.ToolResponse, error) {
	// Validate required parameters
	pageID, err := getStringParam(args, "pageId", true)
	if err != nil {
		return nil, err
	}

	// Call the Confluence client
	err = h.client.DeletePage(pageID)
	if err != nil {
		return nil, h.mapper.MapError(err)
	}

	// Return success response
	return h.mapper.MapToToolResponse(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Page %s deleted successfully", pageID),
	})
}

// handleSearchCQL handles the confluence_search_cql tool call.
func (h *ConfluenceHandler) handleSearchCQL(ctx context.Context, args map[string]interface{}) (*domain.ToolResponse, error) {
	// Validate required parameters
	cql, err := getStringParam(args, "cql", true)
	if err != nil {
		return nil, err
	}

	// Optional parameters
	start, err := getIntParam(args, "start", false)
	if err != nil {
		return nil, err
	}
	limit, err := getIntParam(args, "limit", false)
	if err != nil {
		return nil, err
	}
	expand, _ := getStringParam(args, "expand", false)

	// Build search options
	options := &infrastructure.ConfluenceSearchOptions{
		CQL:    cql,
		Start:  start,
		Limit:  limit,
		Expand: expand,
	}

	// Call the Confluence client
	results, err := h.client.SearchCQL(cql, options)
	if err != nil {
		return nil, h.mapper.MapError(err)
	}

	// Transform the response
	return h.mapper.MapToToolResponse(results)
}

// handleGetSpaces handles the confluence_get_spaces tool call.
func (h *ConfluenceHandler) handleGetSpaces(ctx context.Context, args map[string]interface{}) (*domain.ToolResponse, error) {
	// Call the Confluence client
	spaces, err := h.client.GetSpaces()
	if err != nil {
		return nil, h.mapper.MapError(err)
	}

	// Transform the response
	return h.mapper.MapToToolResponse(spaces)
}

// handleGetPageHistory handles the confluence_get_page_history tool call.
func (h *ConfluenceHandler) handleGetPageHistory(ctx context.Context, args map[string]interface{}) (*domain.ToolResponse, error) {
	// Validate required parameters
	pageID, err := getStringParam(args, "pageId", true)
	if err != nil {
		return nil, err
	}

	// Call the Confluence client
	history, err := h.client.GetPageHistory(pageID)
	if err != nil {
		return nil, h.mapper.MapError(err)
	}

	// Transform the response
	return h.mapper.MapToToolResponse(history)
}
