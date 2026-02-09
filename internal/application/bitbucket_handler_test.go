package application

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"atlassian-mcp-server/internal/domain"
	"atlassian-mcp-server/internal/infrastructure"
)

// setupMockBitbucketServer creates a mock Bitbucket server for testing
func setupMockBitbucketServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		// Get repositories
		case r.Method == "GET" && r.URL.Path == "/rest/api/1.0/projects/PROJ/repos":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []domain.Repository{
					{
						ID:   1,
						Slug: "test-repo",
						Name: "Test Repository",
						Project: domain.Project{
							ID:   "10000",
							Key:  "PROJ",
							Name: "Test Project",
						},
						Public: false,
					},
				},
				"size":       1,
				"limit":      25,
				"isLastPage": true,
				"start":      0,
			})

		// Get branches
		case r.Method == "GET" && r.URL.Path == "/rest/api/1.0/projects/PROJ/repos/test-repo/branches":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []domain.Branch{
					{
						ID:           "refs/heads/main",
						DisplayID:    "main",
						Type:         "BRANCH",
						LatestCommit: "abc123",
					},
				},
				"size":       1,
				"limit":      25,
				"isLastPage": true,
				"start":      0,
			})

		// Create branch
		case r.Method == "POST" && r.URL.Path == "/rest/api/1.0/projects/PROJ/repos/test-repo/branches":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(domain.Branch{
				ID:           "refs/heads/feature-branch",
				DisplayID:    "feature-branch",
				Type:         "BRANCH",
				LatestCommit: "def456",
			})

		// Get pull request
		case r.Method == "GET" && r.URL.Path == "/rest/api/1.0/projects/PROJ/repos/test-repo/pull-requests/1":
			json.NewEncoder(w).Encode(domain.PullRequest{
				ID:          1,
				Version:     1,
				Title:       "Test PR",
				Description: "Test description",
				State:       "OPEN",
				Open:        true,
				Closed:      false,
			})

		// Create pull request
		case r.Method == "POST" && r.URL.Path == "/rest/api/1.0/projects/PROJ/repos/test-repo/pull-requests":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(domain.PullRequest{
				ID:          2,
				Version:     0,
				Title:       "New PR",
				Description: "New description",
				State:       "OPEN",
				Open:        true,
				Closed:      false,
			})

		// Merge pull request
		case r.Method == "POST" && r.URL.Path == "/rest/api/1.0/projects/PROJ/repos/test-repo/pull-requests/1/merge":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(domain.PullRequest{
				ID:     1,
				State:  "MERGED",
				Open:   false,
				Closed: true,
			})

		// Get commits
		case r.Method == "GET" && r.URL.Path == "/rest/api/1.0/projects/PROJ/repos/test-repo/commits":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []domain.Commit{
					{
						ID:        "abc123",
						DisplayID: "abc123",
						Message:   "Test commit",
					},
				},
				"size":       1,
				"limit":      25,
				"isLastPage": true,
				"start":      0,
			})

		// Get file content
		case r.Method == "GET" && r.URL.Path == "/rest/api/1.0/projects/PROJ/repos/test-repo/browse/README.md":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"lines": []map[string]string{
					{"text": "# Test Repository"},
					{"text": "This is a test"},
				},
			})

		default:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Not found",
			})
		}
	}))
}

func TestBitbucketHandler_ToolName(t *testing.T) {
	handler := NewBitbucketHandler(nil, nil)
	if handler.ToolName() != "bitbucket" {
		t.Errorf("expected tool name 'bitbucket', got '%s'", handler.ToolName())
	}
}

func TestBitbucketHandler_ListTools(t *testing.T) {
	handler := NewBitbucketHandler(nil, nil)
	tools := handler.ListTools()

	expectedTools := []string{
		ToolBitbucketGetRepositories,
		ToolBitbucketGetBranches,
		ToolBitbucketCreateBranch,
		ToolBitbucketGetPullRequest,
		ToolBitbucketCreatePullRequest,
		ToolBitbucketMergePullRequest,
		ToolBitbucketGetCommits,
		ToolBitbucketGetFileContent,
	}

	if len(tools) != len(expectedTools) {
		t.Fatalf("expected %d tools, got %d", len(expectedTools), len(tools))
	}

	// Check that all expected tools are present
	toolMap := make(map[string]bool)
	for _, tool := range tools {
		toolMap[tool.Name] = true
	}

	for _, expectedTool := range expectedTools {
		if !toolMap[expectedTool] {
			t.Errorf("expected tool '%s' not found", expectedTool)
		}
	}

	// Verify that each tool has a description and input schema
	for _, tool := range tools {
		if tool.Description == "" {
			t.Errorf("tool '%s' has no description", tool.Name)
		}
		if tool.InputSchema.Type == "" {
			t.Errorf("tool '%s' has no input schema", tool.Name)
		}
	}
}

