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

// ConfluenceClient handles Confluence Server 8.15 API interactions.
// It implements the AtlassianClient interface and provides methods
// for all Confluence operations required by the MCP server.
type ConfluenceClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewConfluenceClient creates a new Confluence API client.
// The baseURL should be the root URL of the Confluence Server instance (e.g., "https://confluence.example.com").
// The httpClient should be an authenticated client from the AuthenticationManager.
func NewConfluenceClient(baseURL string, httpClient *http.Client) *ConfluenceClient {
	return &ConfluenceClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// BaseURL returns the configured base URL for the Confluence instance.
func (c *ConfluenceClient) BaseURL() string {
	return c.baseURL
}

// Do executes an HTTP request with authentication.
// This method is part of the AtlassianClient interface.
func (c *ConfluenceClient) Do(req *http.Request) (*http.Response, error) {
	// Set common headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Execute the request using the authenticated HTTP client
	return c.httpClient.Do(req)
}

// GetPage retrieves a Confluence page by its ID.
// Returns the page details or an error if the page doesn't exist or cannot be retrieved.
func (c *ConfluenceClient) GetPage(pageID string) (*domain.ConfluencePage, error) {
	// Construct the API endpoint
	// Confluence REST API v1: /rest/api/content/{id}
	endpoint := fmt.Sprintf("%s/rest/api/content/%s", c.baseURL, pageID)

	// Add expand parameter to get full page details including body and version
	params := url.Values{}
	params.Set("expand", "body.storage,version,space")
	endpoint = endpoint + "?" + params.Encode()

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
	var page domain.ConfluencePage
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &page, nil
}

// CreatePage creates a new Confluence page.
// Returns the created page with its assigned ID.
func (c *ConfluenceClient) CreatePage(page *domain.PageCreate) (*domain.ConfluencePage, error) {
	// Construct the API endpoint
	endpoint := fmt.Sprintf("%s/rest/api/content", c.baseURL)

	// Marshal the page to JSON
	body, err := json.Marshal(page)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal page: %w", err)
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
	var createdPage domain.ConfluencePage
	if err := json.NewDecoder(resp.Body).Decode(&createdPage); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &createdPage, nil
}

// UpdatePage updates an existing Confluence page.
// The pageID identifies the page to update.
// Returns an error if the update fails.
func (c *ConfluenceClient) UpdatePage(pageID string, update *domain.PageUpdate) (*domain.ConfluencePage, error) {
	// Construct the API endpoint
	endpoint := fmt.Sprintf("%s/rest/api/content/%s", c.baseURL, pageID)

	// Marshal the update to JSON
	body, err := json.Marshal(update)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal update: %w", err)
	}

	// Create the HTTP request
	req, err := http.NewRequest("PUT", endpoint, bytes.NewReader(body))
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

	// Parse the response - Confluence returns the updated page
	var updatedPage domain.ConfluencePage
	if err := json.NewDecoder(resp.Body).Decode(&updatedPage); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &updatedPage, nil
}

// DeletePage deletes a Confluence page.
// The pageID identifies the page to delete.
// Returns an error if the deletion fails.
func (c *ConfluenceClient) DeletePage(pageID string) error {
	// Construct the API endpoint
	endpoint := fmt.Sprintf("%s/rest/api/content/%s", c.baseURL, pageID)

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

// ConfluenceSearchOptions contains options for CQL search operations.
type ConfluenceSearchOptions struct {
	CQL    string // The CQL query string
	Start  int    // The index of the first result to return (0-based)
	Limit  int    // The maximum number of results to return
	Expand string // Comma-separated list of properties to expand (optional)
}

// ConfluenceSearchResults represents the results of a CQL search.
type ConfluenceSearchResults struct {
	Results []domain.ConfluencePage `json:"results"`
	Start   int                     `json:"start"`
	Limit   int                     `json:"limit"`
	Size    int                     `json:"size"`
	Links   map[string]string       `json:"_links,omitempty"`
}

// SearchCQL performs a CQL (Confluence Query Language) search.
// Returns search results including pages and pagination metadata.
func (c *ConfluenceClient) SearchCQL(cql string, options *ConfluenceSearchOptions) (*ConfluenceSearchResults, error) {
	// Construct the API endpoint
	endpoint := fmt.Sprintf("%s/rest/api/content/search", c.baseURL)

	// Build query parameters
	params := url.Values{}
	params.Set("cql", cql)

	if options != nil {
		if options.Start > 0 {
			params.Set("start", fmt.Sprintf("%d", options.Start))
		}
		if options.Limit > 0 {
			params.Set("limit", fmt.Sprintf("%d", options.Limit))
		}
		if options.Expand != "" {
			params.Set("expand", options.Expand)
		}
	}

	// Add query parameters to endpoint
	endpoint = endpoint + "?" + params.Encode()

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
	var results ConfluenceSearchResults
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &results, nil
}

// GetSpaces retrieves all spaces accessible to the authenticated user.
// Returns a list of spaces or an error if the request fails.
func (c *ConfluenceClient) GetSpaces() ([]domain.Space, error) {
	// Construct the API endpoint
	endpoint := fmt.Sprintf("%s/rest/api/space", c.baseURL)

	// Add query parameter to get more results (default is 25)
	params := url.Values{}
	params.Set("limit", "100")
	endpoint = endpoint + "?" + params.Encode()

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

	// Parse the response - Confluence returns a paginated result
	var response struct {
		Results []domain.Space `json:"results"`
		Start   int            `json:"start"`
		Limit   int            `json:"limit"`
		Size    int            `json:"size"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Results, nil
}

// GetPageHistory retrieves the history information for a Confluence page.
// The pageID identifies the page.
// Returns the page history or an error if the request fails.
func (c *ConfluenceClient) GetPageHistory(pageID string) (*domain.PageHistory, error) {
	// Construct the API endpoint
	endpoint := fmt.Sprintf("%s/rest/api/content/%s/history", c.baseURL, pageID)

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
	var history domain.PageHistory
	if err := json.NewDecoder(resp.Body).Decode(&history); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &history, nil
}
