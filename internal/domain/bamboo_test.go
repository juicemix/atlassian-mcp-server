package domain

import (
	"encoding/json"
	"testing"
)

func TestBuildPlanJSONSerialization(t *testing.T) {
	plan := BuildPlan{
		Key:       "PROJ-PLAN",
		Name:      "Project Build Plan",
		ShortName: "Plan",
		ShortKey:  "PLAN",
		Type:      "chain",
		Enabled:   true,
	}

	// Marshal to JSON
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("Failed to marshal BuildPlan: %v", err)
	}

	// Unmarshal back
	var decoded BuildPlan
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal BuildPlan: %v", err)
	}

	// Verify fields
	if decoded.Key != plan.Key {
		t.Errorf("Expected Key %s, got %s", plan.Key, decoded.Key)
	}
	if decoded.Name != plan.Name {
		t.Errorf("Expected Name %s, got %s", plan.Name, decoded.Name)
	}
	if decoded.ShortName != plan.ShortName {
		t.Errorf("Expected ShortName %s, got %s", plan.ShortName, decoded.ShortName)
	}
	if decoded.ShortKey != plan.ShortKey {
		t.Errorf("Expected ShortKey %s, got %s", plan.ShortKey, decoded.ShortKey)
	}
	if decoded.Type != plan.Type {
		t.Errorf("Expected Type %s, got %s", plan.Type, decoded.Type)
	}
	if decoded.Enabled != plan.Enabled {
		t.Errorf("Expected Enabled %v, got %v", plan.Enabled, decoded.Enabled)
	}
}

func TestBuildResultJSONSerialization(t *testing.T) {
	result := BuildResult{
		Key:                "PROJ-PLAN-123",
		Number:             123,
		State:              "Successful",
		LifeCycleState:     "Finished",
		BuildStartedTime:   "2024-01-15T10:00:00.000Z",
		BuildCompletedTime: "2024-01-15T10:05:00.000Z",
		BuildDuration:      300000,
		BuildReason:        "Manual build",
	}

	// Marshal to JSON
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal BuildResult: %v", err)
	}

	// Unmarshal back
	var decoded BuildResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal BuildResult: %v", err)
	}

	// Verify fields
	if decoded.Key != result.Key {
		t.Errorf("Expected Key %s, got %s", result.Key, decoded.Key)
	}
	if decoded.Number != result.Number {
		t.Errorf("Expected Number %d, got %d", result.Number, decoded.Number)
	}
	if decoded.State != result.State {
		t.Errorf("Expected State %s, got %s", result.State, decoded.State)
	}
	if decoded.LifeCycleState != result.LifeCycleState {
		t.Errorf("Expected LifeCycleState %s, got %s", result.LifeCycleState, decoded.LifeCycleState)
	}
	if decoded.BuildDuration != result.BuildDuration {
		t.Errorf("Expected BuildDuration %d, got %d", result.BuildDuration, decoded.BuildDuration)
	}
}

func TestDeploymentProjectJSONSerialization(t *testing.T) {
	project := DeploymentProject{
		ID:      1,
		Name:    "My Deployment",
		PlanKey: "PROJ-PLAN",
		Environments: []Environment{
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
	}

	// Marshal to JSON
	data, err := json.Marshal(project)
	if err != nil {
		t.Fatalf("Failed to marshal DeploymentProject: %v", err)
	}

	// Unmarshal back
	var decoded DeploymentProject
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal DeploymentProject: %v", err)
	}

	// Verify fields
	if decoded.ID != project.ID {
		t.Errorf("Expected ID %d, got %d", project.ID, decoded.ID)
	}
	if decoded.Name != project.Name {
		t.Errorf("Expected Name %s, got %s", project.Name, decoded.Name)
	}
	if decoded.PlanKey != project.PlanKey {
		t.Errorf("Expected PlanKey %s, got %s", project.PlanKey, decoded.PlanKey)
	}
	if len(decoded.Environments) != len(project.Environments) {
		t.Errorf("Expected %d environments, got %d", len(project.Environments), len(decoded.Environments))
	}
}

func TestEnvironmentJSONSerialization(t *testing.T) {
	env := Environment{
		ID:          100,
		Name:        "Production",
		Description: "Production environment",
	}

	// Marshal to JSON
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("Failed to marshal Environment: %v", err)
	}

	// Unmarshal back
	var decoded Environment
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Environment: %v", err)
	}

	// Verify fields
	if decoded.ID != env.ID {
		t.Errorf("Expected ID %d, got %d", env.ID, decoded.ID)
	}
	if decoded.Name != env.Name {
		t.Errorf("Expected Name %s, got %s", env.Name, decoded.Name)
	}
	if decoded.Description != env.Description {
		t.Errorf("Expected Description %s, got %s", env.Description, decoded.Description)
	}
}

func TestDeploymentResultJSONSerialization(t *testing.T) {
	result := DeploymentResult{
		ID:                    1001,
		DeploymentVersionName: "release-1.0.0",
		DeploymentState:       "SUCCESS",
		LifeCycleState:        "FINISHED",
		StartedDate:           "2024-01-15T10:00:00.000Z",
		FinishedDate:          "2024-01-15T10:10:00.000Z",
	}

	// Marshal to JSON
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal DeploymentResult: %v", err)
	}

	// Unmarshal back
	var decoded DeploymentResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal DeploymentResult: %v", err)
	}

	// Verify fields
	if decoded.ID != result.ID {
		t.Errorf("Expected ID %d, got %d", result.ID, decoded.ID)
	}
	if decoded.DeploymentVersionName != result.DeploymentVersionName {
		t.Errorf("Expected DeploymentVersionName %s, got %s", result.DeploymentVersionName, decoded.DeploymentVersionName)
	}
	if decoded.DeploymentState != result.DeploymentState {
		t.Errorf("Expected DeploymentState %s, got %s", result.DeploymentState, decoded.DeploymentState)
	}
	if decoded.LifeCycleState != result.LifeCycleState {
		t.Errorf("Expected LifeCycleState %s, got %s", result.LifeCycleState, decoded.LifeCycleState)
	}
}

func TestBuildPlanEmptyValues(t *testing.T) {
	// Test with minimal fields
	plan := BuildPlan{
		Key:  "PLAN-KEY",
		Name: "Plan Name",
	}

	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("Failed to marshal BuildPlan: %v", err)
	}

	var decoded BuildPlan
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal BuildPlan: %v", err)
	}

	if decoded.Key != plan.Key {
		t.Errorf("Expected Key %s, got %s", plan.Key, decoded.Key)
	}
	if decoded.Enabled != false {
		t.Errorf("Expected Enabled to be false, got %v", decoded.Enabled)
	}
}

func TestDeploymentProjectWithoutEnvironments(t *testing.T) {
	// Test deployment project without environments
	project := DeploymentProject{
		ID:      1,
		Name:    "My Deployment",
		PlanKey: "PROJ-PLAN",
	}

	data, err := json.Marshal(project)
	if err != nil {
		t.Fatalf("Failed to marshal DeploymentProject: %v", err)
	}

	var decoded DeploymentProject
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal DeploymentProject: %v", err)
	}

	if decoded.ID != project.ID {
		t.Errorf("Expected ID %d, got %d", project.ID, decoded.ID)
	}
	if decoded.Environments != nil && len(decoded.Environments) != 0 {
		t.Errorf("Expected nil or empty Environments, got %v", decoded.Environments)
	}
}
