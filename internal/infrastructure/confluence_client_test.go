package infrastructure

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"atlassian-mcp-server/internal/domain"
)

// mockConfluenceServer creates a test HTTP server that simulates Confluence API responses.
func mockConfluenceServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authentication header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"message":"Authentication required"}`))
			return
		}

		// Route based on path and method
		switch {
		// GET /rest/api/content/{pageID}
		case r.Method == "GET" && r.URL.Path == "/rest/api/content/12345":
			page := domain.ConfluencePage{
				ID:    "12345",
				Type:  "page",
				Title: "Test Page",
				Space: domain.Space{
					ID:   "1",
					Key:  "TEST",
					Name: "Test Space",
				},
				Body: domain.Body{
					Storage: domain.Storage{
						Value:          "<p>Test content</p>",
						Representation: "storage",
					},
				},
				Version: domain.Version{
					Number: 1,
					When:   "2024-01-01T10:00:00.000Z",
					By: domain.User{
						Name:         "testuser",
						DisplayName:  "Test User",
						EmailAddress: "test@example.com",
					},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(page)

		// GET /rest/api/content/{pageID} - Not Found
		case r.Method == "GET" && r.URL.Path == "/rest/api/content/99999":
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message":"Page not found"}`))

		// POST /rest/api/content
		case r.Method == "POST" && r.URL.Path == "/rest/api/content":
			var createReq domain.PageCreate
			if err := json.NewDecoder(r.Body).Decode(&createReq); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"message":"Invalid request body"}`))
				return
			}

			// Return created page
			page := domain.ConfluencePage{
				ID:    "12346",
				Type:  createReq.Type,
				Title: createReq.Title,
				Space: domain.Space{
					ID:   "1",
					Key:  createReq.Space.Key,
					Name: "Test Space",
				},
				Body: domain.Body{
					Storage: domain.Storage{
						Value:          createReq.Body.Storage.Value,
						Representation: createReq.Body.Storage.Representation,
					},
				},
				Version: domain.Version{
					Number: 1,
					When:   "2024-01-03T10:00:00.000Z",
					By: domain.User{
						Name:        "testuser",
						DisplayName: "Test User",
					},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(page)

		// PUT /rest/api/content/{pageID}
		case r.Method == "PUT" && r.URL.Path == "/rest/api/content/12345":
			var updateReq domain.PageUpdate
			if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"message":"Invalid request body"}`))
				return
			}

			// Return updated page
			page := domain.ConfluencePage{
				ID:    "12345",
				Type:  "page",
				Title: updateReq.Title,
				Space: domain.Space{
					ID:   "1",
					Key:  "TEST",
					Name: "Test Space",
				},
				Body: domain.Body{
					Storage: domain.Storage{
						Value:          updateReq.Body.Storage.Value,
						Representation: updateReq.Body.Storage.Representation,
					},
				},
				Version: domain.Version{
					Number: updateReq.Version.Number,
					When:   "2024-01-03T11:00:00.000Z",
					By: domain.User{
						Name:        "testuser",
						DisplayName: "Test User",
					},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(page)

		// DELETE /rest/api/content/{pageID}
		case r.Method == "DELETE" && r.URL.Path == "/rest/api/content/12345":
			w.WriteHeader(http.StatusNoContent)

		// GET /rest/api/content/search
		case r.Method == "GET" && r.URL.Path == "/rest/api/content/search":
			results := ConfluenceSearchResults{
				Results: []domain.ConfluencePage{
					{
						ID:    "12345",
						Type:  "page",
						Title: "Test Page",
						Space: domain.Space{
							ID:   "1",
							Key:  "TEST",
							Name: "Test Space",
						},
						Body: domain.Body{
							Storage: domain.Storage{
								Value:          "<p>Test content</p>",
								Representation: "storage",
							},
						},
						Version: domain.Version{
							Number: 1,
							When:   "2024-01-01T10:00:00.000Z",
						},
					},
				},
				Start: 0,
				Limit: 25,
				Size:  1,
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(results)

		// GET /rest/api/space
		case r.Method == "GET" && r.URL.Path == "/rest/api/space":
			response := struct {
				Results []domain.Space `json:"results"`
				Start   int            `json:"start"`
				Limit   int            `json:"limit"`
				Size    int            `json:"size"`
			}{
				Results: []domain.Space{
					{
						ID:   "1",
						Key:  "TEST",
						Name: "Test Space",
					},
					{
						ID:   "2",
						Key:  "DEMO",
						Name: "Demo Space",
					},
				},
				Start: 0,
				Limit: 100,
				Size:  2,
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)

		// GET /rest/api/content/{pageID}/history
		case r.Method == "GET" && r.URL.Path == "/rest/api/content/12345/history":
			history := domain.PageHistory{
				Latest:      true,
				CreatedBy:   domain.User{Name: "creator", DisplayName: "Creator User"},
				CreatedDate: "2024-01-01T10:00:00.000Z",
				LastUpdated: domain.LastUpdated{
					By:   domain.User{Name: "updater", DisplayName: "Updater User"},
					When: "2024-01-02T15:30:00.000Z",
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(history)

		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message":"Endpoint not found"}`))
		}
	}))
}

