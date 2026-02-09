package application

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"atlassian-mcp-server/internal/domain"
	"atlassian-mcp-server/internal/infrastructure"
)

// setupMockBambooServer creates a mock Bamboo server for testing
func setupMockBambooServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		// Get plans
		case r.Method == "GET" && r.URL.Path == "/rest/api/latest/plan":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"plans": map[string]interface{}{
					"plan": []domain.BuildPlan{
						{
							Key:       "PROJ-PLAN",
							Name:      "Test Plan",
							ShortName: "Plan",
							ShortKey:  "PLAN",
							Type:      "chain",
							Enabled:   true,
						},
					},
				},
			})

		// Get plan
		case r.Method == "GET" && r.URL.Path == "/rest/api/latest/plan/PROJ-PLAN":
			json.NewEncoder(w).Encode(domain.BuildPlan{
				Key:       "PROJ-PLAN",
				Name:      "Test Plan",
				ShortName: "Plan",
				ShortKey:  "PLAN",
				Type:      "chain",
				Enabled:   true,
			})

		// Trigger build
		case r.Method == "POST" && r.URL.Path == "/rest/api/latest/queue/PROJ-PLAN":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(domain.BuildResult{
				Key:                "PROJ-PLAN-123",
				Number:             123,
				State:              "Successful",
				LifeCycleState:     "Queued",
				BuildStartedTime:   "2024-01-01T10:00:00.000Z",
				BuildCompletedTime: "",
				BuildDuration:      0,
				BuildReason:        "Manual build",
			})

		// Get build result
		case r.Method == "GET" && r.URL.Path == "/rest/api/latest/result/PROJ-PLAN-123":
			json.NewEncoder(w).Encode(domain.BuildResult{
				Key:                "PROJ-PLAN-123",
				Number:             123,
				State:              "Successful",
				LifeCycleState:     "Finished",
				BuildStartedTime:   "2024-01-01T10:00:00.000Z",
				BuildCompletedTime: "2024-01-01T10:05:00.000Z",
				BuildDuration:      300000,
				BuildReason:        "Manual build",
			})

		// Get build log
		case r.Method == "GET" && r.URL.Path == "/rest/api/latest/result/PROJ-PLAN-123/log":
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("Build log output\nLine 2\nLine 3"))

		// Get deployment projects
		case r.Method == "GET" && r.URL.Path == "/rest/api/latest/deploy/project/all":
			json.NewEncoder(w).Encode([]domain.DeploymentProject{
				{
					ID:      1,
					Name:    "Test Deployment",
					PlanKey: "PROJ-PLAN",
					Environments: []domain.Environment{
						{
							ID:          10,
							Name:        "Production",
							Description: "Production environment",
						},
					},
				},
			})

		// Trigger deployment
		case r.Method == "POST" && r.URL.Path == "/rest/api/latest/deploy/environment/10/start":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(domain.DeploymentResult{
				ID:                    100,
				DeploymentVersionName: "release-1.0",
				DeploymentState:       "Success",
				LifeCycleState:        "Queued",
				StartedDate:           "2024-01-01T11:00:00.000Z",
				FinishedDate:          "",
			})

		default:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Not found",
			})
		}
	}))
}

func TestBambooHandler_ToolName(t *testing.T) {
	handler := NewBambooHandler(nil, nil)
	if handler.ToolName() != "bamboo" {
		t.Errorf("expected tool name 'bamboo', got '%s'", handler.ToolName())
	}
}

func TestBambooHandler_ListTools(t *testing.T) {
	handler := NewBambooHandler(nil, nil)
	tools := handler.ListTools()

	expectedTools := []string{
		ToolBambooGetPlans,
		ToolBambooGetPlan,
		ToolBambooTriggerBuild,
		ToolBambooGetBuildResult,
		ToolBambooGetBuildLog,
		ToolBambooGetDeploymentProjects,
		ToolBambooTriggerDeployment,
	}

	if len(tools) != len(expectedTools) {
		t.Fatalf("expected %d tools, got %d", len(expectedTools), len(tools))
	}

	// Check that all expected tools are present
	toolMap := make(map[string]bool)
	for _, tool := range tools {
		toolMap[tool.Name] = true
	}

	for _, expectedTool := range expectedTools {
		if !toolMap[expectedTool] {
			t.Errorf("expected tool '%s' not found", expectedTool)
		}
	}

	// Verify that each tool has a description and input schema
	for _, tool := range tools {
		if tool.Description == "" {
			t.Errorf("tool '%s' has no description", tool.Name)
		}
		if tool.InputSchema.Type == "" {
			t.Errorf("tool '%s' has no input schema", tool.Name)
		}
	}
}

