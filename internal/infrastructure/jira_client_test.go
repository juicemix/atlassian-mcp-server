package infrastructure

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"atlassian-mcp-server/internal/domain"
)

// mockAuthTransport is a test transport that adds a mock Authorization header.
type mockAuthTransport struct {
	base http.RoundTripper
}

func (t *mockAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request and add auth header
	clonedReq := req.Clone(req.Context())
	clonedReq.Header.Set("Authorization", "Bearer test-token")
	return t.base.RoundTrip(clonedReq)
}

// getAuthenticatedClient returns an HTTP client with mock authentication.
func getAuthenticatedClient() *http.Client {
	return &http.Client{
		Transport: &mockAuthTransport{base: http.DefaultTransport},
	}
}

// mockJiraServer creates a test HTTP server that simulates Jira API responses.
func mockJiraServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authentication header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"errorMessages":["Authentication required"]}`))
			return
		}

		// Route based on path and method
		switch {
		// GET /rest/api/2/issue/{issueKey}
		case r.Method == "GET" && r.URL.Path == "/rest/api/2/issue/TEST-123":
			issue := domain.JiraIssue{
				ID:  "10001",
				Key: "TEST-123",
				Fields: domain.JiraFields{
					Summary:     "Test issue",
					Description: "Test description",
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
					Updated: "2024-01-02T15:30:00.000+0000",
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(issue)

		// GET /rest/api/2/issue/{issueKey} - Not Found
		case r.Method == "GET" && r.URL.Path == "/rest/api/2/issue/NOTFOUND-1":
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"errorMessages":["Issue does not exist"]}`))

		// POST /rest/api/2/issue
		case r.Method == "POST" && r.URL.Path == "/rest/api/2/issue":
			var createReq domain.JiraIssueCreate
			if err := json.NewDecoder(r.Body).Decode(&createReq); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"errorMessages":["Invalid request body"]}`))
				return
			}

			// Return created issue
			issue := domain.JiraIssue{
				ID:  "10002",
				Key: "TEST-124",
				Fields: domain.JiraFields{
					Summary:     createReq.Fields.Summary,
					Description: createReq.Fields.Description,
					IssueType: domain.IssueType{
						ID:   domain.FlexibleID(createReq.Fields.IssueType.ID),
						Name: "Bug",
					},
					Project: domain.Project{
						ID:   "10000",
						Key:  createReq.Fields.Project.Key,
						Name: "Test Project",
					},
					Status: domain.Status{
						ID:   "1",
						Name: "Open",
					},
					Created: "2024-01-03T10:00:00.000+0000",
					Updated: "2024-01-03T10:00:00.000+0000",
				},
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(issue)

		// PUT /rest/api/2/issue/{issueKey}
		case r.Method == "PUT" && r.URL.Path == "/rest/api/2/issue/TEST-123":
			w.WriteHeader(http.StatusNoContent)

		// DELETE /rest/api/2/issue/{issueKey}
		case r.Method == "DELETE" && r.URL.Path == "/rest/api/2/issue/TEST-123":
			w.WriteHeader(http.StatusNoContent)

		// GET /rest/api/2/search
		case r.Method == "GET" && r.URL.Path == "/rest/api/2/search":
			results := domain.SearchResults{
				Issues: []domain.JiraIssue{
					{
						ID:  "10001",
						Key: "TEST-123",
						Fields: domain.JiraFields{
							Summary: "Test issue",
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
					},
				},
				Total:      1,
				StartAt:    0,
				MaxResults: 50,
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(results)

		// POST /rest/api/2/issue/{issueKey}/transitions
		case r.Method == "POST" && r.URL.Path == "/rest/api/2/issue/TEST-123/transitions":
			w.WriteHeader(http.StatusNoContent)

		// POST /rest/api/2/issue/{issueKey}/comment
		case r.Method == "POST" && r.URL.Path == "/rest/api/2/issue/TEST-123/comment":
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id":"10000","body":"Test comment"}`))

		// GET /rest/api/2/project
		case r.Method == "GET" && r.URL.Path == "/rest/api/2/project":
			projects := []domain.Project{
				{
					ID:   "10000",
					Key:  "TEST",
					Name: "Test Project",
				},
				{
					ID:   "10001",
					Key:  "DEMO",
					Name: "Demo Project",
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(projects)

		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"errorMessages":["Endpoint not found"]}`))
		}
	}))
}

func TestNewJiraClient(t *testing.T) {
	baseURL := "https://jira.example.com"
	httpClient := &http.Client{}

	client := NewJiraClient(baseURL, httpClient)

	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	if client.BaseURL() != baseURL {
		t.Errorf("Expected BaseURL %s, got %s", baseURL, client.BaseURL())
	}
	if client.httpClient != httpClient {
		t.Error("Expected httpClient to be set correctly")
	}
}

func TestJiraClient_GetIssue(t *testing.T) {
	server := mockJiraServer()
	defer server.Close()

	// Create client with mock server and authenticated client
	client := NewJiraClient(server.URL, getAuthenticatedClient())

	// Test successful retrieval
	issue, err := client.GetIssue("TEST-123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if issue == nil {
		t.Fatal("Expected non-nil issue")
	}
	if issue.Key != "TEST-123" {
		t.Errorf("Expected issue key TEST-123, got %s", issue.Key)
	}
	if issue.Fields.Summary != "Test issue" {
		t.Errorf("Expected summary 'Test issue', got %s", issue.Fields.Summary)
	}
}

func TestJiraClient_GetIssue_NotFound(t *testing.T) {
	server := mockJiraServer()
	defer server.Close()

	client := NewJiraClient(server.URL, getAuthenticatedClient())

	// Test issue not found
	_, err := client.GetIssue("NOTFOUND-1")
	if err == nil {
		t.Fatal("Expected error for non-existent issue")
	}
}

func TestJiraClient_CreateIssue(t *testing.T) {
	server := mockJiraServer()
	defer server.Close()

	client := NewJiraClient(server.URL, getAuthenticatedClient())

	// Create issue request
	issueCreate := &domain.JiraIssueCreate{
		Fields: domain.JiraFieldsCreate{
			Summary:     "New test issue",
			Description: "New test description",
			IssueType: domain.IssueTypeRef{
				ID: "1",
			},
			Project: domain.ProjectRef{
				Key: "TEST",
			},
		},
	}

	// Test successful creation
	issue, err := client.CreateIssue(issueCreate)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if issue == nil {
		t.Fatal("Expected non-nil issue")
	}
	if issue.Key != "TEST-124" {
		t.Errorf("Expected issue key TEST-124, got %s", issue.Key)
	}
	if issue.Fields.Summary != "New test issue" {
		t.Errorf("Expected summary 'New test issue', got %s", issue.Fields.Summary)
	}
}

func TestJiraClient_UpdateIssue(t *testing.T) {
	server := mockJiraServer()
	defer server.Close()

	client := NewJiraClient(server.URL, getAuthenticatedClient())

	// Update issue request
	issueUpdate := &domain.JiraIssueUpdate{
		Fields: domain.JiraFieldsUpdate{
			Summary:     "Updated summary",
			Description: "Updated description",
		},
	}

	// Test successful update
	err := client.UpdateIssue("TEST-123", issueUpdate)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestJiraClient_DeleteIssue(t *testing.T) {
	server := mockJiraServer()
	defer server.Close()

	client := NewJiraClient(server.URL, getAuthenticatedClient())

	// Test successful deletion
	err := client.DeleteIssue("TEST-123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestJiraClient_SearchJQL(t *testing.T) {
	server := mockJiraServer()
	defer server.Close()

	client := NewJiraClient(server.URL, getAuthenticatedClient())

	// Test successful search
	results, err := client.SearchJQL("project = TEST", nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if results == nil {
		t.Fatal("Expected non-nil results")
	}
	if results.Total != 1 {
		t.Errorf("Expected total 1, got %d", results.Total)
	}
	if len(results.Issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(results.Issues))
	}
	if len(results.Issues) > 0 && results.Issues[0].Key != "TEST-123" {
		t.Errorf("Expected issue key TEST-123, got %s", results.Issues[0].Key)
	}
}

func TestJiraClient_SearchJQL_WithOptions(t *testing.T) {
	server := mockJiraServer()
	defer server.Close()

	client := NewJiraClient(server.URL, getAuthenticatedClient())

	// Test search with options
	options := &SearchOptions{
		StartAt:    0,
		MaxResults: 10,
		Fields:     []string{"summary", "status"},
	}

	results, err := client.SearchJQL("project = TEST", options)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if results == nil {
		t.Fatal("Expected non-nil results")
	}
}

func TestJiraClient_TransitionIssue(t *testing.T) {
	server := mockJiraServer()
	defer server.Close()

	client := NewJiraClient(server.URL, getAuthenticatedClient())

	// Transition request
	transition := &domain.IssueTransition{
		Transition: domain.TransitionRef{
			ID: "21",
		},
	}

	// Test successful transition
	err := client.TransitionIssue("TEST-123", transition)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestJiraClient_AddComment(t *testing.T) {
	server := mockJiraServer()
	defer server.Close()

	client := NewJiraClient(server.URL, getAuthenticatedClient())

	// Comment request
	comment := &domain.Comment{
		Body: "Test comment",
	}

	// Test successful comment addition
	err := client.AddComment("TEST-123", comment)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestJiraClient_GetProjects(t *testing.T) {
	server := mockJiraServer()
	defer server.Close()

	client := NewJiraClient(server.URL, getAuthenticatedClient())

	// Test successful project retrieval
	projects, err := client.GetProjects()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if projects == nil {
		t.Fatal("Expected non-nil projects")
	}
	if len(projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(projects))
	}
	if len(projects) > 0 && projects[0].Key != "TEST" {
		t.Errorf("Expected first project key TEST, got %s", projects[0].Key)
	}
}

func TestJiraClient_AuthenticationRequired(t *testing.T) {
	server := mockJiraServer()
	defer server.Close()

	// Create client with a client that doesn't send auth headers
	client := NewJiraClient(server.URL, &http.Client{})

	// Test that requests without authentication fail
	_, err := client.GetIssue("TEST-123")
	if err == nil {
		t.Fatal("Expected error for unauthenticated request")
	}
}

func TestJiraClient_Do_SetsHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers are set
		contentType := r.Header.Get("Content-Type")
		accept := r.Header.Get("Accept")

		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}
		if accept != "application/json" {
			t.Errorf("Expected Accept application/json, got %s", accept)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewJiraClient(server.URL, server.Client())

	// Make a request to verify headers
	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	_, err := client.Do(req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestJiraClient_ErrorHandling(t *testing.T) {
	// Test with invalid URL
	client := NewJiraClient("http://invalid-url-that-does-not-exist.local", &http.Client{})

	_, err := client.GetIssue("TEST-123")
	if err == nil {
		t.Fatal("Expected error for invalid URL")
	}
}

// TestJiraClient_4xxErrors tests handling of various 4xx client errors
func TestJiraClient_4xxErrors(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		errorMessage   string
		setupHandler   func(w http.ResponseWriter, r *http.Request)
		testFunc       func(client *JiraClient) error
		expectedErrMsg string
	}{
		{
			name:         "400 Bad Request on CreateIssue",
			statusCode:   http.StatusBadRequest,
			errorMessage: "Field 'summary' is required",
			setupHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"errorMessages":["Field 'summary' is required"]}`))
			},
			testFunc: func(client *JiraClient) error {
				_, err := client.CreateIssue(&domain.JiraIssueCreate{})
				return err
			},
			expectedErrMsg: "API error (status 400)",
		},
		{
			name:         "403 Forbidden on GetIssue",
			statusCode:   http.StatusForbidden,
			errorMessage: "You do not have permission to view this issue",
			setupHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"errorMessages":["You do not have permission to view this issue"]}`))
			},
			testFunc: func(client *JiraClient) error {
				_, err := client.GetIssue("TEST-123")
				return err
			},
			expectedErrMsg: "API error (status 403)",
		},
		{
			name:         "404 Not Found on UpdateIssue",
			statusCode:   http.StatusNotFound,
			errorMessage: "Issue does not exist",
			setupHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"errorMessages":["Issue does not exist"]}`))
			},
			testFunc: func(client *JiraClient) error {
				return client.UpdateIssue("NOTFOUND-1", &domain.JiraIssueUpdate{})
			},
			expectedErrMsg: "API error (status 404)",
		},
		{
			name:         "409 Conflict on CreateIssue",
			statusCode:   http.StatusConflict,
			errorMessage: "Issue already exists",
			setupHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(`{"errorMessages":["Issue already exists"]}`))
			},
			testFunc: func(client *JiraClient) error {
				_, err := client.CreateIssue(&domain.JiraIssueCreate{})
				return err
			},
			expectedErrMsg: "API error (status 409)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.setupHandler(w, r)
			}))
			defer server.Close()

			client := NewJiraClient(server.URL, getAuthenticatedClient())
			err := tt.testFunc(client)

			if err == nil {
				t.Fatalf("Expected error for %s, got nil", tt.name)
			}
			if !contains(err.Error(), tt.expectedErrMsg) {
				t.Errorf("Expected error to contain '%s', got '%s'", tt.expectedErrMsg, err.Error())
			}
		})
	}
}

