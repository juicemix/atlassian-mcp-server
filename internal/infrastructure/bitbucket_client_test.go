package infrastructure

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"atlassian-mcp-server/internal/domain"
)

// mockBitbucketServer creates a mock Bitbucket server for testing.
func mockBitbucketServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authentication header
		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"errors":[{"message":"Authentication required"}]}`))
			return
		}

		// Route based on path and method
		switch {
		// Get repositories
		case r.Method == "GET" && r.URL.Path == "/rest/api/1.0/projects/PROJ/repos":
			response := BitbucketRepositoriesResponse{
				Values: []domain.Repository{
					{
						ID:   123,
						Slug: "my-repo",
						Name: "My Repository",
						Project: domain.Project{
							ID:   "1",
							Key:  "PROJ",
							Name: "My Project",
						},
						Public: false,
					},
				},
				Size:       1,
				Limit:      25,
				IsLastPage: true,
				Start:      0,
			}
			json.NewEncoder(w).Encode(response)

		// Get branches
		case r.Method == "GET" && r.URL.Path == "/rest/api/1.0/projects/PROJ/repos/my-repo/branches":
			response := BitbucketBranchesResponse{
				Values: []domain.Branch{
					{
						ID:           "refs/heads/main",
						DisplayID:    "main",
						Type:         "BRANCH",
						LatestCommit: "abc123def456",
						IsDefault:    true,
					},
				},
				Size:       1,
				Limit:      25,
				IsLastPage: true,
				Start:      0,
			}
			json.NewEncoder(w).Encode(response)

		// Create branch
		case r.Method == "POST" && r.URL.Path == "/rest/api/1.0/projects/PROJ/repos/my-repo/branches":
			var branchCreate domain.BranchCreate
			if err := json.NewDecoder(r.Body).Decode(&branchCreate); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			branch := domain.Branch{
				ID:           "refs/heads/" + branchCreate.Name,
				DisplayID:    branchCreate.Name,
				Type:         "BRANCH",
				LatestCommit: "abc123def456",
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(branch)

		// Get pull request
		case r.Method == "GET" && r.URL.Path == "/rest/api/1.0/projects/PROJ/repos/my-repo/pull-requests/1":
			pr := domain.PullRequest{
				ID:      1,
				Version: 1,
				Title:   "Test PR",
				State:   "OPEN",
				Open:    true,
				Closed:  false,
				FromRef: domain.Ref{
					ID:        "refs/heads/feature",
					DisplayID: "feature",
				},
				ToRef: domain.Ref{
					ID:        "refs/heads/main",
					DisplayID: "main",
				},
				Author: domain.User{
					Name:        "jsmith",
					DisplayName: "John Smith",
				},
			}
			json.NewEncoder(w).Encode(pr)

		// Create pull request
		case r.Method == "POST" && r.URL.Path == "/rest/api/1.0/projects/PROJ/repos/my-repo/pull-requests":
			var prCreate domain.PullRequestCreate
			if err := json.NewDecoder(r.Body).Decode(&prCreate); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			pr := domain.PullRequest{
				ID:      1,
				Version: 1,
				Title:   prCreate.Title,
				State:   "OPEN",
				Open:    true,
				Closed:  false,
				FromRef: domain.Ref{
					ID:        prCreate.FromRef.ID,
					DisplayID: "feature",
				},
				ToRef: domain.Ref{
					ID:        prCreate.ToRef.ID,
					DisplayID: "main",
				},
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(pr)

		// Merge pull request
		case r.Method == "POST" && r.URL.Path == "/rest/api/1.0/projects/PROJ/repos/my-repo/pull-requests/1/merge":
			w.WriteHeader(http.StatusOK)
			pr := domain.PullRequest{
				ID:      1,
				Version: 2,
				Title:   "Test PR",
				State:   "MERGED",
				Open:    false,
				Closed:  true,
			}
			json.NewEncoder(w).Encode(pr)

		// Get commits
		case r.Method == "GET" && r.URL.Path == "/rest/api/1.0/projects/PROJ/repos/my-repo/commits":
			response := BitbucketCommitsResponse{
				Values: []domain.Commit{
					{
						ID:              "abc123def456",
						DisplayID:       "abc123d",
						Message:         "Initial commit",
						AuthorTimestamp: 1234567890000,
						Author: domain.User{
							Name:        "jsmith",
							DisplayName: "John Smith",
						},
					},
				},
				Size:       1,
				Limit:      25,
				IsLastPage: true,
				Start:      0,
			}
			json.NewEncoder(w).Encode(response)

		// Get file content
		case r.Method == "GET" && r.URL.Path == "/rest/api/1.0/projects/PROJ/repos/my-repo/browse/README.md":
			response := struct {
				Lines []struct {
					Text string `json:"text"`
				} `json:"lines"`
			}{
				Lines: []struct {
					Text string `json:"text"`
				}{
					{Text: "# My Repository"},
					{Text: "This is a test repository."},
				},
			}
			json.NewEncoder(w).Encode(response)

		// Not found
		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"errors":[{"message":"Not found"}]}`))
		}
	}))
}

