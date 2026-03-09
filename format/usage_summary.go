package format

import (
	"sort"
	"strings"

	"github.com/geoffreywiseman/gh-actions-usage/client"
)

type workflowSummary struct {
	Workflow client.Workflow
	Usage    uint
}

type repoSummary struct {
	Repo      *client.Repository
	Owner     string
	Private   bool
	Workflows []workflowSummary
	Total     uint
}

type ownerSummary struct {
	Owner         string
	RepoCount     int
	WorkflowCount int
	Total         uint
}

type usageSummary struct {
	Repos         []repoSummary
	Owners        []ownerSummary
	RepoCount     int
	WorkflowCount int
	Total         uint
}

// summarizeUsage builds owner and total rollups for human-readable output.
// Collection intentionally stays as raw RepoUsage so each formatter can choose
// how much reorganization it needs without coupling API collection to
// presentation-specific summary rules.
func summarizeUsage(usage client.RepoUsage) usageSummary {
	repos := make([]repoSummary, 0, len(usage))
	owners := make(map[string]*ownerSummary)

	for repo, flowUsage := range usage {
		workflows := sortedWorkflowUsage(flowUsage)
		var repoTotal uint
		for _, workflow := range workflows {
			repoTotal += workflow.Usage
		}

		owner := ownerName(repo)
		repos = append(repos, repoSummary{
			Repo:      repo,
			Owner:     owner,
			Private:   repo.Private,
			Workflows: workflows,
			Total:     repoTotal,
		})

		summary := owners[owner]
		if summary == nil {
			summary = &ownerSummary{Owner: owner}
			owners[owner] = summary
		}
		summary.RepoCount++
		summary.WorkflowCount += len(workflows)
		summary.Total += repoTotal
	}

	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Repo.FullName < repos[j].Repo.FullName
	})

	ownerTotals := make([]ownerSummary, 0, len(owners))
	var workflowCount int
	var total uint
	for _, owner := range owners {
		ownerTotals = append(ownerTotals, *owner)
		workflowCount += owner.WorkflowCount
		total += owner.Total
	}
	sort.Slice(ownerTotals, func(i, j int) bool {
		return ownerTotals[i].Owner < ownerTotals[j].Owner
	})

	return usageSummary{
		Repos:         repos,
		Owners:        ownerTotals,
		RepoCount:     len(repos),
		WorkflowCount: workflowCount,
		Total:         total,
	}
}

func sortedWorkflowUsage(flowUsage client.WorkflowUsage) []workflowSummary {
	workflows := make([]workflowSummary, 0, len(flowUsage))
	for workflow, workflowUsage := range flowUsage {
		workflows = append(workflows, workflowSummary{
			Workflow: workflow,
			Usage:    workflowUsage,
		})
	}
	sort.Slice(workflows, func(i, j int) bool {
		if workflows[i].Workflow.Path != workflows[j].Workflow.Path {
			return workflows[i].Workflow.Path < workflows[j].Workflow.Path
		}
		if workflows[i].Workflow.Name != workflows[j].Workflow.Name {
			return workflows[i].Workflow.Name < workflows[j].Workflow.Name
		}
		return workflows[i].Workflow.ID < workflows[j].Workflow.ID
	})
	return workflows
}

func ownerName(repo *client.Repository) string {
	if repo != nil && repo.Owner != nil && repo.Owner.Login != "" {
		return repo.Owner.Login
	}
	if repo != nil {
		owner, _, found := strings.Cut(repo.FullName, "/")
		if found {
			return owner
		}
		return repo.FullName
	}
	return ""
}
