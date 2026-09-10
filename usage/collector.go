// Package usage computes GitHub Actions usage for repositories from the durations of their jobs.
package usage

import (
	"context"
	"fmt"
	"math"
	"path"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/geoffreywiseman/gh-actions-usage/client"
)

const (
	// defaultConcurrency is the number of check-run requests kept in flight at once.
	defaultConcurrency = 8
	// callsPerCommit is the estimated number of API calls needed per commit: one, plus extra pages
	// for commits that carry many jobs.
	callsPerCommit = 1.3
	// progressThreshold is the number of commits above which a repository's collection is announced.
	progressThreshold = 20
	// statusCompleted and conclusionSkipped are the check-run states that decide whether a job's time counts.
	statusCompleted      = "completed"
	conclusionSkipped    = "skipped"
	deletedWorkflowState = "deleted"
)

// API is the subset of the GitHub client the collector depends on.
type API interface {
	GetWorkflows(repository client.Repository) ([]client.Workflow, error)
	GetWorkflowRuns(repository client.Repository, from, to time.Time) ([]client.WorkflowRun, error)
	GetCheckRuns(repository client.Repository, sha string) ([]client.CheckRun, error)
	GetRateLimit() (*client.RateLimit, error)
}

// RateLimitError is returned when the remaining API budget is too small for the requested collection.
type RateLimitError struct {
	Needed    uint
	Remaining uint
	Reset     time.Time
}

// Error returns a message describing the shortfall and when the budget resets.
func (e RateLimitError) Error() string {
	return fmt.Sprintf("GitHub API rate limit too low: about %d calls needed, %d remaining until %s",
		e.Needed, e.Remaining, e.Reset.Format(time.Kitchen))
}

// Collector computes usage for repositories over a period by summing the durations of their jobs.
type Collector struct {
	API  API
	From time.Time
	To   time.Time
	// Concurrency is the number of check-run requests kept in flight at once; zero means defaultConcurrency.
	Concurrency int
	// Notify, if set, receives progress messages for long collections.
	Notify func(message string)
}

// CurrentPeriod returns the bounds of the current metered billing period, which is the calendar month in UTC,
// from its first instant up to now.
func CurrentPeriod(now time.Time) (time.Time, time.Time) {
	now = now.UTC().Truncate(time.Second)
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC), now
}

// repoPlan holds what was learned about a repository from the cheap listing calls, before check runs are fetched.
type repoPlan struct {
	repo      *client.Repository
	usage     client.WorkflowUsage
	workflows map[uint]client.Workflow // by check suite id
	commits   []string
}

// Collect returns the usage of every workflow in each repository for the collector's period.
func (c *Collector) Collect(ctx context.Context, repos []*client.Repository) (client.RepoUsage, error) {
	plans := make([]*repoPlan, 0, len(repos))
	var commits uint
	for _, repo := range repos {
		plan, err := c.plan(repo)
		if err != nil {
			return nil, err
		}
		plans = append(plans, plan)
		commits += uint(len(plan.commits))
	}

	err := c.preflight(commits)
	if err != nil {
		return nil, err
	}

	result := make(client.RepoUsage, len(plans))
	for _, plan := range plans {
		err = c.fetchDurations(ctx, plan)
		if err != nil {
			return nil, err
		}
		result[plan.repo] = plan.usage
	}
	return result, nil
}

// plan lists a repository's workflows and the runs in the period, seeding every workflow at zero usage and
// mapping each run's check suite to its workflow.
func (c *Collector) plan(repo *client.Repository) (*repoPlan, error) {
	workflows, err := c.API.GetWorkflows(*repo)
	if err != nil {
		return nil, fmt.Errorf("listing workflows of %s: %w", repo.FullName, err)
	}
	runs, err := c.API.GetWorkflowRuns(*repo, c.From, c.To)
	if err != nil {
		return nil, fmt.Errorf("listing workflow runs of %s: %w", repo.FullName, err)
	}

	plan := &repoPlan{
		repo:      repo,
		usage:     make(client.WorkflowUsage, len(workflows)),
		workflows: make(map[uint]client.Workflow, len(runs)),
	}
	byID := make(map[uint]client.Workflow, len(workflows))
	for _, workflow := range workflows {
		byID[workflow.ID] = workflow
		plan.usage[workflow] = 0
	}
	seen := make(map[string]bool, len(runs))
	for _, run := range runs {
		workflow, ok := byID[run.WorkflowID]
		if !ok {
			workflow = deletedWorkflow(run)
			byID[run.WorkflowID] = workflow
			plan.usage[workflow] = 0
		}
		plan.workflows[run.CheckSuiteID] = workflow
		if !seen[run.HeadSHA] {
			seen[run.HeadSHA] = true
			plan.commits = append(plan.commits, run.HeadSHA)
		}
	}
	return plan, nil
}

// deletedWorkflow describes a workflow that has runs in the period but no longer exists in the repository.
func deletedWorkflow(run client.WorkflowRun) client.Workflow {
	return client.Workflow{ID: run.WorkflowID, Name: path.Base(run.Path), Path: run.Path, State: deletedWorkflowState}
}

// preflight checks that the remaining API budget covers the check-run calls the collection will need.
func (c *Collector) preflight(commits uint) error {
	if commits == 0 {
		return nil
	}
	limit, err := c.API.GetRateLimit()
	if err != nil {
		return fmt.Errorf("checking rate limit: %w", err)
	}
	needed := uint(math.Ceil(float64(commits) * callsPerCommit))
	if needed > limit.Remaining {
		return RateLimitError{Needed: needed, Remaining: limit.Remaining, Reset: limit.ResetAt()}
	}
	return nil
}

// fetchDurations fetches the check runs of every commit in the plan and adds each job's duration to its workflow.
func (c *Collector) fetchDurations(ctx context.Context, plan *repoPlan) error {
	if len(plan.commits) > progressThreshold && c.Notify != nil {
		c.Notify(fmt.Sprintf("%s: fetching job durations for %d commits...", plan.repo.FullName, len(plan.commits)))
	}
	var mu sync.Mutex
	group, _ := errgroup.WithContext(ctx)
	group.SetLimit(c.concurrency())
	for _, sha := range plan.commits {
		group.Go(func() error {
			checkRuns, err := c.API.GetCheckRuns(*plan.repo, sha)
			if err != nil {
				return fmt.Errorf("listing check runs of %s@%s: %w", plan.repo.FullName, sha, err)
			}
			mu.Lock()
			defer mu.Unlock()
			for _, checkRun := range checkRuns {
				workflow, ok := plan.workflows[checkRun.CheckSuite.ID]
				if ok {
					plan.usage[workflow] += duration(checkRun)
				}
			}
			return nil
		})
	}
	return group.Wait() //nolint:wrapcheck // errors are wrapped where they arise
}

func (c *Collector) concurrency() int {
	if c.Concurrency > 0 {
		return c.Concurrency
	}
	return defaultConcurrency
}

// duration returns the milliseconds a check run consumed. Jobs that did not complete, or were skipped, count for
// nothing; skipped jobs in particular report a completion time before their start time.
func duration(checkRun client.CheckRun) uint {
	if checkRun.Status != statusCompleted || checkRun.Conclusion == conclusionSkipped {
		return 0
	}
	elapsed := checkRun.CompletedAt.Sub(checkRun.StartedAt)
	if elapsed <= 0 {
		return 0
	}
	return uint(elapsed / time.Millisecond)
}
