package usage

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/geoffreywiseman/gh-actions-usage/client"
)

var (
	errAPI = errors.New("api failed")
	from   = time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	to     = time.Date(2026, time.September, 9, 12, 0, 0, 0, time.UTC)
	t0     = time.Date(2026, time.September, 5, 10, 0, 0, 0, time.UTC)

	ci      = client.Workflow{ID: 1, Name: "CI", Path: ".github/workflows/ci.yml", State: "active"}
	release = client.Workflow{ID: 2, Name: "release", Path: ".github/workflows/release.yml", State: "active"}
	repo    = &client.Repository{FullName: "codiform/gh-actions-usage", Name: "gh-actions-usage"}
)

type apiMock struct {
	mock.Mock
}

func (m *apiMock) GetWorkflows(repository client.Repository) ([]client.Workflow, error) {
	args := m.Called(repository)
	return args.Get(0).([]client.Workflow), args.Error(1) //nolint:wrapcheck
}

func (m *apiMock) GetWorkflowRuns(repository client.Repository, from, to time.Time) ([]client.WorkflowRun, error) {
	args := m.Called(repository, from, to)
	return args.Get(0).([]client.WorkflowRun), args.Error(1) //nolint:wrapcheck
}

func (m *apiMock) GetCheckRuns(repository client.Repository, sha string) ([]client.CheckRun, error) {
	args := m.Called(repository, sha)
	return args.Get(0).([]client.CheckRun), args.Error(1) //nolint:wrapcheck
}

func (m *apiMock) GetRateLimit() (*client.RateLimit, error) {
	args := m.Called()
	return args.Get(0).(*client.RateLimit), args.Error(1) //nolint:wrapcheck
}

func job(suite uint, seconds int) client.CheckRun {
	return client.CheckRun{
		Status: "completed", Conclusion: "success",
		StartedAt: t0, CompletedAt: t0.Add(time.Duration(seconds) * time.Second),
		CheckSuite: client.CheckSuite{ID: suite},
	}
}

func newCollector(api *apiMock) *Collector {
	return &Collector{API: api, From: from, To: to}
}

func expectListing(api *apiMock, workflows []client.Workflow, runs []client.WorkflowRun) {
	api.On("GetWorkflows", *repo).Return(workflows, nil)
	api.On("GetWorkflowRuns", *repo, from, to).Return(runs, nil)
}

func plentyOfBudget(api *apiMock) {
	api.On("GetRateLimit").Return(&client.RateLimit{Limit: 5000, Remaining: 4000}, nil)
}

func TestPeriod(t *testing.T) {
	utc := func(year int, month time.Month, day, hour, minute, second int) time.Time {
		return time.Date(year, month, day, hour, minute, second, 0, time.UTC)
	}
	est := time.FixedZone("EST", -5*3600)
	now := time.Date(2026, time.September, 9, 12, 34, 56, 789, time.UTC)
	tests := []struct {
		name       string
		month, now time.Time
		from, to   time.Time
	}{
		{
			name: "current month runs up to now", month: utc(2026, time.September, 1, 0, 0, 0), now: now,
			from: utc(2026, time.September, 1, 0, 0, 0), to: utc(2026, time.September, 9, 12, 34, 56),
		},
		{
			name: "any instant in the month selects it", month: utc(2026, time.September, 23, 8, 15, 0), now: now,
			from: utc(2026, time.September, 1, 0, 0, 0), to: utc(2026, time.September, 9, 12, 34, 56),
		},
		{
			name: "first instant of month", month: utc(2026, time.February, 1, 0, 0, 0), now: utc(2026, time.February, 1, 0, 0, 0),
			from: utc(2026, time.February, 1, 0, 0, 0), to: utc(2026, time.February, 1, 0, 0, 0),
		},
		{
			name:  "local evening is next month in UTC",
			month: time.Date(2026, time.August, 31, 22, 0, 0, 0, est), now: time.Date(2026, time.August, 31, 22, 0, 0, 0, est),
			from: utc(2026, time.September, 1, 0, 0, 0), to: utc(2026, time.September, 1, 3, 0, 0),
		},
		{
			name: "past month runs through its last second", month: utc(2026, time.August, 1, 0, 0, 0), now: now,
			from: utc(2026, time.August, 1, 0, 0, 0), to: utc(2026, time.August, 31, 23, 59, 59),
		},
		{
			name: "leap February", month: utc(2024, time.February, 1, 0, 0, 0), now: now,
			from: utc(2024, time.February, 1, 0, 0, 0), to: utc(2024, time.February, 29, 23, 59, 59),
		},
		{
			name: "December ends before the new year", month: utc(2025, time.December, 1, 0, 0, 0), now: now,
			from: utc(2025, time.December, 1, 0, 0, 0), to: utc(2025, time.December, 31, 23, 59, 59),
		},
		{
			name: "previous month is complete moments after it ended", month: utc(2026, time.August, 1, 0, 0, 0),
			now:  utc(2026, time.September, 1, 0, 0, 0),
			from: utc(2026, time.August, 1, 0, 0, 0), to: utc(2026, time.August, 31, 23, 59, 59),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			from, to := Period(tc.month, tc.now)
			assert.Equal(t, tc.from, from)
			assert.Equal(t, tc.to, to)
			assert.Equal(t, time.UTC, from.Location())
		})
	}
}