// TestBitbucketClientGetRepositories tests the GetRepositories method.
func TestBitbucketClientGetRepositories(t *testing.T) {
	server := mockBitbucketServer()
	defer server.Close()

	// Create authenticated HTTP client
	httpClient := getAuthenticatedClient()

	client := NewBitbucketClient(server.URL, httpClient)

	// Test successful retrieval
	repos, err := client.GetRepositories("PROJ")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(repos) != 1 {
		t.Fatalf("Expected 1 repository, got %d", len(repos))
	}

	if repos[0].Slug != "my-repo" {
		t.Errorf("Expected slug 'my-repo', got '%s'", repos[0].Slug)
	}
}

// TestBitbucketClientGetBranches tests the GetBranches method.
func TestBitbucketClientGetBranches(t *testing.T) {
	server := mockBitbucketServer()
	defer server.Close()

	httpClient := getAuthenticatedClient()

	client := NewBitbucketClient(server.URL, httpClient)

	// Test successful retrieval
	branches, err := client.GetBranches("PROJ", "my-repo")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(branches) != 1 {
		t.Fatalf("Expected 1 branch, got %d", len(branches))
	}

	if branches[0].DisplayID != "main" {
		t.Errorf("Expected branch 'main', got '%s'", branches[0].DisplayID)
	}
}

// TestBitbucketClientCreateBranch tests the CreateBranch method.
func TestBitbucketClientCreateBranch(t *testing.T) {
	server := mockBitbucketServer()
	defer server.Close()

	httpClient := getAuthenticatedClient()

	client := NewBitbucketClient(server.URL, httpClient)

	// Test successful creation
	branchCreate := &domain.BranchCreate{
		Name:       "feature/new-feature",
		StartPoint: "main",
	}

	branch, err := client.CreateBranch("PROJ", "my-repo", branchCreate)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if branch.DisplayID != "feature/new-feature" {
		t.Errorf("Expected branch 'feature/new-feature', got '%s'", branch.DisplayID)
	}
}

