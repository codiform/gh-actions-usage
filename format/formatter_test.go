package format

import "github.com/geoffreywiseman/gh-actions-usage/client"

func sampleMultipleRepositoriesUsage() client.RepoUsage {
	codiform := &client.User{Login: "codiform"}
	geoffreywiseman := &client.User{Login: "geoffreywiseman"}

	ci := client.Workflow{Name: "CI", Path: ".github/workflows/ci.yml", State: "active"}
	release := client.Workflow{Name: "Release", Path: ".github/workflows/release.yml", State: "active"}

	firstRepo := &client.Repository{Owner: codiform, FullName: "codiform/gh-actions-usage", Private: true}
	secondRepo := &client.Repository{Owner: codiform, FullName: "codiform/terraform-tools", Private: true}
	thirdRepo := &client.Repository{Owner: geoffreywiseman, FullName: "geoffreywiseman/gh-actuse"}

	return client.RepoUsage{
		firstRepo: {
			release: 1500,
			ci:      500,
		},
		secondRepo: {
			ci: 1000,
		},
		thirdRepo: {},
	}
}
