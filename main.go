// Package main is the entry point for the gh-actions-usage extension.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"
	"time"

	gogherrors "github.com/cli/go-gh/v2/pkg/api"
	"github.com/geoffreywiseman/gh-actions-usage/client"
	"github.com/geoffreywiseman/gh-actions-usage/format"
	"github.com/geoffreywiseman/gh-actions-usage/usage"
)

var gh client.Client

// monthLayout is the format of the --month flag, a year and month such as 2026-09.
const monthLayout = "2006-01"

type config struct {
	format  format.Formatter
	output  string
	month   string
	from    time.Time
	to      time.Time
	skip    bool
	verbose bool
	w       io.Writer
}

// UnknownRepoError is an error condition when a repository cannot be found
type UnknownRepoError string

// Error returns a formatted error message for UnknownRepoError
func (e UnknownRepoError) Error() string {
	return "Unknown repository: " + string(e)
}

// UnknownUserError is an error condition where the user cannot be found
type UnknownUserError string

// Error returns a formatted error message for UnknownUserError
func (e UnknownUserError) Error() string {
	return "Unknown user: " + string(e)
}

// InvalidMonthError is an error condition where the --month flag does not name a current or past month
type InvalidMonthError struct {
	Value  string
	Reason string
}

// Error returns a formatted error message for InvalidMonthError
func (e InvalidMonthError) Error() string {
	return fmt.Sprintf("Invalid month %q: %s", e.Value, e.Reason)
}

func main() {
	fmt.Printf("GitHub Actions Usage (%s)\n\n", getVersion())

	gh = client.New()

	now := time.Now()
	cfg := &config{w: os.Stdout}
	flag.BoolVar(&cfg.skip, "skip", false, "Skips displaying repositories with no workflows")
	flag.BoolVar(&cfg.verbose, "verbose", false, "Print verbose output including additional error details")
	flag.StringVar(&cfg.output, "output", "human", "Output format: human or TSV (machine readable)")
	flag.StringVar(&cfg.month, "month", now.UTC().Format(monthLayout), "Billing period to report, as YYYY-MM")
	flag.Parse()

	var err error
	cfg.format, err = format.GetFormatter(cfg.output)
	if err != nil {
		fmt.Printf("Invalid Option: %s\n\n", err)
		printHelp()
		return
	}

	month, err := parseMonth(cfg.month, now)
	if err != nil {
		printError(*cfg, "Invalid option", err)
		printHelp()
		return
	}
	cfg.from, cfg.to = usage.Period(month, now)

	if len(flag.Args()) < 1 {
		tryDisplayCurrentRepo(*cfg)
	} else {
		tryDisplayAllSpecified(*cfg, flag.Args())
	}
}

func getVersion() string {
	const minShaLen = 7
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				hash := setting.Value
				if len(hash) > minShaLen {
					return hash[:minShaLen]
				}
				if len(hash) > 0 {
					return hash
				}
			}
		}
	}
	return "?"
}

func tryDisplayCurrentRepo(cfg config) {
	repo, err := gh.GetCurrentRepository()
	if repo == nil {
		if err != nil {
			printError(cfg, "No current repository", err)
		} else {
			fmt.Printf("No current repository found.\n\n")
		}
		printHelp()
		return
	}
	repoFlowUsage, err := collectUsage(cfg, []*client.Repository{repo})
	if err != nil {
		printError(cfg, "Error collecting usage", err)
		return
	}
	cfg.format.PrintUsage(repoFlowUsage)
}

func tryDisplayAllSpecified(cfg config, targets []string) {
	repos, err := getRepositories(targets)
	if err != nil {
		printError(cfg, "Error getting targets", err)
		printHelp()
		return
	}
	var targetsList []*client.Repository
	for _, list := range repos {
		targetsList = append(targetsList, list...)
	}
	repoFlowUsage, err := collectUsage(cfg, targetsList)
	if err != nil {
		printError(cfg, "Error collecting usage", err)
		return
	}
	if cfg.skip {
		for repo, flows := range repoFlowUsage {
			if len(flows) == 0 {
				delete(repoFlowUsage, repo)
			}
		}
	}
	cfg.format.PrintUsage(repoFlowUsage)
}

// parseMonth returns the first instant, in UTC, of the month named by value as YYYY-MM. Months that have not
// started yet as of now are rejected, since there can be no usage to report for them.
func parseMonth(value string, now time.Time) (time.Time, error) {
	month, err := time.ParseInLocation(monthLayout, value, time.UTC)
	if err != nil {
		return time.Time{}, InvalidMonthError{Value: value, Reason: "expected a year and month such as 2026-09"}
	}
	if month.After(now) {
		return time.Time{}, InvalidMonthError{Value: value, Reason: "the month has not started yet"}
	}
	return month, nil
}

