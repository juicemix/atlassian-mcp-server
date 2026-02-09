package application

import (
	"context"
	"fmt"

	"atlassian-mcp-server/internal/domain"
	"atlassian-mcp-server/internal/infrastructure"
)

// BambooHandler implements ToolHandler for Bamboo operations.
// It routes MCP tool calls to the appropriate BambooClient methods and
// transforms responses using the ResponseMapper.
type BambooHandler struct {
	client *infrastructure.BambooClient
	mapper domain.ResponseMapper
}

// NewBambooHandler creates a new BambooHandler instance.
func NewBambooHandler(client *infrastructure.BambooClient, mapper domain.ResponseMapper) *BambooHandler {
	return &BambooHandler{
		client: client,
		mapper: mapper,
	}
}

// Tool name constants for Bamboo operations
const (
	ToolBambooGetPlans              = "bamboo_get_plans"
	ToolBambooGetPlan               = "bamboo_get_plan"
	ToolBambooTriggerBuild          = "bamboo_trigger_build"
	ToolBambooGetBuildResult        = "bamboo_get_build_result"
	ToolBambooGetBuildLog           = "bamboo_get_build_log"
	ToolBambooGetDeploymentProjects = "bamboo_get_deployment_projects"
	ToolBambooTriggerDeployment     = "bamboo_trigger_deployment"
)

// ToolName returns the identifier for this handler.
func (h *BambooHandler) ToolName() string {
	return "bamboo"
}

// ListTools returns available tools for Bamboo operations.
func (h *BambooHandler) ListTools() []domain.ToolDefinition {
	return []domain.ToolDefinition{
		{
			Name:        ToolBambooGetPlans,
			Description: "Retrieve all build plans from Bamboo",
			InputSchema: domain.JSONSchema{
				Type:       "object",
				Properties: map[string]interface{}{},
				Required:   []string{},
			},
		},
		{
			Name:        ToolBambooGetPlan,
			Description: "Retrieve a specific build plan by its key (e.g., PROJ-PLAN)",
			InputSchema: domain.JSONSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"planKey": map[string]interface{}{
						"type":        "string",
						"description": "The plan key (e.g., PROJ-PLAN)",
					},
				},
				Required: []string{"planKey"},
			},
		},
		{
			Name:        ToolBambooTriggerBuild,
			Description: "Trigger a build for a specific plan",
			InputSchema: domain.JSONSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"planKey": map[string]interface{}{
						"type":        "string",
						"description": "The plan key to trigger (e.g., PROJ-PLAN)",
					},
				},
				Required: []string{"planKey"},
			},
		},
		{
			Name:        ToolBambooGetBuildResult,
			Description: "Retrieve the result of a specific build",
			InputSchema: domain.JSONSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"buildKey": map[string]interface{}{
						"type":        "string",
						"description": "The build result key (e.g., PROJ-PLAN-123)",
					},
				},
				Required: []string{"buildKey"},
			},
		},
		{
			Name:        ToolBambooGetBuildLog,
			Description: "Retrieve the log output for a specific build",
			InputSchema: domain.JSONSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"buildKey": map[string]interface{}{
						"type":        "string",
						"description": "The build result key (e.g., PROJ-PLAN-123)",
					},
				},
				Required: []string{"buildKey"},
			},
		},
		{
			Name:        ToolBambooGetDeploymentProjects,
			Description: "Retrieve all deployment projects from Bamboo",
			InputSchema: domain.JSONSchema{
				Type:       "object",
				Properties: map[string]interface{}{},
				Required:   []string{},
			},
		},
		{
			Name:        ToolBambooTriggerDeployment,
			Description: "Trigger a deployment to a specific environment",
			InputSchema: domain.JSONSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"projectId": map[string]interface{}{
						"type":        "integer",
						"description": "The deployment project ID",
					},
					"environmentId": map[string]interface{}{
						"type":        "integer",
						"description": "The environment ID to deploy to",
					},
				},
				Required: []string{"projectId", "environmentId"},
			},
		},
	}
}

