package domain

import (
	"encoding/json"
	"testing"
)

func TestJiraIssueJSONSerialization(t *testing.T) {
	issue := JiraIssue{
		ID:  "10001",
		Key: "TEST-123",
		Fields: JiraFields{
			Summary:     "Test issue summary",
			Description: "Test issue description",
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
			Assignee: &User{
				Name:         "jsmith",
				DisplayName:  "John Smith",
				EmailAddress: "jsmith@example.com",
			},
			Reporter: &User{
				Name:         "jdoe",
				DisplayName:  "Jane Doe",
				EmailAddress: "jdoe@example.com",
			},
			Created: "2024-01-01T10:00:00.000+0000",
			Updated: "2024-01-02T15:30:00.000+0000",
		},
	}

	// Test serialization
	data, err := json.Marshal(issue)
	if err != nil {
		t.Fatalf("Failed to marshal JiraIssue: %v", err)
	}

	// Test deserialization
	var decoded JiraIssue
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal JiraIssue: %v", err)
	}

	// Verify key fields
	if decoded.ID != issue.ID {
		t.Errorf("Expected ID %s, got %s", issue.ID, decoded.ID)
	}
	if decoded.Key != issue.Key {
		t.Errorf("Expected Key %s, got %s", issue.Key, decoded.Key)
	}
	if decoded.Fields.Summary != issue.Fields.Summary {
		t.Errorf("Expected Summary %s, got %s", issue.Fields.Summary, decoded.Fields.Summary)
	}
	if decoded.Fields.IssueType.Name != issue.Fields.IssueType.Name {
		t.Errorf("Expected IssueType.Name %s, got %s", issue.Fields.IssueType.Name, decoded.Fields.IssueType.Name)
	}
	if decoded.Fields.Project.Key != issue.Fields.Project.Key {
		t.Errorf("Expected Project.Key %s, got %s", issue.Fields.Project.Key, decoded.Fields.Project.Key)
	}
	if decoded.Fields.Status.Name != issue.Fields.Status.Name {
		t.Errorf("Expected Status.Name %s, got %s", issue.Fields.Status.Name, decoded.Fields.Status.Name)
	}
	if decoded.Fields.Assignee == nil || decoded.Fields.Assignee.Name != issue.Fields.Assignee.Name {
		t.Errorf("Assignee not properly deserialized")
	}
}

func TestJiraIssueCreateJSONSerialization(t *testing.T) {
	issueCreate := JiraIssueCreate{
		Fields: JiraFieldsCreate{
			Summary:     "New issue",
			Description: "New issue description",
			IssueType: IssueTypeRef{
				ID: "1",
			},
			Project: ProjectRef{
				Key: "TEST",
			},
			Assignee: &UserRef{
				Name: "jsmith",
			},
		},
	}

	// Test serialization
	data, err := json.Marshal(issueCreate)
	if err != nil {
		t.Fatalf("Failed to marshal JiraIssueCreate: %v", err)
	}

	// Test deserialization
	var decoded JiraIssueCreate
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal JiraIssueCreate: %v", err)
	}

	// Verify fields
	if decoded.Fields.Summary != issueCreate.Fields.Summary {
		t.Errorf("Expected Summary %s, got %s", issueCreate.Fields.Summary, decoded.Fields.Summary)
	}
	if decoded.Fields.IssueType.ID != issueCreate.Fields.IssueType.ID {
		t.Errorf("Expected IssueType.ID %s, got %s", issueCreate.Fields.IssueType.ID, decoded.Fields.IssueType.ID)
	}
	if decoded.Fields.Project.Key != issueCreate.Fields.Project.Key {
		t.Errorf("Expected Project.Key %s, got %s", issueCreate.Fields.Project.Key, decoded.Fields.Project.Key)
	}
	if decoded.Fields.Assignee == nil || decoded.Fields.Assignee.Name != issueCreate.Fields.Assignee.Name {
		t.Errorf("Assignee not properly deserialized")
	}
}