func TestNewConfluenceClient(t *testing.T) {
	baseURL := "https://confluence.example.com"
	httpClient := &http.Client{}

	client := NewConfluenceClient(baseURL, httpClient)

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

func TestConfluenceClient_GetPage(t *testing.T) {
	server := mockConfluenceServer()
	defer server.Close()

	client := NewConfluenceClient(server.URL, getAuthenticatedClient())

	// Test successful retrieval
	page, err := client.GetPage("12345")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if page == nil {
		t.Fatal("Expected non-nil page")
	}
	if page.ID != "12345" {
		t.Errorf("Expected page ID 12345, got %s", page.ID)
	}
	if page.Title != "Test Page" {
		t.Errorf("Expected title 'Test Page', got %s", page.Title)
	}
	if page.Space.Key != "TEST" {
		t.Errorf("Expected space key TEST, got %s", page.Space.Key)
	}
}

func TestConfluenceClient_GetPage_NotFound(t *testing.T) {
	server := mockConfluenceServer()
	defer server.Close()

	client := NewConfluenceClient(server.URL, getAuthenticatedClient())

	// Test page not found
	_, err := client.GetPage("99999")
	if err == nil {
		t.Fatal("Expected error for non-existent page")
	}
}

func TestConfluenceClient_CreatePage(t *testing.T) {
	server := mockConfluenceServer()
	defer server.Close()

	client := NewConfluenceClient(server.URL, getAuthenticatedClient())

	// Create page request
	pageCreate := &domain.PageCreate{
		Type:  "page",
		Title: "New Test Page",
		Space: domain.SpaceRef{
			Key: "TEST",
		},
		Body: domain.BodyCreate{
			Storage: domain.StorageCreate{
				Value:          "<p>New content</p>",
				Representation: "storage",
			},
		},
	}

	// Test successful creation
	page, err := client.CreatePage(pageCreate)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if page == nil {
		t.Fatal("Expected non-nil page")
	}
	if page.ID != "12346" {
		t.Errorf("Expected page ID 12346, got %s", page.ID)
	}
	if page.Title != "New Test Page" {
		t.Errorf("Expected title 'New Test Page', got %s", page.Title)
	}
}

func TestConfluenceClient_UpdatePage(t *testing.T) {
	server := mockConfluenceServer()
	defer server.Close()

	client := NewConfluenceClient(server.URL, getAuthenticatedClient())

	// Update page request
	pageUpdate := &domain.PageUpdate{
		Version: domain.VersionUpdate{
			Number: 2,
		},
		Title: "Updated Page Title",
		Type:  "page",
		Body: &domain.BodyCreate{
			Storage: domain.StorageCreate{
				Value:          "<p>Updated content</p>",
				Representation: "storage",
			},
		},
	}

	// Test successful update
	page, err := client.UpdatePage("12345", pageUpdate)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if page == nil {
		t.Fatal("Expected non-nil page")
	}
	if page.Title != "Updated Page Title" {
		t.Errorf("Expected title 'Updated Page Title', got %s", page.Title)
	}
	if page.Version.Number != 2 {
		t.Errorf("Expected version 2, got %d", page.Version.Number)
	}
}

func TestConfluenceClient_DeletePage(t *testing.T) {
	server := mockConfluenceServer()
	defer server.Close()

	client := NewConfluenceClient(server.URL, getAuthenticatedClient())

	// Test successful deletion
	err := client.DeletePage("12345")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestConfluenceClient_SearchCQL(t *testing.T) {
	server := mockConfluenceServer()
	defer server.Close()

	client := NewConfluenceClient(server.URL, getAuthenticatedClient())

	// Test successful search
	results, err := client.SearchCQL("space = TEST", nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if results == nil {
		t.Fatal("Expected non-nil results")
	}
	if results.Size != 1 {
		t.Errorf("Expected size 1, got %d", results.Size)
	}
	if len(results.Results) != 1 {
		t.Errorf("Expected 1 page, got %d", len(results.Results))
	}
	if len(results.Results) > 0 && results.Results[0].ID != "12345" {
		t.Errorf("Expected page ID 12345, got %s", results.Results[0].ID)
	}
}

func TestConfluenceClient_SearchCQL_WithOptions(t *testing.T) {
	server := mockConfluenceServer()
	defer server.Close()

	client := NewConfluenceClient(server.URL, getAuthenticatedClient())

	// Test search with options
	options := &ConfluenceSearchOptions{
		Start:  0,
		Limit:  10,
		Expand: "body.storage,version",
	}

	results, err := client.SearchCQL("space = TEST", options)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if results == nil {
		t.Fatal("Expected non-nil results")
	}
}

func TestConfluenceClient_GetSpaces(t *testing.T) {
	server := mockConfluenceServer()
	defer server.Close()

	client := NewConfluenceClient(server.URL, getAuthenticatedClient())

	// Test successful space retrieval
	spaces, err := client.GetSpaces()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if spaces == nil {
		t.Fatal("Expected non-nil spaces")
	}
	if len(spaces) != 2 {
		t.Errorf("Expected 2 spaces, got %d", len(spaces))
	}
	if len(spaces) > 0 && spaces[0].Key != "TEST" {
		t.Errorf("Expected first space key TEST, got %s", spaces[0].Key)
	}
}

func TestConfluenceClient_GetPageHistory(t *testing.T) {
	server := mockConfluenceServer()
	defer server.Close()

	client := NewConfluenceClient(server.URL, getAuthenticatedClient())

	// Test successful history retrieval
	history, err := client.GetPageHistory("12345")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if history == nil {
		t.Fatal("Expected non-nil history")
	}
	if !history.Latest {
		t.Error("Expected Latest to be true")
	}
	if history.CreatedBy.Name != "creator" {
		t.Errorf("Expected creator name 'creator', got %s", history.CreatedBy.Name)
	}
}

func TestConfluenceClient_AuthenticationRequired(t *testing.T) {
	server := mockConfluenceServer()
	defer server.Close()

	// Create client with a client that doesn't send auth headers
	client := NewConfluenceClient(server.URL, &http.Client{})

	// Test that requests without authentication fail
	_, err := client.GetPage("12345")
	if err == nil {
		t.Fatal("Expected error for unauthenticated request")
	}
}

func TestConfluenceClient_Do_SetsHeaders(t *testing.T) {
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

	client := NewConfluenceClient(server.URL, server.Client())

	// Make a request to verify headers
	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	_, err := client.Do(req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestConfluenceClient_ErrorHandling(t *testing.T) {
	// Test with invalid URL
	client := NewConfluenceClient("http://invalid-url-that-does-not-exist.local", &http.Client{})

	_, err := client.GetPage("12345")
	if err == nil {
		t.Fatal("Expected error for invalid URL")
	}
}

// TestConfluenceClient_4xxErrors tests handling of various 4xx client errors
func TestConfluenceClient_4xxErrors(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		errorMessage   string
		setupHandler   func(w http.ResponseWriter, r *http.Request)
		testFunc       func(client *ConfluenceClient) error
		expectedErrMsg string
	}{
		{
			name:         "400 Bad Request on CreatePage",
			statusCode:   http.StatusBadRequest,
			errorMessage: "Field 'title' is required",
			setupHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"message":"Field 'title' is required"}`))
			},
			testFunc: func(client *ConfluenceClient) error {
				_, err := client.CreatePage(&domain.PageCreate{})
				return err
			},
			expectedErrMsg: "API error (status 400)",
		},
		{
			name:         "403 Forbidden on GetPage",
			statusCode:   http.StatusForbidden,
			errorMessage: "You do not have permission to view this page",
			setupHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"message":"You do not have permission to view this page"}`))
			},
			testFunc: func(client *ConfluenceClient) error {
				_, err := client.GetPage("12345")
				return err
			},
			expectedErrMsg: "API error (status 403)",
		},
		{
			name:         "404 Not Found on UpdatePage",
			statusCode:   http.StatusNotFound,
			errorMessage: "Page does not exist",
			setupHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"message":"Page does not exist"}`))
			},
			testFunc: func(client *ConfluenceClient) error {
				_, err := client.UpdatePage("99999", &domain.PageUpdate{})
				return err
			},
			expectedErrMsg: "API error (status 404)",
		},
		{
			name:         "409 Conflict on CreatePage",
			statusCode:   http.StatusConflict,
			errorMessage: "Page already exists",
			setupHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(`{"message":"Page already exists"}`))
			},
			testFunc: func(client *ConfluenceClient) error {
				_, err := client.CreatePage(&domain.PageCreate{})
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

			client := NewConfluenceClient(server.URL, getAuthenticatedClient())
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

// TestConfluenceClient_5xxErrors tests handling of various 5xx server errors
func TestConfluenceClient_5xxErrors(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		errorMessage   string
		method         string
		testFunc       func(client *ConfluenceClient) error
		expectedErrMsg string
	}{
		{
			name:         "500 Internal Server Error on GetPage",
			statusCode:   http.StatusInternalServerError,
			errorMessage: "Internal server error",
			method:       "GET",
			testFunc: func(client *ConfluenceClient) error {
				_, err := client.GetPage("12345")
				return err
			},
			expectedErrMsg: "API error (status 500)",
		},
		{
			name:         "502 Bad Gateway on SearchCQL",
			statusCode:   http.StatusBadGateway,
			errorMessage: "Bad gateway",
			method:       "GET",
			testFunc: func(client *ConfluenceClient) error {
				_, err := client.SearchCQL("space = TEST", nil)
				return err
			},
			expectedErrMsg: "API error (status 502)",
		},
		{
			name:         "503 Service Unavailable on CreatePage",
			statusCode:   http.StatusServiceUnavailable,
			errorMessage: "Service temporarily unavailable",
			method:       "POST",
			testFunc: func(client *ConfluenceClient) error {
				_, err := client.CreatePage(&domain.PageCreate{})
				return err
			},
			expectedErrMsg: "API error (status 503)",
		},
		{
			name:         "504 Gateway Timeout on GetSpaces",
			statusCode:   http.StatusGatewayTimeout,
			errorMessage: "Gateway timeout",
			method:       "GET",
			testFunc: func(client *ConfluenceClient) error {
				_, err := client.GetSpaces()
				return err
			},
			expectedErrMsg: "API error (status 504)",
		},
		{
			name:         "500 Internal Server Error on UpdatePage",
			statusCode:   http.StatusInternalServerError,
			errorMessage: "Database connection failed",
			method:       "PUT",
			testFunc: func(client *ConfluenceClient) error {
				_, err := client.UpdatePage("12345", &domain.PageUpdate{})
				return err
			},
			expectedErrMsg: "API error (status 500)",
		},
		{
			name:         "503 Service Unavailable on DeletePage",
			statusCode:   http.StatusServiceUnavailable,
			errorMessage: "Service maintenance",
			method:       "DELETE",
			testFunc: func(client *ConfluenceClient) error {
				return client.DeletePage("12345")
			},
			expectedErrMsg: "API error (status 503)",
		},
		{
			name:         "502 Bad Gateway on GetPageHistory",
			statusCode:   http.StatusBadGateway,
			errorMessage: "Upstream server error",
			method:       "GET",
			testFunc: func(client *ConfluenceClient) error {
				_, err := client.GetPageHistory("12345")
				return err
			},
			expectedErrMsg: "API error (status 502)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(`{"message":"` + tt.errorMessage + `"}`))
			}))
			defer server.Close()

			client := NewConfluenceClient(server.URL, getAuthenticatedClient())
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

// TestConfluenceClient_AuthenticationHeaderInclusion tests that authentication headers are included in all API calls
func TestConfluenceClient_AuthenticationHeaderInclusion(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(client *ConfluenceClient, server *httptest.Server) error
	}{
		{
			name: "GetPage includes auth header",
			testFunc: func(client *ConfluenceClient, server *httptest.Server) error {
				_, err := client.GetPage("12345")
				return err
			},
		},
		{
			name: "CreatePage includes auth header",
			testFunc: func(client *ConfluenceClient, server *httptest.Server) error {
				_, err := client.CreatePage(&domain.PageCreate{
					Type:  "page",
					Title: "Test",
					Space: domain.SpaceRef{Key: "TEST"},
					Body: domain.BodyCreate{
						Storage: domain.StorageCreate{
							Value:          "<p>Test</p>",
							Representation: "storage",
						},
					},
				})
				return err
			},
		},
		{
			name: "UpdatePage includes auth header",
			testFunc: func(client *ConfluenceClient, server *httptest.Server) error {
				_, err := client.UpdatePage("12345", &domain.PageUpdate{})
				return err
			},
		},
		{
			name: "DeletePage includes auth header",
			testFunc: func(client *ConfluenceClient, server *httptest.Server) error {
				return client.DeletePage("12345")
			},
		},
		{
			name: "SearchCQL includes auth header",
			testFunc: func(client *ConfluenceClient, server *httptest.Server) error {
				_, err := client.SearchCQL("space = TEST", nil)
				return err
			},
		},
		{
			name: "GetSpaces includes auth header",
			testFunc: func(client *ConfluenceClient, server *httptest.Server) error {
				_, err := client.GetSpaces()
				return err
			},
		},
		{
			name: "GetPageHistory includes auth header",
			testFunc: func(client *ConfluenceClient, server *httptest.Server) error {
				_, err := client.GetPageHistory("12345")
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
				w.Write([]byte(`{"id":"12345","type":"page","title":"Test"}`))
			}))
			defer server.Close()

			client := NewConfluenceClient(server.URL, getAuthenticatedClient())
			_ = tt.testFunc(client, server)

			if !authHeaderReceived {
				t.Errorf("Expected Authorization header to be included in %s", tt.name)
			}
		})
	}
}

// TestConfluenceClient_MalformedJSONResponse tests handling of malformed JSON responses
func TestConfluenceClient_MalformedJSONResponse(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		statusCode int
		testFunc   func(client *ConfluenceClient) error
	}{
		{
			name:       "GetPage with malformed JSON",
			response:   `{"id":"12345","title":"Test",invalid}`,
			statusCode: http.StatusOK,
			testFunc: func(client *ConfluenceClient) error {
				_, err := client.GetPage("12345")
				return err
			},
		},
		{
			name:       "CreatePage with malformed JSON",
			response:   `{"id":"12345","title":"Test"incomplete`,
			statusCode: http.StatusOK,
			testFunc: func(client *ConfluenceClient) error {
				_, err := client.CreatePage(&domain.PageCreate{
					Type:  "page",
					Title: "Test",
					Space: domain.SpaceRef{Key: "TEST"},
					Body: domain.BodyCreate{
						Storage: domain.StorageCreate{
							Value:          "<p>Test</p>",
							Representation: "storage",
						},
					},
				})
				return err
			},
		},
		{
			name:       "SearchCQL with malformed JSON",
			response:   `{"results":[{"id":"12345"}],"size":1,malformed}`,
			statusCode: http.StatusOK,
			testFunc: func(client *ConfluenceClient) error {
				_, err := client.SearchCQL("space = TEST", nil)
				return err
			},
		},
		{
			name:       "GetSpaces with malformed JSON",
			response:   `{"results":[{"id":"1","key":"TEST"invalid]}`,
			statusCode: http.StatusOK,
			testFunc: func(client *ConfluenceClient) error {
				_, err := client.GetSpaces()
				return err
			},
		},
		{
			name:       "GetPageHistory with malformed JSON",
			response:   `{"latest":true,"createdBy":{"name":"test"incomplete}`,
			statusCode: http.StatusOK,
			testFunc: func(client *ConfluenceClient) error {
				_, err := client.GetPageHistory("12345")
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

			client := NewConfluenceClient(server.URL, getAuthenticatedClient())
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

// TestConfluenceClient_EmptyResponse tests handling of empty responses where data is expected
func TestConfluenceClient_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(``))
	}))
	defer server.Close()

	client := NewConfluenceClient(server.URL, getAuthenticatedClient())

	// Test GetPage with empty response
	_, err := client.GetPage("12345")
	if err == nil {
		t.Fatal("Expected error for empty response")
	}
}

// TestConfluenceClient_ContentTypeHeaders tests that Content-Type and Accept headers are set correctly
func TestConfluenceClient_ContentTypeHeaders(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(client *ConfluenceClient) error
	}{
		{
			name: "GetPage sets headers",
			testFunc: func(client *ConfluenceClient) error {
				_, err := client.GetPage("12345")
				return err
			},
		},
		{
			name: "CreatePage sets headers",
			testFunc: func(client *ConfluenceClient) error {
				_, err := client.CreatePage(&domain.PageCreate{
					Type:  "page",
					Title: "Test",
					Space: domain.SpaceRef{Key: "TEST"},
					Body: domain.BodyCreate{
						Storage: domain.StorageCreate{
							Value:          "<p>Test</p>",
							Representation: "storage",
						},
					},
				})
				return err
			},
		},
		{
			name: "UpdatePage sets headers",
			testFunc: func(client *ConfluenceClient) error {
				_, err := client.UpdatePage("12345", &domain.PageUpdate{})
				return err
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
				w.Write([]byte(`{"id":"12345","type":"page","title":"Test"}`))
			}))
			defer server.Close()

			client := NewConfluenceClient(server.URL, getAuthenticatedClient())
			_ = tt.testFunc(client)

			if !headersCorrect {
				t.Errorf("Expected Content-Type and Accept headers to be application/json in %s", tt.name)
			}
		})
	}
}
