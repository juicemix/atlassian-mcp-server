package application

import (
	"context"
	"fmt"

	"atlassian-mcp-server/internal/domain"
	"atlassian-mcp-server/internal/infrastructure"
)

// BitbucketHandler implements ToolHandler for Bitbucket operations.
// It routes MCP tool calls to the appropriate BitbucketClient methods and
// transforms responses using the ResponseMapper.
type BitbucketHandler struct {
	client *infrastructure.BitbucketClient
	mapper domain.ResponseMapper
}

// NewBitbucketHandler creates a new BitbucketHandler instance.
func NewBitbucketHandler(client *infrastructure.BitbucketClient, mapper domain.ResponseMapper) *BitbucketHandler {
	return &BitbucketHandler{
		client: client,
		mapper: mapper,
	}
}

// Tool name constants for Bitbucket operations
const (
	ToolBitbucketGetRepositories   = "bitbucket_get_repositories"
	ToolBitbucketGetBranches       = "bitbucket_get_branches"
	ToolBitbucketCreateBranch      = "bitbucket_create_branch"
	ToolBitbucketGetPullRequest    = "bitbucket_get_pull_request"
	ToolBitbucketCreatePullRequest = "bitbucket_create_pull_request"
	ToolBitbucketMergePullRequest  = "bitbucket_merge_pull_request"
	ToolBitbucketGetCommits        = "bitbucket_get_commits"
	ToolBitbucketGetFileContent    = "bitbucket_get_file_content"
)

// ToolName returns the identifier for this handler.
func (h *BitbucketHandler) ToolName() string {
	return "bitbucket"
}