// Handle processes an MCP tool call request for Bamboo operations.
func (h *BambooHandler) Handle(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error) {
	// Validate that we have arguments
	if req.Arguments == nil {
		req.Arguments = make(map[string]interface{})
	}

	// Route to the appropriate handler based on tool name
	switch req.Name {
	case ToolBambooGetPlans:
		return h.handleGetPlans(ctx, req.Arguments)
	case ToolBambooGetPlan:
		return h.handleGetPlan(ctx, req.Arguments)
	case ToolBambooTriggerBuild:
		return h.handleTriggerBuild(ctx, req.Arguments)
	case ToolBambooGetBuildResult:
		return h.handleGetBuildResult(ctx, req.Arguments)
	case ToolBambooGetBuildLog:
		return h.handleGetBuildLog(ctx, req.Arguments)
	case ToolBambooGetDeploymentProjects:
		return h.handleGetDeploymentProjects(ctx, req.Arguments)
	case ToolBambooTriggerDeployment:
		return h.handleTriggerDeployment(ctx, req.Arguments)
	default:
		return nil, &domain.Error{
			Code:    domain.MethodNotFound,
			Message: fmt.Sprintf("unknown Bamboo tool: %s", req.Name),
		}
	}
}

// handleGetPlans handles the bamboo_get_plans tool call.
func (h *BambooHandler) handleGetPlans(ctx context.Context, args map[string]interface{}) (*domain.ToolResponse, error) {
	// Call the Bamboo client
	plans, err := h.client.GetPlans()
	if err != nil {
		return nil, h.mapper.MapError(err)
	}

	// Transform the response
	return h.mapper.MapToToolResponse(plans)
}

// handleGetPlan handles the bamboo_get_plan tool call.
func (h *BambooHandler) handleGetPlan(ctx context.Context, args map[string]interface{}) (*domain.ToolResponse, error) {
	// Validate required parameters
	planKey, err := getStringParam(args, "planKey", true)
	if err != nil {
		return nil, err
	}

	// Call the Bamboo client
	plan, err := h.client.GetPlan(planKey)
	if err != nil {
		return nil, h.mapper.MapError(err)
	}

	// Transform the response
	return h.mapper.MapToToolResponse(plan)
}

// handleTriggerBuild handles the bamboo_trigger_build tool call.
func (h *BambooHandler) handleTriggerBuild(ctx context.Context, args map[string]interface{}) (*domain.ToolResponse, error) {
	// Validate required parameters
	planKey, err := getStringParam(args, "planKey", true)
	if err != nil {
		return nil, err
	}

	// Call the Bamboo client
	result, err := h.client.TriggerBuild(planKey)
	if err != nil {
		return nil, h.mapper.MapError(err)
	}

	// Transform the response
	return h.mapper.MapToToolResponse(result)
}

// handleGetBuildResult handles the bamboo_get_build_result tool call.
func (h *BambooHandler) handleGetBuildResult(ctx context.Context, args map[string]interface{}) (*domain.ToolResponse, error) {
	// Validate required parameters
	buildKey, err := getStringParam(args, "buildKey", true)
	if err != nil {
		return nil, err
	}

	// Call the Bamboo client
	result, err := h.client.GetBuildResult(buildKey)
	if err != nil {
		return nil, h.mapper.MapError(err)
	}

	// Transform the response
	return h.mapper.MapToToolResponse(result)
}

// handleGetBuildLog handles the bamboo_get_build_log tool call.
func (h *BambooHandler) handleGetBuildLog(ctx context.Context, args map[string]interface{}) (*domain.ToolResponse, error) {
	// Validate required parameters
	buildKey, err := getStringParam(args, "buildKey", true)
	if err != nil {
		return nil, err
	}

	// Call the Bamboo client
	log, err := h.client.GetBuildLog(buildKey)
	if err != nil {
		return nil, h.mapper.MapError(err)
	}

	// Transform the response - wrap the log string in a map for consistent response format
	return h.mapper.MapToToolResponse(map[string]interface{}{
		"buildKey": buildKey,
		"log":      log,
	})
}

// handleGetDeploymentProjects handles the bamboo_get_deployment_projects tool call.
func (h *BambooHandler) handleGetDeploymentProjects(ctx context.Context, args map[string]interface{}) (*domain.ToolResponse, error) {
	// Call the Bamboo client
	projects, err := h.client.GetDeploymentProjects()
	if err != nil {
		return nil, h.mapper.MapError(err)
	}

	// Transform the response
	return h.mapper.MapToToolResponse(projects)
}

// handleTriggerDeployment handles the bamboo_trigger_deployment tool call.
func (h *BambooHandler) handleTriggerDeployment(ctx context.Context, args map[string]interface{}) (*domain.ToolResponse, error) {
	// Validate required parameters
	projectID, err := getIntParam(args, "projectId", true)
	if err != nil {
		return nil, err
	}
	environmentID, err := getIntParam(args, "environmentId", true)
	if err != nil {
		return nil, err
	}

	// Call the Bamboo client
	result, err := h.client.TriggerDeployment(projectID, environmentID)
	if err != nil {
		return nil, h.mapper.MapError(err)
	}

	// Transform the response
	return h.mapper.MapToToolResponse(result)
}
