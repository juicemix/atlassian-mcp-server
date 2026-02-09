package domain

// BuildPlan represents a Bamboo build plan.
// This is the main entity returned by build plan API operations.
type BuildPlan struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	ShortName string `json:"shortName"`
	ShortKey  string `json:"shortKey"`
	Type      string `json:"type"`
	Enabled   bool   `json:"enabled"`
}

// BuildResult represents the result of a build execution.
type BuildResult struct {
	Key                string `json:"key"`
	Number             int    `json:"number"`
	State              string `json:"state"`          // Successful, Failed, Unknown
	LifeCycleState     string `json:"lifeCycleState"` // Pending, Queued, InProgress, Finished
	BuildStartedTime   string `json:"buildStartedTime"`
	BuildCompletedTime string `json:"buildCompletedTime"`
	BuildDuration      int64  `json:"buildDuration"`
	BuildReason        string `json:"buildReason"`
}

// DeploymentProject represents a Bamboo deployment project.
type DeploymentProject struct {
	ID           int           `json:"id"`
	Name         string        `json:"name"`
	PlanKey      string        `json:"planKey"`
	Environments []Environment `json:"environments,omitempty"`
}

// Environment represents a deployment environment within a deployment project.
type Environment struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// DeploymentResult represents the result of a deployment execution.
type DeploymentResult struct {
	ID                    int    `json:"id"`
	DeploymentVersionName string `json:"deploymentVersionName"`
	DeploymentState       string `json:"deploymentState"`
	LifeCycleState        string `json:"lifeCycleState"`
	StartedDate           string `json:"startedDate"`
	FinishedDate          string `json:"finishedDate"`
}