func TestBambooHandler_HandleGetPlans(t *testing.T) {
	server := setupMockBambooServer()
	defer server.Close()

	client := infrastructure.NewBambooClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBambooHandler(client, mapper)

	req := &domain.ToolRequest{
		Name:      ToolBambooGetPlans,
		Arguments: map[string]interface{}{},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	if len(resp.Content) == 0 {
		t.Fatal("expected content in response")
	}

	if resp.Content[0].Type != "text" {
		t.Errorf("expected content type 'text', got '%s'", resp.Content[0].Type)
	}
}

func TestBambooHandler_HandleGetPlan(t *testing.T) {
	server := setupMockBambooServer()
	defer server.Close()

	client := infrastructure.NewBambooClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBambooHandler(client, mapper)

	req := &domain.ToolRequest{
		Name: ToolBambooGetPlan,
		Arguments: map[string]interface{}{
			"planKey": "PROJ-PLAN",
		},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

func TestBambooHandler_HandleGetPlan_MissingParameter(t *testing.T) {
	server := setupMockBambooServer()
	defer server.Close()

	client := infrastructure.NewBambooClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBambooHandler(client, mapper)

	req := &domain.ToolRequest{
		Name:      ToolBambooGetPlan,
		Arguments: map[string]interface{}{},
	}

	_, err := handler.Handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing parameter, got nil")
	}

	domainErr, ok := err.(*domain.Error)
	if !ok {
		t.Fatalf("expected domain.Error, got %T", err)
	}

	if domainErr.Code != domain.InvalidParams {
		t.Errorf("expected error code %d, got %d", domain.InvalidParams, domainErr.Code)
	}
}

func TestBambooHandler_HandleTriggerBuild(t *testing.T) {
	server := setupMockBambooServer()
	defer server.Close()

	client := infrastructure.NewBambooClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBambooHandler(client, mapper)

	req := &domain.ToolRequest{
		Name: ToolBambooTriggerBuild,
		Arguments: map[string]interface{}{
			"planKey": "PROJ-PLAN",
		},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

func TestBambooHandler_HandleTriggerBuild_MissingParameter(t *testing.T) {
	server := setupMockBambooServer()
	defer server.Close()

	client := infrastructure.NewBambooClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBambooHandler(client, mapper)

	req := &domain.ToolRequest{
		Name:      ToolBambooTriggerBuild,
		Arguments: map[string]interface{}{},
	}

	_, err := handler.Handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing parameter, got nil")
	}

	domainErr, ok := err.(*domain.Error)
	if !ok {
		t.Fatalf("expected domain.Error, got %T", err)
	}

	if domainErr.Code != domain.InvalidParams {
		t.Errorf("expected error code %d, got %d", domain.InvalidParams, domainErr.Code)
	}
}

func TestBambooHandler_HandleGetBuildResult(t *testing.T) {
	server := setupMockBambooServer()
	defer server.Close()

	client := infrastructure.NewBambooClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBambooHandler(client, mapper)

	req := &domain.ToolRequest{
		Name: ToolBambooGetBuildResult,
		Arguments: map[string]interface{}{
			"buildKey": "PROJ-PLAN-123",
		},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

func TestBambooHandler_HandleGetBuildResult_MissingParameter(t *testing.T) {
	server := setupMockBambooServer()
	defer server.Close()

	client := infrastructure.NewBambooClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBambooHandler(client, mapper)

	req := &domain.ToolRequest{
		Name:      ToolBambooGetBuildResult,
		Arguments: map[string]interface{}{},
	}

	_, err := handler.Handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing parameter, got nil")
	}

	domainErr, ok := err.(*domain.Error)
	if !ok {
		t.Fatalf("expected domain.Error, got %T", err)
	}

	if domainErr.Code != domain.InvalidParams {
		t.Errorf("expected error code %d, got %d", domain.InvalidParams, domainErr.Code)
	}
}

func TestBambooHandler_HandleGetBuildLog(t *testing.T) {
	server := setupMockBambooServer()
	defer server.Close()

	client := infrastructure.NewBambooClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBambooHandler(client, mapper)

	req := &domain.ToolRequest{
		Name: ToolBambooGetBuildLog,
		Arguments: map[string]interface{}{
			"buildKey": "PROJ-PLAN-123",
		},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

func TestBambooHandler_HandleGetBuildLog_MissingParameter(t *testing.T) {
	server := setupMockBambooServer()
	defer server.Close()

	client := infrastructure.NewBambooClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBambooHandler(client, mapper)

	req := &domain.ToolRequest{
		Name:      ToolBambooGetBuildLog,
		Arguments: map[string]interface{}{},
	}

	_, err := handler.Handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing parameter, got nil")
	}

	domainErr, ok := err.(*domain.Error)
	if !ok {
		t.Fatalf("expected domain.Error, got %T", err)
	}

	if domainErr.Code != domain.InvalidParams {
		t.Errorf("expected error code %d, got %d", domain.InvalidParams, domainErr.Code)
	}
}

func TestBambooHandler_HandleGetDeploymentProjects(t *testing.T) {
	server := setupMockBambooServer()
	defer server.Close()

	client := infrastructure.NewBambooClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBambooHandler(client, mapper)

	req := &domain.ToolRequest{
		Name:      ToolBambooGetDeploymentProjects,
		Arguments: map[string]interface{}{},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

func TestBambooHandler_HandleTriggerDeployment(t *testing.T) {
	server := setupMockBambooServer()
	defer server.Close()

	client := infrastructure.NewBambooClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBambooHandler(client, mapper)

	req := &domain.ToolRequest{
		Name: ToolBambooTriggerDeployment,
		Arguments: map[string]interface{}{
			"projectId":     float64(1),
			"environmentId": float64(10),
		},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

func TestBambooHandler_HandleTriggerDeployment_MissingParameters(t *testing.T) {
	server := setupMockBambooServer()
	defer server.Close()

	client := infrastructure.NewBambooClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBambooHandler(client, mapper)

	testCases := []struct {
		name      string
		arguments map[string]interface{}
		missing   string
	}{
		{
			name:      "missing projectId",
			arguments: map[string]interface{}{"environmentId": float64(10)},
			missing:   "projectId",
		},
		{
			name:      "missing environmentId",
			arguments: map[string]interface{}{"projectId": float64(1)},
			missing:   "environmentId",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &domain.ToolRequest{
				Name:      ToolBambooTriggerDeployment,
				Arguments: tc.arguments,
			}

			_, err := handler.Handle(context.Background(), req)
			if err == nil {
				t.Fatalf("expected error for missing %s, got nil", tc.missing)
			}

			domainErr, ok := err.(*domain.Error)
			if !ok {
				t.Fatalf("expected domain.Error, got %T", err)
			}

			if domainErr.Code != domain.InvalidParams {
				t.Errorf("expected error code %d, got %d", domain.InvalidParams, domainErr.Code)
			}
		})
	}
}

func TestBambooHandler_HandleUnknownTool(t *testing.T) {
	server := setupMockBambooServer()
	defer server.Close()

	client := infrastructure.NewBambooClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBambooHandler(client, mapper)

	req := &domain.ToolRequest{
		Name:      "bamboo_unknown_tool",
		Arguments: map[string]interface{}{},
	}

	_, err := handler.Handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for unknown tool, got nil")
	}

	domainErr, ok := err.(*domain.Error)
	if !ok {
		t.Fatalf("expected domain.Error, got %T", err)
	}

	if domainErr.Code != domain.MethodNotFound {
		t.Errorf("expected error code %d, got %d", domain.MethodNotFound, domainErr.Code)
	}
}

func TestBambooHandler_ParameterValidation_InvalidType(t *testing.T) {
	server := setupMockBambooServer()
	defer server.Close()

	client := infrastructure.NewBambooClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBambooHandler(client, mapper)

	testCases := []struct {
		name      string
		toolName  string
		arguments map[string]interface{}
		paramName string
	}{
		{
			name:      "getPlan with non-string planKey",
			toolName:  ToolBambooGetPlan,
			arguments: map[string]interface{}{"planKey": 123},
			paramName: "planKey",
		},
		{
			name:      "triggerBuild with non-string planKey",
			toolName:  ToolBambooTriggerBuild,
			arguments: map[string]interface{}{"planKey": 123},
			paramName: "planKey",
		},
		{
			name:      "getBuildResult with non-string buildKey",
			toolName:  ToolBambooGetBuildResult,
			arguments: map[string]interface{}{"buildKey": 123},
			paramName: "buildKey",
		},
		{
			name:      "getBuildLog with non-string buildKey",
			toolName:  ToolBambooGetBuildLog,
			arguments: map[string]interface{}{"buildKey": 123},
			paramName: "buildKey",
		},
		{
			name:      "triggerDeployment with non-integer projectId",
			toolName:  ToolBambooTriggerDeployment,
			arguments: map[string]interface{}{"projectId": "invalid", "environmentId": float64(10)},
			paramName: "projectId",
		},
		{
			name:      "triggerDeployment with non-integer environmentId",
			toolName:  ToolBambooTriggerDeployment,
			arguments: map[string]interface{}{"projectId": float64(1), "environmentId": "invalid"},
			paramName: "environmentId",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &domain.ToolRequest{
				Name:      tc.toolName,
				Arguments: tc.arguments,
			}

			_, err := handler.Handle(context.Background(), req)
			if err == nil {
				t.Fatalf("expected error for invalid type of %s, got nil", tc.paramName)
			}

			domainErr, ok := err.(*domain.Error)
			if !ok {
				t.Fatalf("expected domain.Error, got %T", err)
			}

			if domainErr.Code != domain.InvalidParams {
				t.Errorf("expected error code %d, got %d", domain.InvalidParams, domainErr.Code)
			}
		})
	}
}

func TestBambooHandler_IntegerParameterValidation(t *testing.T) {
	server := setupMockBambooServer()
	defer server.Close()

	client := infrastructure.NewBambooClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBambooHandler(client, mapper)

	// Test with float64 (from JSON)
	req := &domain.ToolRequest{
		Name: ToolBambooTriggerDeployment,
		Arguments: map[string]interface{}{
			"projectId":     float64(1),
			"environmentId": float64(10),
		},
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error with float64: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	// Test with int
	req2 := &domain.ToolRequest{
		Name: ToolBambooTriggerDeployment,
		Arguments: map[string]interface{}{
			"projectId":     1,
			"environmentId": 10,
		},
	}

	resp2, err := handler.Handle(context.Background(), req2)
	if err != nil {
		t.Fatalf("unexpected error with int: %v", err)
	}

	if resp2 == nil {
		t.Fatal("expected response, got nil")
	}

	// Test with invalid type
	req3 := &domain.ToolRequest{
		Name: ToolBambooTriggerDeployment,
		Arguments: map[string]interface{}{
			"projectId":     "invalid",
			"environmentId": float64(10),
		},
	}

	_, err = handler.Handle(context.Background(), req3)
	if err == nil {
		t.Fatal("expected error for invalid integer type, got nil")
	}

	domainErr, ok := err.(*domain.Error)
	if !ok {
		t.Fatalf("expected domain.Error, got %T", err)
	}

	if domainErr.Code != domain.InvalidParams {
		t.Errorf("expected error code %d, got %d", domain.InvalidParams, domainErr.Code)
	}
}

func TestBambooHandler_NilArguments(t *testing.T) {
	server := setupMockBambooServer()
	defer server.Close()

	client := infrastructure.NewBambooClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBambooHandler(client, mapper)

	req := &domain.ToolRequest{
		Name:      ToolBambooGetPlans,
		Arguments: nil,
	}

	resp, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

func TestBambooHandler_HTTPErrorHandling(t *testing.T) {
	testCases := []struct {
		name           string
		statusCode     int
		expectedErrMsg string
	}{
		{
			name:           "404 Not Found",
			statusCode:     http.StatusNotFound,
			expectedErrMsg: "404",
		},
		{
			name:           "401 Unauthorized",
			statusCode:     http.StatusUnauthorized,
			expectedErrMsg: "401",
		},
		{
			name:           "403 Forbidden",
			statusCode:     http.StatusForbidden,
			expectedErrMsg: "403",
		},
		{
			name:           "500 Internal Server Error",
			statusCode:     http.StatusInternalServerError,
			expectedErrMsg: "500",
		},
		{
			name:           "503 Service Unavailable",
			statusCode:     http.StatusServiceUnavailable,
			expectedErrMsg: "503",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock server that returns the specific error code
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "Test error",
				})
			}))
			defer server.Close()

			client := infrastructure.NewBambooClient(server.URL, server.Client())
			mapper := &mockResponseMapper{}
			handler := NewBambooHandler(client, mapper)

			req := &domain.ToolRequest{
				Name: ToolBambooGetPlan,
				Arguments: map[string]interface{}{
					"planKey": "PROJ-PLAN",
				},
			}

			_, err := handler.Handle(context.Background(), req)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			// Verify the error contains the status code
			if !contains(err.Error(), tc.expectedErrMsg) {
				t.Errorf("expected error to contain '%s', got: %v", tc.expectedErrMsg, err)
			}
		})
	}
}

func TestBambooHandler_ClientErrorPropagation(t *testing.T) {
	// Create a server that closes immediately to simulate network error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Close the connection immediately
		hj, ok := w.(http.Hijacker)
		if ok {
			conn, _, _ := hj.Hijack()
			conn.Close()
		}
	}))
	server.Close() // Close the server to ensure connection failures

	client := infrastructure.NewBambooClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBambooHandler(client, mapper)

	req := &domain.ToolRequest{
		Name: ToolBambooGetPlan,
		Arguments: map[string]interface{}{
			"planKey": "PROJ-PLAN",
		},
	}

	_, err := handler.Handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for closed connection, got nil")
	}
}