// TestJiraClient_5xxErrors tests handling of various 5xx server errors
func TestJiraClient_5xxErrors(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		errorMessage   string
		method         string
		testFunc       func(client *JiraClient) error
		expectedErrMsg string
	}{
		{
			name:         "500 Internal Server Error on GetIssue",
			statusCode:   http.StatusInternalServerError,
			errorMessage: "Internal server error",
			method:       "GET",
			testFunc: func(client *JiraClient) error {
				_, err := client.GetIssue("TEST-123")
				return err
			},
			expectedErrMsg: "API error (status 500)",
		},
		{
			name:         "502 Bad Gateway on SearchJQL",
			statusCode:   http.StatusBadGateway,
			errorMessage: "Bad gateway",
			method:       "GET",
			testFunc: func(client *JiraClient) error {
				_, err := client.SearchJQL("project = TEST", nil)
				return err
			},
			expectedErrMsg: "API error (status 502)",
		},
		{
			name:         "503 Service Unavailable on CreateIssue",
			statusCode:   http.StatusServiceUnavailable,
			errorMessage: "Service temporarily unavailable",
			method:       "POST",
			testFunc: func(client *JiraClient) error {
				_, err := client.CreateIssue(&domain.JiraIssueCreate{})
				return err
			},
			expectedErrMsg: "API error (status 503)",
		},
		{
			name:         "504 Gateway Timeout on GetProjects",
			statusCode:   http.StatusGatewayTimeout,
			errorMessage: "Gateway timeout",
			method:       "GET",
			testFunc: func(client *JiraClient) error {
				_, err := client.GetProjects()
				return err
			},
			expectedErrMsg: "API error (status 504)",
		},
		{
			name:         "500 Internal Server Error on UpdateIssue",
			statusCode:   http.StatusInternalServerError,
			errorMessage: "Database connection failed",
			method:       "PUT",
			testFunc: func(client *JiraClient) error {
				return client.UpdateIssue("TEST-123", &domain.JiraIssueUpdate{})
			},
			expectedErrMsg: "API error (status 500)",
		},
		{
			name:         "503 Service Unavailable on DeleteIssue",
			statusCode:   http.StatusServiceUnavailable,
			errorMessage: "Service maintenance",
			method:       "DELETE",
			testFunc: func(client *JiraClient) error {
				return client.DeleteIssue("TEST-123")
			},
			expectedErrMsg: "API error (status 503)",
		},
		{
			name:         "500 Internal Server Error on TransitionIssue",
			statusCode:   http.StatusInternalServerError,
			errorMessage: "Workflow error",
			method:       "POST",
			testFunc: func(client *JiraClient) error {
				return client.TransitionIssue("TEST-123", &domain.IssueTransition{})
			},
			expectedErrMsg: "API error (status 500)",
		},
		{
			name:         "502 Bad Gateway on AddComment",
			statusCode:   http.StatusBadGateway,
			errorMessage: "Upstream server error",
			method:       "POST",
			testFunc: func(client *JiraClient) error {
				return client.AddComment("TEST-123", &domain.Comment{Body: "test"})
			},
			expectedErrMsg: "API error (status 502)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(`{"errorMessages":["` + tt.errorMessage + `"]}`))
			}))
			defer server.Close()

			client := NewJiraClient(server.URL, getAuthenticatedClient())
			err := tt.testFunc(client)

			if err == nil {
				t.Fatalf("Expected error for %s, got nil", tt.name)
			}
			if !contains(err.Error(), tt.expectedErrMsg) {
				t.Errorf("Expected error to contain '%s', got '%s'", tt.expectedErrMsg, err.Error())
			}
		})
	}
}

