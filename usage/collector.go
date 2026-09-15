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
	// callsPerCommit is the estimated number of API calls needed per commit: one, plus an allowance for the
	// extra pages of commits that carry more than a hundred jobs. Job counts are unknown before fetching, so
	// this is an estimate rather than a bound; if it proves short, the transport waits for the limit to reset.
	callsPerCommit = 1.3
	// callsPerRepository is the estimated number of API calls needed to list a repository's workflows and runs:
	// one page of each. Busy repositories need more pages of runs, and the fetch is preflighted separately.
	callsPerRepository = 2
	// listingThreshold is the number of repositories above which the listing phase is announced, since each
	// repository costs a few paced calls before anything else can be said about the collection.
	listingThreshold = 10
	// progressThreshold is the number of commits, across all repositories, above which the fetch of job durations
	// is announced and its progress reported.
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
	// Notify, if set, receives a message as each phase of a long collection begins: listing many repositories,
	// and fetching job durations for many commits, with the size of the job ahead.
	Notify func(message string)
	// ListingProgress, if set, receives the number of repositories listed so far against the total, as each
	// repository of a long collection is listed.
	ListingProgress func(done, total uint)
	// FetchProgress, if set, receives the number of commits whose job durations have been fetched so far, against
	// the total across all repositories, as each commit of a long collection completes. It is called from the fetch
	// goroutines, but never concurrently, and always ends with done equal to total unless the fetch fails.
	FetchProgress func(done, total uint)
}

// Period returns the bounds of the metered billing period containing the instant month, which is the calendar
// month in UTC: from its first instant through its last second, or up to now when the month is the current one.
// The bounds of a month that has not started yet are not meaningful; callers should reject those first.
func Period(month, now time.Time) (time.Time, time.Time) {
	month = month.UTC()
	now = now.UTC().Truncate(time.Second)
	from := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0).Add(-time.Second)
	if to.After(now) {
		to = now
	}
	return from, to
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
	var listing *progress
	if len(repos) > listingThreshold {
		err := c.preflight(uint(len(repos)) * callsPerRepository)
		if err != nil {
			return nil, err
		}
		c.notify(fmt.Sprintf("Listing workflows and runs for %d repositories...", len(repos)))
		listing = &progress{total: uint(len(repos)), report: c.ListingProgress}
	}
	plans := make([]*repoPlan, 0, len(repos))
	var commits uint
	for _, repo := range repos {
		plan, err := c.plan(repo)
		if err != nil {
			return nil, err
		}
		plans = append(plans, plan)
		commits += uint(len(plan.commits))
		listing.advance()
	}

	needed := callsNeeded(commits)
	if commits > 0 {
		err := c.preflight(needed)
		if err != nil {
			return nil, err
		}
	}

	var fetching *progress
	if commits > progressThreshold {
		c.notify(forewarning(plans, commits, needed))
		fetching = &progress{total: commits, report: c.FetchProgress}
	}

	result := make(client.RepoUsage, len(plans))
	for _, plan := range plans {
		err := c.fetchDurations(ctx, plan, fetching)
		if err != nil {
			return nil, err
		}
		result[plan.repo] = plan.usage
	}
	return result, nil
}

func (c *Collector) notify(message string) {
	if c.Notify != nil {
		c.Notify(message)
	}
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

// callsNeeded estimates the API calls that fetching the job durations of the given number of commits will take.
func callsNeeded(commits uint) uint {
	return uint(math.Ceil(float64(commits) * callsPerCommit))
}

// preflight checks that the remaining API budget covers the calls the next phase of the collection will need.
func (c *Collector) preflight(needed uint) error {
	limit, err := c.API.GetRateLimit()
	if err != nil {
		return fmt.Errorf("checking rate limit: %w", err)
	}
	if needed > limit.Remaining {
		return RateLimitError{Needed: needed, Remaining: limit.Remaining, Reset: limit.ResetAt()}
	}
	return nil
}

// forewarning describes the fetch about to begin: how many commits it covers, in which or how many repositories,
// and roughly how many API calls and how long that will take at the transport's pacing rate. The duration is a
// lower bound, since a secondary rate limit can pause the fetch, but the transport announces those pauses itself.
func forewarning(plans []*repoPlan, commits, needed uint) string {
	var busy []*repoPlan
	for _, plan := range plans {
		if len(plan.commits) > 0 {
			busy = append(busy, plan)
		}
	}
	where := fmt.Sprintf("%d repositories", len(busy))
	if len(busy) == 1 {
		where = busy[0].repo.FullName
	}
	estimate := time.Duration(float64(needed) / client.RequestsPerSecond * float64(time.Second))
	return fmt.Sprintf("Fetching job durations for %d commits in %s (about %d API calls, %s)...",
		commits, where, needed, roughly(estimate))
}

// roughly renders a duration estimate in the coarsest unit that still says something: seconds under a minute,
// minutes under an hour, and hours and minutes beyond that.
func roughly(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Minute {
		return plural(int(d/time.Second), "second")
	}
	d = d.Round(time.Minute)
	if d < time.Hour {
		return plural(int(d/time.Minute), "minute")
	}
	hours := plural(int(d/time.Hour), "hour")
	if minutes := int(d % time.Hour / time.Minute); minutes > 0 {
		return hours + " " + plural(minutes, "minute")
	}
	return hours
}

func plural(n int, unit string) string {
	if n == 1 {
		return "1 " + unit
	}
	return fmt.Sprintf("%d %ss", n, unit)
}

// progress counts the commits fetched so far across every repository of a collection and reports each one.
// A nil progress counts nothing, for collections too small to be worth reporting.
type progress struct {
	done, total uint
	report      func(done, total uint)
}

// advance records one more fetched commit. Callers serialize it, so the report never runs concurrently.
func (p *progress) advance() {
	if p == nil {
		return
	}
	p.done++
	if p.report != nil {
		p.report(p.done, p.total)
	}
}

// fetchDurations fetches the check runs of every commit in the plan and adds each job's duration to its workflow.
func (c *Collector) fetchDurations(ctx context.Context, plan *repoPlan, tracker *progress) error {
	var mu sync.Mutex
	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(c.concurrency())
	for _, sha := range plan.commits {
		group.Go(func() error {
			// The API seam carries no context, so cancellation is observed between requests rather than
			// within one: once a sibling fails or the caller cancels, queued commits are skipped.
			if ctx.Err() != nil {
				return ctx.Err()
			}
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
			tracker.advance()
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

// duration returns the milliseconds a check run consumed. Jobs that did not complete, were skipped, or lack
// either timestamp count for nothing; skipped jobs in particular report a completion time before their start time.
func duration(checkRun client.CheckRun) uint {
	if checkRun.Status != statusCompleted || checkRun.Conclusion == conclusionSkipped {
		return 0
	}
	if checkRun.StartedAt.IsZero() || checkRun.CompletedAt.IsZero() {
		return 0
	}
	elapsed := checkRun.CompletedAt.Sub(checkRun.StartedAt)
	if elapsed <= 0 {
		return 0
	}
	return uint(elapsed / time.Millisecond)
}
