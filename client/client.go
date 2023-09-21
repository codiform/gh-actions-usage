package client

import (
	"errors"
	"fmt"
	"github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/api"
)

// New creates a new Client instance, initialized with a GH RESTClient
func New() Client {
	rest, err := gh.RESTClient(nil)
	if err != nil {
		panic(err)
	}

	return Client{Rest: rest}
}

// Client is a GH API client customized for the specifics of `gh-actions-usage`.
type Client struct {
	Rest api.RESTClient
}

// Workflow represents a GitHub Actions workflow
type Workflow struct {
	ID    uint
	Name  string
	Path  string
	State string
}

// WorkflowUsage is a map of usage by Workflow
type WorkflowUsage map[Workflow]uint

// RepoUsage is a map of WorkflowUsage by Repo
type RepoUsage map[*Repository]WorkflowUsage

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
	TotalCount uint64 `json:"total_count"`
	Workflows  []Workflow
}

func (c *Client) getWorkflowPage(repository Repository, page uint8) ([]Workflow, error) {
	response := workflowPage{}
	url := fmt.Sprintf("repos/%s/actions/workflows?page=%d", repository.FullName, page)
	err := c.Rest.Get(url, &response)
	if err != nil {
		return nil, err
	}
	return response.Workflows, nil
}

// Usage represents the usage of a workflow within the billing period
type Usage struct {
	Billable struct {
		Ubuntu  *UsageDetails
		Macos   *UsageDetails
		Windows *UsageDetails
	}
}

// UsageDetails is a sub-item of Usage which is basically just a container for the total milliseconds of usage
type UsageDetails struct {
	TotalMs uint `json:"total_ms"`
}

// GetWorkflowUsage returns the Usage for a Workflow in a Repository
func (c *Client) GetWorkflowUsage(repository Repository, workflow Workflow) (*Usage, error) {
	response := Usage{}
	path := fmt.Sprintf("repos/%s/actions/workflows/%d/timing", repository.FullName, workflow.ID)
	err := c.Rest.Get(path, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// TotalMs sums the milliseconds of each UsageDetails instance
func (u *Usage) TotalMs() uint {
	var total uint = 0
	if u.Billable.Windows != nil {
		total += u.Billable.Windows.TotalMs
	}
	if u.Billable.Macos != nil {
		total += u.Billable.Macos.TotalMs
	}
	if u.Billable.Ubuntu != nil {
		total += u.Billable.Ubuntu.TotalMs
	}
	return total
}

// Repository represents a GitHub Repository
type Repository struct {
	ID       uint
	Name     string
	FullName string `json:"full_name"`
	Owner    *User
}

// User represents a GitHub User that can act as the Owner of a GitHub Repository, which might be an Organization
type User struct {
	ID    uint
	Login string
	Type  string
}

// GetRepository gets a Repository instance corresponding to the specified fullName
func (c *Client) GetRepository(fullName string) (*Repository, error) {
	response := Repository{}
	err := c.Rest.Get("repos/"+fullName, &response)
	if err != nil {
		if is404(err) {
			return nil, nil
		}
		return nil, err
	}
	return &response, nil
}

// GetCurrentRepository gets the Repository that corresponds to the current working directory, or nil if there is none
func (c *Client) GetCurrentRepository() (*Repository, error) {
	repo, err := gh.CurrentRepository()
	if err != nil {
		return nil, err
	}

	if repo.Host() != "github.com" {
		return nil, fmt.Errorf("not sure how to handle host %s", repo.Host())
	}

	return c.GetRepository(fmt.Sprintf("%s/%s", repo.Owner(), repo.Name()))
}

func is404(err error) bool {
	var httpError api.HTTPError
	return errors.As(err, &httpError) && httpError.StatusCode == 404
}

// GetUser returns a User corresponding to the specified name, or nil if the user was not found
func (c *Client) GetUser(name string) (*User, error) {
	response := User{}
	err := c.Rest.Get("users/"+name, &response)
	if err != nil {
		if is404(err) {
			return nil, nil
		}
		return nil, err
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
	if user.Type == "Organization" {
		return fmt.Sprintf("orgs/%s/repos?page=%d", user.Login, page), nil
	} else if user.Type == "User" {
		return fmt.Sprintf("users/%s/repos?page=%d", user.Login, page), nil
	} else {
		return "", fmt.Errorf("unknown user type: %s", user.Type)
	}
}

func (c *Client) getAllRepositoriesPage(pagePath string) ([]*Repository, error) {
	var response []*Repository
	err := c.Rest.Get(pagePath, &response)
	if err != nil {
		if is404(err) {
			return nil, nil
		}
		return nil, err
	}
	return response, nil
}
