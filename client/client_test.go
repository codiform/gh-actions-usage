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
			StatusCode:  404,
			RequestURL:  requestUrl,
			Message:     "Couldn't find repo",
			OAuthScopes: "gh",
			Errors:      nil,
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
			StatusCode:  501,
			RequestURL:  requestUrl,
			Message:     "Server Error",
			OAuthScopes: "gh",
			Errors:      nil,
		})

	//Then
	repo, err := client.GetRepository(expectedName)
	assert.NotNil(t, err)
	assert.Nil(t, repo)
}
