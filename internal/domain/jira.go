package domain

import (
	"encoding/json"
	"fmt"
)

// FlexibleID is a type that can unmarshal both string and numeric IDs from JSON.
type FlexibleID string

// UnmarshalJSON implements custom unmarshaling to handle both string and numeric IDs.
func (f *FlexibleID) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = FlexibleID(s)
		return nil
	}

	// Try to unmarshal as number
	var n json.Number
	if err := json.Unmarshal(data, &n); err == nil {
		*f = FlexibleID(n.String())
		return nil
	}

	return fmt.Errorf("id must be a string or number")
}

// String returns the string representation of the ID.
func (f FlexibleID) String() string {
	return string(f)
}

// JiraIssue represents a Jira issue with all its fields.
// This is the main entity returned by Jira API operations.
type JiraIssue struct {
	ID     FlexibleID `json:"id"`
	Key    string     `json:"key"`
	Fields JiraFields `json:"fields"`
}

// JiraFields contains all the field data for a Jira issue.
type JiraFields struct {
	Summary     string    `json:"summary"`
	Description string    `json:"description"`
	IssueType   IssueType `json:"issuetype"`
	Project     Project   `json:"project"`
	Status      Status    `json:"status"`
	Assignee    *User     `json:"assignee,omitempty"`
	Reporter    *User     `json:"reporter,omitempty"`
	Created     string    `json:"created"`
	Updated     string    `json:"updated"`
}

// IssueType represents a Jira issue type (e.g., Bug, Story, Task).
type IssueType struct {
	ID   FlexibleID `json:"id"`
	Name string     `json:"name"`
}

// Project represents a Jira project.
type Project struct {
	ID   FlexibleID `json:"id"`
	Key  string     `json:"key"`
	Name string     `json:"name"`
}

// Status represents a Jira issue status (e.g., Open, In Progress, Done).
type Status struct {
	ID   FlexibleID `json:"id"`
	Name string     `json:"name"`
}

// User represents a Jira user.
type User struct {
	Name         string `json:"name"`
	DisplayName  string `json:"displayName"`
	EmailAddress string `json:"emailAddress"`
}

// SearchResults represents the results of a JQL search.
type SearchResults struct {
	Issues     []JiraIssue `json:"issues"`
	Total      int         `json:"total"`
	StartAt    int         `json:"startAt"`
	MaxResults int         `json:"maxResults"`
}

// JiraIssueCreate represents the request body for creating a new Jira issue.
type JiraIssueCreate struct {
	Fields JiraFieldsCreate `json:"fields"`
}

// JiraFieldsCreate contains the fields required to create a new issue.
type JiraFieldsCreate struct {
	Summary     string       `json:"summary"`
	Description string       `json:"description,omitempty"`
	IssueType   IssueTypeRef `json:"issuetype"`
	Project     ProjectRef   `json:"project"`
	Assignee    *UserRef     `json:"assignee,omitempty"`
}

// IssueTypeRef is a reference to an issue type (used in create/update operations).
type IssueTypeRef struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// ProjectRef is a reference to a project (used in create/update operations).
type ProjectRef struct {
	ID  string `json:"id,omitempty"`
	Key string `json:"key,omitempty"`
}

// UserRef is a reference to a user (used in create/update operations).
type UserRef struct {
	Name string `json:"name"`
}

// JiraIssueUpdate represents the request body for updating a Jira issue.
type JiraIssueUpdate struct {
	Fields JiraFieldsUpdate `json:"fields,omitempty"`
	Update JiraUpdateOps    `json:"update,omitempty"`
}

// JiraFieldsUpdate contains the fields that can be updated on an issue.
type JiraFieldsUpdate struct {
	Summary     string   `json:"summary,omitempty"`
	Description string   `json:"description,omitempty"`
	Assignee    *UserRef `json:"assignee,omitempty"`
}

// JiraUpdateOps contains update operations for complex field updates.
type JiraUpdateOps struct {
	// This can be extended with specific update operations as needed
}

// IssueTransition represents a workflow transition request.
type IssueTransition struct {
	Transition TransitionRef `json:"transition"`
}

// TransitionRef is a reference to a workflow transition.
type TransitionRef struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// Comment represents a comment on a Jira issue.
type Comment struct {
	Body string `json:"body"`
}
