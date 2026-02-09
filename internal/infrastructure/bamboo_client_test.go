package infrastructure

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"atlassian-mcp-server/internal/domain"
)

// mockBambooServer creates a mock Bamboo server for testing.
func mockBambooServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authentication header
		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"message":"Authentication required"}`))
			return
		}

		// Route based on path and method
		switch {
		// Get all plans
		case r.Method == "GET" && r.URL.Path == "/rest/api/latest/plan":
			response := BambooPlansResponse{
				Plans: struct {
					Plan []domain.BuildPlan `json:"plan"`
					Size int                `json:"size"`
				}{
					Plan: []domain.BuildPlan{
						{
							Key:       "PROJ-PLAN",
							Name:      "Project Build Plan",
							ShortName: "Plan",
							ShortKey:  "PLAN",
							Type:      "chain",
							Enabled:   true,
						},
						{
							Key:       "PROJ-TEST",
							Name:      "Project Test Plan",
							ShortName: "Test",
							ShortKey:  "TEST",
							Type:      "chain",
							Enabled:   false,
						},
					},
					Size: 2,
				},
			}
			json.NewEncoder(w).Encode(response)

		// Get specific plan
		case r.Method == "GET" && r.URL.Path == "/rest/api/latest/plan/PROJ-PLAN":
			plan := domain.BuildPlan{
				Key:       "PROJ-PLAN",
				Name:      "Project Build Plan",
				ShortName: "Plan",
				ShortKey:  "PLAN",
				Type:      "chain",
				Enabled:   true,
			}
			json.NewEncoder(w).Encode(plan)

		// Get non-existent plan
		case r.Method == "GET" && r.URL.Path == "/rest/api/latest/plan/NOTFOUND":
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message":"Plan not found"}`))

		// Trigger build
		case r.Method == "POST" && r.URL.Path == "/rest/api/latest/queue/PROJ-PLAN":
			result := domain.BuildResult{
				Key:              "PROJ-PLAN-123",
				Number:           123,
				State:            "Unknown",
				LifeCycleState:   "Queued",
				BuildStartedTime: "",
				BuildReason:      "Manual build",
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(result)

		// Trigger build for non-existent plan
		case r.Method == "POST" && r.URL.Path == "/rest/api/latest/queue/NOTFOUND":
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message":"Plan not found"}`))

		// Get build result
		case r.Method == "GET" && r.URL.Path == "/rest/api/latest/result/PROJ-PLAN-123":
			result := domain.BuildResult{
				Key:                "PROJ-PLAN-123",
				Number:             123,
				State:              "Successful",
				LifeCycleState:     "Finished",
				BuildStartedTime:   "2024-01-15T10:00:00.000Z",
				BuildCompletedTime: "2024-01-15T10:05:00.000Z",
				BuildDuration:      300000,
				BuildReason:        "Manual build",
			}
			json.NewEncoder(w).Encode(result)

		// Get build result for non-existent build
		case r.Method == "GET" && r.URL.Path == "/rest/api/latest/result/NOTFOUND":
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message":"Build result not found"}`))

		// Get build log
		case r.Method == "GET" && r.URL.Path == "/rest/api/latest/result/PROJ-PLAN-123/log":
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("Build log output\nLine 2\nLine 3"))

		// Get build log for non-existent build
		case r.Method == "GET" && r.URL.Path == "/rest/api/latest/result/NOTFOUND/log":
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message":"Build result not found"}`))

		// Get deployment projects
		case r.Method == "GET" && r.URL.Path == "/rest/api/latest/deploy/project/all":
			projects := []domain.DeploymentProject{
				{
					ID:      1,
					Name:    "My Deployment",
					PlanKey: "PROJ-PLAN",
					Environments: []domain.Environment{
						{
							ID:          10,
							Name:        "Production",
							Description: "Production environment",
						},
						{
							ID:          20,
							Name:        "Staging",
							Description: "Staging environment",
						},
					},
				},
			}
			json.NewEncoder(w).Encode(projects)

		// Trigger deployment
		case r.Method == "POST" && r.URL.Path == "/rest/api/latest/deploy/environment/10/start":
			result := domain.DeploymentResult{
				ID:                    1001,
				DeploymentVersionName: "release-1.0.0",
				DeploymentState:       "PENDING",
				LifeCycleState:        "QUEUED",
				StartedDate:           "2024-01-15T10:00:00.000Z",
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(result)

		// Trigger deployment for non-existent environment
		case r.Method == "POST" && r.URL.Path == "/rest/api/latest/deploy/environment/999/start":
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message":"Environment not found"}`))

		// Get deployment result
		case r.Method == "GET" && r.URL.Path == "/rest/api/latest/deploy/result/1001":
			result := domain.DeploymentResult{
				ID:                    1001,
				DeploymentVersionName: "release-1.0.0",
				DeploymentState:       "SUCCESS",
				LifeCycleState:        "FINISHED",
				StartedDate:           "2024-01-15T10:00:00.000Z",
				FinishedDate:          "2024-01-15T10:10:00.000Z",
			}
			json.NewEncoder(w).Encode(result)

		// Get deployment result for non-existent deployment
		case r.Method == "GET" && r.URL.Path == "/rest/api/latest/deploy/result/9999":
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message":"Deployment result not found"}`))

		// Return 500 for server error test
		case r.URL.Path == "/rest/api/latest/plan/SERVERERROR":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"message":"Internal server error"}`))

		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message":"Not found"}`))
		}
	}))
}

