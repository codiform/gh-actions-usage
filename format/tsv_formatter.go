package format

import (
	"fmt"
	"io"
	"sort"

	"github.com/geoffreywiseman/gh-actions-usage/client"
)

type tsvFormatter struct {
	w io.Writer
}

func (tf tsvFormatter) PrintUsage(usage client.RepoUsage) {
	_, _ = fmt.Fprintf(tf.w, "%s\t%s\t%s\n", "Repo", "Workflow", "Milliseconds")
	repos := sortedRepositories(usage)
	for _, repo := range repos {
		workflows := sortedWorkflowUsage(usage[repo])
		if len(workflows) == 0 {
			_, _ = fmt.Fprintf(tf.w, "%s\tn/a\t0\n", repo.FullName)
		} else {
			for _, workflow := range workflows {
				_, _ = fmt.Fprintf(tf.w, "%s\t%s\t%d\n", repo.FullName, workflow.Workflow.Path, workflow.Usage)
			}
		}
	}
}

func sortedRepositories(usage client.RepoUsage) []*client.Repository {
	repos := make([]*client.Repository, 0, len(usage))
	for repo := range usage {
		repos = append(repos, repo)
	}
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].FullName < repos[j].FullName
	})
	return repos
}
