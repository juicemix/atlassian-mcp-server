package domain

import (
	"errors"
	"net/http"
	"testing"
)

func TestDefaultResponseMapper_MapToToolResponse(t *testing.T) {
	mapper := NewResponseMapper()

	t.Run("nil response", func(t *testing.T) {
		response, err := mapper.MapToToolResponse(nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if response == nil {
			t.Fatal("expected non-nil response")
		}
		if len(response.Content) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(response.Content))
		}
		if response.Content[0].Type != "text" {
			t.Errorf("expected type 'text', got %s", response.Content[0].Type)
		}
		if response.Content[0].Text != "{}" {
			t.Errorf("expected empty JSON object, got %s", response.Content[0].Text)
		}
	})

	t.Run("Jira issue response", func(t *testing.T) {
		issue := &JiraIssue{
			ID:  "10001",
			Key: "TEST-1",
			Fields: JiraFields{
				Summary:     "Test issue",
				Description: "Test description",
				IssueType: IssueType{
					ID:   "1",
					Name: "Bug",
				},
				Project: Project{
					ID:   "10000",
					Key:  "TEST",
					Name: "Test Project",
				},
				Status: Status{
					ID:   "1",
					Name: "Open",
				},
				Created: "2024-01-01T00:00:00.000Z",
				Updated: "2024-01-02T00:00:00.000Z",
			},
		}

		response, err := mapper.MapToToolResponse(issue)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if response == nil {
			t.Fatal("expected non-nil response")
		}
		if len(response.Content) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(response.Content))
		}
		if response.Content[0].Type != "text" {
			t.Errorf("expected type 'text', got %s", response.Content[0].Type)
		}
		// Verify the JSON contains key fields
		text := response.Content[0].Text
		if text == "" {
			t.Error("expected non-empty text content")
		}
		// Basic validation that JSON contains expected fields
		if !containsSubstring(text, "TEST-1") || !containsSubstring(text, "Test issue") {
			t.Errorf("expected JSON to contain issue key and summary, got: %s", text)
		}
	})

	t.Run("Jira search results with pagination", func(t *testing.T) {
		searchResults := &SearchResults{
			Issues: []JiraIssue{
				{
					ID:  "10001",
					Key: "TEST-1",
					Fields: JiraFields{
						Summary: "First issue",
					},
				},
				{
					ID:  "10002",
					Key: "TEST-2",
					Fields: JiraFields{
						Summary: "Second issue",
					},
				},
			},
			Total:      100,
			StartAt:    0,
			MaxResults: 50,
		}

		response, err := mapper.MapToToolResponse(searchResults)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if response == nil {
			t.Fatal("expected non-nil response")
		}
		// Should have 2 content blocks: one for data, one for pagination
		if len(response.Content) != 2 {
			t.Fatalf("expected 2 content blocks, got %d", len(response.Content))
		}
		// First block should contain the search results
		if response.Content[0].Type != "text" {
			t.Errorf("expected type 'text', got %s", response.Content[0].Type)
		}
		// Second block should contain pagination info
		if response.Content[1].Type != "text" {
			t.Errorf("expected type 'text', got %s", response.Content[1].Type)
		}
		paginationText := response.Content[1].Text
		if !containsSubstring(paginationText, "Pagination") || !containsSubstring(paginationText, "100 total") {
			t.Errorf("expected pagination info, got: %s", paginationText)
		}
	})

	t.Run("Confluence page response", func(t *testing.T) {
		page := &ConfluencePage{
			ID:    "12345",
			Type:  "page",
			Title: "Test Page",
			Space: Space{
				ID:   "1",
				Key:  "TEST",
				Name: "Test Space",
			},
			Body: Body{
				Storage: Storage{
					Value:          "<p>Test content</p>",
					Representation: "storage",
				},
			},
			Version: Version{
				Number: 1,
				When:   "2024-01-01T00:00:00.000Z",
				By: User{
					Name:        "testuser",
					DisplayName: "Test User",
				},
			},
		}

		response, err := mapper.MapToToolResponse(page)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if response == nil {
			t.Fatal("expected non-nil response")
		}
		if len(response.Content) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(response.Content))
		}
		text := response.Content[0].Text
		if !containsSubstring(text, "Test Page") || !containsSubstring(text, "Test content") {
			t.Errorf("expected JSON to contain page title and content, got: %s", text)
		}
	})

	t.Run("Bitbucket pull request response", func(t *testing.T) {
		pr := &PullRequest{
			ID:          1,
			Version:     1,
			Title:       "Test PR",
			Description: "Test description",
			State:       "OPEN",
			Open:        true,
			Closed:      false,
			FromRef: Ref{
				ID: "refs/heads/feature",
				Repository: Repository{
					Slug: "test-repo",
				},
			},
			ToRef: Ref{
				ID: "refs/heads/main",
				Repository: Repository{
					Slug: "test-repo",
				},
			},
			Author: User{
				Name:        "testuser",
				DisplayName: "Test User",
			},
		}

		response, err := mapper.MapToToolResponse(pr)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if response == nil {
			t.Fatal("expected non-nil response")
		}
		if len(response.Content) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(response.Content))
		}
		text := response.Content[0].Text
		if !containsSubstring(text, "Test PR") || !containsSubstring(text, "OPEN") {
			t.Errorf("expected JSON to contain PR title and state, got: %s", text)
		}
	})

	t.Run("Bamboo build result response", func(t *testing.T) {
		buildResult := &BuildResult{
			Key:                "TEST-PLAN-1",
			Number:             1,
			State:              "Successful",
			LifeCycleState:     "Finished",
			BuildStartedTime:   "2024-01-01T00:00:00.000Z",
			BuildCompletedTime: "2024-01-01T00:05:00.000Z",
			BuildDuration:      300000,
			BuildReason:        "Manual build",
		}

		response, err := mapper.MapToToolResponse(buildResult)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if response == nil {
			t.Fatal("expected non-nil response")
		}
		if len(response.Content) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(response.Content))
		}
		text := response.Content[0].Text
		if !containsSubstring(text, "TEST-PLAN-1") || !containsSubstring(text, "Successful") {
			t.Errorf("expected JSON to contain build key and state, got: %s", text)
		}
	})

	t.Run("list response - projects", func(t *testing.T) {
		projects := []Project{
			{ID: "1", Key: "PROJ1", Name: "Project 1"},
			{ID: "2", Key: "PROJ2", Name: "Project 2"},
		}

		response, err := mapper.MapToToolResponse(projects)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if response == nil {
			t.Fatal("expected non-nil response")
		}
		if len(response.Content) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(response.Content))
		}
		text := response.Content[0].Text
		if !containsSubstring(text, "PROJ1") || !containsSubstring(text, "PROJ2") {
			t.Errorf("expected JSON to contain both projects, got: %s", text)
		}
	})

	t.Run("Bamboo build plan response", func(t *testing.T) {
		buildPlan := &BuildPlan{
			Key:       "PROJ-PLAN",
			Name:      "Project Build Plan",
			ShortName: "Plan",
			ShortKey:  "PLAN",
			Type:      "chain",
			Enabled:   true,
		}

		response, err := mapper.MapToToolResponse(buildPlan)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if response == nil {
			t.Fatal("expected non-nil response")
		}
		if len(response.Content) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(response.Content))
		}
		text := response.Content[0].Text
		if !containsSubstring(text, "PROJ-PLAN") || !containsSubstring(text, "Project Build Plan") {
			t.Errorf("expected JSON to contain build plan key and name, got: %s", text)
		}
	})

	t.Run("Bamboo deployment project response", func(t *testing.T) {
		deploymentProject := &DeploymentProject{
			ID:      123,
			Name:    "Production Deployment",
			PlanKey: "PROJ-PLAN",
			Environments: []Environment{
				{ID: 1, Name: "Staging", Description: "Staging environment"},
				{ID: 2, Name: "Production", Description: "Production environment"},
			},
		}

		response, err := mapper.MapToToolResponse(deploymentProject)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if response == nil {
			t.Fatal("expected non-nil response")
		}
		if len(response.Content) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(response.Content))
		}
		text := response.Content[0].Text
		if !containsSubstring(text, "Production Deployment") || !containsSubstring(text, "Staging") {
			t.Errorf("expected JSON to contain deployment project name and environments, got: %s", text)
		}
	})

	t.Run("Bamboo deployment result response", func(t *testing.T) {
		deploymentResult := &DeploymentResult{
			ID:                    456,
			DeploymentVersionName: "release-1.0.0",
			DeploymentState:       "Success",
			LifeCycleState:        "Finished",
			StartedDate:           "2024-01-01T00:00:00.000Z",
			FinishedDate:          "2024-01-01T00:10:00.000Z",
		}

		response, err := mapper.MapToToolResponse(deploymentResult)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if response == nil {
			t.Fatal("expected non-nil response")
		}
		if len(response.Content) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(response.Content))
		}
		text := response.Content[0].Text
		if !containsSubstring(text, "release-1.0.0") || !containsSubstring(text, "Success") {
			t.Errorf("expected JSON to contain deployment version and state, got: %s", text)
		}
	})

	t.Run("Bitbucket repository response", func(t *testing.T) {
		repository := &Repository{
			ID:   123,
			Slug: "my-repo",
			Name: "My Repository",
			Project: Project{
				ID:   "1",
				Key:  "PROJ",
				Name: "My Project",
			},
			Public: false,
		}

		response, err := mapper.MapToToolResponse(repository)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if response == nil {
			t.Fatal("expected non-nil response")
		}
		if len(response.Content) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(response.Content))
		}
		text := response.Content[0].Text
		if !containsSubstring(text, "my-repo") || !containsSubstring(text, "My Repository") {
			t.Errorf("expected JSON to contain repository slug and name, got: %s", text)
		}
	})

	t.Run("Bitbucket branch response", func(t *testing.T) {
		branch := &Branch{
			ID:           "refs/heads/feature-branch",
			DisplayID:    "feature-branch",
			Type:         "BRANCH",
			LatestCommit: "abc123def456",
			IsDefault:    false,
		}

		response, err := mapper.MapToToolResponse(branch)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if response == nil {
			t.Fatal("expected non-nil response")
		}
		if len(response.Content) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(response.Content))
		}
		text := response.Content[0].Text
		if !containsSubstring(text, "feature-branch") || !containsSubstring(text, "abc123def456") {
			t.Errorf("expected JSON to contain branch name and commit, got: %s", text)
		}
	})

	t.Run("Bitbucket commit response", func(t *testing.T) {
		commit := &Commit{
			ID:              "abc123def456789",
			DisplayID:       "abc123d",
			Author:          User{Name: "developer", DisplayName: "Developer Name"},
			AuthorTimestamp: 1704067200000,
			Message:         "Fix bug in authentication",
		}

		response, err := mapper.MapToToolResponse(commit)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if response == nil {
			t.Fatal("expected non-nil response")
		}
		if len(response.Content) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(response.Content))
		}
		text := response.Content[0].Text
		if !containsSubstring(text, "abc123d") || !containsSubstring(text, "Fix bug in authentication") {
			t.Errorf("expected JSON to contain commit ID and message, got: %s", text)
		}
	})

	t.Run("Confluence space response", func(t *testing.T) {
		space := &Space{
			ID:   "12345",
			Key:  "DOCS",
			Name: "Documentation Space",
		}

		response, err := mapper.MapToToolResponse(space)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if response == nil {
			t.Fatal("expected non-nil response")
		}
		if len(response.Content) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(response.Content))
		}
		text := response.Content[0].Text
		if !containsSubstring(text, "DOCS") || !containsSubstring(text, "Documentation Space") {
			t.Errorf("expected JSON to contain space key and name, got: %s", text)
		}
	})

	t.Run("empty list response", func(t *testing.T) {
		emptyProjects := []Project{}

		response, err := mapper.MapToToolResponse(emptyProjects)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if response == nil {
			t.Fatal("expected non-nil response")
		}
		if len(response.Content) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(response.Content))
		}
		text := response.Content[0].Text
		// Should contain empty array representation
		if !containsSubstring(text, "[") {
			t.Errorf("expected JSON array representation, got: %s", text)
		}
	})

	t.Run("response with special characters", func(t *testing.T) {
		issue := &JiraIssue{
			ID:  "10001",
			Key: "TEST-1",
			Fields: JiraFields{
				Summary:     "Issue with \"quotes\" and <html> tags",
				Description: "Description with\nnewlines\tand\ttabs",
			},
		}

		response, err := mapper.MapToToolResponse(issue)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if response == nil {
			t.Fatal("expected non-nil response")
		}
		if len(response.Content) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(response.Content))
		}
		text := response.Content[0].Text
		// JSON should properly escape special characters
		if text == "" {
			t.Error("expected non-empty text content")
		}
	})

	t.Run("list of build plans", func(t *testing.T) {
		buildPlans := []BuildPlan{
			{Key: "PROJ-PLAN1", Name: "Build Plan 1", Enabled: true},
			{Key: "PROJ-PLAN2", Name: "Build Plan 2", Enabled: false},
		}

		response, err := mapper.MapToToolResponse(buildPlans)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if response == nil {
			t.Fatal("expected non-nil response")
		}
		if len(response.Content) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(response.Content))
		}
		text := response.Content[0].Text
		if !containsSubstring(text, "PROJ-PLAN1") || !containsSubstring(text, "PROJ-PLAN2") {
			t.Errorf("expected JSON to contain both build plans, got: %s", text)
		}
	})

	t.Run("list of repositories", func(t *testing.T) {
		repositories := []Repository{
			{ID: 1, Slug: "repo1", Name: "Repository 1"},
			{ID: 2, Slug: "repo2", Name: "Repository 2"},
		}

		response, err := mapper.MapToToolResponse(repositories)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if response == nil {
			t.Fatal("expected non-nil response")
		}
		if len(response.Content) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(response.Content))
		}
		text := response.Content[0].Text
		if !containsSubstring(text, "repo1") || !containsSubstring(text, "repo2") {
			t.Errorf("expected JSON to contain both repositories, got: %s", text)
		}
	})

	t.Run("list of spaces", func(t *testing.T) {
		spaces := []Space{
			{ID: "1", Key: "SPACE1", Name: "Space 1"},
			{ID: "2", Key: "SPACE2", Name: "Space 2"},
		}

		response, err := mapper.MapToToolResponse(spaces)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if response == nil {
			t.Fatal("expected non-nil response")
		}
		if len(response.Content) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(response.Content))
		}
		text := response.Content[0].Text
		if !containsSubstring(text, "SPACE1") || !containsSubstring(text, "SPACE2") {
			t.Errorf("expected JSON to contain both spaces, got: %s", text)
		}
	})
}

