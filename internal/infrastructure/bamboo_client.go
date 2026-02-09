package infrastructure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"atlassian-mcp-server/internal/domain"
)

// BambooClient handles Bamboo 9.2.7 API interactions.
// It implements the AtlassianClient interface and provides methods
// for all Bamboo operations required by the MCP server.
type BambooClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewBambooClient creates a new Bamboo API client.
// The baseURL should be the root URL of the Bamboo instance (e.g., "https://bamboo.example.com").
// The httpClient should be an authenticated client from the AuthenticationManager.
func NewBambooClient(baseURL string, httpClient *http.Client) *BambooClient {
	return &BambooClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// BaseURL returns the configured base URL for the Bamboo instance.
func (c *BambooClient) BaseURL() string {
	return c.baseURL
}

// Do executes an HTTP request with authentication.
// This method is part of the AtlassianClient interface.
func (c *BambooClient) Do(req *http.Request) (*http.Response, error) {
	// Set common headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Execute the request using the authenticated HTTP client
	return c.httpClient.Do(req)
}

// BambooPlansResponse represents the paginated response from the plans API.
type BambooPlansResponse struct {
	Plans struct {
		Plan []domain.BuildPlan `json:"plan"`
		Size int                `json:"size"`
	} `json:"plans"`
}

// GetPlans retrieves all build plans.
// Returns a list of build plans or an error if the request fails.
func (c *BambooClient) GetPlans() ([]domain.BuildPlan, error) {
	// Construct the API endpoint
	// Bamboo REST API: /rest/api/latest/plan
	endpoint := fmt.Sprintf("%s/rest/api/latest/plan", c.baseURL)

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
	var response BambooPlansResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Plans.Plan, nil
}

// GetPlan retrieves a specific build plan by its key.
// The planKey parameter is the plan key (e.g., "PROJ-PLAN").
// Returns the build plan details or an error if the request fails.
func (c *BambooClient) GetPlan(planKey string) (*domain.BuildPlan, error) {
	// Construct the API endpoint
	// Bamboo REST API: /rest/api/latest/plan/{planKey}
	endpoint := fmt.Sprintf("%s/rest/api/latest/plan/%s", c.baseURL, planKey)

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
	var plan domain.BuildPlan
	if err := json.NewDecoder(resp.Body).Decode(&plan); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &plan, nil
}

// TriggerBuild triggers a build for the specified plan.
// The planKey parameter is the plan key (e.g., "PROJ-PLAN").
// Returns the build result or an error if the trigger fails.
func (c *BambooClient) TriggerBuild(planKey string) (*domain.BuildResult, error) {
	// Construct the API endpoint
	// Bamboo REST API: /rest/api/latest/queue/{planKey}
	endpoint := fmt.Sprintf("%s/rest/api/latest/queue/%s", c.baseURL, planKey)

	// Create the HTTP request with empty body
	req, err := http.NewRequest("POST", endpoint, nil)
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

	// Parse the response - Bamboo returns build result information
	var result domain.BuildResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetBuildResult retrieves the result of a specific build.
// The buildKey parameter is the build result key (e.g., "PROJ-PLAN-123").
// Returns the build result details or an error if the request fails.
func (c *BambooClient) GetBuildResult(buildKey string) (*domain.BuildResult, error) {
	// Construct the API endpoint
	// Bamboo REST API: /rest/api/latest/result/{buildKey}
	endpoint := fmt.Sprintf("%s/rest/api/latest/result/%s", c.baseURL, buildKey)

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
	var result domain.BuildResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetBuildLog retrieves the log output for a specific build.
// The buildKey parameter is the build result key (e.g., "PROJ-PLAN-123").
// Returns the build log as a string or an error if the request fails.
func (c *BambooClient) GetBuildLog(buildKey string) (string, error) {
	// Construct the API endpoint
	// Bamboo REST API: /rest/api/latest/result/{buildKey}/log
	endpoint := fmt.Sprintf("%s/rest/api/latest/result/%s/log", c.baseURL, buildKey)

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

	// Read the log content - Bamboo may return plain text or JSON
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

// BambooDeploymentProjectsResponse represents the response from the deployment projects API.
type BambooDeploymentProjectsResponse struct {
	Projects []domain.DeploymentProject `json:"projects"`
	Size     int                        `json:"size"`
}

// GetDeploymentProjects retrieves all deployment projects.
// Returns a list of deployment projects or an error if the request fails.
func (c *BambooClient) GetDeploymentProjects() ([]domain.DeploymentProject, error) {
	// Construct the API endpoint
	// Bamboo REST API: /rest/api/latest/deploy/project/all
	endpoint := fmt.Sprintf("%s/rest/api/latest/deploy/project/all", c.baseURL)

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

	// Parse the response - could be array or object with array
	// Try to parse as array first
	var projects []domain.DeploymentProject
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		// If that fails, the response might be empty or in a different format
		return []domain.DeploymentProject{}, nil
	}

	return projects, nil
}

// TriggerDeployment triggers a deployment to a specific environment.
// The projectID parameter is the deployment project ID.
// The envID parameter is the environment ID.
// Returns the deployment result or an error if the trigger fails.
func (c *BambooClient) TriggerDeployment(projectID int, envID int) (*domain.DeploymentResult, error) {
	// Construct the API endpoint
	// Bamboo REST API: /rest/api/latest/deploy/environment/{envId}/start
	endpoint := fmt.Sprintf("%s/rest/api/latest/deploy/environment/%d/start", c.baseURL, envID)

	// Create request body with deployment version (using latest)
	requestBody := map[string]interface{}{
		"deploymentProjectId": projectID,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
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

	// Parse the response
	var result domain.DeploymentResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetDeploymentResult retrieves the result of a specific deployment.
// The deploymentResultID parameter is the deployment result ID.
// Returns the deployment result details or an error if the request fails.
func (c *BambooClient) GetDeploymentResult(deploymentResultID int) (*domain.DeploymentResult, error) {
	// Construct the API endpoint
	// Bamboo REST API: /rest/api/latest/deploy/result/{deploymentResultId}
	endpoint := fmt.Sprintf("%s/rest/api/latest/deploy/result/%s", c.baseURL, strconv.Itoa(deploymentResultID))

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
	var result domain.DeploymentResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
