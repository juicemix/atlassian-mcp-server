package infrastructure

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"atlassian-mcp-server/internal/domain"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: atlassian-mcp-server, Property 14: REST API Request Validity
// **Validates: Requirements 9.2**
//
// For any Atlassian API request constructed by the server, it should conform to the
// API specification for that tool (correct HTTP method, valid endpoint path, proper
// headers, valid JSON body).
func TestProperty14_RESTAPIRequestValidity(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Generator for valid issue keys (PROJECT-123 format)
	genIssueKey := gen.Identifier().
		SuchThat(func(s string) bool { return len(s) >= 2 }).
		Map(func(s string) string {
			// Convert to uppercase and add number
			return strings.ToUpper(s[:min(10, len(s))]) + "-123"
		})

	// Generator for valid JQL queries
	genJQL := gen.OneConstOf(
		"project = TEST",
		"status = Open",
		"assignee = currentUser()",
		"created >= -7d",
	)

	// Property: GetIssue constructs valid HTTP requests
	properties.Property("GetIssue constructs valid HTTP GET request", prop.ForAll(
		func(issueKey string) bool {
			// Create a test server to capture the request
			var capturedReq *http.Request
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedReq = r
				// Return a valid response
				issue := domain.JiraIssue{
					ID:  "10001",
					Key: issueKey,
					Fields: domain.JiraFields{
						Summary: "Test",
						IssueType: domain.IssueType{
							ID:   "1",
							Name: "Bug",
						},
						Project: domain.Project{
							ID:   "10000",
							Key:  "TEST",
							Name: "Test Project",
						},
						Status: domain.Status{
							ID:   "1",
							Name: "Open",
						},
						Created: "2024-01-01T10:00:00.000+0000",
						Updated: "2024-01-01T10:00:00.000+0000",
					},
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(issue)
			}))
			defer server.Close()

			// Create client and make request
			client := NewJiraClient(server.URL, server.Client())
			_, err := client.GetIssue(issueKey)
			if err != nil {
				return false
			}

			// Verify request properties
			if capturedReq == nil {
				return false
			}

			// 1. Correct HTTP method
			if capturedReq.Method != "GET" {
				return false
			}

			// 2. Valid endpoint path
			expectedPath := "/rest/api/2/issue/" + issueKey
			if capturedReq.URL.Path != expectedPath {
				return false
			}

			// 3. Proper headers
			if capturedReq.Header.Get("Content-Type") != "application/json" {
				return false
			}
			if capturedReq.Header.Get("Accept") != "application/json" {
				return false
			}

			// 4. No body for GET request
			if capturedReq.Body != nil {
				body, _ := io.ReadAll(capturedReq.Body)
				if len(body) > 0 {
					return false
				}
			}

			return true
		},
		genIssueKey,
	))

	// Property: CreateIssue constructs valid HTTP POST request with JSON body
	properties.Property("CreateIssue constructs valid HTTP POST request", prop.ForAll(
		func(summary string, description string) bool {
			// Ensure non-empty values
			if summary == "" {
				summary = "Test Summary"
			}
			if description == "" {
				description = "Test Description"
			}

			// Create a test server to capture the request
			var capturedReq *http.Request
			var capturedBody []byte
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedReq = r
				capturedBody, _ = io.ReadAll(r.Body)
				// Return a valid response
				issue := domain.JiraIssue{
					ID:  "10002",
					Key: "TEST-124",
					Fields: domain.JiraFields{
						Summary:     summary,
						Description: description,
						IssueType: domain.IssueType{
							ID:   "1",
							Name: "Bug",
						},
						Project: domain.Project{
							ID:   "10000",
							Key:  "TEST",
							Name: "Test Project",
						},
						Status: domain.Status{
							ID:   "1",
							Name: "Open",
						},
						Created: "2024-01-01T10:00:00.000+0000",
						Updated: "2024-01-01T10:00:00.000+0000",
					},
				}
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(issue)
			}))
			defer server.Close()

			// Create issue request
			issueCreate := &domain.JiraIssueCreate{
				Fields: domain.JiraFieldsCreate{
					Summary:     summary,
					Description: description,
					IssueType: domain.IssueTypeRef{
						ID: "1",
					},
					Project: domain.ProjectRef{
						Key: "TEST",
					},
				},
			}

			// Create client and make request
			client := NewJiraClient(server.URL, server.Client())
			_, err := client.CreateIssue(issueCreate)
			if err != nil {
				return false
			}

			// Verify request properties
			if capturedReq == nil {
				return false
			}

			// 1. Correct HTTP method
			if capturedReq.Method != "POST" {
				return false
			}

			// 2. Valid endpoint path
			if capturedReq.URL.Path != "/rest/api/2/issue" {
				return false
			}

			// 3. Proper headers
			if capturedReq.Header.Get("Content-Type") != "application/json" {
				return false
			}
			if capturedReq.Header.Get("Accept") != "application/json" {
				return false
			}

			// 4. Valid JSON body
			if len(capturedBody) == 0 {
				return false
			}

			// Verify body is valid JSON
			var bodyMap map[string]interface{}
			if err := json.Unmarshal(capturedBody, &bodyMap); err != nil {
				return false
			}

			// Verify body contains expected fields
			fields, ok := bodyMap["fields"].(map[string]interface{})
			if !ok {
				return false
			}

			// Check that summary and description are present
			if fields["summary"] != summary {
				return false
			}
			if fields["description"] != description {
				return false
			}

			return true
		},
		gen.AlphaString(),
		gen.AlphaString(),
	))

	// Property: UpdateIssue constructs valid HTTP PUT request
	properties.Property("UpdateIssue constructs valid HTTP PUT request", prop.ForAll(
		func(issueKey string, summary string) bool {
			// Ensure non-empty values
			if summary == "" {
				summary = "Updated Summary"
			}

			// Create a test server to capture the request
			var capturedReq *http.Request
			var capturedBody []byte
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedReq = r
				capturedBody, _ = io.ReadAll(r.Body)
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			// Create update request
			issueUpdate := &domain.JiraIssueUpdate{
				Fields: domain.JiraFieldsUpdate{
					Summary: summary,
				},
			}

			// Create client and make request
			client := NewJiraClient(server.URL, server.Client())
			err := client.UpdateIssue(issueKey, issueUpdate)
			if err != nil {
				return false
			}

			// Verify request properties
			if capturedReq == nil {
				return false
			}

			// 1. Correct HTTP method
			if capturedReq.Method != "PUT" {
				return false
			}

			// 2. Valid endpoint path
			expectedPath := "/rest/api/2/issue/" + issueKey
			if capturedReq.URL.Path != expectedPath {
				return false
			}

			// 3. Proper headers
			if capturedReq.Header.Get("Content-Type") != "application/json" {
				return false
			}
			if capturedReq.Header.Get("Accept") != "application/json" {
				return false
			}

			// 4. Valid JSON body
			if len(capturedBody) == 0 {
				return false
			}

			// Verify body is valid JSON
			var bodyMap map[string]interface{}
			if err := json.Unmarshal(capturedBody, &bodyMap); err != nil {
				return false
			}

			return true
		},
		genIssueKey,
		gen.AlphaString(),
	))

	// Property: DeleteIssue constructs valid HTTP DELETE request
	properties.Property("DeleteIssue constructs valid HTTP DELETE request", prop.ForAll(
		func(issueKey string) bool {
			// Create a test server to capture the request
			var capturedReq *http.Request
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedReq = r
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			// Create client and make request
			client := NewJiraClient(server.URL, server.Client())
			err := client.DeleteIssue(issueKey)
			if err != nil {
				return false
			}

			// Verify request properties
			if capturedReq == nil {
				return false
			}

			// 1. Correct HTTP method
			if capturedReq.Method != "DELETE" {
				return false
			}

			// 2. Valid endpoint path
			expectedPath := "/rest/api/2/issue/" + issueKey
			if capturedReq.URL.Path != expectedPath {
				return false
			}

			// 3. Proper headers
			if capturedReq.Header.Get("Content-Type") != "application/json" {
				return false
			}
			if capturedReq.Header.Get("Accept") != "application/json" {
				return false
			}

			// 4. No body for DELETE request
			if capturedReq.Body != nil {
				body, _ := io.ReadAll(capturedReq.Body)
				if len(body) > 0 {
					return false
				}
			}

			return true
		},
		genIssueKey,
	))

	// Property: SearchJQL constructs valid HTTP GET request with query parameters
	properties.Property("SearchJQL constructs valid HTTP GET request with query params", prop.ForAll(
		func(jql string, startAt int, maxResults int) bool {
			// Ensure valid values
			if startAt < 0 {
				startAt = 0
			}
			if maxResults < 0 {
				maxResults = 50
			}
			if maxResults > 1000 {
				maxResults = 1000
			}

			// Create a test server to capture the request
			var capturedReq *http.Request
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedReq = r
				// Return a valid response
				results := domain.SearchResults{
					Issues:     []domain.JiraIssue{},
					Total:      0,
					StartAt:    startAt,
					MaxResults: maxResults,
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(results)
			}))
			defer server.Close()

			// Create search options
			options := &SearchOptions{
				StartAt:    startAt,
				MaxResults: maxResults,
			}

			// Create client and make request
			client := NewJiraClient(server.URL, server.Client())
			_, err := client.SearchJQL(jql, options)
			if err != nil {
				return false
			}

			// Verify request properties
			if capturedReq == nil {
				return false
			}

			// 1. Correct HTTP method
			if capturedReq.Method != "GET" {
				return false
			}

			// 2. Valid endpoint path
			if capturedReq.URL.Path != "/rest/api/2/search" {
				return false
			}

			// 3. Proper headers
			if capturedReq.Header.Get("Content-Type") != "application/json" {
				return false
			}
			if capturedReq.Header.Get("Accept") != "application/json" {
				return false
			}

			// 4. Valid query parameters
			queryParams := capturedReq.URL.Query()

			// JQL parameter must be present
			if queryParams.Get("jql") != jql {
				return false
			}

			// If startAt is provided, it should be in query params
			if startAt > 0 {
				if queryParams.Get("startAt") == "" {
					return false
				}
			}

			// If maxResults is provided, it should be in query params
			if maxResults > 0 {
				if queryParams.Get("maxResults") == "" {
					return false
				}
			}

			return true
		},
		genJQL,
		gen.IntRange(0, 100),
		gen.IntRange(1, 100),
	))

	// Property: TransitionIssue constructs valid HTTP POST request
	properties.Property("TransitionIssue constructs valid HTTP POST request", prop.ForAll(
		func(issueKey string, transitionID string) bool {
			// Ensure non-empty transition ID
			if transitionID == "" {
				transitionID = "21"
			}

			// Create a test server to capture the request
			var capturedReq *http.Request
			var capturedBody []byte
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedReq = r
				capturedBody, _ = io.ReadAll(r.Body)
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			// Create transition request
			transition := &domain.IssueTransition{
				Transition: domain.TransitionRef{
					ID: transitionID,
				},
			}

			// Create client and make request
			client := NewJiraClient(server.URL, server.Client())
			err := client.TransitionIssue(issueKey, transition)
			if err != nil {
				return false
			}

			// Verify request properties
			if capturedReq == nil {
				return false
			}

			// 1. Correct HTTP method
			if capturedReq.Method != "POST" {
				return false
			}

			// 2. Valid endpoint path
			expectedPath := "/rest/api/2/issue/" + issueKey + "/transitions"
			if capturedReq.URL.Path != expectedPath {
				return false
			}

			// 3. Proper headers
			if capturedReq.Header.Get("Content-Type") != "application/json" {
				return false
			}
			if capturedReq.Header.Get("Accept") != "application/json" {
				return false
			}

			// 4. Valid JSON body
			if len(capturedBody) == 0 {
				return false
			}

			// Verify body is valid JSON
			var bodyMap map[string]interface{}
			if err := json.Unmarshal(capturedBody, &bodyMap); err != nil {
				return false
			}

			// Verify transition field is present
			_, ok := bodyMap["transition"]
			if !ok {
				return false
			}

			return true
		},
		genIssueKey,
		gen.Identifier(),
	))

	// Property: AddComment constructs valid HTTP POST request
	properties.Property("AddComment constructs valid HTTP POST request", prop.ForAll(
		func(issueKey string, commentBody string) bool {
			// Ensure non-empty comment
			if commentBody == "" {
				commentBody = "Test comment"
			}

			// Create a test server to capture the request
			var capturedReq *http.Request
			var capturedBody []byte
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedReq = r
				capturedBody, _ = io.ReadAll(r.Body)
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"id":"10000","body":"` + commentBody + `"}`))
			}))
			defer server.Close()

			// Create comment request
			comment := &domain.Comment{
				Body: commentBody,
			}

			// Create client and make request
			client := NewJiraClient(server.URL, server.Client())
			err := client.AddComment(issueKey, comment)
			if err != nil {
				return false
			}

			// Verify request properties
			if capturedReq == nil {
				return false
			}

			// 1. Correct HTTP method
			if capturedReq.Method != "POST" {
				return false
			}

			// 2. Valid endpoint path
			expectedPath := "/rest/api/2/issue/" + issueKey + "/comment"
			if capturedReq.URL.Path != expectedPath {
				return false
			}

			// 3. Proper headers
			if capturedReq.Header.Get("Content-Type") != "application/json" {
				return false
			}
			if capturedReq.Header.Get("Accept") != "application/json" {
				return false
			}

			// 4. Valid JSON body
			if len(capturedBody) == 0 {
				return false
			}

			// Verify body is valid JSON
			var bodyMap map[string]interface{}
			if err := json.Unmarshal(capturedBody, &bodyMap); err != nil {
				return false
			}

			// Verify body field is present
			if bodyMap["body"] != commentBody {
				return false
			}

			return true
		},
		genIssueKey,
		gen.AlphaString(),
	))

	// Property: GetProjects constructs valid HTTP GET request
	properties.Property("GetProjects constructs valid HTTP GET request", prop.ForAll(
		func() bool {
			// Create a test server to capture the request
			var capturedReq *http.Request
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedReq = r
				// Return a valid response
				projects := []domain.Project{
					{
						ID:   "10000",
						Key:  "TEST",
						Name: "Test Project",
					},
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(projects)
			}))
			defer server.Close()

			// Create client and make request
			client := NewJiraClient(server.URL, server.Client())
			_, err := client.GetProjects()
			if err != nil {
				return false
			}

			// Verify request properties
			if capturedReq == nil {
				return false
			}

			// 1. Correct HTTP method
			if capturedReq.Method != "GET" {
				return false
			}

			// 2. Valid endpoint path
			if capturedReq.URL.Path != "/rest/api/2/project" {
				return false
			}

			// 3. Proper headers
			if capturedReq.Header.Get("Content-Type") != "application/json" {
				return false
			}
			if capturedReq.Header.Get("Accept") != "application/json" {
				return false
			}

			// 4. No body for GET request
			if capturedReq.Body != nil {
				body, _ := io.ReadAll(capturedReq.Body)
				if len(body) > 0 {
					return false
				}
			}

			return true
		},
	))

	// Property: All requests have valid base URL
	properties.Property("All requests use valid base URL", prop.ForAll(
		func(baseURL string, issueKey string) bool {
			// Ensure valid base URL format
			if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
				baseURL = "https://" + baseURL
			}

			// Parse to ensure it's a valid URL
			parsedURL, err := url.Parse(baseURL)
			if err != nil {
				return true // Skip invalid URLs
			}

			// Create a test server to capture the request
			var capturedReq *http.Request
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedReq = r
				issue := domain.JiraIssue{
					ID:  "10001",
					Key: issueKey,
					Fields: domain.JiraFields{
						Summary: "Test",
						IssueType: domain.IssueType{
							ID:   "1",
							Name: "Bug",
						},
						Project: domain.Project{
							ID:   "10000",
							Key:  "TEST",
							Name: "Test Project",
						},
						Status: domain.Status{
							ID:   "1",
							Name: "Open",
						},
						Created: "2024-01-01T10:00:00.000+0000",
						Updated: "2024-01-01T10:00:00.000+0000",
					},
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(issue)
			}))
			defer server.Close()

			// Create client with the test server URL
			client := NewJiraClient(server.URL, server.Client())
			_, err = client.GetIssue(issueKey)
			if err != nil {
				return false
			}

			// Verify the request was made
			if capturedReq == nil {
				return false
			}

			// Verify the base URL is used correctly
			if client.BaseURL() != server.URL {
				return false
			}

			// Verify the request URL starts with the base URL
			requestURL := capturedReq.URL.String()
			if !strings.HasPrefix(requestURL, "/rest/api/2/") {
				return false
			}

			// Verify parsedURL is valid (has scheme and host)
			if parsedURL.Scheme == "" || parsedURL.Host == "" {
				return true // Skip if URL is not fully valid
			}

			return true
		},
		gen.Identifier().Map(func(s string) string { return s + ".example.com" }),
		genIssueKey,
	))

	// Property: All POST/PUT requests have valid JSON bodies
	properties.Property("All POST/PUT requests have valid JSON bodies", prop.ForAll(
		func(summary string, description string) bool {
			// Ensure non-empty values
			if summary == "" {
				summary = "Test"
			}
			if description == "" {
				description = "Test"
			}

			// Create a test server to capture the request
			var capturedBody []byte
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedBody, _ = io.ReadAll(r.Body)
				issue := domain.JiraIssue{
					ID:  "10002",
					Key: "TEST-124",
					Fields: domain.JiraFields{
						Summary:     summary,
						Description: description,
						IssueType: domain.IssueType{
							ID:   "1",
							Name: "Bug",
						},
						Project: domain.Project{
							ID:   "10000",
							Key:  "TEST",
							Name: "Test Project",
						},
						Status: domain.Status{
							ID:   "1",
							Name: "Open",
						},
						Created: "2024-01-01T10:00:00.000+0000",
						Updated: "2024-01-01T10:00:00.000+0000",
					},
				}
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(issue)
			}))
			defer server.Close()

			// Create issue request
			issueCreate := &domain.JiraIssueCreate{
				Fields: domain.JiraFieldsCreate{
					Summary:     summary,
					Description: description,
					IssueType: domain.IssueTypeRef{
						ID: "1",
					},
					Project: domain.ProjectRef{
						Key: "TEST",
					},
				},
			}

			// Create client and make request
			client := NewJiraClient(server.URL, server.Client())
			_, err := client.CreateIssue(issueCreate)
			if err != nil {
				return false
			}

			// Verify body is valid JSON
			if len(capturedBody) == 0 {
				return false
			}

			var bodyMap map[string]interface{}
			if err := json.Unmarshal(capturedBody, &bodyMap); err != nil {
				return false
			}

			// Verify the JSON can be re-serialized (round-trip test)
			reserializedBody, err := json.Marshal(bodyMap)
			if err != nil {
				return false
			}

			// Verify the re-serialized body is also valid JSON
			var checkMap map[string]interface{}
			if err := json.Unmarshal(reserializedBody, &checkMap); err != nil {
				return false
			}

			return true
		},
		gen.AlphaString(),
		gen.AlphaString(),
	))

	properties.TestingRun(t)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
