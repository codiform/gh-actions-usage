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

func summarizeUsage(usage client.RepoUsage) usageSummary {
	repos := make([]repoSummary, 0, len(usage))
	owners := make(map[string]*ownerSummary)

	for repo, flowUsage := range usage {
		workflows := make([]workflowSummary, 0, len(flowUsage))
		var repoTotal uint
		for workflow, workflowUsage := range flowUsage {
			repoTotal += workflowUsage
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

		owner := ownerName(repo)
		repos = append(repos, repoSummary{
			Repo:      repo,
			Owner:     owner,
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
