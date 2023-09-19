package main

import (
	"flag"
	"fmt"
	"log"
	"runtime/debug"
	"strings"
	"time"

	"github.com/geoffreywiseman/gh-actions-usage/client"
	"github.com/geoffreywiseman/gh-actions-usage/format"
)

var gh client.Client

type Config struct {
	Output string
	Skip   bool
}

func main() {
	fmt.Printf("GitHub Actions Usage (%s)\n\n", getVersion())

	gh = client.New()

	config := &Config{}
	flag.BoolVar(&config.Skip, "skip", false, "Skips displaying repositories with no workflows")
	flag.StringVar(&config.Output, "output", "human", "output format: human or TSV (machine readable)")
	flag.Parse()
	if config.Output != "human" && config.Output != "tsv" {
		log.Fatal("Invalid output format. Choose 'human' or 'tsv'.")
	}

	if len(flag.Args()) < 1 {
		tryDisplayCurrentRepo(*config)
	} else {
		tryDisplayAllSpecified(*config, flag.Args())
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

func tryDisplayCurrentRepo(config Config) {
	repo, err := gh.GetCurrentRepository()
	if repo == nil {
		fmt.Printf("No current repository: %s\n\n", err)
		printUsage()
		return
	}
	var repoFlowUsage = make(map[*client.Repository]workflowUsage)
	r := getRepoUsage(false, repo)
	repoFlowUsage[repo] = r
	printRepoFlowUsage(config, repoFlowUsage)
}

func tryDisplayAllSpecified(config Config, targets []string) {
	repos, err := getRepositories(targets)
	if err != nil {
		fmt.Printf("Error getting targets: %s\n\n", err)
		printUsage()
		return
	}
	var repoFlowUsage = make(map[*client.Repository]workflowUsage)
	for _, list := range repos {
		for _, item := range list {
			r := getRepoUsage(config.Skip, item)
			repoFlowUsage[item] = r
		}
	}
	printRepoFlowUsage(config, repoFlowUsage)
}

func printRepoFlowUsage(config Config, results repoFlowUsage) {
	switch config.Output {
	case "tsv":
		printRepoFlowUsage_tsv(results)
	default:
		printRepoFlowUsage_human(results)
	}

}

func printRepoFlowUsage_tsv(results repoFlowUsage) {
	fmt.Printf("%s\t%s\t%s\n", "Repo", "Workflow", "Minutes")
	for repo, flowUsage := range results {
		for workflow, usage := range flowUsage {
			d := time.Duration(usage) * time.Millisecond
			fmt.Printf("%s\t%s\t%d\n", repo.FullName, workflow.Path, int(d.Minutes()))
		}
	}
	fmt.Println("")
}

func printRepoFlowUsage_human(results repoFlowUsage) {

	for repo, flowUsage := range results {
		var lines = make([]string, 0, len(flowUsage))
		var repoTotal uint
		for flow, usage := range flowUsage {
			repoTotal += usage
			line := fmt.Sprintf("- %s (%s, %s, %s)", flow.Name, flow.Path, flow.State, format.Humanize(usage))
			lines = append(lines, line)
			// result[flow] = usage.TotalMs()
		}
		fmt.Printf("%s (%d workflows; %s): \n", repo.FullName, len(results[repo]), format.Humanize(repoTotal))
		for _, line := range lines {
			fmt.Println(line)
		}
		fmt.Println()

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

type workflowUsage map[client.Workflow]uint
type repoFlowUsage map[*client.Repository]workflowUsage

func getRepoUsage(skip bool, repo *client.Repository) workflowUsage {
	workflows, err := gh.GetWorkflows(*repo)
	if err != nil {
		panic(err)
	}

	var result = make(workflowUsage)
	for _, flow := range workflows {
		usage, err := gh.GetWorkflowUsage(*repo, flow)
		if err != nil {
			panic(err)
		}
		result[flow] = usage.TotalMs()

	}

	return result
}

func printUsage() {
	fmt.Println("USAGE: gh actions-usage [--output=human|tsv] [--skip] [target]...\n\n" +
		"Gets the usage for all workflows in one or more GitHub repositories.\n\n" +
		"If target is not specified, actions-usage will attempt to get usage for a git repo in the current working directory.\n" +
		"Target can be one of:\n" +
		"- username (e.g. geoffreywiseman)\n" +
		"- organization (e.g. codiform)\n" +
		"- repository (e.g. codiform/gh-actions-usage)")
}
