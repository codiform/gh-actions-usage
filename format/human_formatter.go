package format

import (
	"fmt"
	"github.com/geoffreywiseman/gh-actions-usage/client"
	"io"
)

type humanFormatter struct {
	w io.Writer
}

func (hf humanFormatter) PrintUsage(usage client.RepoUsage) {
	for repo, flowUsage := range usage {
		var lines = make([]string, 0, len(flowUsage))
		var repoTotal uint
		for flow, usage := range flowUsage {
			repoTotal += usage
			line := fmt.Sprintf("- %s (%s, %s, %s)", flow.Name, flow.Path, flow.State, Humanize(usage))
			lines = append(lines, line)
		}
		if len(lines) == 0 {
			_, _ = fmt.Fprintf(hf.w, "%s (0 workflows; 0ms)\n", repo.FullName)
		} else {
			_, _ = fmt.Fprintf(hf.w, "%s (%d workflows; %s):\n", repo.FullName, len(usage[repo]), Humanize(repoTotal))
			for _, line := range lines {
				_, _ = fmt.Fprintln(hf.w, line)
			}
		}
		_, _ = fmt.Fprintln(hf.w)
	}
}
