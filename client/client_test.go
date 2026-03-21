package client

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/cli/go-gh/pkg/api"
	mocks "github.com/geoffreywiseman/gh-actions-usage/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const testRepoFullName = "codiform/gh-actions-usage"

func TestClient_GetRepository(t *testing.T) {
	// Given
	rest, client := getTestClient()
	expectedName := testRepoFullName
	rest.On("Get", "repos/"+testRepoFullName, mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			repo := args.Get(1).(*Repository)
			repo.ID = 1
			repo.Name = "gh-actions-usage"
			repo.FullName = testRepoFullName
		})

	// When
	repo, err := client.GetRepository(expectedName)

	// Then
	require.NoError(t, err)
	assert.Equal(t, expectedName, repo.FullName)
}

func TestClient_GetRepository_NotFound(t *testing.T) {
	// Given
	rest, client := getTestClient()
	expectedName := testRepoFullName
	requestURL, _ := url.Parse("https://github.com/" + testRepoFullName)
	rest.On("Get", "repos/"+testRepoFullName, mock.Anything).
		Return(api.HTTPError{
			Errors:     nil,
			Headers:    nil,
			Message:    "Couldn't find repo",
			RequestURL: requestURL,
			StatusCode: 404,
		})

	// When
	repo, err := client.GetRepository(expectedName)

	// Then
	require.NoError(t, err)
	assert.Nil(t, repo)
}

func TestClient_GetRepository_Failure(t *testing.T) {
	// Given
	rest, client := getTestClient()
	expectedName := testRepoFullName
	requestURL, _ := url.Parse("https://github.com/" + testRepoFullName)
	rest.On("Get", "repos/"+testRepoFullName, mock.Anything).
		Return(api.HTTPError{
			Errors:     nil,
			Headers:    nil,
			Message:    "Server Error",
			RequestURL: requestURL,
			StatusCode: 501,
		})

	// When
	repo, err := client.GetRepository(expectedName)

	// Then
	require.Error(t, err)
	assert.Nil(t, repo)
}

func TestClient_GetWorkflows(t *testing.T) {
	// Given
	rest, client := getTestClient()
	repo := Repository{ID: 1, Name: "gh-actions-usage", FullName: testRepoFullName}
	rest.On("Get", "repos/"+testRepoFullName+"/actions/workflows?page=1", mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			wp := args.Get(1).(*workflowPage)
			wp.Workflows = append(wp.Workflows, Workflow{ID: 1, Name: "Build", Path: ".github/workflows/build.yml", State: "active"})
			wp.TotalCount = 1
		})
	rest.On("Get", "repos/"+testRepoFullName+"/actions/workflows?page=2", mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			wp := args.Get(1).(*workflowPage)
			wp.TotalCount = 0
		})

	// When
	repos, err := client.GetWorkflows(repo)

	// Then
	require.NoError(t, err)
	assert.Len(t, repos, 1)
	assert.Equal(t, "Build", repos[0].Name)
}

func TestClient_GetWorkflowUsage(t *testing.T) {
	// Given
	rest, client := getTestClient()
	repo := Repository{ID: 1, Name: "gh-actions-usage", FullName: testRepoFullName}
	flow := Workflow{ID: 2, Name: "CI", Path: "repos/" + testRepoFullName + "/actions/workflows/2", State: "active"}
	rest.On("Get", "repos/"+testRepoFullName+"/actions/workflows/2/timing", mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			u := args.Get(1).(*Usage)
			u.Billable = map[string]*UsageDetails{
				"WINDOWS": {TotalMs: 4},
				"UBUNTU":  {TotalMs: 180},
				"MACOS":   {TotalMs: 16},
			}
		})

	// When
	usage, err := client.GetWorkflowUsage(repo, flow)

	// Then
	require.NoError(t, err)
	assert.Equal(t, uint(200), usage.TotalMs())
}

func TestUsage_TotalMs_ApiFormat(t *testing.T) {
	// Verify that the Usage struct correctly deserializes the GitHub API response format,
	// which uses uppercase environment keys like UBUNTU, MACOS, WINDOWS.
	data := `{"billable":{"UBUNTU":{"total_ms":180000},"MACOS":{"total_ms":240000},"WINDOWS":{"total_ms":300000}}}`
	var u Usage
	err := json.Unmarshal([]byte(data), &u)
	require.NoError(t, err)
	assert.Equal(t, uint(720000), u.TotalMs())
}

func TestUsage_TotalMs_AdditionalRunnerTypes(t *testing.T) {
	// Verify that the Usage struct correctly captures additional runner environment keys
	// that GitHub may return for larger or ARM64 runners.
	data := `{"billable":{"UBUNTU":{"total_ms":180000},"UBUNTU_ARM":{"total_ms":60000},"MACOS":{"total_ms":240000}}}`
	var u Usage
	err := json.Unmarshal([]byte(data), &u)
	require.NoError(t, err)
	assert.Equal(t, uint(480000), u.TotalMs())
}

