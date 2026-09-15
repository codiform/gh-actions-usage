package client

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
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
		Return(&api.HTTPError{
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
		Return(&api.HTTPError{
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
	rest.On("Get", "repos/"+testRepoFullName+"/actions/workflows?per_page=100&page=1", mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			wp := args.Get(1).(*workflowPage)
			wp.Workflows = append(wp.Workflows, Workflow{ID: 1, Name: "Build", Path: ".github/workflows/build.yml", State: "active"})
			wp.TotalCount = 1
		})

	// When
	repos, err := client.GetWorkflows(repo)

	// Then
	require.NoError(t, err)
	assert.Len(t, repos, 1)
	assert.Equal(t, "Build", repos[0].Name)
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
			u.Type = "Organization"
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
		Return(&api.HTTPError{
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

func TestClient_GetWorkflows_FetchesTheNextPageAfterAFullOne(t *testing.T) {
	// Given: a full first page and a short second one
	rest, client := getTestClient()
	repo := Repository{ID: 1, Name: "gh-actions-usage", FullName: testRepoFullName}
	rest.On("Get", "repos/"+testRepoFullName+"/actions/workflows?per_page=100&page=1", mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			wp := args.Get(1).(*workflowPage)
			for i := range perPage {
				wp.Workflows = append(wp.Workflows, Workflow{ID: uint(i), Name: fmt.Sprintf("Workflow %d", i)})
			}
		})
	rest.On("Get", "repos/"+testRepoFullName+"/actions/workflows?per_page=100&page=2", mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			wp := args.Get(1).(*workflowPage)
			wp.Workflows = append(wp.Workflows, Workflow{ID: perPage, Name: "Workflow 100"})
		})

	// When
	workflows, err := client.GetWorkflows(repo)

	// Then: both pages are returned, and no third page is requested
	require.NoError(t, err)
	assert.Len(t, workflows, perPage+1)
	rest.AssertNumberOfCalls(t, "Get", 2)
}

func TestClient_GetAllRepositories_FetchesTheNextPageAfterAFullOne(t *testing.T) {
	// Given: a full first page and a short second one
	rest, client := getTestClient()
	rest.On("Get", "orgs/codiform/repos?per_page=100&page=1", mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			ars := args.Get(1).(*[]*Repository)
			for i := range perPage {
				*ars = append(*ars, &Repository{ID: uint(i), Name: fmt.Sprintf("repo-%d", i)})
			}
		})
	rest.On("Get", "orgs/codiform/repos?per_page=100&page=2", mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			ars := args.Get(1).(*[]*Repository)
			*ars = append(*ars, &Repository{ID: perPage, Name: "repo-100"})
		})
	owner := &User{ID: 1, Login: "codiform", Type: "Organization"}

	// When
	repos, err := client.GetAllRepositories(owner)

	// Then: both pages are returned, and no third page is requested
	require.NoError(t, err)
	assert.Len(t, repos, perPage+1)
	rest.AssertNumberOfCalls(t, "Get", 2)
}

// Success Case
func TestClient_GetAllRepositories(t *testing.T) {
	// Given
	rest, client := getTestClient()
	rest.On("Get", "users/geoffreywiseman/repos?per_page=100&page=1", mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			ars := args.Get(1).(*[]*Repository)
			*ars = append(*ars, &Repository{ID: 427462569, Name: "gh-actuse", FullName: "geoffreywiseman/gh-actuse"})
		})
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

func getTestClient() (*mocks.RestMock, Client) {
	rest := new(mocks.RestMock)
	return rest, Client{Rest: rest}
}