func TestDefaultResponseMapper_MapError(t *testing.T) {
	mapper := NewResponseMapper()

	t.Run("nil error", func(t *testing.T) {
		result := mapper.MapError(nil)
		if result != nil {
			t.Errorf("expected nil for nil error, got %v", result)
		}
	})

	t.Run("HTTP 401 Unauthorized", func(t *testing.T) {
		httpErr := NewHTTPError(http.StatusUnauthorized, "Unauthorized", "Invalid credentials")
		result := mapper.MapError(httpErr)
		if result == nil {
			t.Fatal("expected non-nil error")
		}
		if result.Code != AuthenticationError {
			t.Errorf("expected code %d, got %d", AuthenticationError, result.Code)
		}
		if result.Message != "Authentication failed" {
			t.Errorf("expected 'Authentication failed', got %s", result.Message)
		}
		if result.Data == nil {
			t.Error("expected non-nil data")
		}
	})

	t.Run("HTTP 403 Forbidden", func(t *testing.T) {
		httpErr := NewHTTPError(http.StatusForbidden, "Forbidden", "Access denied")
		result := mapper.MapError(httpErr)
		if result == nil {
			t.Fatal("expected non-nil error")
		}
		if result.Code != AuthenticationError {
			t.Errorf("expected code %d, got %d", AuthenticationError, result.Code)
		}
		if !containsSubstring(result.Message, "forbidden") {
			t.Errorf("expected message to contain 'forbidden', got %s", result.Message)
		}
	})

	t.Run("HTTP 404 Not Found", func(t *testing.T) {
		httpErr := NewHTTPError(http.StatusNotFound, "Not Found", "Resource not found")
		result := mapper.MapError(httpErr)
		if result == nil {
			t.Fatal("expected non-nil error")
		}
		if result.Code != APIError {
			t.Errorf("expected code %d, got %d", APIError, result.Code)
		}
		if !containsSubstring(result.Message, "not found") {
			t.Errorf("expected message to contain 'not found', got %s", result.Message)
		}
	})

	t.Run("HTTP 400 Bad Request", func(t *testing.T) {
		httpErr := NewHTTPError(http.StatusBadRequest, "Bad Request", "Invalid parameters")
		result := mapper.MapError(httpErr)
		if result == nil {
			t.Fatal("expected non-nil error")
		}
		if result.Code != InvalidParams {
			t.Errorf("expected code %d, got %d", InvalidParams, result.Code)
		}
		if !containsSubstring(result.Message, "Bad request") {
			t.Errorf("expected message to contain 'Bad request', got %s", result.Message)
		}
	})

	t.Run("HTTP 409 Conflict", func(t *testing.T) {
		httpErr := NewHTTPError(http.StatusConflict, "Conflict", "Version mismatch")
		result := mapper.MapError(httpErr)
		if result == nil {
			t.Fatal("expected non-nil error")
		}
		if result.Code != APIError {
			t.Errorf("expected code %d, got %d", APIError, result.Code)
		}
		if !containsSubstring(result.Message, "Conflict") {
			t.Errorf("expected message to contain 'Conflict', got %s", result.Message)
		}
	})

	t.Run("HTTP 429 Too Many Requests", func(t *testing.T) {
		httpErr := NewHTTPError(http.StatusTooManyRequests, "Too Many Requests", "Rate limit exceeded")
		result := mapper.MapError(httpErr)
		if result == nil {
			t.Fatal("expected non-nil error")
		}
		if result.Code != RateLimitError {
			t.Errorf("expected code %d, got %d", RateLimitError, result.Code)
		}
		if !containsSubstring(result.Message, "Rate limit") {
			t.Errorf("expected message to contain 'Rate limit', got %s", result.Message)
		}
	})

	t.Run("HTTP 500 Internal Server Error", func(t *testing.T) {
		httpErr := NewHTTPError(http.StatusInternalServerError, "Internal Server Error", "Server error")
		result := mapper.MapError(httpErr)
		if result == nil {
			t.Fatal("expected non-nil error")
		}
		if result.Code != APIError {
			t.Errorf("expected code %d, got %d", APIError, result.Code)
		}
		if !containsSubstring(result.Message, "Internal server error") {
			t.Errorf("expected message to contain 'Internal server error', got %s", result.Message)
		}
	})

	t.Run("HTTP 503 Service Unavailable", func(t *testing.T) {
		httpErr := NewHTTPError(http.StatusServiceUnavailable, "Service Unavailable", "Service down")
		result := mapper.MapError(httpErr)
		if result == nil {
			t.Fatal("expected non-nil error")
		}
		if result.Code != NetworkError {
			t.Errorf("expected code %d, got %d", NetworkError, result.Code)
		}
		if !containsSubstring(result.Message, "Service unavailable") {
			t.Errorf("expected message to contain 'Service unavailable', got %s", result.Message)
		}
	})

	t.Run("HTTP 504 Gateway Timeout", func(t *testing.T) {
		httpErr := NewHTTPError(http.StatusGatewayTimeout, "Gateway Timeout", "Timeout")
		result := mapper.MapError(httpErr)
		if result == nil {
			t.Fatal("expected non-nil error")
		}
		if result.Code != NetworkError {
			t.Errorf("expected code %d, got %d", NetworkError, result.Code)
		}
		if !containsSubstring(result.Message, "timeout") {
			t.Errorf("expected message to contain 'timeout', got %s", result.Message)
		}
	})

	t.Run("generic 4xx error", func(t *testing.T) {
		httpErr := NewHTTPError(418, "I'm a teapot", "Cannot brew coffee")
		result := mapper.MapError(httpErr)
		if result == nil {
			t.Fatal("expected non-nil error")
		}
		if result.Code != APIError {
			t.Errorf("expected code %d, got %d", APIError, result.Code)
		}
		if !containsSubstring(result.Message, "Client error") {
			t.Errorf("expected message to contain 'Client error', got %s", result.Message)
		}
	})

	t.Run("generic 5xx error", func(t *testing.T) {
		httpErr := NewHTTPError(599, "Custom Server Error", "Something went wrong")
		result := mapper.MapError(httpErr)
		if result == nil {
			t.Fatal("expected non-nil error")
		}
		if result.Code != APIError {
			t.Errorf("expected code %d, got %d", APIError, result.Code)
		}
		if !containsSubstring(result.Message, "Server error") {
			t.Errorf("expected message to contain 'Server error', got %s", result.Message)
		}
	})

	t.Run("domain Error passthrough", func(t *testing.T) {
		domainErr := &Error{
			Code:    InvalidRequest,
			Message: "Invalid request",
			Data:    map[string]string{"field": "value"},
		}
		result := mapper.MapError(domainErr)
		if result == nil {
			t.Fatal("expected non-nil error")
		}
		if result.Code != InvalidRequest {
			t.Errorf("expected code %d, got %d", InvalidRequest, result.Code)
		}
		if result.Message != "Invalid request" {
			t.Errorf("expected 'Invalid request', got %s", result.Message)
		}
	})

	t.Run("generic error", func(t *testing.T) {
		genericErr := errors.New("something went wrong")
		result := mapper.MapError(genericErr)
		if result == nil {
			t.Fatal("expected non-nil error")
		}
		if result.Code != InternalError {
			t.Errorf("expected code %d, got %d", InternalError, result.Code)
		}
		if result.Message != "something went wrong" {
			t.Errorf("expected 'something went wrong', got %s", result.Message)
		}
	})

	t.Run("HTTP error with body data preservation", func(t *testing.T) {
		httpErr := NewHTTPError(http.StatusBadRequest, "Bad Request", `{"error":"Invalid field value"}`)
		result := mapper.MapError(httpErr)
		if result == nil {
			t.Fatal("expected non-nil error")
		}
		if result.Data == nil {
			t.Fatal("expected non-nil data")
		}
		dataMap, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Fatal("expected data to be a map")
		}
		if dataMap["statusCode"] != http.StatusBadRequest {
			t.Errorf("expected statusCode %d in data, got %v", http.StatusBadRequest, dataMap["statusCode"])
		}
		if dataMap["body"] != `{"error":"Invalid field value"}` {
			t.Errorf("expected body to be preserved in data, got %v", dataMap["body"])
		}
	})

	t.Run("HTTP error without body", func(t *testing.T) {
		httpErr := NewHTTPError(http.StatusNotFound, "Not Found", "")
		result := mapper.MapError(httpErr)
		if result == nil {
			t.Fatal("expected non-nil error")
		}
		if result.Data == nil {
			t.Fatal("expected non-nil data")
		}
		dataMap, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Fatal("expected data to be a map")
		}
		if _, hasBody := dataMap["body"]; hasBody {
			t.Error("expected no body field in data when body is empty")
		}
	})

	t.Run("HTTP 402 Payment Required - generic 4xx", func(t *testing.T) {
		httpErr := NewHTTPError(http.StatusPaymentRequired, "Payment Required", "")
		result := mapper.MapError(httpErr)
		if result == nil {
			t.Fatal("expected non-nil error")
		}
		if result.Code != APIError {
			t.Errorf("expected code %d, got %d", APIError, result.Code)
		}
		if !containsSubstring(result.Message, "Client error") {
			t.Errorf("expected message to contain 'Client error', got %s", result.Message)
		}
	})

	t.Run("HTTP 502 Bad Gateway - generic 5xx", func(t *testing.T) {
		httpErr := NewHTTPError(http.StatusBadGateway, "Bad Gateway", "")
		result := mapper.MapError(httpErr)
		if result == nil {
			t.Fatal("expected non-nil error")
		}
		if result.Code != APIError {
			t.Errorf("expected code %d, got %d", APIError, result.Code)
		}
		if !containsSubstring(result.Message, "Server error") {
			t.Errorf("expected message to contain 'Server error', got %s", result.Message)
		}
	})
}