func TestCollector_IdleWorkflowsAppearAtZero(t *testing.T) {
	// Given
	api := new(apiMock)
	expectListing(api, []client.Workflow{ci, release}, nil)

	// When
	usage, err := newCollector(api).Collect(context.Background(), []*client.Repository{repo})

	// Then
	require.NoError(t, err)
	assert.Equal(t, client.WorkflowUsage{ci: 0, release: 0}, usage[repo])
	api.AssertNotCalled(t, "GetRateLimit")
	api.AssertNotCalled(t, "GetCheckRuns", mock.Anything, mock.Anything)
}

func TestCollector_AttributesJobsToWorkflowsBySuite(t *testing.T) {
	// Given
	api := new(apiMock)
	expectListing(api, []client.Workflow{ci, release}, []client.WorkflowRun{
		{ID: 10, WorkflowID: ci.ID, CheckSuiteID: 100, HeadSHA: "aaa"},
		{ID: 11, WorkflowID: release.ID, CheckSuiteID: 110, HeadSHA: "aaa"},
		{ID: 12, WorkflowID: ci.ID, CheckSuiteID: 120, HeadSHA: "bbb"},
	})
	plentyOfBudget(api)
	api.On("GetCheckRuns", *repo, "aaa").Return([]client.CheckRun{
		job(100, 60), job(100, 30), // two CI jobs
		job(110, 10),  // one release job
		job(999, 500), // some other run on the same commit, outside the period
	}, nil)
	api.On("GetCheckRuns", *repo, "bbb").Return([]client.CheckRun{job(120, 45)}, nil)

	// When
	usage, err := newCollector(api).Collect(context.Background(), []*client.Repository{repo})

	// Then
	require.NoError(t, err)
	assert.Equal(t, client.WorkflowUsage{ci: 135_000, release: 10_000}, usage[repo])
	api.AssertNumberOfCalls(t, "GetCheckRuns", 2)
}

func TestCollector_IgnoresJobsWithoutUsableDurations(t *testing.T) {
	// Given
	api := new(apiMock)
	expectListing(api, []client.Workflow{ci}, []client.WorkflowRun{{ID: 10, WorkflowID: ci.ID, CheckSuiteID: 100, HeadSHA: "aaa"}})
	plentyOfBudget(api)
	skipped := job(100, -8)
	skipped.Conclusion = "skipped"
	inProgress := job(100, 0)
	inProgress.Status = "in_progress"
	inProgress.CompletedAt = time.Time{}
	negative := job(100, -3)
	noStart := job(100, 60)
	noStart.StartedAt = time.Time{}
	api.On("GetCheckRuns", *repo, "aaa").Return([]client.CheckRun{skipped, inProgress, negative, noStart, job(100, 7)}, nil)

	// When
	usage, err := newCollector(api).Collect(context.Background(), []*client.Repository{repo})

	// Then
	require.NoError(t, err)
	assert.Equal(t, client.WorkflowUsage{ci: 7_000}, usage[repo])
}

func TestCollector_KeepsRunsOfDeletedWorkflows(t *testing.T) {
	// Given
	api := new(apiMock)
	expectListing(api, []client.Workflow{ci}, []client.WorkflowRun{
		{ID: 10, WorkflowID: 77, Path: ".github/workflows/old.yml", CheckSuiteID: 100, HeadSHA: "aaa"},
	})
	plentyOfBudget(api)
	api.On("GetCheckRuns", *repo, "aaa").Return([]client.CheckRun{job(100, 20)}, nil)

	// When
	usage, err := newCollector(api).Collect(context.Background(), []*client.Repository{repo})

	// Then
	require.NoError(t, err)
	old := client.Workflow{ID: 77, Name: "old.yml", Path: ".github/workflows/old.yml", State: "deleted"}
	assert.Equal(t, client.WorkflowUsage{ci: 0, old: 20_000}, usage[repo])
}

