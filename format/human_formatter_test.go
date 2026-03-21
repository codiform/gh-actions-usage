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
	r := client.Repository{FullName: "codiform/gh-actions-usage", Private: true}
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

	r := client.Repository{FullName: "geoffreywiseman/Moo", Private: true}
	ru := make(client.RepoUsage)
	ru[&r] = make(client.WorkflowUsage)

	// When
	formatter.PrintUsage(ru)

	// Then
	assert.Equal(t, `geoffreywiseman/Moo (0 workflows; 0ms)

`, output.String())
}

func TestHumanFormatter_PublicRepo(t *testing.T) {
	// Given
	var output bytes.Buffer
	formatter := humanFormatter{&output}

	wf := client.Workflow{Name: "CI", Path: ".github/workflows/ci.yml", State: "active"}
	wfu := make(client.WorkflowUsage)
	wfu[wf] = 0
	r := client.Repository{FullName: "geoffreywiseman/gh-actuse"}
	ru := make(client.RepoUsage)
	ru[&r] = wfu

	// When
	formatter.PrintUsage(ru)

	// Then
	assert.Equal(t, `geoffreywiseman/gh-actuse (1 workflows; 0ms; public):
- CI (.github/workflows/ci.yml, active, 0ms)

`, output.String())
}

func TestHumanFormatter_PublicRepo_NoWorkflows(t *testing.T) {
	// Given
	var output bytes.Buffer
	formatter := humanFormatter{&output}

	r := client.Repository{FullName: "geoffreywiseman/public-empty"}
	ru := make(client.RepoUsage)
	ru[&r] = make(client.WorkflowUsage)

	// When
	formatter.PrintUsage(ru)

	// Then
	assert.Equal(t, `geoffreywiseman/public-empty (0 workflows; 0ms; public)

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
	assert.Equal(t, `codiform/gh-actions-usage (2 workflows; 2s):
- CI (.github/workflows/ci.yml, active, 500ms)
- Release (.github/workflows/release.yml, active, 1s 500ms)

codiform/terraform-tools (1 workflows; 1s):
- CI (.github/workflows/ci.yml, active, 1s)

geoffreywiseman/gh-actuse (0 workflows; 0ms; public)

Totals:
- codiform (2 repositories; 3 workflows; 3s)
- geoffreywiseman (1 repositories; 0 workflows; 0ms)
- all repositories (3 repositories; 3 workflows; 3s)
`, output.String())
}

func TestHumanFormatter_BillingSkus(t *testing.T) {
	// Given
	var output bytes.Buffer
	formatter := humanFormatter{&output}

	// SKU-based workflows have no Path or State (billing API data)
	linux := client.Workflow{Name: "Actions Linux"}
	macos := client.Workflow{Name: "Actions macOS"}
	wfu := client.WorkflowUsage{
		linux: 100 * 60000, // 100 minutes in ms
		macos: 30 * 60000,  // 30 minutes in ms
	}
	r := client.Repository{FullName: "codiform/gh-actions-usage", Private: true}
	ru := client.RepoUsage{&r: wfu}

	// When
	formatter.PrintUsage(ru)

	// Then
	assert.Equal(t, `codiform/gh-actions-usage (2 workflows; 2h 10m):
- Actions Linux (1h 40m)
- Actions macOS (30m)

`, output.String())
}