func TestHTTPError_Error(t *testing.T) {
	t.Run("with body", func(t *testing.T) {
		err := NewHTTPError(404, "Not Found", "Resource does not exist")
		expected := "HTTP 404: Not Found - Resource does not exist"
		if err.Error() != expected {
			t.Errorf("expected %s, got %s", expected, err.Error())
		}
	})

	t.Run("without body", func(t *testing.T) {
		err := NewHTTPError(500, "Internal Server Error", "")
		expected := "HTTP 500: Internal Server Error"
		if err.Error() != expected {
			t.Errorf("expected %s, got %s", expected, err.Error())
		}
	})
}

func TestExtractPaginationInfo(t *testing.T) {
	t.Run("SearchResults pointer with pagination", func(t *testing.T) {
		searchResults := &SearchResults{
			Issues:     make([]JiraIssue, 10),
			Total:      100,
			StartAt:    20,
			MaxResults: 50,
		}
		info := extractPaginationInfo(searchResults)
		if info == "" {
			t.Error("expected non-empty pagination info")
		}
		if !containsSubstring(info, "21-30") || !containsSubstring(info, "100 total") {
			t.Errorf("expected pagination info with correct range, got: %s", info)
		}
	})

	t.Run("SearchResults value with pagination", func(t *testing.T) {
		searchResults := SearchResults{
			Issues:     make([]JiraIssue, 5),
			Total:      50,
			StartAt:    0,
			MaxResults: 10,
		}
		info := extractPaginationInfo(searchResults)
		if info == "" {
			t.Error("expected non-empty pagination info")
		}
		if !containsSubstring(info, "1-5") || !containsSubstring(info, "50 total") {
			t.Errorf("expected pagination info with correct range, got: %s", info)
		}
	})

	t.Run("non-paginated response", func(t *testing.T) {
		issue := &JiraIssue{
			ID:  "10001",
			Key: "TEST-1",
		}
		info := extractPaginationInfo(issue)
		if info != "" {
			t.Errorf("expected empty pagination info for non-paginated response, got: %s", info)
		}
	})
}

// Helper function to check if a string contains a substring
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
