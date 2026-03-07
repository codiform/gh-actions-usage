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

func TestTsvFormatter_MultipleRepositories(t *testing.T) {
	// Given
	var output bytes.Buffer
	formatter := tsvFormatter{&output}
	ru := sampleMultipleRepositoriesUsage()

	// When
	formatter.PrintUsage(ru)

	// Then
	assert.Equal(t, `Repo	Workflow	Milliseconds
codiform/gh-actions-usage	.github/workflows/ci.yml	500
codiform/gh-actions-usage	.github/workflows/release.yml	1500
codiform/terraform-tools	.github/workflows/ci.yml	1000
geoffreywiseman/gh-actuse	n/a	0
`, output.String())
}