// ListTools returns available tools for Bitbucket operations.
func (h *BitbucketHandler) ListTools() []domain.ToolDefinition {
	return []domain.ToolDefinition{
		{
			Name:        ToolBitbucketGetRepositories,
			Description: "List all repositories in a Bitbucket project",
			InputSchema: domain.JSONSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project": map[string]interface{}{
						"type":        "string",
						"description": "The project key (e.g., PROJ)",
					},
				},
				Required: []string{"project"},
			},
		},
		{
			Name:        ToolBitbucketGetBranches,
			Description: "List all branches in a Bitbucket repository",
			InputSchema: domain.JSONSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project": map[string]interface{}{
						"type":        "string",
						"description": "The project key (e.g., PROJ)",
					},
					"repo": map[string]interface{}{
						"type":        "string",
						"description": "The repository slug (e.g., my-repo)",
					},
				},
				Required: []string{"project", "repo"},
			},
		},
		{
			Name:        ToolBitbucketCreateBranch,
			Description: "Create a new branch in a Bitbucket repository",
			InputSchema: domain.JSONSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project": map[string]interface{}{
						"type":        "string",
						"description": "The project key (e.g., PROJ)",
					},
					"repo": map[string]interface{}{
						"type":        "string",
						"description": "The repository slug (e.g., my-repo)",
					},
					"name": map[string]interface{}{
						"type":        "string",
						"description": "The branch name (e.g., feature/new-feature)",
					},
					"startPoint": map[string]interface{}{
						"type":        "string",
						"description": "The commit ID or branch name to branch from",
					},
				},
				Required: []string{"project", "repo", "name", "startPoint"},
			},
		},
		{
			Name:        ToolBitbucketGetPullRequest,
			Description: "Retrieve a pull request by its ID",
			InputSchema: domain.JSONSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project": map[string]interface{}{
						"type":        "string",
						"description": "The project key (e.g., PROJ)",
					},
					"repo": map[string]interface{}{
						"type":        "string",
						"description": "The repository slug (e.g., my-repo)",
					},
					"prId": map[string]interface{}{
						"type":        "integer",
						"description": "The pull request ID",
					},
				},
				Required: []string{"project", "repo", "prId"},
			},
		},
		{
			Name:        ToolBitbucketCreatePullRequest,
			Description: "Create a new pull request in a Bitbucket repository",
			InputSchema: domain.JSONSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project": map[string]interface{}{
						"type":        "string",
						"description": "The project key (e.g., PROJ)",
					},
					"repo": map[string]interface{}{
						"type":        "string",
						"description": "The repository slug (e.g., my-repo)",
					},
					"title": map[string]interface{}{
						"type":        "string",
						"description": "The pull request title",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "The pull request description (optional)",
					},
					"fromRef": map[string]interface{}{
						"type":        "string",
						"description": "The source branch name",
					},
					"toRef": map[string]interface{}{
						"type":        "string",
						"description": "The target branch name",
					},
				},
				Required: []string{"project", "repo", "title", "fromRef", "toRef"},
			},
		},
		{
			Name:        ToolBitbucketMergePullRequest,
			Description: "Merge a pull request in a Bitbucket repository",
			InputSchema: domain.JSONSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project": map[string]interface{}{
						"type":        "string",
						"description": "The project key (e.g., PROJ)",
					},
					"repo": map[string]interface{}{
						"type":        "string",
						"description": "The repository slug (e.g., my-repo)",
					},
					"prId": map[string]interface{}{
						"type":        "integer",
						"description": "The pull request ID",
					},
					"version": map[string]interface{}{
						"type":        "integer",
						"description": "The pull request version (for optimistic locking)",
					},
				},
				Required: []string{"project", "repo", "prId", "version"},
			},
		},
		{
			Name:        ToolBitbucketGetCommits,
			Description: "Retrieve commit history for a Bitbucket repository",
			InputSchema: domain.JSONSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project": map[string]interface{}{
						"type":        "string",
						"description": "The project key (e.g., PROJ)",
					},
					"repo": map[string]interface{}{
						"type":        "string",
						"description": "The repository slug (e.g., my-repo)",
					},
					"until": map[string]interface{}{
						"type":        "string",
						"description": "The commit ID or branch name to retrieve commits up to (optional)",
					},
					"since": map[string]interface{}{
						"type":        "string",
						"description": "The commit ID or branch name to retrieve commits since (optional)",
					},
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Filter commits by file path (optional)",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of commits to return (optional)",
					},
					"start": map[string]interface{}{
						"type":        "integer",
						"description": "Starting index for pagination (optional)",
					},
				},
				Required: []string{"project", "repo"},
			},
		},
		{
			Name:        ToolBitbucketGetFileContent,
			Description: "Retrieve the content of a file from a Bitbucket repository",
			InputSchema: domain.JSONSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project": map[string]interface{}{
						"type":        "string",
						"description": "The project key (e.g., PROJ)",
					},
					"repo": map[string]interface{}{
						"type":        "string",
						"description": "The repository slug (e.g., my-repo)",
					},
					"path": map[string]interface{}{
						"type":        "string",
						"description": "The file path within the repository",
					},
					"ref": map[string]interface{}{
						"type":        "string",
						"description": "The branch name or commit ID (optional, defaults to default branch)",
					},
				},
				Required: []string{"project", "repo", "path"},
			},
		},
	}
}

// Handle processes an MCP tool call request for Bitbucket operations.
func (h *BitbucketHandler) Handle(ctx context.Context, req *domain.ToolRequest) (*domain.ToolResponse, error) {
	// Validate that we have arguments
	if req.Arguments == nil {
		req.Arguments = make(map[string]interface{})
	}

	// Route to the appropriate handler based on tool name
	switch req.Name {
	case ToolBitbucketGetRepositories:
		return h.handleGetRepositories(ctx, req.Arguments)
	case ToolBitbucketGetBranches:
		return h.handleGetBranches(ctx, req.Arguments)
	case ToolBitbucketCreateBranch:
		return h.handleCreateBranch(ctx, req.Arguments)
	case ToolBitbucketGetPullRequest:
		return h.handleGetPullRequest(ctx, req.Arguments)
	case ToolBitbucketCreatePullRequest:
		return h.handleCreatePullRequest(ctx, req.Arguments)
	case ToolBitbucketMergePullRequest:
		return h.handleMergePullRequest(ctx, req.Arguments)
	case ToolBitbucketGetCommits:
		return h.handleGetCommits(ctx, req.Arguments)
	case ToolBitbucketGetFileContent:
		return h.handleGetFileContent(ctx, req.Arguments)
	default:
		return nil, &domain.Error{
			Code:    domain.MethodNotFound,
			Message: fmt.Sprintf("unknown Bitbucket tool: %s", req.Name),
		}
	}
}