// collectUsage computes the usage of each repository's workflows for the configured billing period.
func collectUsage(cfg config, repos []*client.Repository) (client.RepoUsage, error) {
	collector := usage.Collector{
		API:  &gh,
		From: cfg.from,
		To:   cfg.to,
		Notify: func(message string) {
			_, _ = fmt.Fprintln(os.Stderr, message)
		},
	}
	return collector.Collect(context.Background(), repos) //nolint:wrapcheck // caller prints the error as-is
}

type repoMap map[*client.User][]*client.Repository

// printError prints an error message with varying detail based on error type and verbosity.
// Known typed errors (UnknownRepoError, UnknownUserError, etc.) always print a clean,
// self-describing message without the prefix, as their messages already include full context.
// HTTP errors from the GitHub API print the status code and message.
// Other errors are only shown in full when --verbose is set; otherwise a brief message is shown.
func printError(cfg config, prefix string, err error) {
	if cfg.verbose {
		_, _ = fmt.Fprintf(cfg.w, "%s: %s\n\n", prefix, err)
		return
	}
	if msg, ok := knownErrorMessage(err); ok {
		_, _ = fmt.Fprintf(cfg.w, "%s\n\n", msg)
		return
	}
	if httpErr, ok := errors.AsType[*gogherrors.HTTPError](err); ok {
		_, _ = fmt.Fprintf(cfg.w, "%s: HTTP %d: %s\n\n", prefix, httpErr.StatusCode, httpErr.Message)
		return
	}
	_, _ = fmt.Fprintf(cfg.w, "%s (use --verbose for details)\n\n", prefix)
}

// knownErrorMessage checks if err contains a well-typed, self-describing error and returns
// its clean message. These errors do not require --verbose to produce a useful message.
func knownErrorMessage(err error) (string, bool) {
	if unknownRepo, ok := errors.AsType[UnknownRepoError](err); ok {
		return unknownRepo.Error(), true
	}
	if unknownUser, ok := errors.AsType[UnknownUserError](err); ok {
		return unknownUser.Error(), true
	}
	if invalidMonth, ok := errors.AsType[InvalidMonthError](err); ok {
		return invalidMonth.Error(), true
	}
	if unexpectedHost, ok := errors.AsType[client.UnexpectedHostError](err); ok {
		return unexpectedHost.Error(), true
	}
	if unexpectedUserType, ok := errors.AsType[client.UnexpectedUserTypeError](err); ok {
		return unexpectedUserType.Error(), true
	}
	if rateLimit, ok := errors.AsType[usage.RateLimitError](err); ok {
		return rateLimit.Error(), true
	}
	return "", false
}

func getRepositories(targets []string) (repoMap, error) {
	repos := make(repoMap)
	for _, target := range targets {
		if strings.ContainsRune(target, '/') {
			err := mapRepository(repos, target)
			if err != nil {
				return nil, err
			}
		} else {
			err := mapOwner(repos, target)
			if err != nil {
				return nil, err
			}
		}
	}
	return repos, nil
}

func mapRepository(repos repoMap, repoName string) error {
	repo, err := gh.GetRepository(repoName)
	if err != nil {
		return fmt.Errorf("could not get repository: %w", err)
	}
	if repo == nil {
		return UnknownRepoError(repoName)
	}

	owner := repo.Owner
	list := repos[owner]
	if list == nil {
		list = make([]*client.Repository, 0)
	}
	repos[owner] = append(list, repo)
	return nil
}

func mapOwner(repos repoMap, userName string) error {
	user, err := gh.GetUser(userName)
	if err != nil {
		return fmt.Errorf("could not get user: %w", err)
	}
	if user == nil {
		return UnknownUserError(userName)
	}

	list := repos[user]
	if list == nil {
		list = make([]*client.Repository, 0)
	}

	ors, err := gh.GetAllRepositories(user)
	if err != nil {
		return fmt.Errorf("could not get repositories: %w", err)
	}

	list = append(list, ors...)
	repos[user] = list
	return nil
}

func printHelp() {
	fmt.Println("USAGE: gh actions-usage [--month=YYYY-MM] [--output=human|tsv] [--skip] [--verbose] [target]...\n\n" +
		"Gets the usage for all workflows in one or more GitHub repositories for the selected billing period,\n" +
		"the current month by default.\n\n" +
		"If target is not specified, actions-usage will attempt to get usage for a git repo in the current working directory.\n" +
		"Target can be one of:\n" +
		"- username (e.g. geoffreywiseman)\n" +
		"- organization (e.g. codiform)\n" +
		"- repository (e.g. codiform/gh-actions-usage)")
}
