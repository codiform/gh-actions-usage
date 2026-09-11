// Package client provides a GitHub API client for the gh-actions-usage extension.
package client

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
)

// New creates a new Client instance, initialized with a GH RESTClient that paces requests to stay within
// GitHub's rate limits and retries requests that are rejected as rate limited.
func New() Client {
	rest, err := api.NewRESTClient(api.ClientOptions{Transport: newThrottle(http.DefaultTransport)})
	if err != nil {
		panic(err)
	}

	return Client{Rest: rest}
}

// REST is the subset of the go-gh REST client used by Client, expressed as an interface so that it can be mocked.
type REST interface {
	// Get issues a GET request to the specified path and decodes the JSON response into response.
	Get(path string, response any) error
}

// Client is a GH API client customized for the specifics of `gh-actions-usage`.
type Client struct {
	Rest REST
}

// Workflow represents a GitHub Actions workflow
type Workflow struct {
	Name  string
	Path  string
	State string
	ID    uint
}

// WorkflowUsage is a map of usage by Workflow
type WorkflowUsage map[Workflow]uint

// RepoUsage is a map of WorkflowUsage by Repo
type RepoUsage map[*Repository]WorkflowUsage

// UnexpectedUserTypeError is an error when the user type is unexpected
type UnexpectedUserTypeError string

// Error returns a formatted error message for UnexpectedUserTypeError
func (e UnexpectedUserTypeError) Error() string {
	return "Unexpected user type: " + string(e)
}

// UnexpectedHostError is an error when the host is unexpected
type UnexpectedHostError string

// Error returns a formatted error message for UnexpectedHostError
func (e UnexpectedHostError) Error() string {
	return "Unexpected host: " + string(e)
}

// GetWorkflows returns a slice of Workflow instances, one for each workflow in the repository
func (c *Client) GetWorkflows(repository Repository) ([]Workflow, error) {
	var page uint8 = 1
	var workflows = make([]Workflow, 0)

	for {
		wfp, err := c.getWorkflowPage(repository, page)
		if err != nil {
			return nil, err
		}
		if len(wfp) == 0 {
			break
		}
		workflows = append(workflows, wfp...)
		page++
	}
	return workflows, nil
}

type workflowPage struct {
	Workflows  []Workflow
	TotalCount uint64 `json:"total_count"`
}

func (c *Client) getWorkflowPage(repository Repository, page uint8) ([]Workflow, error) {
	response := workflowPage{}
	url := fmt.Sprintf("repos/%s/actions/workflows?page=%d", repository.FullName, page)
	err := c.Rest.Get(url, &response)
	if err != nil {
		return nil, fmt.Errorf("could not get workflow page: %w", err)
	}
	return response.Workflows, nil
}

// Repository represents a GitHub Repository
type Repository struct {
	Owner    *User
	FullName string `json:"full_name"`
	Name     string
	ID       uint
	Private  bool `json:"private"`
}

// User represents a GitHub User that can act as the Owner of a GitHub Repository, which might be an Organization
type User struct {
	Login string
	Type  string
	ID    uint
}

// GetRepository gets a Repository instance corresponding to the specified fullName
func (c *Client) GetRepository(fullName string) (*Repository, error) {
	response := Repository{}
	err := c.Rest.Get("repos/"+fullName, &response)
	if err != nil {
		if is404(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("could not get repository: %w", err)
	}
	return &response, nil
}

// GetCurrentRepository gets the Repository that corresponds to the current working directory, or nil if there is none
func (c *Client) GetCurrentRepository() (*Repository, error) {
	repo, err := repository.Current()
	if err != nil {
		return nil, fmt.Errorf("could not get current repository: %w", err)
	}

	if repo.Host != "github.com" {
		return nil, UnexpectedHostError(repo.Host)
	}

	return c.GetRepository(fmt.Sprintf("%s/%s", repo.Owner, repo.Name))
}

func is404(err error) bool {
	var httpError *api.HTTPError
	return errors.As(err, &httpError) && httpError.StatusCode == http.StatusNotFound
}

// GetUser returns a User corresponding to the specified name, or nil if the user was not found
func (c *Client) GetUser(name string) (*User, error) {
	response := User{}
	err := c.Rest.Get("users/"+name, &response)
	if err != nil {
		if is404(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("could not get user: %w", err)
	}
	return &response, nil
}

// GetAllRepositories returns a list of repositories for the specified user
func (c *Client) GetAllRepositories(user *User) ([]*Repository, error) {
	var page uint8 = 1
	var repos = make([]*Repository, 0)

	for {
		path, err := c.getAllRepositoriesPath(user, page)
		if err != nil {
			return nil, err
		}
		rp, err := c.getAllRepositoriesPage(path)
		if err != nil {
			return nil, err
		}
		if len(rp) == 0 {
			break
		}
		repos = append(repos, rp...)
		page++
	}
	return repos, nil
}

func (c *Client) getAllRepositoriesPath(user *User, page uint8) (string, error) {
	switch user.Type {
	case "Organization":
		return fmt.Sprintf("orgs/%s/repos?page=%d", user.Login, page), nil
	case "User":
		return fmt.Sprintf("users/%s/repos?page=%d", user.Login, page), nil
	default:
		return "", UnexpectedUserTypeError(user.Type)
	}
}

func (c *Client) getAllRepositoriesPage(pagePath string) ([]*Repository, error) {
	var response []*Repository
	err := c.Rest.Get(pagePath, &response)
	if err != nil {
		if is404(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("could not get repositories: %w", err)
	}
	return response, nil
}