func TestBambooHandler_ResponseMapperIntegration(t *testing.T) {
	server := setupMockBambooServer()
	defer server.Close()

	client := infrastructure.NewBambooClient(server.URL, server.Client())

	// Use the real response mapper
	mapper := domain.NewResponseMapper()
	handler := NewBambooHandler(client, mapper)

	t.Run("successful response mapping", func(t *testing.T) {
		req := &domain.ToolRequest{
			Name: ToolBambooGetPlan,
			Arguments: map[string]interface{}{
				"planKey": "PROJ-PLAN",
			},
		}

		resp, err := handler.Handle(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp == nil {
			t.Fatal("expected response, got nil")
		}

		if len(resp.Content) == 0 {
			t.Fatal("expected content in response")
		}

		// Verify the content is valid JSON
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(resp.Content[0].Text), &jsonData); err != nil {
			t.Fatalf("response is not valid JSON: %v", err)
		}
	})

	t.Run("error response mapping", func(t *testing.T) {
		// Create a server that returns 404
		errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Plan not found",
			})
		}))
		defer errorServer.Close()

		errorClient := infrastructure.NewBambooClient(errorServer.URL, errorServer.Client())
		errorHandler := NewBambooHandler(errorClient, mapper)

		req := &domain.ToolRequest{
			Name: ToolBambooGetPlan,
			Arguments: map[string]interface{}{
				"planKey": "NOTFOUND-PLAN",
			},
		}

		_, err := errorHandler.Handle(context.Background(), req)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		// Verify it's a domain error with proper structure
		domainErr, ok := err.(*domain.Error)
		if !ok {
			t.Fatalf("expected domain.Error, got %T", err)
		}

		if domainErr.Code == 0 {
			t.Error("expected non-zero error code")
		}

		if domainErr.Message == "" {
			t.Error("expected error message")
		}
	})
}