func TestNewBambooClient(t *testing.T) {
	baseURL := "https://bamboo.example.com"
	httpClient := &http.Client{}

	client := NewBambooClient(baseURL, httpClient)

	if client.BaseURL() != baseURL {
		t.Errorf("Expected base URL %s, got %s", baseURL, client.BaseURL())
	}
	if client.httpClient != httpClient {
		t.Error("Expected httpClient to be set correctly")
	}
}

func TestBambooClient_GetPlans(t *testing.T) {
	server := mockBambooServer()
	defer server.Close()

	httpClient := getAuthenticatedClient()
	client := NewBambooClient(server.URL, httpClient)

	// Test successful retrieval
	plans, err := client.GetPlans()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(plans) != 2 {
		t.Errorf("Expected 2 plans, got %d", len(plans))
	}

	if plans[0].Key != "PROJ-PLAN" {
		t.Errorf("Expected first plan key PROJ-PLAN, got %s", plans[0].Key)
	}
	if plans[0].Name != "Project Build Plan" {
		t.Errorf("Expected first plan name 'Project Build Plan', got %s", plans[0].Name)
	}
	if !plans[0].Enabled {
		t.Error("Expected first plan to be enabled")
	}

	if plans[1].Key != "PROJ-TEST" {
		t.Errorf("Expected second plan key PROJ-TEST, got %s", plans[1].Key)
	}
	if plans[1].Enabled {
		t.Error("Expected second plan to be disabled")
	}
}

func TestBambooClient_GetPlans_Unauthenticated(t *testing.T) {
	server := mockBambooServer()
	defer server.Close()

	// Create unauthenticated client
	client := NewBambooClient(server.URL, &http.Client{})

	// Test unauthenticated request
	_, err := client.GetPlans()
	if err == nil {
		t.Error("Expected error for unauthenticated request, got nil")
	}
}

func TestBambooClient_GetPlan(t *testing.T) {
	server := mockBambooServer()
	defer server.Close()

	httpClient := getAuthenticatedClient()
	client := NewBambooClient(server.URL, httpClient)

	// Test successful retrieval
	plan, err := client.GetPlan("PROJ-PLAN")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if plan.Key != "PROJ-PLAN" {
		t.Errorf("Expected plan key PROJ-PLAN, got %s", plan.Key)
	}
	if plan.Name != "Project Build Plan" {
		t.Errorf("Expected plan name 'Project Build Plan', got %s", plan.Name)
	}
	if plan.ShortName != "Plan" {
		t.Errorf("Expected short name 'Plan', got %s", plan.ShortName)
	}
	if !plan.Enabled {
		t.Error("Expected plan to be enabled")
	}
}

func TestBambooClient_GetPlan_NotFound(t *testing.T) {
	server := mockBambooServer()
	defer server.Close()

	httpClient := getAuthenticatedClient()
	client := NewBambooClient(server.URL, httpClient)

	// Test non-existent plan
	_, err := client.GetPlan("NOTFOUND")
	if err == nil {
		t.Error("Expected error for non-existent plan, got nil")
	}
}

