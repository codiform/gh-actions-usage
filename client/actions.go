package client

import (
	"fmt"
	"time"
)

const (
	// perPage is the largest page size GitHub allows on the endpoints used here.
	perPage = 100
	// maxRunsPerQuery is the most results GitHub will return for one workflow runs query, regardless of paging.
	maxRunsPerQuery = 1000
	// githubActionsAppID identifies the GitHub Actions app, so that check runs from other apps can be excluded.
	githubActionsAppID = 15368
	// day is the window used to split a runs query that would otherwise exceed maxRunsPerQuery.
	day = 24 * time.Hour
)

// WorkflowRun is one run of a workflow, as listed by the workflow runs endpoint. Only the fields needed to
// join runs to their workflow and to their check runs are decoded.
type WorkflowRun struct {
	ID           uint      `json:"id"`
	WorkflowID   uint      `json:"workflow_id"`
	Path         string    `json:"path"`
	HeadSHA      string    `json:"head_sha"`
	CheckSuiteID uint      `json:"check_suite_id"`
	RunStartedAt time.Time `json:"run_started_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type workflowRunPage struct {
	TotalCount   uint          `json:"total_count"`
	WorkflowRuns []WorkflowRun `json:"workflow_runs"`
}

// GetWorkflowRuns returns every workflow run in the repository created between from and to (inclusive).
//
// GitHub returns at most maxRunsPerQuery results for a single query, so when the window holds more runs than
// that and spans at least a day, the window is re-queried one UTC day at a time. A window shorter than a day
// holding more than maxRunsPerQuery runs is truncated to the first maxRunsPerQuery.
func (c *Client) GetWorkflowRuns(repository Repository, from, to time.Time) ([]WorkflowRun, error) {
	first, total, err := c.getWorkflowRunPage(repository, from, to, 1)
	if err != nil {
		return nil, err
	}
	if total > maxRunsPerQuery && to.Sub(from) >= day {
		return c.getWorkflowRunsByDay(repository, from, to)
	}
	runs := first
	for page := 2; uint(len(runs)) < total && len(first) == perPage; page++ {
		var next []WorkflowRun
		next, _, err = c.getWorkflowRunPage(repository, from, to, page)
		if err != nil {
			return nil, err
		}
		if len(next) == 0 {
			break
		}
		runs = append(runs, next...)
	}
	return runs, nil
}

func (c *Client) getWorkflowRunsByDay(repository Repository, from, to time.Time) ([]WorkflowRun, error) {
	var runs []WorkflowRun
	for start := from; !start.After(to); start = start.Add(day) {
		end := start.Add(day - time.Second)
		if end.After(to) {
			end = to
		}
		dayRuns, err := c.GetWorkflowRuns(repository, start, end)
		if err != nil {
			return nil, err
		}
		runs = append(runs, dayRuns...)
	}
	return runs, nil
}

func (c *Client) getWorkflowRunPage(repository Repository, from, to time.Time, page int) ([]WorkflowRun, uint, error) {
	response := workflowRunPage{}
	path := fmt.Sprintf("repos/%s/actions/runs?per_page=%d&page=%d&created=%s..%s",
		repository.FullName, perPage, page, from.UTC().Format(time.RFC3339), to.UTC().Format(time.RFC3339))
	err := c.Rest.Get(path, &response)
	if err != nil {
		if is404(err) {
			return nil, 0, nil
		}
		return nil, 0, fmt.Errorf("could not get workflow runs: %w", err)
	}
	return response.WorkflowRuns, response.TotalCount, nil
}

// CheckRun is one job of a workflow run, as reported by the check runs endpoint for a commit.
type CheckRun struct {
	ID          uint       `json:"id"`
	Name        string     `json:"name"`
	Status      string     `json:"status"`
	Conclusion  string     `json:"conclusion"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt time.Time  `json:"completed_at"`
	CheckSuite  CheckSuite `json:"check_suite"`
}

// CheckSuite identifies the check suite a CheckRun belongs to; a workflow run's CheckSuiteID refers to the same suite.
type CheckSuite struct {
	ID uint `json:"id"`
}

type checkRunPage struct {
	TotalCount uint       `json:"total_count"`
	CheckRuns  []CheckRun `json:"check_runs"`
}

// GetCheckRuns returns every GitHub Actions check run for the specified commit.
func (c *Client) GetCheckRuns(repository Repository, sha string) ([]CheckRun, error) {
	var checkRuns []CheckRun
	for page := 1; ; page++ {
		response := checkRunPage{}
		path := fmt.Sprintf("repos/%s/commits/%s/check-runs?app_id=%d&per_page=%d&page=%d",
			repository.FullName, sha, githubActionsAppID, perPage, page)
		err := c.Rest.Get(path, &response)
		if err != nil {
			return nil, fmt.Errorf("could not get check runs: %w", err)
		}
		checkRuns = append(checkRuns, response.CheckRuns...)
		if len(response.CheckRuns) < perPage || uint(len(checkRuns)) >= response.TotalCount {
			return checkRuns, nil
		}
	}
}

// RateLimit is the state of the authenticated user's core REST API rate limit.
type RateLimit struct {
	Limit     uint  `json:"limit"`
	Remaining uint  `json:"remaining"`
	Used      uint  `json:"used"`
	Reset     int64 `json:"reset"`
}

// ResetAt returns the time at which the rate limit window resets.
func (r RateLimit) ResetAt() time.Time {
	return time.Unix(r.Reset, 0)
}

type rateLimitResponse struct {
	Resources struct {
		Core RateLimit `json:"core"`
	} `json:"resources"`
}

// GetRateLimit returns the current state of the core REST API rate limit. Calling it does not count against the limit.
func (c *Client) GetRateLimit() (*RateLimit, error) {
	response := rateLimitResponse{}
	err := c.Rest.Get("rate_limit", &response)
	if err != nil {
		return nil, fmt.Errorf("could not get rate limit: %w", err)
	}
	return &response.Resources.Core, nil
}