func TestCollector_RefusesWhenBudgetIsShort(t *testing.T) {
	// Given
	api := new(apiMock)
	runs := make([]client.WorkflowRun, 0, 100)
	for i := range 100 {
		runs = append(runs, client.WorkflowRun{ID: uint(i), WorkflowID: ci.ID, CheckSuiteID: uint(i), HeadSHA: string(rune('a' + i))})
	}
	expectListing(api, []client.Workflow{ci}, runs)
	reset := time.Date(2026, time.September, 9, 13, 0, 0, 0, time.UTC)
	api.On("GetRateLimit").Return(&client.RateLimit{Limit: 5000, Remaining: 50, Reset: reset.Unix()}, nil)

	// When
	usage, err := newCollector(api).Collect(context.Background(), []*client.Repository{repo})

	// Then
	var limitErr RateLimitError
	require.ErrorAs(t, err, &limitErr)
	assert.Equal(t, uint(130), limitErr.Needed)
	assert.Equal(t, uint(50), limitErr.Remaining)
	assert.True(t, reset.Equal(limitErr.Reset), "reset %s should equal %s", limitErr.Reset, reset)
	assert.Nil(t, usage)
	api.AssertNotCalled(t, "GetCheckRuns", mock.Anything, mock.Anything)
}

func TestCollector_PreflightsAllRepositoriesTogether(t *testing.T) {
	// Given
	api := new(apiMock)
	other := &client.Repository{FullName: "codiform/other", Name: "other"}
	expectListing(api, []client.Workflow{ci}, []client.WorkflowRun{{ID: 10, WorkflowID: ci.ID, CheckSuiteID: 100, HeadSHA: "aaa"}})
	api.On("GetWorkflows", *other).Return([]client.Workflow{release}, nil)
	api.On("GetWorkflowRuns", *other, from, to).Return([]client.WorkflowRun{{ID: 20, WorkflowID: release.ID, CheckSuiteID: 200, HeadSHA: "ccc"}}, nil)
	plentyOfBudget(api)
	api.On("GetCheckRuns", *repo, "aaa").Return([]client.CheckRun{job(100, 1)}, nil)
	api.On("GetCheckRuns", *other, "ccc").Return([]client.CheckRun{job(200, 2)}, nil)

	// When
	usage, err := newCollector(api).Collect(context.Background(), []*client.Repository{repo, other})

	// Then
	require.NoError(t, err)
	assert.Equal(t, client.WorkflowUsage{ci: 1_000}, usage[repo])
	assert.Equal(t, client.WorkflowUsage{release: 2_000}, usage[other])
	api.AssertNumberOfCalls(t, "GetRateLimit", 1)
}

// busyRepo expects the listing of a repository with the given number of commits, each with an empty fetch.
func busyRepo(api *apiMock, repository *client.Repository, commits int) {
	runs := make([]client.WorkflowRun, 0, commits)
	for i := range commits {
		sha := fmt.Sprintf("%s-%d", repository.Name, i)
		runs = append(runs, client.WorkflowRun{ID: uint(i), WorkflowID: ci.ID, CheckSuiteID: uint(i), HeadSHA: sha})
		api.On("GetCheckRuns", *repository, sha).Return([]client.CheckRun{}, nil)
	}
	api.On("GetWorkflows", *repository).Return([]client.Workflow{ci}, nil)
	api.On("GetWorkflowRuns", *repository, from, to).Return(runs, nil)
}

// recorder captures what a collector reports through its hooks.
type recorder struct {
	messages []string
	listed   []uint
	done     []uint
	totals   map[uint]bool
}

func record(collector *Collector) *recorder {
	r := &recorder{totals: make(map[uint]bool)}
	collector.Notify = func(message string) { r.messages = append(r.messages, message) }
	collector.ListingProgress = func(done, _ uint) { r.listed = append(r.listed, done) }
	collector.FetchProgress = func(done, total uint) {
		r.done = append(r.done, done)
		r.totals[total] = true
	}
	return r
}

// manyRepos expects the listing of enough idle repositories to pass the listing threshold.
func manyRepos(api *apiMock) []*client.Repository {
	repos := make([]*client.Repository, 0, listingThreshold+1)
	for i := range listingThreshold + 1 {
		repository := &client.Repository{FullName: fmt.Sprintf("codiform/repo-%d", i), Name: fmt.Sprintf("repo-%d", i)}
		busyRepo(api, repository, 0)
		repos = append(repos, repository)
	}
	return repos
}

