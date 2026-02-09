package infrastructure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"atlassian-mcp-server/internal/domain"
)

// JiraClient handles Jira Server 9.12 API interactions.
// It implements the AtlassianClient interface and provides methods
// for all Jira operations required by the MCP server.
type JiraClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewJiraClient creates a new Jira API client.
// The baseURL should be the root URL of the Jira Server instance (e.g., "https://jira.example.com").
// The httpClient should be an authenticated client from the AuthenticationManager.
func NewJiraClient(baseURL string, httpClient *http.Client) *JiraClient {
	return &JiraClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// BaseURL returns the configured base URL for the Jira instance.
func (c *JiraClient) BaseURL() string {
	return c.baseURL
}

// Do executes an HTTP request with authentication.
// This method is part of the AtlassianClient interface.
func (c *JiraClient) Do(req *http.Request) (*http.Response, error) {
	// Set common headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Execute the request using the authenticated HTTP client
	return c.httpClient.Do(req)
}

// GetIssue retrieves a Jira issue by its key (e.g., "TEST-123").
// Returns the issue details or an error if the issue doesn't exist or cannot be retrieved.
func (c *JiraClient) GetIssue(issueKey string) (*domain.JiraIssue, error) {
	// Construct the API endpoint
	endpoint := fmt.Sprintf("%s/rest/api/2/issue/%s", c.baseURL, issueKey)

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
	var issue domain.JiraIssue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &issue, nil
}

// CreateIssue creates a new Jira issue.
// Returns the created issue with its assigned key and ID.
func (c *JiraClient) CreateIssue(issue *domain.JiraIssueCreate) (*domain.JiraIssue, error) {
	// Construct the API endpoint
	endpoint := fmt.Sprintf("%s/rest/api/2/issue", c.baseURL)

	// Marshal the issue to JSON
	body, err := json.Marshal(issue)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal issue: %w", err)
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
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var createdIssue domain.JiraIssue
	if err := json.NewDecoder(resp.Body).Decode(&createdIssue); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &createdIssue, nil
}

// UpdateIssue updates an existing Jira issue.
// The issueKey identifies the issue to update (e.g., "TEST-123").
// Returns an error if the update fails.
func (c *JiraClient) UpdateIssue(issueKey string, update *domain.JiraIssueUpdate) error {
	// Construct the API endpoint
	endpoint := fmt.Sprintf("%s/rest/api/2/issue/%s", c.baseURL, issueKey)

	// Marshal the update to JSON
	body, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("failed to marshal update: %w", err)
	}

	// Create the HTTP request
	req, err := http.NewRequest("PUT", endpoint, bytes.NewReader(body))
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
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteIssue deletes a Jira issue.
// The issueKey identifies the issue to delete (e.g., "TEST-123").
// Returns an error if the deletion fails.
func (c *JiraClient) DeleteIssue(issueKey string) error {
	// Construct the API endpoint
	endpoint := fmt.Sprintf("%s/rest/api/2/issue/%s", c.baseURL, issueKey)

	// Create the HTTP request
	req, err := http.NewRequest("DELETE", endpoint, nil)
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
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// SearchOptions contains options for JQL search operations.
type SearchOptions struct {
	JQL        string   // The JQL query string
	StartAt    int      // The index of the first issue to return (0-based)
	MaxResults int      // The maximum number of issues to return
	Fields     []string // The fields to include in the response (optional)
}

// SearchJQL performs a JQL (Jira Query Language) search.
// Returns search results including issues and pagination metadata.
func (c *JiraClient) SearchJQL(jql string, options *SearchOptions) (*domain.SearchResults, error) {
	// Construct the API endpoint
	endpoint := fmt.Sprintf("%s/rest/api/2/search", c.baseURL)

	// Build query parameters
	params := url.Values{}
	params.Set("jql", jql)

	if options != nil {
		if options.StartAt > 0 {
			params.Set("startAt", fmt.Sprintf("%d", options.StartAt))
		}
		if options.MaxResults > 0 {
			params.Set("maxResults", fmt.Sprintf("%d", options.MaxResults))
		}
		if len(options.Fields) > 0 {
			for _, field := range options.Fields {
				params.Add("fields", field)
			}
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
	var results domain.SearchResults
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &results, nil
}

// TransitionIssue transitions a Jira issue to a new status.
// The issueKey identifies the issue (e.g., "TEST-123").
// The transition specifies the workflow transition to perform.
func (c *JiraClient) TransitionIssue(issueKey string, transition *domain.IssueTransition) error {
	// Construct the API endpoint
	endpoint := fmt.Sprintf("%s/rest/api/2/issue/%s/transitions", c.baseURL, issueKey)

	// Marshal the transition to JSON
	body, err := json.Marshal(transition)
	if err != nil {
		return fmt.Errorf("failed to marshal transition: %w", err)
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
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
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// AddComment adds a comment to a Jira issue.
// The issueKey identifies the issue (e.g., "TEST-123").
// Returns an error if the comment cannot be added.
func (c *JiraClient) AddComment(issueKey string, comment *domain.Comment) error {
	// Construct the API endpoint
	endpoint := fmt.Sprintf("%s/rest/api/2/issue/%s/comment", c.baseURL, issueKey)

	// Marshal the comment to JSON
	body, err := json.Marshal(comment)
	if err != nil {
		return fmt.Errorf("failed to marshal comment: %w", err)
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
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
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetProjects retrieves all projects accessible to the authenticated user.
// Returns a list of projects or an error if the request fails.
func (c *JiraClient) GetProjects() ([]domain.Project, error) {
	// Construct the API endpoint
	endpoint := fmt.Sprintf("%s/rest/api/2/project", c.baseURL)

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
	var projects []domain.Project
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return projects, nil
}
