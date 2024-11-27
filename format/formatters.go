package format

import (
	"fmt"
	"os"

	"github.com/geoffreywiseman/gh-actions-usage/client"
)

var formatters = map[string]Formatter{
	"human": humanFormatter{os.Stdout},
	"tsv":   tsvFormatter{os.Stdout},
}

// Formatter is an interface for formatting output from the extension, allowing the user to pick one of several output styles
type Formatter interface {
	PrintUsage(usage client.RepoUsage)
}

// UnknownFormatterError is an error when the specified formatter can't be found
type UnknownFormatterError string

// Error returns a formatted error message for UnknownFormatterError
func (e UnknownFormatterError) Error() string {
	return fmt.Sprintf("Unknown formatter: %s", string(e))
}

// GetFormatter returns a formatter by name, or an error if the name is unknown
func GetFormatter(name string) (Formatter, error) {
	f, ok := formatters[name]
	if !ok {
		return nil, UnknownFormatterError(name)
	}
	return f, nil
}
