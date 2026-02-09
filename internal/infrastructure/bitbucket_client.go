package infrastructure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"atlassian-mcp-server/internal/domain"
)

// BitbucketClient handles Bitbucket 8.9 API interactions.
// It implements the AtlassianClient interface and provides methods
// for all Bitbucket operations required by the MCP server.
type BitbucketClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewBitbucketClient creates a new Bitbucket API client.
// The baseURL should be the root URL of the Bitbucket Server instance (e.g., "https://bitbucket.example.com").
// The httpClient should be an authenticated client from the AuthenticationManager.
func NewBitbucketClient(baseURL string, httpClient *http.Client) *BitbucketClient {
	return &BitbucketClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// BaseURL returns the configured base URL for the Bitbucket instance.
func (c *BitbucketClient) BaseURL() string {
	return c.baseURL
}

// Do executes an HTTP request with authentication.
// This method is part of the AtlassianClient interface.
func (c *BitbucketClient) Do(req *http.Request) (*http.Response, error) {
	// Set common headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Execute the request using the authenticated HTTP client
	return c.httpClient.Do(req)
}

// BitbucketRepositoriesResponse represents the paginated response from the repositories API.
type BitbucketRepositoriesResponse struct {
	Values        []domain.Repository `json:"values"`
	Size          int                 `json:"size"`
	Limit         int                 `json:"limit"`
	IsLastPage    bool                `json:"isLastPage"`
	Start         int                 `json:"start"`
	NextPageStart int                 `json:"nextPageStart,omitempty"`
}

// GetRepositories retrieves all repositories for a given project.
// The project parameter is the project key (e.g., "PROJ").
// Returns a list of repositories or an error if the request fails.
func (c *BitbucketClient) GetRepositories(project string) ([]domain.Repository, error) {
	// Construct the API endpoint
	// Bitbucket REST API: /rest/api/1.0/projects/{projectKey}/repos
	endpoint := fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos", c.baseURL, project)

	// Create the HTTP request
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Execute the request
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check for error status codes
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var response BitbucketRepositoriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Values, nil
}

// BitbucketBranchesResponse represents the paginated response from the branches API.
type BitbucketBranchesResponse struct {
	Values        []domain.Branch `json:"values"`
	Size          int             `json:"size"`
	Limit         int             `json:"limit"`
	IsLastPage    bool            `json:"isLastPage"`
	Start         int             `json:"start"`
	NextPageStart int             `json:"nextPageStart,omitempty"`
}

// GetBranches retrieves all branches for a given repository.
// The project parameter is the project key (e.g., "PROJ").
// The repo parameter is the repository slug (e.g., "my-repo").
// Returns a list of branches or an error if the request fails.
func (c *BitbucketClient) GetBranches(project, repo string) ([]domain.Branch, error) {
	// Construct the API endpoint
	// Bitbucket REST API: /rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}/branches
	endpoint := fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s/branches", c.baseURL, project, repo)

	// Create the HTTP request
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Execute the request
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check for error status codes
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var response BitbucketBranchesResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Values, nil
}

// CreateBranch creates a new branch in a repository.
// The project parameter is the project key (e.g., "PROJ").
// The repo parameter is the repository slug (e.g., "my-repo").
// Returns an error if the branch creation fails.
func (c *BitbucketClient) CreateBranch(project, repo string, branch *domain.BranchCreate) (*domain.Branch, error) {
	// Construct the API endpoint
	// Bitbucket REST API: /rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}/branches
	endpoint := fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s/branches", c.baseURL, project, repo)

	// Marshal the branch to JSON
	body, err := json.Marshal(branch)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal branch: %w", err)
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Execute the request
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check for error status codes
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse the response - Bitbucket returns the created branch
	var createdBranch domain.Branch
	if err := json.NewDecoder(resp.Body).Decode(&createdBranch); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &createdBranch, nil
}

// GetPullRequest retrieves a pull request by its ID.
// The project parameter is the project key (e.g., "PROJ").
// The repo parameter is the repository slug (e.g., "my-repo").
// The prID parameter is the pull request ID.
// Returns the pull request details or an error if the request fails.
func (c *BitbucketClient) GetPullRequest(project, repo string, prID int) (*domain.PullRequest, error) {
	// Construct the API endpoint
	// Bitbucket REST API: /rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}/pull-requests/{pullRequestId}
	endpoint := fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d", c.baseURL, project, repo, prID)

	// Create the HTTP request
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Execute the request
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check for error status codes
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var pr domain.PullRequest
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &pr, nil
}

// CreatePullRequest creates a new pull request.
// The project parameter is the project key (e.g., "PROJ").
// The repo parameter is the repository slug (e.g., "my-repo").
// Returns the created pull request or an error if the creation fails.
func (c *BitbucketClient) CreatePullRequest(project, repo string, pr *domain.PullRequestCreate) (*domain.PullRequest, error) {
	// Construct the API endpoint
	// Bitbucket REST API: /rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}/pull-requests
	endpoint := fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s/pull-requests", c.baseURL, project, repo)

	// Marshal the pull request to JSON
	body, err := json.Marshal(pr)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pull request: %w", err)
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Execute the request
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check for error status codes
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var createdPR domain.PullRequest
	if err := json.NewDecoder(resp.Body).Decode(&createdPR); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &createdPR, nil
}

// MergePullRequest merges a pull request.
// The project parameter is the project key (e.g., "PROJ").
// The repo parameter is the repository slug (e.g., "my-repo").
// The prID parameter is the pull request ID.
// Returns an error if the merge fails.
func (c *BitbucketClient) MergePullRequest(project, repo string, prID int, version int) error {
	// Construct the API endpoint
	// Bitbucket REST API: /rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}/pull-requests/{pullRequestId}/merge
	endpoint := fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/merge", c.baseURL, project, repo, prID)

	// Add version as query parameter (required for optimistic locking)
	params := url.Values{}
	params.Set("version", strconv.Itoa(version))
	endpoint = endpoint + "?" + params.Encode()

	// Create the HTTP request with empty body
	req, err := http.NewRequest("POST", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Execute the request
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check for error status codes
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// BitbucketCommitsResponse represents the paginated response from the commits API.
type BitbucketCommitsResponse struct {
	Values        []domain.Commit `json:"values"`
	Size          int             `json:"size"`
	Limit         int             `json:"limit"`
	IsLastPage    bool            `json:"isLastPage"`
	Start         int             `json:"start"`
	NextPageStart int             `json:"nextPageStart,omitempty"`
}

// GetCommits retrieves commit history for a repository.
// The project parameter is the project key (e.g., "PROJ").
// The repo parameter is the repository slug (e.g., "my-repo").
// Returns a list of commits or an error if the request fails.
func (c *BitbucketClient) GetCommits(project, repo string, options *domain.CommitOptions) ([]domain.Commit, error) {
	// Construct the API endpoint
	// Bitbucket REST API: /rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}/commits
	endpoint := fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s/commits", c.baseURL, project, repo)

	// Build query parameters
	params := url.Values{}
	if options != nil {
		if options.Until != "" {
			params.Set("until", options.Until)
		}
		if options.Since != "" {
			params.Set("since", options.Since)
		}
		if options.Path != "" {
			params.Set("path", options.Path)
		}
		if options.Limit > 0 {
			params.Set("limit", strconv.Itoa(options.Limit))
		}
		if options.Start > 0 {
			params.Set("start", strconv.Itoa(options.Start))
		}
	}

	// Add query parameters to endpoint
	if len(params) > 0 {
		endpoint = endpoint + "?" + params.Encode()
	}

	// Create the HTTP request
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Execute the request
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check for error status codes
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var response BitbucketCommitsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Values, nil
}

// GetFileContent retrieves the content of a file from a repository.
// The project parameter is the project key (e.g., "PROJ").
// The repo parameter is the repository slug (e.g., "my-repo").
// The path parameter is the file path within the repository.
// The ref parameter is the branch name or commit ID (optional, defaults to default branch).
// Returns the file content as a string or an error if the request fails.
func (c *BitbucketClient) GetFileContent(project, repo, path, ref string) (string, error) {
	// Construct the API endpoint
	// Bitbucket REST API: /rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}/browse/{path}
	endpoint := fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s/browse/%s", c.baseURL, project, repo, path)

	// Build query parameters
	params := url.Values{}
	if ref != "" {
		params.Set("at", ref)
	}

	// Add query parameters to endpoint
	if len(params) > 0 {
		endpoint = endpoint + "?" + params.Encode()
	}

	// Create the HTTP request
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Execute the request
	resp, err := c.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check for error status codes
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse the response - Bitbucket returns file content in a specific format
	var response struct {
		Lines []struct {
			Text string `json:"text"`
		} `json:"lines"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Concatenate all lines into a single string
	var content string
	for i, line := range response.Lines {
		content += line.Text
		if i < len(response.Lines)-1 {
			content += "\n"
		}
	}

	return content, nil
}
