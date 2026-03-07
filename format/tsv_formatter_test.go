package format

import (
	"bytes"
	"testing"

	"github.com/geoffreywiseman/gh-actions-usage/client"
	"github.com/stretchr/testify/assert"
)

func TestTsvFormatter(t *testing.T) {
	// Given
	var output bytes.Buffer
	formatter := tsvFormatter{&output}

	wf := client.Workflow{Name: "Security", Path: ".github/workflows/DevSecOps.yaml", State: "alert"}
	wfu := make(client.WorkflowUsage)
	wfu[wf] = 2500
	r := client.Repository{FullName: "codiform/gh-actions-usage"}
	ru := make(client.RepoUsage)
	ru[&r] = wfu

	// When
	formatter.PrintUsage(ru)

	// Then
	assert.Equal(t, `Repo	Workflow	Milliseconds
codiform/gh-actions-usage	.github/workflows/DevSecOps.yaml	2500
`, output.String())
}

func TestTsvFormatter_Empty(t *testing.T) {
	// Given
	var output bytes.Buffer
	formatter := tsvFormatter{&output}

	wfu := make(client.WorkflowUsage)
	r := client.Repository{FullName: "kim0/salt-states"}
	ru := make(client.RepoUsage)
	ru[&r] = wfu

	// When
	formatter.PrintUsage(ru)

	// Then
	assert.Equal(t, `Repo	Workflow	Milliseconds
kim0/salt-states	n/a	0
`, output.String())
}

func TestTsvFormatter_Totals(t *testing.T) {
	// Given
	var output bytes.Buffer
	formatter := tsvFormatter{&output}

	codiform := &client.User{Login: "codiform"}
	geoffreywiseman := &client.User{Login: "geoffreywiseman"}

	ci := client.Workflow{Name: "CI", Path: ".github/workflows/ci.yml", State: "active"}
	release := client.Workflow{Name: "Release", Path: ".github/workflows/release.yml", State: "active"}

	firstRepo := &client.Repository{Owner: codiform, FullName: "codiform/gh-actions-usage"}
	secondRepo := &client.Repository{Owner: codiform, FullName: "codiform/terraform-tools"}
	thirdRepo := &client.Repository{Owner: geoffreywiseman, FullName: "geoffreywiseman/gh-actuse"}

	ru := client.RepoUsage{
		firstRepo: {
			release: 1500,
			ci:      500,
		},
		secondRepo: {
			ci: 1000,
		},
		thirdRepo: {},
	}

	// When
	formatter.PrintUsage(ru)

	// Then
	assert.Equal(t, `Repo	Workflow	Milliseconds
codiform/gh-actions-usage	.github/workflows/ci.yml	500
codiform/gh-actions-usage	.github/workflows/release.yml	1500
codiform/terraform-tools	.github/workflows/ci.yml	1000
geoffreywiseman/gh-actuse	n/a	0
codiform	TOTAL	3000
geoffreywiseman	TOTAL	0
all repositories	TOTAL	3000
`, output.String())
}