func TestBambooHandler_AllToolsHaveValidSchemas(t *testing.T) {
	handler := NewBambooHandler(nil, nil)
	tools := handler.ListTools()

	for _, tool := range tools {
		t.Run(tool.Name, func(t *testing.T) {
			// Verify basic schema structure
			if tool.InputSchema.Type != "object" {
				t.Errorf("expected schema type 'object', got '%s'", tool.InputSchema.Type)
			}

			// Verify required fields are in properties
			for _, requiredField := range tool.InputSchema.Required {
				if _, exists := tool.InputSchema.Properties[requiredField]; !exists {
					t.Errorf("required field '%s' not found in properties", requiredField)
				}
			}

			// Verify all properties have type and description
			for propName, propValue := range tool.InputSchema.Properties {
				propMap, ok := propValue.(map[string]interface{})
				if !ok {
					t.Errorf("property '%s' is not a map", propName)
					continue
				}

				if _, hasType := propMap["type"]; !hasType {
					t.Errorf("property '%s' missing type", propName)
				}

				if _, hasDesc := propMap["description"]; !hasDesc {
					t.Errorf("property '%s' missing description", propName)
				}
			}
		})
	}
}

func TestBambooHandler_ContextPropagation(t *testing.T) {
	server := setupMockBambooServer()
	defer server.Close()

	client := infrastructure.NewBambooClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBambooHandler(client, mapper)

	// Create a context with a value
	ctx := context.WithValue(context.Background(), "test-key", "test-value")

	req := &domain.ToolRequest{
		Name: ToolBambooGetPlan,
		Arguments: map[string]interface{}{
			"planKey": "PROJ-PLAN",
		},
	}

	// This should not panic or error due to context
	resp, err := handler.Handle(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

func TestBambooHandler_EdgeCases(t *testing.T) {
	server := setupMockBambooServer()
	defer server.Close()

	client := infrastructure.NewBambooClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBambooHandler(client, mapper)

	t.Run("empty string parameters", func(t *testing.T) {
		// Empty strings should be accepted (they're still strings)
		req := &domain.ToolRequest{
			Name: ToolBambooGetPlan,
			Arguments: map[string]interface{}{
				"planKey": "",
			},
		}

		// This should not fail validation (empty strings are valid)
		_, err := handler.Handle(context.Background(), req)
		// The error here would be from the API, not from validation
		// We're just checking that validation doesn't reject empty strings
		if err != nil {
			domainErr, ok := err.(*domain.Error)
			if ok && domainErr.Code == domain.InvalidParams {
				t.Error("empty strings should not fail parameter validation")
			}
		}
	})

	t.Run("special characters in parameters", func(t *testing.T) {
		req := &domain.ToolRequest{
			Name: ToolBambooGetPlan,
			Arguments: map[string]interface{}{
				"planKey": "PROJ-PLAN!@#$%",
			},
		}

		// Should not fail validation
		_, err := handler.Handle(context.Background(), req)
		if err != nil {
			domainErr, ok := err.(*domain.Error)
			if ok && domainErr.Code == domain.InvalidParams {
				t.Error("special characters should not fail parameter validation")
			}
		}
	})
}

func TestBambooHandler_AllOperations(t *testing.T) {
	server := setupMockBambooServer()
	defer server.Close()

	client := infrastructure.NewBambooClient(server.URL, server.Client())
	mapper := &mockResponseMapper{}
	handler := NewBambooHandler(client, mapper)

	testCases := []struct {
		name      string
		toolName  string
		arguments map[string]interface{}
	}{
		{
			name:      "get plans",
			toolName:  ToolBambooGetPlans,
			arguments: map[string]interface{}{},
		},
		{
			name:     "get plan",
			toolName: ToolBambooGetPlan,
			arguments: map[string]interface{}{
				"planKey": "PROJ-PLAN",
			},
		},
		{
			name:     "trigger build",
			toolName: ToolBambooTriggerBuild,
			arguments: map[string]interface{}{
				"planKey": "PROJ-PLAN",
			},
		},
		{
			name:     "get build result",
			toolName: ToolBambooGetBuildResult,
			arguments: map[string]interface{}{
				"buildKey": "PROJ-PLAN-123",
			},
		},
		{
			name:     "get build log",
			toolName: ToolBambooGetBuildLog,
			arguments: map[string]interface{}{
				"buildKey": "PROJ-PLAN-123",
			},
		},
		{
			name:      "get deployment projects",
			toolName:  ToolBambooGetDeploymentProjects,
			arguments: map[string]interface{}{},
		},
		{
			name:     "trigger deployment",
			toolName: ToolBambooTriggerDeployment,
			arguments: map[string]interface{}{
				"projectId":     float64(1),
				"environmentId": float64(10),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &domain.ToolRequest{
				Name:      tc.toolName,
				Arguments: tc.arguments,
			}

			resp, err := handler.Handle(context.Background(), req)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tc.name, err)
			}

			if resp == nil {
				t.Fatalf("expected response for %s, got nil", tc.name)
			}

			if len(resp.Content) == 0 {
				t.Fatalf("expected content in response for %s", tc.name)
			}
		})
	}
}
