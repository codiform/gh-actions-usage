package format

import (
	"fmt"
	"io"

	"github.com/geoffreywiseman/gh-actions-usage/client"
)

type tsvFormatter struct {
	w io.Writer
}

func (tf tsvFormatter) PrintUsage(usage client.RepoUsage) {
	summary := summarizeUsage(usage)
	_, _ = fmt.Fprintf(tf.w, "%s\t%s\t%s\n", "Repo", "Workflow", "Milliseconds")
	for _, repo := range summary.Repos {
		if len(repo.Workflows) == 0 {
			_, _ = fmt.Fprintf(tf.w, "%s\tn/a\t0\n", repo.Repo.FullName)
		} else {
			for _, workflow := range repo.Workflows {
				_, _ = fmt.Fprintf(tf.w, "%s\t%s\t%d\n", repo.Repo.FullName, workflow.Workflow.Path, workflow.Usage)
			}
		}
	}
	if summary.RepoCount <= 1 {
		return
	}
	for _, owner := range summary.Owners {
		_, _ = fmt.Fprintf(tf.w, "%s\tTOTAL\t%d\n", owner.Owner, owner.Total)
	}
	_, _ = fmt.Fprintf(tf.w, "all repositories\tTOTAL\t%d\n", summary.Total)
}
