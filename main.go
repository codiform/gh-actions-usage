package main

import (
	"flag"
	"fmt"
	"github.com/geoffreywiseman/gh-actions-usage/client"
	"github.com/geoffreywiseman/gh-actions-usage/format"
	"log"
	"runtime/debug"
	"strings"
)

var gh client.Client

type config struct {
	output string
	skip   bool
	format format.Formatter
}

func main() {
	fmt.Printf("GitHub Actions Usage (%s)\n\n", getVersion())

	gh = client.New()

	cfg := &config{}
	flag.BoolVar(&cfg.skip, "skip", false, "Skips displaying repositories with no workflows")
	flag.StringVar(&cfg.output, "output", "human", "Output format: human or TSV (machine readable)")
	flag.Parse()

	var err error
	cfg.format, err = format.GetFormatter(cfg.output)
	if err != nil {
		log.Fatal(err)
	}

	if len(flag.Args()) < 1 {
		tryDisplayCurrentRepo(*cfg)
	} else {
		tryDisplayAllSpecified(*cfg, flag.Args())
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

func tryDisplayCurrentRepo(cfg config) {
	repo, err := gh.GetCurrentRepository()
	if repo == nil {
		fmt.Printf("No current repository: %s\n\n", err)
		printHelp()
		return
	}
	var repoFlowUsage = make(map[*client.Repository]client.WorkflowUsage)
	r := getRepoUsage(false, repo)
	repoFlowUsage[repo] = r
	cfg.format.PrintUsage(repoFlowUsage)
}

func tryDisplayAllSpecified(cfg config, targets []string) {
	repos, err := getRepositories(targets)
	if err != nil {
		fmt.Printf("Error getting targets: %s\n\n", err)
		printHelp()
		return
	}
	var repoFlowUsage = make(map[*client.Repository]client.WorkflowUsage)
	for _, list := range repos {
		for _, item := range list {
			r := getRepoUsage(cfg.skip, item)
			repoFlowUsage[item] = r
		}
	}
	cfg.format.PrintUsage(repoFlowUsage)
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

func getRepoUsage(skip bool, repo *client.Repository) client.WorkflowUsage {
	workflows, err := gh.GetWorkflows(*repo)
	if err != nil {
		panic(err)
	}

	var result = make(client.WorkflowUsage)
	for _, flow := range workflows {
		usage, err := gh.GetWorkflowUsage(*repo, flow)
		if err != nil {
			panic(err)
		}
		result[flow] = usage.TotalMs()

	}

	return result
}

func printHelp() {
	fmt.Println("USAGE: gh actions-usage [--output=human|tsv] [--skip] [target]...\n\n" +
		"Gets the usage for all workflows in one or more GitHub repositories.\n\n" +
		"If target is not specified, actions-usage will attempt to get usage for a git repo in the current working directory.\n" +
		"Target can be one of:\n" +
		"- username (e.g. geoffreywiseman)\n" +
		"- organization (e.g. codiform)\n" +
		"- repository (e.g. codiform/gh-actions-usage)")
}