// handleGetRepositories handles the bitbucket_get_repositories tool call.
func (h *BitbucketHandler) handleGetRepositories(ctx context.Context, args map[string]interface{}) (*domain.ToolResponse, error) {
	// Validate required parameters
	project, err := getStringParam(args, "project", true)
	if err != nil {
		return nil, err
	}

	// Call the Bitbucket client
	repos, err := h.client.GetRepositories(project)
	if err != nil {
		return nil, h.mapper.MapError(err)
	}

	// Transform the response
	return h.mapper.MapToToolResponse(repos)
}

// handleGetBranches handles the bitbucket_get_branches tool call.
func (h *BitbucketHandler) handleGetBranches(ctx context.Context, args map[string]interface{}) (*domain.ToolResponse, error) {
	// Validate required parameters
	project, err := getStringParam(args, "project", true)
	if err != nil {
		return nil, err
	}
	repo, err := getStringParam(args, "repo", true)
	if err != nil {
		return nil, err
	}

	// Call the Bitbucket client
	branches, err := h.client.GetBranches(project, repo)
	if err != nil {
		return nil, h.mapper.MapError(err)
	}

	// Transform the response
	return h.mapper.MapToToolResponse(branches)
}

// handleCreateBranch handles the bitbucket_create_branch tool call.
func (h *BitbucketHandler) handleCreateBranch(ctx context.Context, args map[string]interface{}) (*domain.ToolResponse, error) {
	// Validate required parameters
	project, err := getStringParam(args, "project", true)
	if err != nil {
		return nil, err
	}
	repo, err := getStringParam(args, "repo", true)
	if err != nil {
		return nil, err
	}
	name, err := getStringParam(args, "name", true)
	if err != nil {
		return nil, err
	}
	startPoint, err := getStringParam(args, "startPoint", true)
	if err != nil {
		return nil, err
	}

	// Build the create request
	createReq := &domain.BranchCreate{
		Name:       name,
		StartPoint: startPoint,
	}

	// Call the Bitbucket client
	branch, err := h.client.CreateBranch(project, repo, createReq)
	if err != nil {
		return nil, h.mapper.MapError(err)
	}

	// Transform the response
	return h.mapper.MapToToolResponse(branch)
}

// handleGetPullRequest handles the bitbucket_get_pull_request tool call.
func (h *BitbucketHandler) handleGetPullRequest(ctx context.Context, args map[string]interface{}) (*domain.ToolResponse, error) {
	// Validate required parameters
	project, err := getStringParam(args, "project", true)
	if err != nil {
		return nil, err
	}
	repo, err := getStringParam(args, "repo", true)
	if err != nil {
		return nil, err
	}
	prID, err := getIntParam(args, "prId", true)
	if err != nil {
		return nil, err
	}

	// Call the Bitbucket client
	pr, err := h.client.GetPullRequest(project, repo, prID)
	if err != nil {
		return nil, h.mapper.MapError(err)
	}

	// Transform the response
	return h.mapper.MapToToolResponse(pr)
}