func TestCollector_ForewarnsAndReportsProgressOfLargeCollections(t *testing.T) {
	// Given
	api := new(apiMock)
	busyRepo(api, repo, progressThreshold+1)
	plentyOfBudget(api)
	collector := newCollector(api)
	reported := record(collector)

	// When
	_, err := collector.Collect(context.Background(), []*client.Repository{repo})

	// Then
	require.NoError(t, err)
	assert.Equal(t, []string{"Fetching job durations for 21 commits in codiform/gh-actions-usage (about 28 API calls, 3 seconds)..."}, reported.messages)
	assert.Empty(t, reported.listed)
	assert.Len(t, reported.done, progressThreshold+1)
	assert.IsIncreasing(t, reported.done)
	assert.Equal(t, uint(progressThreshold+1), reported.done[len(reported.done)-1])
	assert.Equal(t, map[uint]bool{progressThreshold + 1: true}, reported.totals)
}

func TestCollector_CountsProgressAcrossRepositories(t *testing.T) {
	// Given: three repositories, one of them idle, whose commits together pass the threshold
	api := new(apiMock)
	other := &client.Repository{FullName: "codiform/other", Name: "other"}
	idle := &client.Repository{FullName: "codiform/idle", Name: "idle"}
	busyRepo(api, repo, 15)
	busyRepo(api, other, 15)
	busyRepo(api, idle, 0)
	plentyOfBudget(api)
	collector := newCollector(api)
	reported := record(collector)

	// When
	_, err := collector.Collect(context.Background(), []*client.Repository{repo, other, idle})

	// Then: one forewarning for the busy repositories, and one running count that spans both
	require.NoError(t, err)
	assert.Equal(t, []string{"Fetching job durations for 30 commits in 2 repositories (about 39 API calls, 4 seconds)..."}, reported.messages)
	assert.Len(t, reported.done, 30)
	assert.IsIncreasing(t, reported.done)
	assert.Equal(t, map[uint]bool{30: true}, reported.totals)
}

func TestCollector_AnnouncesAndCountsListingOfManyRepositories(t *testing.T) {
	// Given
	api := new(apiMock)
	repos := manyRepos(api)
	plentyOfBudget(api)
	collector := newCollector(api)
	reported := record(collector)

	// When
	_, err := collector.Collect(context.Background(), repos)

	// Then: the listing is preflighted and announced, and counted repository by repository
	require.NoError(t, err)
	assert.Equal(t, []string{"Listing workflows and runs for 11 repositories..."}, reported.messages)
	assert.Equal(t, []uint{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}, reported.listed)
	assert.Empty(t, reported.done)
	api.AssertNumberOfCalls(t, "GetRateLimit", 1)
}

func TestCollector_RefusesToListManyRepositoriesWhenBudgetIsShort(t *testing.T) {
	// Given
	api := new(apiMock)
	repos := manyRepos(api)
	api.On("GetRateLimit").Return(&client.RateLimit{Limit: 5000, Remaining: 21}, nil)
	collector := newCollector(api)
	reported := record(collector)

	// When
	usage, err := collector.Collect(context.Background(), repos)

	// Then: two calls per repository were needed, and none were made
	var limitErr RateLimitError
	require.ErrorAs(t, err, &limitErr)
	assert.Equal(t, uint(22), limitErr.Needed)
	assert.Nil(t, usage)
	assert.Empty(t, reported.messages)
	api.AssertNotCalled(t, "GetWorkflows", mock.Anything)
}

func TestCollector_IsQuietForSmallCollections(t *testing.T) {
	// Given
	api := new(apiMock)
	busyRepo(api, repo, progressThreshold)
	plentyOfBudget(api)
	collector := newCollector(api)
	reported := record(collector)

	// When
	_, err := collector.Collect(context.Background(), []*client.Repository{repo})

	// Then
	require.NoError(t, err)
	assert.Empty(t, reported.messages)
	assert.Empty(t, reported.done)
}

func TestCollector_DoesNotForewarnWhenBudgetIsShort(t *testing.T) {
	// Given
	api := new(apiMock)
	busyRepo(api, repo, progressThreshold+1)
	api.On("GetRateLimit").Return(&client.RateLimit{Limit: 5000, Remaining: 1}, nil)
	collector := newCollector(api)
	reported := record(collector)

	// When
	_, err := collector.Collect(context.Background(), []*client.Repository{repo})

	// Then
	require.ErrorAs(t, err, new(RateLimitError))
	assert.Empty(t, reported.messages)
}

