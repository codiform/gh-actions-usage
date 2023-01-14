package main

import (
	"fmt"
	"github.com/geoffreywiseman/gh-actions-usage/client"
	"github.com/geoffreywiseman/gh-actions-usage/format"
	"os"
)

var gh client.Client

func main() {
	fmt.Println("GitHub Actions Usage")
	fmt.Println()

	gh = client.New()
	if len(os.Args) <= 1 {
		tryDisplayCurrentRepo()
	} else {
		tryDisplayAllSpecified(os.Args[1:])
	}
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
	for _, target := range targets {
		repo, err := gh.GetRepository(target)
		if err != nil {
			panic(err)
		}
		if repo == nil {
			fmt.Printf("Cannot find repo: %s\n", target)
			return
		}

		displayRepoUsage(repo)
	}
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

	var lines []string = make([]string, 0, len(workflows))
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