func TestBitbucketHandler_HandleGetRepositories(t *testing.T) {
	server := setupMockBitbucketServer()
	defer server.Close()

	client := infrastructure.NewBitbucketClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBitbucketHandler(client, mapper)

	req := &domain.ToolRequest{
		Name: ToolBitbucketGetRepositories,
		Arguments: map[string]interface{}{
			"project": "PROJ",
		},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	if len(resp.Content) == 0 {
		t.Fatal("expected content in response")
	}

	if resp.Content[0].Type != "text" {
		t.Errorf("expected content type 'text', got '%s'", resp.Content[0].Type)
	}
}

func TestBitbucketHandler_HandleGetRepositories_MissingParameter(t *testing.T) {
	server := setupMockBitbucketServer()
	defer server.Close()

	client := infrastructure.NewBitbucketClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBitbucketHandler(client, mapper)

	req := &domain.ToolRequest{
		Name:      ToolBitbucketGetRepositories,
		Arguments: map[string]interface{}{},
	}

	_, err := handler.Handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing parameter, got nil")
	}

	domainErr, ok := err.(*domain.Error)
	if !ok {
		t.Fatalf("expected domain.Error, got %T", err)
	}

	if domainErr.Code != domain.InvalidParams {
		t.Errorf("expected error code %d, got %d", domain.InvalidParams, domainErr.Code)
	}
}

