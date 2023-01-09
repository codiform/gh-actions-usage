package client

import (
	"errors"
	"fmt"
	"github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/api"
)

func New() Client {
	rest, err := gh.RESTClient(nil)
	if err != nil {
		panic(err)
	}

	return Client{Rest: rest}
}

type Client struct {
	Rest api.RESTClient
}

type Workflow struct {
	Id    uint
	Name  string
	Path  string
	State string
}

func (c *Client) GetWorkflows(repository Repository) ([]Workflow, error) {
	var page uint8 = 1
	var workflows []Workflow = make([]Workflow, 0)

	for {
		wfp, err := c.GetWorkflowPage(repository, page)
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

type WorkflowPage struct {
	TotalCount uint64 `json:"total_count"`
	Workflows  []Workflow
}

func (c *Client) GetWorkflowPage(repository Repository, page uint8) ([]Workflow, error) {
	response := WorkflowPage{}
	url := fmt.Sprintf("repos/%s/actions/workflows?page=%d", repository.FullName, page)
	err := c.Rest.Get(url, &response)
	if err != nil {
		return nil, err
	} else {
		return response.Workflows, nil
	}
}

type Usage struct {
	Billable struct {
		Ubuntu  *UsageDetails
		Macos   *UsageDetails
		Windows *UsageDetails
	}
}

type UsageDetails struct {
	TotalMs uint `json:"total_ms"`
}

func (c *Client) GetWorkflowUsage(repository Repository, workflow Workflow) (*Usage, error) {
	response := Usage{}
	path := fmt.Sprintf("repos/%s/actions/workflows/%d/timing", repository.FullName, workflow.Id)
	err := c.Rest.Get(path, &response)
	if err != nil {
		return nil, err
	} else {
		return &response, nil
	}
}

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

type Repository struct {
	Id       uint
	Name     string
	FullName string `json:"full_name"`
}

func (c *Client) GetRepository(fullName string) (*Repository, error) {
	response := Repository{}
	err := c.Rest.Get("repos/"+fullName, &response)
	if err != nil {
		if is404(err) {
			return nil, nil
		} else {
			return nil, err
		}
	}
	return &response, nil
}

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
