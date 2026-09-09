package client

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	mocks "github.com/geoffreywiseman/gh-actions-usage/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	windowStart = time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	windowEnd   = time.Date(2026, time.September, 2, 23, 59, 59, 0, time.UTC)
)

func runsPath(from, to time.Time, page int) string {
	return fmt.Sprintf("repos/%s/actions/runs?per_page=100&page=%d&created=%s..%s",
		testRepoFullName, page, from.Format(time.RFC3339), to.Format(time.RFC3339))
}

func checkRunsPath(sha string, page int) string {
	return fmt.Sprintf("repos/%s/commits/%s/check-runs?app_id=15368&per_page=100&page=%d", testRepoFullName, sha, page)
}

func makeRuns(firstID uint, count int) []WorkflowRun {
	runs := make([]WorkflowRun, 0, count)
	for i := range count {
		id := firstID + uint(i)
		runs = append(runs, WorkflowRun{ID: id, WorkflowID: 1, CheckSuiteID: id * 10, HeadSHA: "abc"})
	}
	return runs
}

func makeCheckRuns(firstID uint, count int) []CheckRun {
	checkRuns := make([]CheckRun, 0, count)
	for i := range count {
		id := firstID + uint(i)
		checkRuns = append(checkRuns, CheckRun{ID: id, Name: "job", Status: "completed", Conclusion: "success", CheckSuite: CheckSuite{ID: 7}})
	}
	return checkRuns
}

func expectRunPage(rest *mocks.RestMock, path string, total uint, runs []WorkflowRun) {
	rest.On("Get", path, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		page := args.Get(1).(*workflowRunPage)
		page.TotalCount = total
		page.WorkflowRuns = runs
	})
}

func expectCheckRunPage(rest *mocks.RestMock, path string, total uint, checkRuns []CheckRun) {
	rest.On("Get", path, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		page := args.Get(1).(*checkRunPage)
		page.TotalCount = total
		page.CheckRuns = checkRuns
	})
}

func TestClient_GetWorkflowRuns_SinglePage(t *testing.T) {
	// Given
	rest, client := getTestClient()
	expectRunPage(rest, runsPath(windowStart, windowEnd, 1), 2, makeRuns(1, 2))

	// When
	runs, err := client.GetWorkflowRuns(Repository{FullName: testRepoFullName}, windowStart, windowEnd)

	// Then
	require.NoError(t, err)
	assert.Len(t, runs, 2)
	rest.AssertNumberOfCalls(t, "Get", 1)
}

func TestClient_GetWorkflowRuns_MultiplePages(t *testing.T) {
	// Given
	rest, client := getTestClient()
	expectRunPage(rest, runsPath(windowStart, windowEnd, 1), 150, makeRuns(1, 100))
	expectRunPage(rest, runsPath(windowStart, windowEnd, 2), 150, makeRuns(101, 50))

	// When
	runs, err := client.GetWorkflowRuns(Repository{FullName: testRepoFullName}, windowStart, windowEnd)

	// Then
	require.NoError(t, err)
	assert.Len(t, runs, 150)
	assert.Equal(t, uint(1), runs[0].ID)
	assert.Equal(t, uint(150), runs[149].ID)
	rest.AssertNumberOfCalls(t, "Get", 2)
}

func TestClient_GetWorkflowRuns_SplitsByDayWhenOverCap(t *testing.T) {
	// Given
	rest, client := getTestClient()
	day1End := time.Date(2026, time.September, 1, 23, 59, 59, 0, time.UTC)
	day2Start := time.Date(2026, time.September, 2, 0, 0, 0, 0, time.UTC)
	expectRunPage(rest, runsPath(windowStart, windowEnd, 1), 1500, makeRuns(1, 100))
	expectRunPage(rest, runsPath(windowStart, day1End, 1), 800, makeRuns(1, 100))
	for page := 2; page <= 8; page++ {
		expectRunPage(rest, runsPath(windowStart, day1End, page), 800, makeRuns(uint(page-1)*100+1, 100))
	}
	expectRunPage(rest, runsPath(day2Start, windowEnd, 1), 700, makeRuns(1001, 100))
	for page := 2; page <= 7; page++ {
		expectRunPage(rest, runsPath(day2Start, windowEnd, page), 700, makeRuns(1000+uint(page-1)*100+1, 100))
	}

	// When
	runs, err := client.GetWorkflowRuns(Repository{FullName: testRepoFullName}, windowStart, windowEnd)

	// Then
	require.NoError(t, err)
	assert.Len(t, runs, 1500)
	assert.Equal(t, uint(1), runs[0].ID)
	assert.Equal(t, uint(1700), runs[1499].ID)
	rest.AssertNumberOfCalls(t, "Get", 1+8+7)
}

