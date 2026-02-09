package domain

import (
	"encoding/json"
	"testing"
)

// TestRepositoryJSONSerialization tests that Repository can be marshaled and unmarshaled correctly.
func TestRepositoryJSONSerialization(t *testing.T) {
	repo := Repository{
		ID:   123,
		Slug: "my-repo",
		Name: "My Repository",
		Project: Project{
			ID:   "1",
			Key:  "PROJ",
			Name: "My Project",
		},
		Public: true,
	}

	// Marshal to JSON
	data, err := json.Marshal(repo)
	if err != nil {
		t.Fatalf("Failed to marshal repository: %v", err)
	}

	// Unmarshal back
	var decoded Repository
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal repository: %v", err)
	}

	// Verify fields
	if decoded.ID != repo.ID {
		t.Errorf("Expected ID %d, got %d", repo.ID, decoded.ID)
	}
	if decoded.Slug != repo.Slug {
		t.Errorf("Expected Slug %s, got %s", repo.Slug, decoded.Slug)
	}
	if decoded.Name != repo.Name {
		t.Errorf("Expected Name %s, got %s", repo.Name, decoded.Name)
	}
	if decoded.Project.Key != repo.Project.Key {
		t.Errorf("Expected Project Key %s, got %s", repo.Project.Key, decoded.Project.Key)
	}
	if decoded.Public != repo.Public {
		t.Errorf("Expected Public %v, got %v", repo.Public, decoded.Public)
	}
}

// TestBranchJSONSerialization tests that Branch can be marshaled and unmarshaled correctly.
func TestBranchJSONSerialization(t *testing.T) {
	branch := Branch{
		ID:           "refs/heads/main",
		DisplayID:    "main",
		Type:         "BRANCH",
		LatestCommit: "abc123def456",
		IsDefault:    true,
	}

	// Marshal to JSON
	data, err := json.Marshal(branch)
	if err != nil {
		t.Fatalf("Failed to marshal branch: %v", err)
	}

	// Unmarshal back
	var decoded Branch
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal branch: %v", err)
	}

	// Verify fields
	if decoded.ID != branch.ID {
		t.Errorf("Expected ID %s, got %s", branch.ID, decoded.ID)
	}
	if decoded.DisplayID != branch.DisplayID {
		t.Errorf("Expected DisplayID %s, got %s", branch.DisplayID, decoded.DisplayID)
	}
	if decoded.Type != branch.Type {
		t.Errorf("Expected Type %s, got %s", branch.Type, decoded.Type)
	}
	if decoded.LatestCommit != branch.LatestCommit {
		t.Errorf("Expected LatestCommit %s, got %s", branch.LatestCommit, decoded.LatestCommit)
	}
	if decoded.IsDefault != branch.IsDefault {
		t.Errorf("Expected IsDefault %v, got %v", branch.IsDefault, decoded.IsDefault)
	}
}

// TestPullRequestJSONSerialization tests that PullRequest can be marshaled and unmarshaled correctly.
func TestPullRequestJSONSerialization(t *testing.T) {
	pr := PullRequest{
		ID:          1,
		Version:     2,
		Title:       "Add new feature",
		Description: "This PR adds a new feature",
		State:       "OPEN",
		Open:        true,
		Closed:      false,
		FromRef: Ref{
			ID:        "refs/heads/feature-branch",
			DisplayID: "feature-branch",
			Repository: Repository{
				ID:   123,
				Slug: "my-repo",
				Name: "My Repository",
			},
		},
		ToRef: Ref{
			ID:        "refs/heads/main",
			DisplayID: "main",
			Repository: Repository{
				ID:   123,
				Slug: "my-repo",
				Name: "My Repository",
			},
		},
		Author: User{
			Name:        "jsmith",
			DisplayName: "John Smith",
		},
		Reviewers: []Reviewer{
			{
				User: User{
					Name:        "jdoe",
					DisplayName: "Jane Doe",
				},
				Approved: true,
				Status:   "APPROVED",
			},
		},
		CreatedDate: 1234567890000,
		UpdatedDate: 1234567890000,
	}

	// Marshal to JSON
	data, err := json.Marshal(pr)
	if err != nil {
		t.Fatalf("Failed to marshal pull request: %v", err)
	}

	// Unmarshal back
	var decoded PullRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal pull request: %v", err)
	}

	// Verify fields
	if decoded.ID != pr.ID {
		t.Errorf("Expected ID %d, got %d", pr.ID, decoded.ID)
	}
	if decoded.Title != pr.Title {
		t.Errorf("Expected Title %s, got %s", pr.Title, decoded.Title)
	}
	if decoded.State != pr.State {
		t.Errorf("Expected State %s, got %s", pr.State, decoded.State)
	}
	if decoded.FromRef.DisplayID != pr.FromRef.DisplayID {
		t.Errorf("Expected FromRef DisplayID %s, got %s", pr.FromRef.DisplayID, decoded.FromRef.DisplayID)
	}
	if decoded.ToRef.DisplayID != pr.ToRef.DisplayID {
		t.Errorf("Expected ToRef DisplayID %s, got %s", pr.ToRef.DisplayID, decoded.ToRef.DisplayID)
	}
	if len(decoded.Reviewers) != len(pr.Reviewers) {
		t.Errorf("Expected %d reviewers, got %d", len(pr.Reviewers), len(decoded.Reviewers))
	}
	if len(decoded.Reviewers) > 0 && decoded.Reviewers[0].Status != pr.Reviewers[0].Status {
		t.Errorf("Expected reviewer status %s, got %s", pr.Reviewers[0].Status, decoded.Reviewers[0].Status)
	}
}