func TestBambooClient_TriggerBuild(t *testing.T) {
	server := mockBambooServer()
	defer server.Close()

	httpClient := getAuthenticatedClient()
	client := NewBambooClient(server.URL, httpClient)

	// Test successful build trigger
	result, err := client.TriggerBuild("PROJ-PLAN")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Key != "PROJ-PLAN-123" {
		t.Errorf("Expected build key PROJ-PLAN-123, got %s", result.Key)
	}
	if result.Number != 123 {
		t.Errorf("Expected build number 123, got %d", result.Number)
	}
	if result.LifeCycleState != "Queued" {
		t.Errorf("Expected lifecycle state Queued, got %s", result.LifeCycleState)
	}
}

func TestBambooClient_TriggerBuild_NotFound(t *testing.T) {
	server := mockBambooServer()
	defer server.Close()

	httpClient := getAuthenticatedClient()
	client := NewBambooClient(server.URL, httpClient)

	// Test triggering build for non-existent plan
	_, err := client.TriggerBuild("NOTFOUND")
	if err == nil {
		t.Error("Expected error for non-existent plan, got nil")
	}
}

func TestBambooClient_GetBuildResult(t *testing.T) {
	server := mockBambooServer()
	defer server.Close()

	httpClient := getAuthenticatedClient()
	client := NewBambooClient(server.URL, httpClient)

	// Test successful retrieval
	result, err := client.GetBuildResult("PROJ-PLAN-123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Key != "PROJ-PLAN-123" {
		t.Errorf("Expected build key PROJ-PLAN-123, got %s", result.Key)
	}
	if result.Number != 123 {
		t.Errorf("Expected build number 123, got %d", result.Number)
	}
	if result.State != "Successful" {
		t.Errorf("Expected state Successful, got %s", result.State)
	}
	if result.LifeCycleState != "Finished" {
		t.Errorf("Expected lifecycle state Finished, got %s", result.LifeCycleState)
	}
	if result.BuildDuration != 300000 {
		t.Errorf("Expected build duration 300000, got %d", result.BuildDuration)
	}
}

func TestBambooClient_GetBuildResult_NotFound(t *testing.T) {
	server := mockBambooServer()
	defer server.Close()

	httpClient := getAuthenticatedClient()
	client := NewBambooClient(server.URL, httpClient)

	// Test non-existent build result
	_, err := client.GetBuildResult("NOTFOUND")
	if err == nil {
		t.Error("Expected error for non-existent build result, got nil")
	}
}

