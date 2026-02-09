package domain

import (
	"net/http"
)

// AtlassianClient defines common operations for all Atlassian tools.
// Each Atlassian tool (Jira, Confluence, Bitbucket, Bamboo) implements
// this interface to provide authenticated API access.
type AtlassianClient interface {
	// BaseURL returns the configured base URL for the tool.
	BaseURL() string

	// Do executes an HTTP request with authentication.
	// The request should already be constructed with the appropriate
	// method, path, headers, and body. This method adds authentication
	// and executes the request.
	Do(req *http.Request) (*http.Response, error)
}