// Straightforward Test
func TestClient_GetUser(t *testing.T) {
	// Given
	rest, client := getTestClient()
	rest.On("Get", "users/codiform", mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			u := args.Get(1).(*User)
			u.ID = 103469606
			u.Login = "codiform"
			u.Type = userTypeOrganization
		})

	// When
	owner, err := client.GetUser("codiform")

	// Then
	require.NoError(t, err)
	assert.Equal(t, "codiform", owner.Login)
}

// Not Found
func TestClient_GetUser_NotFound(t *testing.T) {
	// Given
	rest, client := getTestClient()
	expectedName := "codiform2"
	requestURL, _ := url.Parse("https://github.com/users/codiform2")
	rest.On("Get", "users/codiform2", mock.Anything).
		Return(api.HTTPError{
			Errors:     nil,
			Headers:    nil,
			Message:    "Not Found",
			RequestURL: requestURL,
			StatusCode: 404,
		})

	// When
	repo, err := client.GetUser(expectedName)

	// Then
	require.NoError(t, err)
	assert.Nil(t, repo)
}

// Success Case
func TestClient_GetAllRepositories(t *testing.T) {
	// Given
	rest, client := getTestClient()
	rest.On("Get", "users/geoffreywiseman/repos?page=1", mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			ars := args.Get(1).(*[]*Repository)
			*ars = append(*ars, &Repository{ID: 427462569, Name: "gh-actuse", FullName: "geoffreywiseman/gh-actuse"})
		})
	rest.On("Get", "users/geoffreywiseman/repos?page=2", mock.Anything).
		Return(nil) //.
	owner := &User{ID: 49935, Login: "geoffreywiseman", Type: "User"}

	// When
	repos, err := client.GetAllRepositories(owner)

	// Then
	require.NoError(t, err)
	assert.Len(t, repos, 1, repos)
	if len(repos) > 0 {
		assert.Equal(t, "gh-actuse", repos[0].Name)
	}
}

func TestClient_GetActionsUsage_User(t *testing.T) {
	// Given
	rest, client := getTestClient()
	owner := &User{ID: 49935, Login: "geoffreywiseman", Type: userTypeUser}
	rest.On("Get", "users/geoffreywiseman/settings/billing/usage", mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			report := args.Get(1).(*BillingUsageReport)
			report.UsageItems = []BillingUsageItem{
				{Date: "2024-01-01", Product: "Actions", SKU: "Actions Linux", Quantity: 100, UnitType: "minutes", RepositoryName: "geoffreywiseman/gh-actuse"},
				{Date: "2024-01-02", Product: "Actions", SKU: "Actions Linux", Quantity: 50, UnitType: "minutes", RepositoryName: "geoffreywiseman/gh-actuse"},
			}
		})

	// When
	report, err := client.GetActionsUsage(owner)

	// Then
	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Len(t, report.UsageItems, 2)
	assert.Equal(t, "Actions Linux", report.UsageItems[0].SKU)
	assert.InDelta(t, float64(100), report.UsageItems[0].Quantity, 0.001)
}

func TestClient_GetActionsUsage_Organization(t *testing.T) {
	// Given
	rest, client := getTestClient()
	owner := &User{ID: 103469606, Login: "codiform", Type: userTypeOrganization}
	rest.On("Get", "organizations/codiform/settings/billing/usage", mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			report := args.Get(1).(*BillingUsageReport)
			report.UsageItems = []BillingUsageItem{
				{Date: "2024-01-01", Product: "Actions", SKU: "Actions Linux", Quantity: 200, UnitType: "minutes", RepositoryName: "codiform/gh-actions-usage"},
				{Date: "2024-01-01", Product: "Actions", SKU: "Actions macOS", Quantity: 30, UnitType: "minutes", RepositoryName: "codiform/gh-actions-usage"},
			}
		})

	// When
	report, err := client.GetActionsUsage(owner)

	// Then
	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Len(t, report.UsageItems, 2)
	assert.Equal(t, "codiform/gh-actions-usage", report.UsageItems[0].RepositoryName)
}

func TestClient_GetActionsUsage_UnexpectedType(t *testing.T) {
	// Given
	_, client := getTestClient()
	owner := &User{ID: 1, Login: "bot", Type: "Bot"}

	// When
	report, err := client.GetActionsUsage(owner)

	// Then
	assert.Nil(t, report)
	require.Error(t, err)
	var unexpectedType UnexpectedUserTypeError
	assert.ErrorAs(t, err, &unexpectedType)
}

func getTestClient() (*mocks.RestMock, Client) {
	rest := new(mocks.RestMock)
	return rest, Client{Rest: rest}
}
