package format

import (
	"fmt"
	"github.com/geoffreywiseman/gh-actions-usage/client"
	"io"
)

type tsvFormatter struct {
	w io.Writer
}

func (tf tsvFormatter) PrintUsage(usage client.RepoUsage) {
	_, _ = fmt.Fprintf(tf.w, "%s\t%s\t%s\n", "Repo", "Workflow", "Milliseconds")
	for repo, flowUsage := range usage {
		for workflow, usage := range flowUsage {
			_, _ = fmt.Fprintf(tf.w, "%s\t%s\t%d\n", repo.FullName, workflow.Path, usage)
		}
	}
}