func TestCollector_WorksWithoutHooks(t *testing.T) {
	// Given
	api := new(apiMock)
	busyRepo(api, repo, progressThreshold+1)
	plentyOfBudget(api)

	// When
	usage, err := newCollector(api).Collect(context.Background(), []*client.Repository{repo})

	// Then
	require.NoError(t, err)
	assert.Equal(t, client.WorkflowUsage{ci: 0}, usage[repo])
}

func TestRoughly(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{d: 0, want: "0 seconds"},
		{d: time.Second, want: "1 second"},
		{d: 2800 * time.Millisecond, want: "3 seconds"},
		{d: 59 * time.Second, want: "59 seconds"},
		{d: 59500 * time.Millisecond, want: "1 minute"},
		{d: 60 * time.Second, want: "1 minute"},
		{d: 89 * time.Second, want: "1 minute"},
		{d: 90 * time.Second, want: "2 minutes"},
		{d: 59*time.Minute + 29*time.Second, want: "59 minutes"},
		{d: 59*time.Minute + 30*time.Second, want: "1 hour"},
		{d: time.Hour, want: "1 hour"},
		{d: time.Hour + 5*time.Minute, want: "1 hour 5 minutes"},
		{d: 2*time.Hour + 61*time.Minute, want: "3 hours 1 minute"},
		{d: 2*time.Hour + 59*time.Minute + 45*time.Second, want: "3 hours"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			assert.Equal(t, tc.want, roughly(tc.d))
		})
	}
}

func TestCollector_StopsFetchingAfterAFailure(t *testing.T) {
	// Given: two commits fetched one at a time, the first of which fails
	api := new(apiMock)
	expectListing(api, []client.Workflow{ci}, []client.WorkflowRun{
		{ID: 10, WorkflowID: ci.ID, CheckSuiteID: 100, HeadSHA: "aaa"},
		{ID: 11, WorkflowID: ci.ID, CheckSuiteID: 101, HeadSHA: "bbb"},
	})
	plentyOfBudget(api)
	api.On("GetCheckRuns", *repo, "aaa").Return([]client.CheckRun{}, errAPI)
	api.On("GetCheckRuns", *repo, "bbb").Return([]client.CheckRun{job(101, 5)}, nil)
	collector := newCollector(api)
	collector.Concurrency = 1

	// When
	_, err := collector.Collect(context.Background(), []*client.Repository{repo})

	// Then
	require.ErrorIs(t, err, errAPI)
	api.AssertNotCalled(t, "GetCheckRuns", *repo, "bbb")
}

func TestCollector_PropagatesErrors(t *testing.T) {
	type test struct {
		name  string
		setup func(api *apiMock)
		want  string
	}
	listing := func(api *apiMock) {
		expectListing(api, []client.Workflow{ci}, []client.WorkflowRun{{ID: 10, WorkflowID: ci.ID, CheckSuiteID: 100, HeadSHA: "aaa"}})
	}
	tests := []test{
		{
			name:  "workflows",
			setup: func(api *apiMock) { api.On("GetWorkflows", *repo).Return([]client.Workflow(nil), errAPI) },
			want:  "listing workflows of codiform/gh-actions-usage",
		},
		{
			name: "runs",
			setup: func(api *apiMock) {
				api.On("GetWorkflows", *repo).Return([]client.Workflow{ci}, nil)
				api.On("GetWorkflowRuns", *repo, from, to).Return([]client.WorkflowRun(nil), errAPI)
			},
			want: "listing workflow runs of codiform/gh-actions-usage",
		},
		{
			name: "rate limit",
			setup: func(api *apiMock) {
				listing(api)
				api.On("GetRateLimit").Return((*client.RateLimit)(nil), errAPI)
			},
			want: "checking rate limit",
		},
		{
			name: "check runs",
			setup: func(api *apiMock) {
				listing(api)
				plentyOfBudget(api)
				api.On("GetCheckRuns", *repo, "aaa").Return([]client.CheckRun(nil), errAPI)
			},
			want: "listing check runs of codiform/gh-actions-usage@aaa",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := new(apiMock)
			tc.setup(api)

			usage, err := newCollector(api).Collect(context.Background(), []*client.Repository{repo})

			require.ErrorIs(t, err, errAPI)
			require.ErrorContains(t, err, tc.want)
			assert.Nil(t, usage)
		})
	}
}
