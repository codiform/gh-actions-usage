package format

import (
	"bytes"
	"testing"

	"github.com/geoffreywiseman/gh-actions-usage/client"
	"github.com/stretchr/testify/assert"
)

func TestHumanFormatter(t *testing.T) {
	// Given
	var output bytes.Buffer
	formatter := humanFormatter{&output}

	wf := client.Workflow{Name: "CI", Path: ".github/workflows/ci.yml", State: "active"}
	wfu := make(client.WorkflowUsage)
	wfu[wf] = 50
	r := client.Repository{FullName: "codiform/gh-actions-usage"}
	ru := make(client.RepoUsage)
	ru[&r] = wfu

	// When
	formatter.PrintUsage(ru)

	// Then
	assert.Equal(t, `codiform/gh-actions-usage (1 workflows; 50ms):
- CI (.github/workflows/ci.yml, active, 50ms)

`, output.String())
}

func TestHumanFormatter_Empty(t *testing.T) {
	// Given
	var output bytes.Buffer
	formatter := humanFormatter{&output}

	r := client.Repository{FullName: "geoffreywiseman/Moo"}
	ru := make(client.RepoUsage)
	ru[&r] = make(client.WorkflowUsage)

	// When
	formatter.PrintUsage(ru)

	// Then
	assert.Equal(t, `geoffreywiseman/Moo (0 workflows; 0ms)

`, output.String())
}

func TestHumanFormatter_Totals(t *testing.T) {
	// Given
	var output bytes.Buffer
	formatter := humanFormatter{&output}
	ru := sampleMultipleRepositoriesUsage()

	// When
	formatter.PrintUsage(ru)

	// Then
	assert.Equal(t, `codiform/gh-actions-usage (2 workflows; 2s 0ms):
- CI (.github/workflows/ci.yml, active, 500ms)
- Release (.github/workflows/release.yml, active, 1s 500ms)

codiform/terraform-tools (1 workflows; 1s 0ms):
- CI (.github/workflows/ci.yml, active, 1s 0ms)

geoffreywiseman/gh-actuse (0 workflows; 0ms)

Totals:
- codiform (2 repositories; 3 workflows; 3s 0ms)
- geoffreywiseman (1 repositories; 0 workflows; 0ms)
- all repositories (3 repositories; 3 workflows; 3s 0ms)
`, output.String())
}
