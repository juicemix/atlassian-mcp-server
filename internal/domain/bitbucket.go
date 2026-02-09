package domain

// Repository represents a Bitbucket repository.
// This is the main entity returned by repository API operations.
type Repository struct {
	ID      int     `json:"id"`
	Slug    string  `json:"slug"`
	Name    string  `json:"name"`
	Project Project `json:"project"`
	Public  bool    `json:"public"`
}

// Branch represents a Bitbucket branch.
type Branch struct {
	ID              string `json:"id"`
	DisplayID       string `json:"displayId"`
	Type            string `json:"type"`
	LatestCommit    string `json:"latestCommit"`
	LatestChangeset string `json:"latestChangeset,omitempty"`
	IsDefault       bool   `json:"isDefault,omitempty"`
}

// PullRequest represents a Bitbucket pull request.
type PullRequest struct {
	ID          int        `json:"id"`
	Version     int        `json:"version"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	State       string     `json:"state"` // OPEN, MERGED, DECLINED
	Open        bool       `json:"open"`
	Closed      bool       `json:"closed"`
	FromRef     Ref        `json:"fromRef"`
	ToRef       Ref        `json:"toRef"`
	Author      User       `json:"author"`
	Reviewers   []Reviewer `json:"reviewers,omitempty"`
	CreatedDate int64      `json:"createdDate,omitempty"`
	UpdatedDate int64      `json:"updatedDate,omitempty"`
}

// Ref represents a reference (branch or tag) in a repository.
type Ref struct {
	ID         string     `json:"id"`
	DisplayID  string     `json:"displayId,omitempty"`
	Repository Repository `json:"repository"`
}

// Reviewer represents a pull request reviewer.
type Reviewer struct {
	User     User   `json:"user"`
	Approved bool   `json:"approved"`
	Status   string `json:"status"` // APPROVED, UNAPPROVED, NEEDS_WORK
	Role     string `json:"role,omitempty"`
}

// Commit represents a Bitbucket commit.
type Commit struct {
	ID              string `json:"id"`
	DisplayID       string `json:"displayId"`
	Author          User   `json:"author"`
	AuthorTimestamp int64  `json:"authorTimestamp"`
	Message         string `json:"message"`
	Parents         []struct {
		ID        string `json:"id"`
		DisplayID string `json:"displayId"`
	} `json:"parents,omitempty"`
}

// BranchCreate represents the request body for creating a new branch.
type BranchCreate struct {
	Name       string `json:"name"`
	StartPoint string `json:"startPoint"` // The commit ID or branch name to branch from
	Message    string `json:"message,omitempty"`
}

// PullRequestCreate represents the request body for creating a new pull request.
type PullRequestCreate struct {
	Title       string        `json:"title"`
	Description string        `json:"description,omitempty"`
	FromRef     RefCreate     `json:"fromRef"`
	ToRef       RefCreate     `json:"toRef"`
	Reviewers   []ReviewerRef `json:"reviewers,omitempty"`
}

// RefCreate is a reference used in create operations.
type RefCreate struct {
	ID         string        `json:"id"`
	Repository RepositoryRef `json:"repository,omitempty"`
}

// RepositoryRef is a reference to a repository (used in create/update operations).
type RepositoryRef struct {
	Slug    string     `json:"slug,omitempty"`
	Project ProjectRef `json:"project,omitempty"`
}

// ReviewerRef is a reference to a reviewer (used in create operations).
type ReviewerRef struct {
	User UserRef `json:"user"`
}

// CommitOptions contains options for commit history retrieval.
type CommitOptions struct {
	Until string // The commit ID to retrieve commits until (optional)
	Since string // The commit ID to retrieve commits since (optional)
	Path  string // Filter commits by file path (optional)
	Limit int    // Maximum number of commits to return (optional)
	Start int    // The index of the first commit to return (0-based, optional)
}
