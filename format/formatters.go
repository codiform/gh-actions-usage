package format

import (
	"fmt"
	"github.com/geoffreywiseman/gh-actions-usage/client"
	"os"
)

var formatters = map[string]Formatter{
	"human": humanFormatter{os.Stdout},
	"tsv":   tsvFormatter{os.Stdout},
}

// Formatter is an interface for formatting output from the extension, allowing the user to pick one of several output styles
type Formatter interface {
	PrintUsage(usage client.RepoUsage)
}

// GetFormatter returns a formatter by name, or an error if the name is unknown
func GetFormatter(name string) (Formatter, error) {
	f, ok := formatters[name]
	if !ok {
		return nil, fmt.Errorf("unknown formatter: %s", name)
	}
	return f, nil
}
