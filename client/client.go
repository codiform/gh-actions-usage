// Package client provides a GitHub API client for the gh-actions-usage extension.
package client

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/api"
)

const userTypeOrganization = "Organization"
const userTypeUser = "User"

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

// BillingUnavailableError is returned when the billing API cannot be accessed due to missing
// permissions. The value is the HTTP status code returned by the API.
type BillingUnavailableError int

// Error returns a formatted error message for BillingUnavailableError
func (e BillingUnavailableError) Error() string {
	return fmt.Sprintf("Billing API unavailable (HTTP %d): a token with billing permissions is required to retrieve usage data", int(e))
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

// Usage represents the usage of a workflow within the billing period
type Usage struct {
	Billable map[string]*UsageDetails `json:"billable"`
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
		return nil, fmt.Errorf("could not get workflow usage: %w", err)
	}
	return &response, nil
}

// TotalMs sums the milliseconds across all runner environments
func (u *Usage) TotalMs() uint {
	var total uint
	for _, details := range u.Billable {
		if details != nil {
			total += details.TotalMs
		}
	}
	return total
}

// BillingUsageItem represents a single line item in a billing usage report.
// Quantity is in the unit specified by UnitType (minutes for Actions).
// RepositoryName is in "owner/repo" format.
type BillingUsageItem struct {
	Date             string  `json:"date"`
	Product          string  `json:"product"`
	SKU              string  `json:"sku"`
	Quantity         float64 `json:"quantity"`
	UnitType         string  `json:"unitType"`
	PricePerUnit     float64 `json:"pricePerUnit"`
	GrossAmount      float64 `json:"grossAmount"`
	DiscountAmount   float64 `json:"discountAmount"`
	NetAmount        float64 `json:"netAmount"`
	OrganizationName string  `json:"organizationName"`
	RepositoryName   string  `json:"repositoryName"`
}

// BillingUsageReport is the response from the billing usage API
type BillingUsageReport struct {
	UsageItems []BillingUsageItem `json:"usageItems"`
}

// GetActionsUsage returns Actions billing usage for a user or organization for the current billing period.
// Returns nil when the user or organization is not on the enhanced billing platform (404).
func (c *Client) GetActionsUsage(user *User) (*BillingUsageReport, error) {
	var path string
	switch user.Type {
	case userTypeOrganization:
		path = fmt.Sprintf("organizations/%s/settings/billing/usage", user.Login)
	case userTypeUser:
		path = fmt.Sprintf("users/%s/settings/billing/usage", user.Login)
	default:
		return nil, UnexpectedUserTypeError(user.Type)
	}
	response := BillingUsageReport{}
	err := c.Rest.Get(path, &response)
	if err != nil {
		var httpError api.HTTPError
		if errors.As(err, &httpError) {
			switch httpError.StatusCode {
			case http.StatusNotFound:
				return nil, nil
			case http.StatusForbidden:
				return nil, BillingUnavailableError(httpError.StatusCode)
			}
		}
		return nil, fmt.Errorf("could not get actions usage: %w", err)
	}
	return &response, nil
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
	repo, err := gh.CurrentRepository()
	if err != nil {
		return nil, fmt.Errorf("could not get current repository: %w", err)
	}

	if repo.Host() != "github.com" {
		return nil, UnexpectedHostError(repo.Host())
	}

	return c.GetRepository(fmt.Sprintf("%s/%s", repo.Owner(), repo.Name()))
}

func is404(err error) bool {
	var httpError api.HTTPError
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
	case userTypeOrganization:
		return fmt.Sprintf("orgs/%s/repos?page=%d", user.Login, page), nil
	case userTypeUser:
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