func TestClient_GetWorkflowRuns_DoesNotSplitSingleDay(t *testing.T) {
	// Given
	rest, client := getTestClient()
	dayEnd := time.Date(2026, time.September, 1, 23, 59, 59, 0, time.UTC)
	for page := 1; page <= 10; page++ {
		expectRunPage(rest, runsPath(windowStart, dayEnd, page), 1500, makeRuns(uint(page-1)*100+1, 100))
	}
	expectRunPage(rest, runsPath(windowStart, dayEnd, 11), 1500, nil)

	// When
	runs, err := client.GetWorkflowRuns(Repository{FullName: testRepoFullName}, windowStart, dayEnd)

	// Then
	require.NoError(t, err)
	assert.Len(t, runs, 1000)
	rest.AssertNumberOfCalls(t, "Get", 11)
}

func TestClient_GetWorkflowRuns_NotFound(t *testing.T) {
	// Given
	rest, client := getTestClient()
	rest.On("Get", runsPath(windowStart, windowEnd, 1), mock.Anything).
		Return(&api.HTTPError{StatusCode: http.StatusNotFound, Message: "Not Found"})

	// When
	runs, err := client.GetWorkflowRuns(Repository{FullName: testRepoFullName}, windowStart, windowEnd)

	// Then
	require.NoError(t, err)
	assert.Empty(t, runs)
}

func TestClient_GetWorkflowRuns_Error(t *testing.T) {
	// Given
	rest, client := getTestClient()
	rest.On("Get", runsPath(windowStart, windowEnd, 1), mock.Anything).
		Return(&api.HTTPError{StatusCode: http.StatusBadGateway, Message: "Bad Gateway"})

	// When
	runs, err := client.GetWorkflowRuns(Repository{FullName: testRepoFullName}, windowStart, windowEnd)

	// Then
	require.ErrorContains(t, err, "could not get workflow runs")
	assert.Nil(t, runs)
}

func TestClient_GetCheckRuns_MultiplePages(t *testing.T) {
	// Given
	rest, client := getTestClient()
	expectCheckRunPage(rest, checkRunsPath("abc", 1), 103, makeCheckRuns(1, 100))
	expectCheckRunPage(rest, checkRunsPath("abc", 2), 103, makeCheckRuns(101, 3))

	// When
	checkRuns, err := client.GetCheckRuns(Repository{FullName: testRepoFullName}, "abc")

	// Then
	require.NoError(t, err)
	assert.Len(t, checkRuns, 103)
	assert.Equal(t, uint(7), checkRuns[0].CheckSuite.ID)
	rest.AssertNumberOfCalls(t, "Get", 2)
}

func TestClient_GetCheckRuns_Empty(t *testing.T) {
	// Given
	rest, client := getTestClient()
	expectCheckRunPage(rest, checkRunsPath("def", 1), 0, nil)

	// When
	checkRuns, err := client.GetCheckRuns(Repository{FullName: testRepoFullName}, "def")

	// Then
	require.NoError(t, err)
	assert.Empty(t, checkRuns)
}

func TestClient_GetCheckRuns_Error(t *testing.T) {
	// Given
	rest, client := getTestClient()
	rest.On("Get", checkRunsPath("abc", 1), mock.Anything).
		Return(&api.HTTPError{StatusCode: http.StatusUnprocessableEntity, Message: "No commit found"})

	// When
	checkRuns, err := client.GetCheckRuns(Repository{FullName: testRepoFullName}, "abc")

	// Then
	require.ErrorContains(t, err, "could not get check runs")
	assert.Nil(t, checkRuns)
}

func TestClient_GetRateLimit(t *testing.T) {
	// Given
	rest, client := getTestClient()
	rest.On("Get", "rate_limit", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		response := args.Get(1).(*rateLimitResponse)
		response.Resources.Core = RateLimit{Limit: 5000, Remaining: 4321, Used: 679, Reset: 1788978347}
	})

	// When
	limit, err := client.GetRateLimit()

	// Then
	require.NoError(t, err)
	assert.Equal(t, uint(4321), limit.Remaining)
	assert.Equal(t, time.Unix(1788978347, 0), limit.ResetAt())
}

func TestClient_GetRateLimit_Error(t *testing.T) {
	// Given
	rest, client := getTestClient()
	rest.On("Get", "rate_limit", mock.Anything).Return(&api.HTTPError{StatusCode: http.StatusUnauthorized, Message: "Bad credentials"})

	// When
	limit, err := client.GetRateLimit()

	// Then
	require.ErrorContains(t, err, "could not get rate limit")
	assert.Nil(t, limit)
}