func TestBitbucketHandler_HandleGetBranches(t *testing.T) {
	server := setupMockBitbucketServer()
	defer server.Close()

	client := infrastructure.NewBitbucketClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBitbucketHandler(client, mapper)

	req := &domain.ToolRequest{
		Name: ToolBitbucketGetBranches,
		Arguments: map[string]interface{}{
			"project": "PROJ",
			"repo":    "test-repo",
		},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

func TestBitbucketHandler_HandleGetBranches_MissingParameters(t *testing.T) {
	server := setupMockBitbucketServer()
	defer server.Close()

	client := infrastructure.NewBitbucketClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBitbucketHandler(client, mapper)

	testCases := []struct {
		name      string
		arguments map[string]interface{}
		missing   string
	}{
		{
			name:      "missing project",
			arguments: map[string]interface{}{"repo": "test-repo"},
			missing:   "project",
		},
		{
			name:      "missing repo",
			arguments: map[string]interface{}{"project": "PROJ"},
			missing:   "repo",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &domain.ToolRequest{
				Name:      ToolBitbucketGetBranches,
				Arguments: tc.arguments,
			}

			_, err := handler.Handle(context.Background(), req)
			if err == nil {
				t.Fatalf("expected error for missing %s, got nil", tc.missing)
			}

			domainErr, ok := err.(*domain.Error)
			if !ok {
				t.Fatalf("expected domain.Error, got %T", err)
			}

			if domainErr.Code != domain.InvalidParams {
				t.Errorf("expected error code %d, got %d", domain.InvalidParams, domainErr.Code)
			}
		})
	}
}

func TestBitbucketHandler_HandleCreateBranch(t *testing.T) {
	server := setupMockBitbucketServer()
	defer server.Close()

	client := infrastructure.NewBitbucketClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBitbucketHandler(client, mapper)

	req := &domain.ToolRequest{
		Name: ToolBitbucketCreateBranch,
		Arguments: map[string]interface{}{
			"project":    "PROJ",
			"repo":       "test-repo",
			"name":       "feature-branch",
			"startPoint": "main",
		},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

func TestBitbucketHandler_HandleCreateBranch_MissingParameters(t *testing.T) {
	server := setupMockBitbucketServer()
	defer server.Close()

	client := infrastructure.NewBitbucketClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBitbucketHandler(client, mapper)

	testCases := []struct {
		name      string
		arguments map[string]interface{}
		missing   string
	}{
		{
			name:      "missing project",
			arguments: map[string]interface{}{"repo": "test-repo", "name": "feature", "startPoint": "main"},
			missing:   "project",
		},
		{
			name:      "missing repo",
			arguments: map[string]interface{}{"project": "PROJ", "name": "feature", "startPoint": "main"},
			missing:   "repo",
		},
		{
			name:      "missing name",
			arguments: map[string]interface{}{"project": "PROJ", "repo": "test-repo", "startPoint": "main"},
			missing:   "name",
		},
		{
			name:      "missing startPoint",
			arguments: map[string]interface{}{"project": "PROJ", "repo": "test-repo", "name": "feature"},
			missing:   "startPoint",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &domain.ToolRequest{
				Name:      ToolBitbucketCreateBranch,
				Arguments: tc.arguments,
			}

			_, err := handler.Handle(context.Background(), req)
			if err == nil {
				t.Fatalf("expected error for missing %s, got nil", tc.missing)
			}

			domainErr, ok := err.(*domain.Error)
			if !ok {
				t.Fatalf("expected domain.Error, got %T", err)
			}

			if domainErr.Code != domain.InvalidParams {
				t.Errorf("expected error code %d, got %d", domain.InvalidParams, domainErr.Code)
			}
		})
	}
}

func TestBitbucketHandler_HandleGetPullRequest(t *testing.T) {
	server := setupMockBitbucketServer()
	defer server.Close()

	client := infrastructure.NewBitbucketClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBitbucketHandler(client, mapper)

	req := &domain.ToolRequest{
		Name: ToolBitbucketGetPullRequest,
		Arguments: map[string]interface{}{
			"project": "PROJ",
			"repo":    "test-repo",
			"prId":    float64(1),
		},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

func TestBitbucketHandler_HandleGetPullRequest_MissingParameters(t *testing.T) {
	server := setupMockBitbucketServer()
	defer server.Close()

	client := infrastructure.NewBitbucketClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBitbucketHandler(client, mapper)

	testCases := []struct {
		name      string
		arguments map[string]interface{}
		missing   string
	}{
		{
			name:      "missing project",
			arguments: map[string]interface{}{"repo": "test-repo", "prId": float64(1)},
			missing:   "project",
		},
		{
			name:      "missing repo",
			arguments: map[string]interface{}{"project": "PROJ", "prId": float64(1)},
			missing:   "repo",
		},
		{
			name:      "missing prId",
			arguments: map[string]interface{}{"project": "PROJ", "repo": "test-repo"},
			missing:   "prId",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &domain.ToolRequest{
				Name:      ToolBitbucketGetPullRequest,
				Arguments: tc.arguments,
			}

			_, err := handler.Handle(context.Background(), req)
			if err == nil {
				t.Fatalf("expected error for missing %s, got nil", tc.missing)
			}

			domainErr, ok := err.(*domain.Error)
			if !ok {
				t.Fatalf("expected domain.Error, got %T", err)
			}

			if domainErr.Code != domain.InvalidParams {
				t.Errorf("expected error code %d, got %d", domain.InvalidParams, domainErr.Code)
			}
		})
	}
}

func TestBitbucketHandler_HandleCreatePullRequest(t *testing.T) {
	server := setupMockBitbucketServer()
	defer server.Close()

	client := infrastructure.NewBitbucketClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBitbucketHandler(client, mapper)

	req := &domain.ToolRequest{
		Name: ToolBitbucketCreatePullRequest,
		Arguments: map[string]interface{}{
			"project":     "PROJ",
			"repo":        "test-repo",
			"title":       "New PR",
			"description": "New description",
			"fromRef":     "feature-branch",
			"toRef":       "main",
		},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

func TestBitbucketHandler_HandleCreatePullRequest_WithoutDescription(t *testing.T) {
	server := setupMockBitbucketServer()
	defer server.Close()

	client := infrastructure.NewBitbucketClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBitbucketHandler(client, mapper)

	req := &domain.ToolRequest{
		Name: ToolBitbucketCreatePullRequest,
		Arguments: map[string]interface{}{
			"project": "PROJ",
			"repo":    "test-repo",
			"title":   "New PR",
			"fromRef": "feature-branch",
			"toRef":   "main",
		},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

func TestBitbucketHandler_HandleCreatePullRequest_MissingParameters(t *testing.T) {
	server := setupMockBitbucketServer()
	defer server.Close()

	client := infrastructure.NewBitbucketClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBitbucketHandler(client, mapper)

	testCases := []struct {
		name      string
		arguments map[string]interface{}
		missing   string
	}{
		{
			name:      "missing project",
			arguments: map[string]interface{}{"repo": "test-repo", "title": "PR", "fromRef": "feature", "toRef": "main"},
			missing:   "project",
		},
		{
			name:      "missing repo",
			arguments: map[string]interface{}{"project": "PROJ", "title": "PR", "fromRef": "feature", "toRef": "main"},
			missing:   "repo",
		},
		{
			name:      "missing title",
			arguments: map[string]interface{}{"project": "PROJ", "repo": "test-repo", "fromRef": "feature", "toRef": "main"},
			missing:   "title",
		},
		{
			name:      "missing fromRef",
			arguments: map[string]interface{}{"project": "PROJ", "repo": "test-repo", "title": "PR", "toRef": "main"},
			missing:   "fromRef",
		},
		{
			name:      "missing toRef",
			arguments: map[string]interface{}{"project": "PROJ", "repo": "test-repo", "title": "PR", "fromRef": "feature"},
			missing:   "toRef",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &domain.ToolRequest{
				Name:      ToolBitbucketCreatePullRequest,
				Arguments: tc.arguments,
			}

			_, err := handler.Handle(context.Background(), req)
			if err == nil {
				t.Fatalf("expected error for missing %s, got nil", tc.missing)
			}

			domainErr, ok := err.(*domain.Error)
			if !ok {
				t.Fatalf("expected domain.Error, got %T", err)
			}

			if domainErr.Code != domain.InvalidParams {
				t.Errorf("expected error code %d, got %d", domain.InvalidParams, domainErr.Code)
			}
		})
	}
}

func TestBitbucketHandler_HandleMergePullRequest(t *testing.T) {
	server := setupMockBitbucketServer()
	defer server.Close()

	client := infrastructure.NewBitbucketClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBitbucketHandler(client, mapper)

	req := &domain.ToolRequest{
		Name: ToolBitbucketMergePullRequest,
		Arguments: map[string]interface{}{
			"project": "PROJ",
			"repo":    "test-repo",
			"prId":    float64(1),
			"version": float64(1),
		},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

func TestBitbucketHandler_HandleMergePullRequest_MissingParameters(t *testing.T) {
	server := setupMockBitbucketServer()
	defer server.Close()

	client := infrastructure.NewBitbucketClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBitbucketHandler(client, mapper)

	testCases := []struct {
		name      string
		arguments map[string]interface{}
		missing   string
	}{
		{
			name:      "missing project",
			arguments: map[string]interface{}{"repo": "test-repo", "prId": float64(1), "version": float64(1)},
			missing:   "project",
		},
		{
			name:      "missing repo",
			arguments: map[string]interface{}{"project": "PROJ", "prId": float64(1), "version": float64(1)},
			missing:   "repo",
		},
		{
			name:      "missing prId",
			arguments: map[string]interface{}{"project": "PROJ", "repo": "test-repo", "version": float64(1)},
			missing:   "prId",
		},
		{
			name:      "missing version",
			arguments: map[string]interface{}{"project": "PROJ", "repo": "test-repo", "prId": float64(1)},
			missing:   "version",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &domain.ToolRequest{
				Name:      ToolBitbucketMergePullRequest,
				Arguments: tc.arguments,
			}

			_, err := handler.Handle(context.Background(), req)
			if err == nil {
				t.Fatalf("expected error for missing %s, got nil", tc.missing)
			}

			domainErr, ok := err.(*domain.Error)
			if !ok {
				t.Fatalf("expected domain.Error, got %T", err)
			}

			if domainErr.Code != domain.InvalidParams {
				t.Errorf("expected error code %d, got %d", domain.InvalidParams, domainErr.Code)
			}
		})
	}
}

func TestBitbucketHandler_HandleGetCommits(t *testing.T) {
	server := setupMockBitbucketServer()
	defer server.Close()

	client := infrastructure.NewBitbucketClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBitbucketHandler(client, mapper)

	req := &domain.ToolRequest{
		Name: ToolBitbucketGetCommits,
		Arguments: map[string]interface{}{
			"project": "PROJ",
			"repo":    "test-repo",
		},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

func TestBitbucketHandler_HandleGetCommits_WithOptions(t *testing.T) {
	server := setupMockBitbucketServer()
	defer server.Close()

	client := infrastructure.NewBitbucketClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBitbucketHandler(client, mapper)

	req := &domain.ToolRequest{
		Name: ToolBitbucketGetCommits,
		Arguments: map[string]interface{}{
			"project": "PROJ",
			"repo":    "test-repo",
			"until":   "main",
			"since":   "develop",
			"path":    "src/",
			"limit":   float64(10),
			"start":   float64(0),
		},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

func TestBitbucketHandler_HandleGetCommits_MissingParameters(t *testing.T) {
	server := setupMockBitbucketServer()
	defer server.Close()

	client := infrastructure.NewBitbucketClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBitbucketHandler(client, mapper)

	testCases := []struct {
		name      string
		arguments map[string]interface{}
		missing   string
	}{
		{
			name:      "missing project",
			arguments: map[string]interface{}{"repo": "test-repo"},
			missing:   "project",
		},
		{
			name:      "missing repo",
			arguments: map[string]interface{}{"project": "PROJ"},
			missing:   "repo",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &domain.ToolRequest{
				Name:      ToolBitbucketGetCommits,
				Arguments: tc.arguments,
			}

			_, err := handler.Handle(context.Background(), req)
			if err == nil {
				t.Fatalf("expected error for missing %s, got nil", tc.missing)
			}

			domainErr, ok := err.(*domain.Error)
			if !ok {
				t.Fatalf("expected domain.Error, got %T", err)
			}

			if domainErr.Code != domain.InvalidParams {
				t.Errorf("expected error code %d, got %d", domain.InvalidParams, domainErr.Code)
			}
		})
	}
}

func TestBitbucketHandler_HandleGetFileContent(t *testing.T) {
	server := setupMockBitbucketServer()
	defer server.Close()

	client := infrastructure.NewBitbucketClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBitbucketHandler(client, mapper)

	req := &domain.ToolRequest{
		Name: ToolBitbucketGetFileContent,
		Arguments: map[string]interface{}{
			"project": "PROJ",
			"repo":    "test-repo",
			"path":    "README.md",
		},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

func TestBitbucketHandler_HandleGetFileContent_WithRef(t *testing.T) {
	server := setupMockBitbucketServer()
	defer server.Close()

	client := infrastructure.NewBitbucketClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBitbucketHandler(client, mapper)

	req := &domain.ToolRequest{
		Name: ToolBitbucketGetFileContent,
		Arguments: map[string]interface{}{
			"project": "PROJ",
			"repo":    "test-repo",
			"path":    "README.md",
			"ref":     "main",
		},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

func TestBitbucketHandler_HandleGetFileContent_MissingParameters(t *testing.T) {
	server := setupMockBitbucketServer()
	defer server.Close()

	client := infrastructure.NewBitbucketClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBitbucketHandler(client, mapper)

	testCases := []struct {
		name      string
		arguments map[string]interface{}
		missing   string
	}{
		{
			name:      "missing project",
			arguments: map[string]interface{}{"repo": "test-repo", "path": "README.md"},
			missing:   "project",
		},
		{
			name:      "missing repo",
			arguments: map[string]interface{}{"project": "PROJ", "path": "README.md"},
			missing:   "repo",
		},
		{
			name:      "missing path",
			arguments: map[string]interface{}{"project": "PROJ", "repo": "test-repo"},
			missing:   "path",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &domain.ToolRequest{
				Name:      ToolBitbucketGetFileContent,
				Arguments: tc.arguments,
			}

			_, err := handler.Handle(context.Background(), req)
			if err == nil {
				t.Fatalf("expected error for missing %s, got nil", tc.missing)
			}

			domainErr, ok := err.(*domain.Error)
			if !ok {
				t.Fatalf("expected domain.Error, got %T", err)
			}

			if domainErr.Code != domain.InvalidParams {
				t.Errorf("expected error code %d, got %d", domain.InvalidParams, domainErr.Code)
			}
		})
	}
}

func TestBitbucketHandler_HandleUnknownTool(t *testing.T) {
	server := setupMockBitbucketServer()
	defer server.Close()

	client := infrastructure.NewBitbucketClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBitbucketHandler(client, mapper)

	req := &domain.ToolRequest{
		Name:      "bitbucket_unknown_tool",
		Arguments: map[string]interface{}{},
	}

	_, err := handler.Handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for unknown tool, got nil")
	}

	domainErr, ok := err.(*domain.Error)
	if !ok {
		t.Fatalf("expected domain.Error, got %T", err)
	}

	if domainErr.Code != domain.MethodNotFound {
		t.Errorf("expected error code %d, got %d", domain.MethodNotFound, domainErr.Code)
	}
}

func TestBitbucketHandler_ParameterValidation_InvalidType(t *testing.T) {
	server := setupMockBitbucketServer()
	defer server.Close()

	client := infrastructure.NewBitbucketClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBitbucketHandler(client, mapper)

	testCases := []struct {
		name      string
		toolName  string
		arguments map[string]interface{}
		paramName string
	}{
		{
			name:      "getRepositories with non-string project",
			toolName:  ToolBitbucketGetRepositories,
			arguments: map[string]interface{}{"project": 123},
			paramName: "project",
		},
		{
			name:      "getBranches with non-string repo",
			toolName:  ToolBitbucketGetBranches,
			arguments: map[string]interface{}{"project": "PROJ", "repo": 123},
			paramName: "repo",
		},
		{
			name:      "createBranch with non-string name",
			toolName:  ToolBitbucketCreateBranch,
			arguments: map[string]interface{}{"project": "PROJ", "repo": "test-repo", "name": 123, "startPoint": "main"},
			paramName: "name",
		},
		{
			name:      "getPullRequest with non-integer prId",
			toolName:  ToolBitbucketGetPullRequest,
			arguments: map[string]interface{}{"project": "PROJ", "repo": "test-repo", "prId": "invalid"},
			paramName: "prId",
		},
		{
			name:      "createPullRequest with non-string title",
			toolName:  ToolBitbucketCreatePullRequest,
			arguments: map[string]interface{}{"project": "PROJ", "repo": "test-repo", "title": 123, "fromRef": "feature", "toRef": "main"},
			paramName: "title",
		},
		{
			name:      "mergePullRequest with non-integer version",
			toolName:  ToolBitbucketMergePullRequest,
			arguments: map[string]interface{}{"project": "PROJ", "repo": "test-repo", "prId": float64(1), "version": "invalid"},
			paramName: "version",
		},
		{
			name:      "getFileContent with non-string path",
			toolName:  ToolBitbucketGetFileContent,
			arguments: map[string]interface{}{"project": "PROJ", "repo": "test-repo", "path": 123},
			paramName: "path",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &domain.ToolRequest{
				Name:      tc.toolName,
				Arguments: tc.arguments,
			}

			_, err := handler.Handle(context.Background(), req)
			if err == nil {
				t.Fatalf("expected error for invalid type of %s, got nil", tc.paramName)
			}

			domainErr, ok := err.(*domain.Error)
			if !ok {
				t.Fatalf("expected domain.Error, got %T", err)
			}

			if domainErr.Code != domain.InvalidParams {
				t.Errorf("expected error code %d, got %d", domain.InvalidParams, domainErr.Code)
			}
		})
	}
}

func TestBitbucketHandler_IntegerParameterValidation(t *testing.T) {
	server := setupMockBitbucketServer()
	defer server.Close()

	client := infrastructure.NewBitbucketClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBitbucketHandler(client, mapper)

	// Test with float64 (from JSON)
	req := &domain.ToolRequest{
		Name: ToolBitbucketGetPullRequest,
		Arguments: map[string]interface{}{
			"project": "PROJ",
			"repo":    "test-repo",
			"prId":    float64(1),
		},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error with float64: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	// Test with int
	req2 := &domain.ToolRequest{
		Name: ToolBitbucketGetPullRequest,
		Arguments: map[string]interface{}{
			"project": "PROJ",
			"repo":    "test-repo",
			"prId":    1,
		},
	}

	resp2, err := handler.Handle(context.Background(), req2)
	if err != nil {
		t.Fatalf("unexpected error with int: %v", err)
	}

	if resp2 == nil {
		t.Fatal("expected response, got nil")
	}

	// Test with invalid type
	req3 := &domain.ToolRequest{
		Name: ToolBitbucketGetPullRequest,
		Arguments: map[string]interface{}{
			"project": "PROJ",
			"repo":    "test-repo",
			"prId":    "invalid",
		},
	}

	_, err = handler.Handle(context.Background(), req3)
	if err == nil {
		t.Fatal("expected error for invalid integer type, got nil")
	}

	domainErr, ok := err.(*domain.Error)
	if !ok {
		t.Fatalf("expected domain.Error, got %T", err)
	}

	if domainErr.Code != domain.InvalidParams {
		t.Errorf("expected error code %d, got %d", domain.InvalidParams, domainErr.Code)
	}
}

func TestBitbucketHandler_NilArguments(t *testing.T) {
	server := setupMockBitbucketServer()
	defer server.Close()

	client := infrastructure.NewBitbucketClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBitbucketHandler(client, mapper)

	req := &domain.ToolRequest{
		Name:      ToolBitbucketGetRepositories,
		Arguments: nil,
	}

	_, err := handler.Handle(context.Background(), req)
	// Should fail because project is required
	if err == nil {
		t.Fatal("expected error for missing required parameter, got nil")
	}
}

func TestBitbucketHandler_HTTPErrorHandling(t *testing.T) {
	testCases := []struct {
		name           string
		statusCode     int
		expectedErrMsg string
	}{
		{
			name:           "404 Not Found",
			statusCode:     http.StatusNotFound,
			expectedErrMsg: "404",
		},
		{
			name:           "401 Unauthorized",
			statusCode:     http.StatusUnauthorized,
			expectedErrMsg: "401",
		},
		{
			name:           "403 Forbidden",
			statusCode:     http.StatusForbidden,
			expectedErrMsg: "403",
		},
		{
			name:           "500 Internal Server Error",
			statusCode:     http.StatusInternalServerError,
			expectedErrMsg: "500",
		},
		{
			name:           "503 Service Unavailable",
			statusCode:     http.StatusServiceUnavailable,
			expectedErrMsg: "503",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock server that returns the specific error code
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "Test error",
				})
			}))
			defer server.Close()

			client := infrastructure.NewBitbucketClient(server.URL, server.Client())
			mapper := &mockResponseMapper{}
			handler := NewBitbucketHandler(client, mapper)

			req := &domain.ToolRequest{
				Name: ToolBitbucketGetRepositories,
				Arguments: map[string]interface{}{
					"project": "PROJ",
				},
			}

			_, err := handler.Handle(context.Background(), req)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			// Verify the error contains the status code
			if !contains(err.Error(), tc.expectedErrMsg) {
				t.Errorf("expected error to contain '%s', got: %v", tc.expectedErrMsg, err)
			}
		})
	}
}

func TestBitbucketHandler_ClientErrorPropagation(t *testing.T) {
	// Create a server that closes immediately to simulate network error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Close the connection immediately
		hj, ok := w.(http.Hijacker)
		if ok {
			conn, _, _ := hj.Hijack()
			conn.Close()
		}
	}))
	server.Close() // Close the server to ensure connection failures

	client := infrastructure.NewBitbucketClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBitbucketHandler(client, mapper)

	req := &domain.ToolRequest{
		Name: ToolBitbucketGetRepositories,
		Arguments: map[string]interface{}{
			"project": "PROJ",
		},
	}

	_, err := handler.Handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for closed connection, got nil")
	}
}

func TestBitbucketHandler_ResponseMapperIntegration(t *testing.T) {
	server := setupMockBitbucketServer()
	defer server.Close()

	client := infrastructure.NewBitbucketClient(server.URL, server.Client())

	// Use the real response mapper
	mapper := domain.NewResponseMapper()
	handler := NewBitbucketHandler(client, mapper)

	t.Run("successful response mapping", func(t *testing.T) {
		req := &domain.ToolRequest{
			Name: ToolBitbucketGetRepositories,
			Arguments: map[string]interface{}{
				"project": "PROJ",
			},
		}

		resp, err := handler.Handle(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp == nil {
			t.Fatal("expected response, got nil")
		}

		if len(resp.Content) == 0 {
			t.Fatal("expected content in response")
		}

		// Verify the content is valid JSON
		var jsonData interface{}
		if err := json.Unmarshal([]byte(resp.Content[0].Text), &jsonData); err != nil {
			t.Fatalf("response is not valid JSON: %v", err)
		}
	})

	t.Run("error response mapping", func(t *testing.T) {
		// Create a server that returns 404
		errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Repository not found",
			})
		}))
		defer errorServer.Close()

		errorClient := infrastructure.NewBitbucketClient(errorServer.URL, errorServer.Client())
		errorHandler := NewBitbucketHandler(errorClient, mapper)

		req := &domain.ToolRequest{
			Name: ToolBitbucketGetRepositories,
			Arguments: map[string]interface{}{
				"project": "NOTFOUND",
			},
		}

		_, err := errorHandler.Handle(context.Background(), req)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		// Verify it's a domain error with proper structure
		domainErr, ok := err.(*domain.Error)
		if !ok {
			t.Fatalf("expected domain.Error, got %T", err)
		}

		if domainErr.Code == 0 {
			t.Error("expected non-zero error code")
		}

		if domainErr.Message == "" {
			t.Error("expected error message")
		}
	})
}