func TestBambooClient_GetBuildLog(t *testing.T) {
	server := mockBambooServer()
	defer server.Close()

	httpClient := getAuthenticatedClient()
	client := NewBambooClient(server.URL, httpClient)

	// Test successful retrieval
	log, err := client.GetBuildLog("PROJ-PLAN-123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expectedLog := "Build log output\nLine 2\nLine 3"
	if log != expectedLog {
		t.Errorf("Expected log '%s', got '%s'", expectedLog, log)
	}
}

func TestBambooClient_GetBuildLog_NotFound(t *testing.T) {
	server := mockBambooServer()
	defer server.Close()

	httpClient := getAuthenticatedClient()
	client := NewBambooClient(server.URL, httpClient)

	// Test non-existent build log
	_, err := client.GetBuildLog("NOTFOUND")
	if err == nil {
		t.Error("Expected error for non-existent build log, got nil")
	}
}

func TestBambooClient_GetDeploymentProjects(t *testing.T) {
	server := mockBambooServer()
	defer server.Close()

	httpClient := getAuthenticatedClient()
	client := NewBambooClient(server.URL, httpClient)

	// Test successful retrieval
	projects, err := client.GetDeploymentProjects()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(projects) != 1 {
		t.Errorf("Expected 1 deployment project, got %d", len(projects))
	}

	project := projects[0]
	if project.ID != 1 {
		t.Errorf("Expected project ID 1, got %d", project.ID)
	}
	if project.Name != "My Deployment" {
		t.Errorf("Expected project name 'My Deployment', got %s", project.Name)
	}
	if project.PlanKey != "PROJ-PLAN" {
		t.Errorf("Expected plan key PROJ-PLAN, got %s", project.PlanKey)
	}
	if len(project.Environments) != 2 {
		t.Errorf("Expected 2 environments, got %d", len(project.Environments))
	}

	// Check first environment
	if project.Environments[0].ID != 10 {
		t.Errorf("Expected environment ID 10, got %d", project.Environments[0].ID)
	}
	if project.Environments[0].Name != "Production" {
		t.Errorf("Expected environment name 'Production', got %s", project.Environments[0].Name)
	}
}

func TestBambooClient_GetDeploymentProjects_Unauthenticated(t *testing.T) {
	server := mockBambooServer()
	defer server.Close()

	// Create unauthenticated client
	client := NewBambooClient(server.URL, &http.Client{})

	// Test unauthenticated request
	_, err := client.GetDeploymentProjects()
	if err == nil {
		t.Error("Expected error for unauthenticated request, got nil")
	}
}

func TestBambooClient_TriggerDeployment(t *testing.T) {
	server := mockBambooServer()
	defer server.Close()

	httpClient := getAuthenticatedClient()
	client := NewBambooClient(server.URL, httpClient)

	// Test successful deployment trigger
	result, err := client.TriggerDeployment(1, 10)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.ID != 1001 {
		t.Errorf("Expected deployment result ID 1001, got %d", result.ID)
	}
	if result.DeploymentVersionName != "release-1.0.0" {
		t.Errorf("Expected version name 'release-1.0.0', got %s", result.DeploymentVersionName)
	}
	if result.DeploymentState != "PENDING" {
		t.Errorf("Expected deployment state PENDING, got %s", result.DeploymentState)
	}
	if result.LifeCycleState != "QUEUED" {
		t.Errorf("Expected lifecycle state QUEUED, got %s", result.LifeCycleState)
	}
}

func TestBambooClient_TriggerDeployment_NotFound(t *testing.T) {
	server := mockBambooServer()
	defer server.Close()

	httpClient := getAuthenticatedClient()
	client := NewBambooClient(server.URL, httpClient)

	// Test triggering deployment for non-existent environment
	_, err := client.TriggerDeployment(1, 999)
	if err == nil {
		t.Error("Expected error for non-existent environment, got nil")
	}
}

func TestBambooClient_GetDeploymentResult(t *testing.T) {
	server := mockBambooServer()
	defer server.Close()

	httpClient := getAuthenticatedClient()
	client := NewBambooClient(server.URL, httpClient)

	// Test successful retrieval
	result, err := client.GetDeploymentResult(1001)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.ID != 1001 {
		t.Errorf("Expected deployment result ID 1001, got %d", result.ID)
	}
	if result.DeploymentVersionName != "release-1.0.0" {
		t.Errorf("Expected version name 'release-1.0.0', got %s", result.DeploymentVersionName)
	}
	if result.DeploymentState != "SUCCESS" {
		t.Errorf("Expected deployment state SUCCESS, got %s", result.DeploymentState)
	}
	if result.LifeCycleState != "FINISHED" {
		t.Errorf("Expected lifecycle state FINISHED, got %s", result.LifeCycleState)
	}
}

func TestBambooClient_GetDeploymentResult_NotFound(t *testing.T) {
	server := mockBambooServer()
	defer server.Close()

	httpClient := getAuthenticatedClient()
	client := NewBambooClient(server.URL, httpClient)

	// Test non-existent deployment result
	_, err := client.GetDeploymentResult(9999)
	if err == nil {
		t.Error("Expected error for non-existent deployment result, got nil")
	}
}

func TestBambooClient_ServerError(t *testing.T) {
	server := mockBambooServer()
	defer server.Close()

	httpClient := getAuthenticatedClient()
	client := NewBambooClient(server.URL, httpClient)

	// Test 500 error handling
	_, err := client.GetPlan("SERVERERROR")
	if err == nil {
		t.Error("Expected error for server error, got nil")
	}
}

func TestBambooClient_Do(t *testing.T) {
	server := mockBambooServer()
	defer server.Close()

	httpClient := getAuthenticatedClient()
	client := NewBambooClient(server.URL, httpClient)

	// Test Do method directly
	req, err := http.NewRequest("GET", server.URL+"/rest/api/latest/plan/PROJ-PLAN", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify headers are set
	if resp.Request.Header.Get("Content-Type") != "application/json" {
		t.Error("Expected Content-Type header to be set")
	}
	if resp.Request.Header.Get("Accept") != "application/json" {
		t.Error("Expected Accept header to be set")
	}
}