// TestCommitJSONSerialization tests that Commit can be marshaled and unmarshaled correctly.
func TestCommitJSONSerialization(t *testing.T) {
	commit := Commit{
		ID:        "abc123def456",
		DisplayID: "abc123d",
		Author: User{
			Name:        "jsmith",
			DisplayName: "John Smith",
		},
		AuthorTimestamp: 1234567890000,
		Message:         "Initial commit",
	}

	// Marshal to JSON
	data, err := json.Marshal(commit)
	if err != nil {
		t.Fatalf("Failed to marshal commit: %v", err)
	}

	// Unmarshal back
	var decoded Commit
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal commit: %v", err)
	}

	// Verify fields
	if decoded.ID != commit.ID {
		t.Errorf("Expected ID %s, got %s", commit.ID, decoded.ID)
	}
	if decoded.DisplayID != commit.DisplayID {
		t.Errorf("Expected DisplayID %s, got %s", commit.DisplayID, decoded.DisplayID)
	}
	if decoded.Message != commit.Message {
		t.Errorf("Expected Message %s, got %s", commit.Message, decoded.Message)
	}
	if decoded.AuthorTimestamp != commit.AuthorTimestamp {
		t.Errorf("Expected AuthorTimestamp %d, got %d", commit.AuthorTimestamp, decoded.AuthorTimestamp)
	}
}

// TestBranchCreateJSONSerialization tests that BranchCreate can be marshaled correctly.
func TestBranchCreateJSONSerialization(t *testing.T) {
	branchCreate := BranchCreate{
		Name:       "feature/new-feature",
		StartPoint: "main",
		Message:    "Creating new feature branch",
	}

	// Marshal to JSON
	data, err := json.Marshal(branchCreate)
	if err != nil {
		t.Fatalf("Failed to marshal branch create: %v", err)
	}

	// Unmarshal back
	var decoded BranchCreate
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal branch create: %v", err)
	}

	// Verify fields
	if decoded.Name != branchCreate.Name {
		t.Errorf("Expected Name %s, got %s", branchCreate.Name, decoded.Name)
	}
	if decoded.StartPoint != branchCreate.StartPoint {
		t.Errorf("Expected StartPoint %s, got %s", branchCreate.StartPoint, decoded.StartPoint)
	}
	if decoded.Message != branchCreate.Message {
		t.Errorf("Expected Message %s, got %s", branchCreate.Message, decoded.Message)
	}
}

// TestPullRequestCreateJSONSerialization tests that PullRequestCreate can be marshaled correctly.
func TestPullRequestCreateJSONSerialization(t *testing.T) {
	prCreate := PullRequestCreate{
		Title:       "Add new feature",
		Description: "This PR adds a new feature",
		FromRef: RefCreate{
			ID: "refs/heads/feature-branch",
			Repository: RepositoryRef{
				Slug: "my-repo",
				Project: ProjectRef{
					Key: "PROJ",
				},
			},
		},
		ToRef: RefCreate{
			ID: "refs/heads/main",
			Repository: RepositoryRef{
				Slug: "my-repo",
				Project: ProjectRef{
					Key: "PROJ",
				},
			},
		},
		Reviewers: []ReviewerRef{
			{
				User: UserRef{
					Name: "jdoe",
				},
			},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(prCreate)
	if err != nil {
		t.Fatalf("Failed to marshal pull request create: %v", err)
	}

	// Unmarshal back
	var decoded PullRequestCreate
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal pull request create: %v", err)
	}

	// Verify fields
	if decoded.Title != prCreate.Title {
		t.Errorf("Expected Title %s, got %s", prCreate.Title, decoded.Title)
	}
	if decoded.FromRef.ID != prCreate.FromRef.ID {
		t.Errorf("Expected FromRef ID %s, got %s", prCreate.FromRef.ID, decoded.FromRef.ID)
	}
	if decoded.ToRef.ID != prCreate.ToRef.ID {
		t.Errorf("Expected ToRef ID %s, got %s", prCreate.ToRef.ID, decoded.ToRef.ID)
	}
	if len(decoded.Reviewers) != len(prCreate.Reviewers) {
		t.Errorf("Expected %d reviewers, got %d", len(prCreate.Reviewers), len(decoded.Reviewers))
	}
}
