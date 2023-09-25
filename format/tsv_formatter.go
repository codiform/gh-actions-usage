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
	_, _ = fmt.Fprintf(tf.w, "%s\t%s\t%s\n", "Repo", "Workflow", "Milliseconds")
	for repo, flowUsage := range usage {
		if len(flowUsage) == 0 {
			_, _ = fmt.Fprintf(tf.w, "%s\tn/a\t0\n", repo.FullName)
		} else {
			for workflow, usage := range flowUsage {
				_, _ = fmt.Fprintf(tf.w, "%s\t%s\t%d\n", repo.FullName, workflow.Path, usage)
			}
		}
	}
}