// TestJiraClient_AuthenticationHeaderInclusion tests that authentication headers are included in all API calls
func TestJiraClient_AuthenticationHeaderInclusion(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(client *JiraClient, server *httptest.Server) error
	}{
		{
			name: "GetIssue includes auth header",
			testFunc: func(client *JiraClient, server *httptest.Server) error {
				_, err := client.GetIssue("TEST-123")
				return err
			},
		},
		{
			name: "CreateIssue includes auth header",
			testFunc: func(client *JiraClient, server *httptest.Server) error {
				_, err := client.CreateIssue(&domain.JiraIssueCreate{
					Fields: domain.JiraFieldsCreate{
						Summary: "Test",
						IssueType: domain.IssueTypeRef{
							ID: "1",
						},
						Project: domain.ProjectRef{
							Key: "TEST",
						},
					},
				})
				return err
			},
		},
		{
			name: "UpdateIssue includes auth header",
			testFunc: func(client *JiraClient, server *httptest.Server) error {
				return client.UpdateIssue("TEST-123", &domain.JiraIssueUpdate{})
			},
		},
		{
			name: "DeleteIssue includes auth header",
			testFunc: func(client *JiraClient, server *httptest.Server) error {
				return client.DeleteIssue("TEST-123")
			},
		},
		{
			name: "SearchJQL includes auth header",
			testFunc: func(client *JiraClient, server *httptest.Server) error {
				_, err := client.SearchJQL("project = TEST", nil)
				return err
			},
		},
		{
			name: "TransitionIssue includes auth header",
			testFunc: func(client *JiraClient, server *httptest.Server) error {
				return client.TransitionIssue("TEST-123", &domain.IssueTransition{
					Transition: domain.TransitionRef{
						ID: "21",
					},
				})
			},
		},
		{
			name: "AddComment includes auth header",
			testFunc: func(client *JiraClient, server *httptest.Server) error {
				return client.AddComment("TEST-123", &domain.Comment{Body: "test"})
			},
		},
		{
			name: "GetProjects includes auth header",
			testFunc: func(client *JiraClient, server *httptest.Server) error {
				_, err := client.GetProjects()
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authHeaderReceived := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check if Authorization header is present
				authHeader := r.Header.Get("Authorization")
				if authHeader != "" {
					authHeaderReceived = true
				}

				// Return success response
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"id":"10001","key":"TEST-123"}`))
			}))
			defer server.Close()

			client := NewJiraClient(server.URL, getAuthenticatedClient())
			_ = tt.testFunc(client, server)

			if !authHeaderReceived {
				t.Errorf("Expected Authorization header to be included in %s", tt.name)
			}
		})
	}
}

// TestJiraClient_MalformedJSONResponse tests handling of malformed JSON responses
func TestJiraClient_MalformedJSONResponse(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		statusCode int
		testFunc   func(client *JiraClient) error
	}{
		{
			name:       "GetIssue with malformed JSON",
			response:   `{"id":"10001","key":"TEST-123",invalid}`,
			statusCode: http.StatusOK,
			testFunc: func(client *JiraClient) error {
				_, err := client.GetIssue("TEST-123")
				return err
			},
		},
		{
			name:       "CreateIssue with malformed JSON",
			response:   `{"id":"10001","key":"TEST-123"incomplete`,
			statusCode: http.StatusCreated,
			testFunc: func(client *JiraClient) error {
				_, err := client.CreateIssue(&domain.JiraIssueCreate{
					Fields: domain.JiraFieldsCreate{
						Summary: "Test",
						IssueType: domain.IssueTypeRef{
							ID: "1",
						},
						Project: domain.ProjectRef{
							Key: "TEST",
						},
					},
				})
				return err
			},
		},
		{
			name:       "SearchJQL with malformed JSON",
			response:   `{"issues":[{"id":"10001"}],"total":1,malformed}`,
			statusCode: http.StatusOK,
			testFunc: func(client *JiraClient) error {
				_, err := client.SearchJQL("project = TEST", nil)
				return err
			},
		},
		{
			name:       "GetProjects with malformed JSON",
			response:   `[{"id":"10000","key":"TEST"invalid]`,
			statusCode: http.StatusOK,
			testFunc: func(client *JiraClient) error {
				_, err := client.GetProjects()
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewJiraClient(server.URL, getAuthenticatedClient())
			err := tt.testFunc(client)

			if err == nil {
				t.Fatalf("Expected error for malformed JSON in %s, got nil", tt.name)
			}
			if !contains(err.Error(), "failed to decode") {
				t.Errorf("Expected error to contain 'failed to decode', got '%s'", err.Error())
			}
		})
	}
}

// TestJiraClient_EmptyResponse tests handling of empty responses where data is expected
func TestJiraClient_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(``))
	}))
	defer server.Close()

	client := NewJiraClient(server.URL, getAuthenticatedClient())

	// Test GetIssue with empty response
	_, err := client.GetIssue("TEST-123")
	if err == nil {
		t.Fatal("Expected error for empty response")
	}
}

// TestJiraClient_ContentTypeHeaders tests that Content-Type and Accept headers are set correctly
func TestJiraClient_ContentTypeHeaders(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(client *JiraClient) error
	}{
		{
			name: "GetIssue sets headers",
			testFunc: func(client *JiraClient) error {
				_, err := client.GetIssue("TEST-123")
				return err
			},
		},
		{
			name: "CreateIssue sets headers",
			testFunc: func(client *JiraClient) error {
				_, err := client.CreateIssue(&domain.JiraIssueCreate{
					Fields: domain.JiraFieldsCreate{
						Summary: "Test",
						IssueType: domain.IssueTypeRef{
							ID: "1",
						},
						Project: domain.ProjectRef{
							Key: "TEST",
						},
					},
				})
				return err
			},
		},
		{
			name: "UpdateIssue sets headers",
			testFunc: func(client *JiraClient) error {
				return client.UpdateIssue("TEST-123", &domain.JiraIssueUpdate{})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headersCorrect := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				contentType := r.Header.Get("Content-Type")
				accept := r.Header.Get("Accept")

				if contentType == "application/json" && accept == "application/json" {
					headersCorrect = true
				}

				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"id":"10001","key":"TEST-123"}`))
			}))
			defer server.Close()

			client := NewJiraClient(server.URL, getAuthenticatedClient())
			_ = tt.testFunc(client)

			if !headersCorrect {
				t.Errorf("Expected Content-Type and Accept headers to be application/json in %s", tt.name)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
