package format

import (
	"fmt"
	"io"

	"github.com/geoffreywiseman/gh-actions-usage/client"
)

type humanFormatter struct {
	w io.Writer
}

func (hf humanFormatter) PrintUsage(usage client.RepoUsage) {
	summary := summarizeUsage(usage)
	for _, repo := range summary.Repos {
		visibility := ""
		if !repo.Private {
			visibility = "; public"
		}
		if len(repo.Workflows) == 0 {
			_, _ = fmt.Fprintf(hf.w, "%s (0 workflows; 0ms%s)\n", repo.Repo.FullName, visibility)
		} else {
			_, _ = fmt.Fprintf(hf.w, "%s (%d workflows; %s%s):\n", repo.Repo.FullName, len(repo.Workflows), Humanize(repo.Total), visibility)
			for _, workflow := range repo.Workflows {
				if workflow.Workflow.Path == "" {
					_, _ = fmt.Fprintf(hf.w, "- %s (%s)\n", workflow.Workflow.Name, Humanize(workflow.Usage))
				} else {
					_, _ = fmt.Fprintf(hf.w, "- %s (%s, %s, %s)\n", workflow.Workflow.Name, workflow.Workflow.Path, workflow.Workflow.State, Humanize(workflow.Usage))
				}
			}
		}
		_, _ = fmt.Fprintln(hf.w)
	}
	if summary.RepoCount <= 1 {
		return
	}

	_, _ = fmt.Fprintln(hf.w, "Totals:")
	for _, owner := range summary.Owners {
		_, _ = fmt.Fprintf(hf.w, "- %s (%d repositories; %d workflows; %s)\n", owner.Owner, owner.RepoCount, owner.WorkflowCount, Humanize(owner.Total))
	}
	_, _ = fmt.Fprintf(hf.w, "- all repositories (%d repositories; %d workflows; %s)\n", summary.RepoCount, summary.WorkflowCount, Humanize(summary.Total))
}