func TestBitbucketHandler_AllToolsHaveValidSchemas(t *testing.T) {
	handler := NewBitbucketHandler(nil, nil)
	tools := handler.ListTools()

	for _, tool := range tools {
		t.Run(tool.Name, func(t *testing.T) {
			// Verify basic schema structure
			if tool.InputSchema.Type != "object" {
				t.Errorf("expected schema type 'object', got '%s'", tool.InputSchema.Type)
			}

			// Verify required fields are in properties
			for _, requiredField := range tool.InputSchema.Required {
				if _, exists := tool.InputSchema.Properties[requiredField]; !exists {
					t.Errorf("required field '%s' not found in properties", requiredField)
				}
			}

			// Verify all properties have type and description
			for propName, propValue := range tool.InputSchema.Properties {
				propMap, ok := propValue.(map[string]interface{})
				if !ok {
					t.Errorf("property '%s' is not a map", propName)
					continue
				}

				if _, hasType := propMap["type"]; !hasType {
					t.Errorf("property '%s' missing type", propName)
				}

				if _, hasDesc := propMap["description"]; !hasDesc {
					t.Errorf("property '%s' missing description", propName)
				}
			}
		})
	}
}

func TestBitbucketHandler_ContextPropagation(t *testing.T) {
	server := setupMockBitbucketServer()
	defer server.Close()

	client := infrastructure.NewBitbucketClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBitbucketHandler(client, mapper)

	// Create a context with a value
	ctx := context.WithValue(context.Background(), "test-key", "test-value")

	req := &domain.ToolRequest{
		Name: ToolBitbucketGetRepositories,
		Arguments: map[string]interface{}{
			"project": "PROJ",
		},
	}

	// This should not panic or error due to context
	resp, err := handler.Handle(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

func TestBitbucketHandler_EdgeCases(t *testing.T) {
	server := setupMockBitbucketServer()
	defer server.Close()

	client := infrastructure.NewBitbucketClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBitbucketHandler(client, mapper)

	t.Run("empty string parameters", func(t *testing.T) {
		// Empty strings should be accepted (they're still strings)
		req := &domain.ToolRequest{
			Name: ToolBitbucketCreatePullRequest,
			Arguments: map[string]interface{}{
				"project":     "PROJ",
				"repo":        "test-repo",
				"title":       "",
				"description": "",
				"fromRef":     "feature",
				"toRef":       "main",
			},
		}

		// This should not fail validation (empty strings are valid)
		_, err := handler.Handle(context.Background(), req)
		// The error here would be from the API, not from validation
		// We're just checking that validation doesn't reject empty strings
		if err != nil {
			domainErr, ok := err.(*domain.Error)
			if ok && domainErr.Code == domain.InvalidParams {
				t.Error("empty strings should not fail parameter validation")
			}
		}
	})

	t.Run("special characters in parameters", func(t *testing.T) {
		req := &domain.ToolRequest{
			Name: ToolBitbucketGetFileContent,
			Arguments: map[string]interface{}{
				"project": "PROJ",
				"repo":    "test-repo",
				"path":    "src/special-chars!@#$%/file.txt",
			},
		}

		// Should not fail validation
		_, err := handler.Handle(context.Background(), req)
		if err != nil {
			domainErr, ok := err.(*domain.Error)
			if ok && domainErr.Code == domain.InvalidParams {
				t.Error("special characters should not fail parameter validation")
			}
		}
	})
}

func TestBitbucketHandler_OptionalParameterCombinations(t *testing.T) {
	server := setupMockBitbucketServer()
	defer server.Close()

	client := infrastructure.NewBitbucketClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBitbucketHandler(client, mapper)

	t.Run("createPullRequest with all optional parameters", func(t *testing.T) {
		req := &domain.ToolRequest{
			Name: ToolBitbucketCreatePullRequest,
			Arguments: map[string]interface{}{
				"project":     "PROJ",
				"repo":        "test-repo",
				"title":       "Test PR",
				"description": "Test description",
				"fromRef":     "feature-branch",
				"toRef":       "main",
			},
		}

		resp, err := handler.Handle(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected response, got nil")
		}
	})

	t.Run("createPullRequest with no optional parameters", func(t *testing.T) {
		req := &domain.ToolRequest{
			Name: ToolBitbucketCreatePullRequest,
			Arguments: map[string]interface{}{
				"project": "PROJ",
				"repo":    "test-repo",
				"title":   "Test PR",
				"fromRef": "feature-branch",
				"toRef":   "main",
			},
		}

		resp, err := handler.Handle(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected response, got nil")
		}
	})

	t.Run("getCommits with all optional parameters", func(t *testing.T) {
		req := &domain.ToolRequest{
			Name: ToolBitbucketGetCommits,
			Arguments: map[string]interface{}{
				"project": "PROJ",
				"repo":    "test-repo",
				"until":   "main",
				"since":   "develop",
				"path":    "src/",
				"limit":   float64(10),
				"start":   float64(0),
			},
		}

		resp, err := handler.Handle(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected response, got nil")
		}
	})

	t.Run("getCommits with no optional parameters", func(t *testing.T) {
		req := &domain.ToolRequest{
			Name: ToolBitbucketGetCommits,
			Arguments: map[string]interface{}{
				"project": "PROJ",
				"repo":    "test-repo",
			},
		}

		resp, err := handler.Handle(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected response, got nil")
		}
	})

	t.Run("getFileContent with ref", func(t *testing.T) {
		req := &domain.ToolRequest{
			Name: ToolBitbucketGetFileContent,
			Arguments: map[string]interface{}{
				"project": "PROJ",
				"repo":    "test-repo",
				"path":    "README.md",
				"ref":     "main",
			},
		}

		resp, err := handler.Handle(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected response, got nil")
		}
	})

	t.Run("getFileContent without ref", func(t *testing.T) {
		req := &domain.ToolRequest{
			Name: ToolBitbucketGetFileContent,
			Arguments: map[string]interface{}{
				"project": "PROJ",
				"repo":    "test-repo",
				"path":    "README.md",
			},
		}

		resp, err := handler.Handle(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected response, got nil")
		}
	})
}