// handleCreatePullRequest handles the bitbucket_create_pull_request tool call.
func (h *BitbucketHandler) handleCreatePullRequest(ctx context.Context, args map[string]interface{}) (*domain.ToolResponse, error) {
	// Validate required parameters
	project, err := getStringParam(args, "project", true)
	if err != nil {
		return nil, err
	}
	repo, err := getStringParam(args, "repo", true)
	if err != nil {
		return nil, err
	}
	title, err := getStringParam(args, "title", true)
	if err != nil {
		return nil, err
	}
	fromRef, err := getStringParam(args, "fromRef", true)
	if err != nil {
		return nil, err
	}
	toRef, err := getStringParam(args, "toRef", true)
	if err != nil {
		return nil, err
	}

	// Optional parameters
	description, _ := getStringParam(args, "description", false)

	// Build the create request
	createReq := &domain.PullRequestCreate{
		Title:       title,
		Description: description,
		FromRef: domain.RefCreate{
			ID: fromRef,
			Repository: domain.RepositoryRef{
				Slug: repo,
				Project: domain.ProjectRef{
					Key: project,
				},
			},
		},
		ToRef: domain.RefCreate{
			ID: toRef,
			Repository: domain.RepositoryRef{
				Slug: repo,
				Project: domain.ProjectRef{
					Key: project,
				},
			},
		},
	}

	// Call the Bitbucket client
	pr, err := h.client.CreatePullRequest(project, repo, createReq)
	if err != nil {
		return nil, h.mapper.MapError(err)
	}

	// Transform the response
	return h.mapper.MapToToolResponse(pr)
}

// handleMergePullRequest handles the bitbucket_merge_pull_request tool call.
func (h *BitbucketHandler) handleMergePullRequest(ctx context.Context, args map[string]interface{}) (*domain.ToolResponse, error) {
	// Validate required parameters
	project, err := getStringParam(args, "project", true)
	if err != nil {
		return nil, err
	}
	repo, err := getStringParam(args, "repo", true)
	if err != nil {
		return nil, err
	}
	prID, err := getIntParam(args, "prId", true)
	if err != nil {
		return nil, err
	}
	version, err := getIntParam(args, "version", true)
	if err != nil {
		return nil, err
	}

	// Call the Bitbucket client
	err = h.client.MergePullRequest(project, repo, prID, version)
	if err != nil {
		return nil, h.mapper.MapError(err)
	}

	// Return success response
	return h.mapper.MapToToolResponse(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Pull request %d merged successfully", prID),
	})
}

// handleGetCommits handles the bitbucket_get_commits tool call.
func (h *BitbucketHandler) handleGetCommits(ctx context.Context, args map[string]interface{}) (*domain.ToolResponse, error) {
	// Validate required parameters
	project, err := getStringParam(args, "project", true)
	if err != nil {
		return nil, err
	}
	repo, err := getStringParam(args, "repo", true)
	if err != nil {
		return nil, err
	}

	// Optional parameters
	until, _ := getStringParam(args, "until", false)
	since, _ := getStringParam(args, "since", false)
	path, _ := getStringParam(args, "path", false)
	limit, err := getIntParam(args, "limit", false)
	if err != nil {
		return nil, err
	}
	start, err := getIntParam(args, "start", false)
	if err != nil {
		return nil, err
	}

	// Build commit options
	options := &domain.CommitOptions{
		Until: until,
		Since: since,
		Path:  path,
		Limit: limit,
		Start: start,
	}

	// Call the Bitbucket client
	commits, err := h.client.GetCommits(project, repo, options)
	if err != nil {
		return nil, h.mapper.MapError(err)
	}

	// Transform the response
	return h.mapper.MapToToolResponse(commits)
}

// handleGetFileContent handles the bitbucket_get_file_content tool call.
func (h *BitbucketHandler) handleGetFileContent(ctx context.Context, args map[string]interface{}) (*domain.ToolResponse, error) {
	// Validate required parameters
	project, err := getStringParam(args, "project", true)
	if err != nil {
		return nil, err
	}
	repo, err := getStringParam(args, "repo", true)
	if err != nil {
		return nil, err
	}
	path, err := getStringParam(args, "path", true)
	if err != nil {
		return nil, err
	}

	// Optional parameters
	ref, _ := getStringParam(args, "ref", false)

	// Call the Bitbucket client
	content, err := h.client.GetFileContent(project, repo, path, ref)
	if err != nil {
		return nil, h.mapper.MapError(err)
	}

	// Transform the response
	return h.mapper.MapToToolResponse(map[string]interface{}{
		"path":    path,
		"ref":     ref,
		"content": content,
	})
}
