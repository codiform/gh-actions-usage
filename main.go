package main

import (
	"fmt"
	"github.com/geoffreywiseman/gh-actions-usage/client"
	"github.com/geoffreywiseman/gh-actions-usage/format"
	"os"
	"runtime/debug"
	"strings"
)

var gh client.Client

func main() {
	fmt.Printf("GitHub Actions Usage (%s)\n\n", getVersion())

	gh = client.New()
	if len(os.Args) <= 1 {
		tryDisplayCurrentRepo()
	} else {
		tryDisplayAllSpecified(os.Args[1:])
	}
}

func getVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				hash := setting.Value
				if len(hash) > 7 {
					return hash[:7]
				}
				if len(hash) > 0 {
					return hash
				}
			}
		}
	}
	return "?"
}

func tryDisplayCurrentRepo() {
	repo, err := gh.GetCurrentRepository()
	if repo == nil {
		fmt.Printf("No current repository: %s\n\n", err)
		printUsage()
		return
	}
	displayRepoUsage(repo)
}

func tryDisplayAllSpecified(targets []string) {
	repos, err := getRepositories(targets)
	if err != nil {
		fmt.Printf("Error getting targets: %s\n\n", err)
		printUsage()
		return
	}

	for _, list := range repos {
		for _, item := range list {
			displayRepoUsage(item)
		}
	}
}

type repoMap map[*client.User][]*client.Repository

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
		return err
	}
	if repo == nil {
		return fmt.Errorf("unknown repo: %s", repoName)
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
		return err
	}
	if user == nil {
		return fmt.Errorf("unknown user: %s", userName)
	}

	list := repos[user]
	if list == nil {
		list = make([]*client.Repository, 0)
	}

	ors, err := gh.GetAllRepositories(user)
	if err != nil {
		return err
	}

	list = append(list, ors...)
	repos[user] = list
	return nil
}

func displayRepoUsage(repo *client.Repository) {
	workflows, err := gh.GetWorkflows(*repo)
	if err != nil {
		panic(err)
	}

	if len(workflows) == 0 {
		fmt.Printf("%s (0 workflows)\n\n", repo.FullName)
		return
	}

	var lines = make([]string, 0, len(workflows))
	var repoTotal uint
	for _, flow := range workflows {
		usage, err := gh.GetWorkflowUsage(*repo, flow)
		if err != nil {
			panic(err)
		}
		repoTotal += usage.TotalMs()
		line := fmt.Sprintf("- %s (%s, %s, %s)", flow.Name, flow.Path, flow.State, format.Humanize(usage.TotalMs()))
		lines = append(lines, line)
	}

	fmt.Printf("%s (%d workflows; %s): \n", repo.FullName, len(workflows), format.Humanize(repoTotal))
	for _, line := range lines {
		fmt.Println(line)
	}
	fmt.Println()
}

func printUsage() {
	fmt.Println("USAGE: gh actions-usage [target]\n\n" +
		"Gets the usage for all workflows in one or more GitHub repositories.\n\n" +
		"If target is not specified, actions-usage will attempt to get usage for a git repo in the current working directory.\n" +
		"Target can be one of:\n" +
		"- username (e.g. geoffreywiseman)\n" +
		"- organization (e.g. codiform)\n" +
		"- repository (e.g. codiform/gh-actions-usage)")
}
