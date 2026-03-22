// Package main is the entry point for the gh-actions-usage extension.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"runtime/debug"
	"strings"

	gogherrors "github.com/cli/go-gh/pkg/api"
	"github.com/geoffreywiseman/gh-actions-usage/client"
	"github.com/geoffreywiseman/gh-actions-usage/format"
)

// msPerMinute is the number of milliseconds in one minute
const msPerMinute = 60000

var gh client.Client

// errUnknownOwner is returned when the repository owner cannot be determined
var errUnknownOwner = errors.New("repository owner is unknown")

type config struct {
	format  format.Formatter
	output  string
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

func main() {
	fmt.Printf("GitHub Actions Usage (%s)\n\n", getVersion())

	gh = client.New()

	cfg := &config{w: os.Stdout}
	flag.BoolVar(&cfg.skip, "skip", false, "Skips displaying repositories with no workflows")
	flag.BoolVar(&cfg.verbose, "verbose", false, "Print verbose output including additional error details")
	flag.StringVar(&cfg.output, "output", "human", "Output format: human or TSV (machine readable)")
	flag.Parse()

	var err error
	cfg.format, err = format.GetFormatter(cfg.output)
	if err != nil {
		fmt.Printf("Invalid Option: %s\n\n", err)
		printHelp()
		return
	}

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
	ownerUsage, err := getOwnerActionsUsage(repo.Owner)
	if err != nil {
		if isBillingUnavailable(err) {
			printWarning(cfg, err)
		} else {
			printError(cfg, "Error getting billing usage", err)
			return
		}
	}
	repoUsage := make(client.RepoUsage)
	wfUsage := ownerUsage[repo.FullName]
	if wfUsage == nil {
		wfUsage = make(client.WorkflowUsage)
	}
	repoUsage[repo] = wfUsage
	cfg.format.PrintUsage(repoUsage)
}

func tryDisplayAllSpecified(cfg config, targets []string) {
	repos, err := getRepositories(targets)
	if err != nil {
		printError(cfg, "Error getting targets", err)
		printHelp()
		return
	}
	repoFlowUsage := make(client.RepoUsage)
	for owner, repoList := range repos {
		ownerUsage, err := getOwnerActionsUsage(owner)
		if err != nil {
			if isBillingUnavailable(err) {
				printWarning(cfg, err)
				ownerUsage = make(map[string]client.WorkflowUsage)
			} else {
				printError(cfg, "Error getting billing usage", err)
				return
			}
		}
		for _, repo := range repoList {
			wfUsage := ownerUsage[repo.FullName]
			if wfUsage == nil {
				wfUsage = make(client.WorkflowUsage)
			}
			if len(wfUsage) == 0 && cfg.skip {
				continue
			}
			repoFlowUsage[repo] = wfUsage
		}
	}
	cfg.format.PrintUsage(repoFlowUsage)
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
	var httpErr gogherrors.HTTPError
	if errors.As(err, &httpErr) {
		_, _ = fmt.Fprintf(cfg.w, "%s: HTTP %d: %s\n\n", prefix, httpErr.StatusCode, httpErr.Message)
		return
	}
	_, _ = fmt.Fprintf(cfg.w, "%s (use --verbose for details)\n\n", prefix)
}

// knownErrorMessage checks if err contains a well-typed, self-describing error and returns
// its clean message. These errors do not require --verbose to produce a useful message.
func knownErrorMessage(err error) (string, bool) {
	var unknownRepo UnknownRepoError
	if errors.As(err, &unknownRepo) {
		return unknownRepo.Error(), true
	}
	var unknownUser UnknownUserError
	if errors.As(err, &unknownUser) {
		return unknownUser.Error(), true
	}
	var unexpectedHost client.UnexpectedHostError
	if errors.As(err, &unexpectedHost) {
		return unexpectedHost.Error(), true
	}
	var unexpectedUserType client.UnexpectedUserTypeError
	if errors.As(err, &unexpectedUserType) {
		return unexpectedUserType.Error(), true
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

func getOwnerActionsUsage(owner *client.User) (map[string]client.WorkflowUsage, error) {
	if owner == nil {
		return nil, errUnknownOwner
	}
	report, err := gh.GetActionsUsage(owner)
	if err != nil {
		return nil, err
	}
	byRepo := make(map[string]client.WorkflowUsage)
	if report == nil {
		return byRepo, nil
	}
	for _, item := range report.UsageItems {
		if !strings.EqualFold(item.Product, "actions") {
			continue
		}
		repoUsage := byRepo[item.RepositoryName]
		if repoUsage == nil {
			repoUsage = make(client.WorkflowUsage)
			byRepo[item.RepositoryName] = repoUsage
		}
		wf := client.Workflow{Name: item.SKU}
		repoUsage[wf] += uint(math.Round(item.Quantity * msPerMinute))
	}
	return byRepo, nil
}

func printHelp() {
	fmt.Println("USAGE: gh actions-usage [--output=human|tsv] [--skip] [--verbose] [target]...\n\n" +
		"Gets the usage for all workflows in one or more GitHub repositories.\n\n" +
		"If target is not specified, actions-usage will attempt to get usage for a git repo in the current working directory.\n" +
		"Target can be one of:\n" +
		"- username (e.g. geoffreywiseman)\n" +
		"- organization (e.g. codiform)\n" +
		"- repository (e.g. codiform/gh-actions-usage)")
}