// TestBitbucketClientGetPullRequest tests the GetPullRequest method.
func TestBitbucketClientGetPullRequest(t *testing.T) {
	server := mockBitbucketServer()
	defer server.Close()

	httpClient := getAuthenticatedClient()

	client := NewBitbucketClient(server.URL, httpClient)

	// Test successful retrieval
	pr, err := client.GetPullRequest("PROJ", "my-repo", 1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if pr.ID != 1 {
		t.Errorf("Expected PR ID 1, got %d", pr.ID)
	}

	if pr.Title != "Test PR" {
		t.Errorf("Expected title 'Test PR', got '%s'", pr.Title)
	}

	if pr.State != "OPEN" {
		t.Errorf("Expected state 'OPEN', got '%s'", pr.State)
	}
}

// TestBitbucketClientCreatePullRequest tests the CreatePullRequest method.
func TestBitbucketClientCreatePullRequest(t *testing.T) {
	server := mockBitbucketServer()
	defer server.Close()

	httpClient := getAuthenticatedClient()

	client := NewBitbucketClient(server.URL, httpClient)

	// Test successful creation
	prCreate := &domain.PullRequestCreate{
		Title: "New Feature PR",
		FromRef: domain.RefCreate{
			ID: "refs/heads/feature",
		},
		ToRef: domain.RefCreate{
			ID: "refs/heads/main",
		},
	}

	pr, err := client.CreatePullRequest("PROJ", "my-repo", prCreate)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if pr.Title != "New Feature PR" {
		t.Errorf("Expected title 'New Feature PR', got '%s'", pr.Title)
	}

	if pr.State != "OPEN" {
		t.Errorf("Expected state 'OPEN', got '%s'", pr.State)
	}
}

// TestBitbucketClientMergePullRequest tests the MergePullRequest method.
func TestBitbucketClientMergePullRequest(t *testing.T) {
	server := mockBitbucketServer()
	defer server.Close()

	httpClient := getAuthenticatedClient()

	client := NewBitbucketClient(server.URL, httpClient)

	// Test successful merge
	err := client.MergePullRequest("PROJ", "my-repo", 1, 1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

// TestBitbucketClientGetCommits tests the GetCommits method.
func TestBitbucketClientGetCommits(t *testing.T) {
	server := mockBitbucketServer()
	defer server.Close()

	httpClient := getAuthenticatedClient()

	client := NewBitbucketClient(server.URL, httpClient)

	// Test successful retrieval
	commits, err := client.GetCommits("PROJ", "my-repo", nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(commits) != 1 {
		t.Fatalf("Expected 1 commit, got %d", len(commits))
	}

	if commits[0].Message != "Initial commit" {
		t.Errorf("Expected message 'Initial commit', got '%s'", commits[0].Message)
	}
}

// TestBitbucketClientGetFileContent tests the GetFileContent method.
func TestBitbucketClientGetFileContent(t *testing.T) {
	server := mockBitbucketServer()
	defer server.Close()

	httpClient := getAuthenticatedClient()

	client := NewBitbucketClient(server.URL, httpClient)

	// Test successful retrieval
	content, err := client.GetFileContent("PROJ", "my-repo", "README.md", "")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expectedContent := "# My Repository\nThis is a test repository."
	if content != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, content)
	}
}

// TestBitbucketClientErrorHandling tests error handling for various status codes.
func TestBitbucketClientErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		expectedError  bool
		errorSubstring string
	}{
		{
			name:           "Unauthorized",
			statusCode:     http.StatusUnauthorized,
			expectedError:  true,
			errorSubstring: "401",
		},
		{
			name:           "Not Found",
			statusCode:     http.StatusNotFound,
			expectedError:  true,
			errorSubstring: "404",
		},
		{
			name:           "Internal Server Error",
			statusCode:     http.StatusInternalServerError,
			expectedError:  true,
			errorSubstring: "500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(`{"errors":[{"message":"Error occurred"}]}`))
			}))
			defer server.Close()

			httpClient := getAuthenticatedClient()

			client := NewBitbucketClient(server.URL, httpClient)

			// Test GetRepositories error handling
			_, err := client.GetRepositories("PROJ")
			if tt.expectedError && err == nil {
				t.Errorf("Expected error, got nil")
			}
			if tt.expectedError && err != nil {
				if tt.errorSubstring != "" && !contains(err.Error(), tt.errorSubstring) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorSubstring, err.Error())
				}
			}
		})
	}
}

// TestBitbucketClientAuthenticationHeaderInclusion tests that authentication headers are included.
func TestBitbucketClientAuthenticationHeaderInclusion(t *testing.T) {
	authHeaderReceived := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if Authorization header is present
		if r.Header.Get("Authorization") != "" {
			authHeaderReceived = true
		}

		// Return a valid response
		response := BitbucketRepositoriesResponse{
			Values:     []domain.Repository{},
			Size:       0,
			Limit:      25,
			IsLastPage: true,
			Start:      0,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	httpClient := getAuthenticatedClient()

	client := NewBitbucketClient(server.URL, httpClient)

	// Make a request
	_, err := client.GetRepositories("PROJ")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !authHeaderReceived {
		t.Error("Expected Authorization header to be included in request")
	}
}