func TestJiraIssueUpdateJSONSerialization(t *testing.T) {
	issueUpdate := JiraIssueUpdate{
		Fields: JiraFieldsUpdate{
			Summary:     "Updated summary",
			Description: "Updated description",
			Assignee: &UserRef{
				Name: "jdoe",
			},
		},
	}

	// Test serialization
	data, err := json.Marshal(issueUpdate)
	if err != nil {
		t.Fatalf("Failed to marshal JiraIssueUpdate: %v", err)
	}

	// Test deserialization
	var decoded JiraIssueUpdate
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal JiraIssueUpdate: %v", err)
	}

	// Verify fields
	if decoded.Fields.Summary != issueUpdate.Fields.Summary {
		t.Errorf("Expected Summary %s, got %s", issueUpdate.Fields.Summary, decoded.Fields.Summary)
	}
	if decoded.Fields.Assignee == nil || decoded.Fields.Assignee.Name != issueUpdate.Fields.Assignee.Name {
		t.Errorf("Assignee not properly deserialized")
	}
}

func TestIssueTransitionJSONSerialization(t *testing.T) {
	transition := IssueTransition{
		Transition: TransitionRef{
			ID: "21",
		},
	}

	// Test serialization
	data, err := json.Marshal(transition)
	if err != nil {
		t.Fatalf("Failed to marshal IssueTransition: %v", err)
	}

	// Test deserialization
	var decoded IssueTransition
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal IssueTransition: %v", err)
	}

	// Verify fields
	if decoded.Transition.ID != transition.Transition.ID {
		t.Errorf("Expected Transition.ID %s, got %s", transition.Transition.ID, decoded.Transition.ID)
	}
}

func TestCommentJSONSerialization(t *testing.T) {
	comment := Comment{
		Body: "This is a test comment",
	}

	// Test serialization
	data, err := json.Marshal(comment)
	if err != nil {
		t.Fatalf("Failed to marshal Comment: %v", err)
	}

	// Test deserialization
	var decoded Comment
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal Comment: %v", err)
	}

	// Verify fields
	if decoded.Body != comment.Body {
		t.Errorf("Expected Body %s, got %s", comment.Body, decoded.Body)
	}
}

func TestSearchResultsJSONSerialization(t *testing.T) {
	results := SearchResults{
		Issues: []JiraIssue{
			{
				ID:  "10001",
				Key: "TEST-1",
				Fields: JiraFields{
					Summary: "First issue",
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
					Created: "2024-01-01T10:00:00.000+0000",
					Updated: "2024-01-01T10:00:00.000+0000",
				},
			},
			{
				ID:  "10002",
				Key: "TEST-2",
				Fields: JiraFields{
					Summary: "Second issue",
					IssueType: IssueType{
						ID:   "2",
						Name: "Story",
					},
					Project: Project{
						ID:   "10000",
						Key:  "TEST",
						Name: "Test Project",
					},
					Status: Status{
						ID:   "2",
						Name: "In Progress",
					},
					Created: "2024-01-02T10:00:00.000+0000",
					Updated: "2024-01-02T10:00:00.000+0000",
				},
			},
		},
		Total:      2,
		StartAt:    0,
		MaxResults: 50,
	}

	// Test serialization
	data, err := json.Marshal(results)
	if err != nil {
		t.Fatalf("Failed to marshal SearchResults: %v", err)
	}

	// Test deserialization
	var decoded SearchResults
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal SearchResults: %v", err)
	}

	// Verify fields
	if decoded.Total != results.Total {
		t.Errorf("Expected Total %d, got %d", results.Total, decoded.Total)
	}
	if decoded.StartAt != results.StartAt {
		t.Errorf("Expected StartAt %d, got %d", results.StartAt, decoded.StartAt)
	}
	if decoded.MaxResults != results.MaxResults {
		t.Errorf("Expected MaxResults %d, got %d", results.MaxResults, decoded.MaxResults)
	}
	if len(decoded.Issues) != len(results.Issues) {
		t.Errorf("Expected %d issues, got %d", len(results.Issues), len(decoded.Issues))
	}
	if len(decoded.Issues) > 0 {
		if decoded.Issues[0].Key != results.Issues[0].Key {
			t.Errorf("Expected first issue key %s, got %s", results.Issues[0].Key, decoded.Issues[0].Key)
		}
	}
}

