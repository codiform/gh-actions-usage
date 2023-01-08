package client

import (
	"github.com/cli/go-gh/pkg/api"
	mocks "github.com/geoffreywiseman/gh-actions-usage/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/url"
	"testing"
)

func TestGetRepositorySuccess(t *testing.T) {
	// Given
	rest := new(mocks.RestMock)
	client := Client{Rest: rest}
	expectedName := "codiform/gh-actuse"

	// When
	rest.On("Get", "repos/codiform/gh-actuse", mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			repo := args.Get(1).(*Repository)
			repo.Id = 1
			repo.Name = "gh-actuse"
			repo.FullName = "codiform/gh-actuse"
		})

	// Then
	repo, err := client.GetRepository(expectedName)
	assert.Nil(t, err)
	assert.Equal(t, repo.FullName, expectedName)
}

func TestGetRepositoryNotFound(t *testing.T) {
	// Given
	rest := new(mocks.RestMock)
	client := Client{Rest: rest}
	expectedName := "codiform/gh-actuse"

	// When
	requestUrl, _ := url.Parse("https://github.com/codiform/gh-actuse")
	rest.On("Get", "repos/codiform/gh-actuse", mock.Anything).
		Return(api.HTTPError{
			Errors:     nil,
			Headers:    nil,
			Message:    "Couldn't find repo",
			RequestURL: requestUrl,
			StatusCode: 404,
		})

	//Then
	repo, err := client.GetRepository(expectedName)
	assert.Nil(t, err)
	assert.Nil(t, repo)
}

func TestGetRepositoryFailure(t *testing.T) {
	// Given
	rest := new(mocks.RestMock)
	client := Client{Rest: rest}
	expectedName := "codiform/gh-actuse"

	// When
	requestUrl, _ := url.Parse("https://github.com/codiform/gh-actuse")
	rest.On("Get", "repos/codiform/gh-actuse", mock.Anything).
		Return(api.HTTPError{
			Errors:     nil,
			Headers:    nil,
			Message:    "Server Error",
			RequestURL: requestUrl,
			StatusCode: 501,
		})

	//Then
	repo, err := client.GetRepository(expectedName)
	assert.NotNil(t, err)
	assert.Nil(t, repo)
}

func TestGetWorkflows(t *testing.T) {
	// Given
	rest, client := getTestClient()
	repo := Repository{Id: 1, Name: "gh-actions-usage", FullName: "codiform/gh-actions-usage"}
	rest.On("Get", "repos/codiform/gh-actions-usage/actions/workflows?page=1", mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			wp := args.Get(1).(*WorkflowPage)
			wp.Workflows = append(wp.Workflows, Workflow{Id: 1, Name: "Build", Path: ".github/workflows/build.yml", State: "active"})
			wp.TotalCount = 1
		})
	rest.On("Get", "repos/codiform/gh-actions-usage/actions/workflows?page=2", mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			wp := args.Get(1).(*WorkflowPage)
			wp.TotalCount = 0
		})

	// When
	repos, err := client.GetWorkflows(repo)

	// Then
	assert.Nil(t, err)
	assert.Equal(t, 1, len(repos))
	assert.Equal(t, "Build", repos[0].Name)
}

func getTestClient() (*mocks.RestMock, Client) {
	rest := new(mocks.RestMock)
	return rest, Client{Rest: rest}
}