func TestJiraIssueWithNilOptionalFields(t *testing.T) {
	// Test that optional fields can be nil
	issue := JiraIssue{
		ID:  "10001",
		Key: "TEST-123",
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
			Assignee: nil, // Optional field
			Reporter: nil, // Optional field
			Created:  "2024-01-01T10:00:00.000+0000",
			Updated:  "2024-01-02T15:30:00.000+0000",
		},
	}

	// Test serialization
	data, err := json.Marshal(issue)
	if err != nil {
		t.Fatalf("Failed to marshal JiraIssue with nil optional fields: %v", err)
	}

	// Test deserialization
	var decoded JiraIssue
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal JiraIssue with nil optional fields: %v", err)
	}

	// Verify optional fields are nil
	if decoded.Fields.Assignee != nil {
		t.Errorf("Expected Assignee to be nil, got %v", decoded.Fields.Assignee)
	}
	if decoded.Fields.Reporter != nil {
		t.Errorf("Expected Reporter to be nil, got %v", decoded.Fields.Reporter)
	}
}

func TestFlexibleIDWithNumericJSON(t *testing.T) {
	// Test that FlexibleID can handle numeric IDs from JSON
	jsonData := `{
		"id": 10000,
		"key": "TEST",
		"fields": {
			"summary": "Test issue",
			"description": "Test description",
			"issuetype": {
				"id": 1,
				"name": "Bug"
			},
			"project": {
				"id": 10000,
				"key": "TEST",
				"name": "Test Project"
			},
			"status": {
				"id": 1,
				"name": "Open"
			},
			"created": "2024-01-01T10:00:00.000+0000",
			"updated": "2024-01-02T15:30:00.000+0000"
		}
	}`

	var issue JiraIssue
	err := json.Unmarshal([]byte(jsonData), &issue)
	if err != nil {
		t.Fatalf("Failed to unmarshal JiraIssue with numeric IDs: %v", err)
	}

	// Verify numeric IDs are converted to strings
	if issue.ID.String() != "10000" {
		t.Errorf("Expected ID '10000', got '%s'", issue.ID.String())
	}
	if issue.Fields.IssueType.ID.String() != "1" {
		t.Errorf("Expected IssueType.ID '1', got '%s'", issue.Fields.IssueType.ID.String())
	}
	if issue.Fields.Project.ID.String() != "10000" {
		t.Errorf("Expected Project.ID '10000', got '%s'", issue.Fields.Project.ID.String())
	}
	if issue.Fields.Status.ID.String() != "1" {
		t.Errorf("Expected Status.ID '1', got '%s'", issue.Fields.Status.ID.String())
	}
}

func TestFlexibleIDWithStringJSON(t *testing.T) {
	// Test that FlexibleID still works with string IDs
	jsonData := `{
		"id": "10000",
		"key": "TEST",
		"fields": {
			"summary": "Test issue",
			"description": "Test description",
			"issuetype": {
				"id": "1",
				"name": "Bug"
			},
			"project": {
				"id": "10000",
				"key": "TEST",
				"name": "Test Project"
			},
			"status": {
				"id": "1",
				"name": "Open"
			},
			"created": "2024-01-01T10:00:00.000+0000",
			"updated": "2024-01-02T15:30:00.000+0000"
		}
	}`

	var issue JiraIssue
	err := json.Unmarshal([]byte(jsonData), &issue)
	if err != nil {
		t.Fatalf("Failed to unmarshal JiraIssue with string IDs: %v", err)
	}

	// Verify string IDs work correctly
	if issue.ID.String() != "10000" {
		t.Errorf("Expected ID '10000', got '%s'", issue.ID.String())
	}
	if issue.Fields.IssueType.ID.String() != "1" {
		t.Errorf("Expected IssueType.ID '1', got '%s'", issue.Fields.IssueType.ID.String())
	}
	if issue.Fields.Project.ID.String() != "10000" {
		t.Errorf("Expected Project.ID '10000', got '%s'", issue.Fields.Project.ID.String())
	}
	if issue.Fields.Status.ID.String() != "1" {
		t.Errorf("Expected Status.ID '1', got '%s'", issue.Fields.Status.ID.String())
	}
}
