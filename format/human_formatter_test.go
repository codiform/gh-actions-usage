package format

import (
	"bytes"
	"github.com/geoffreywiseman/gh-actions-usage/client"
	"github.com/stretchr/testify/assert"
	"testing"
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
	assert.Equal(t, "codiform/gh-actions-usage (1 workflows; 50ms):\n- CI (.github/workflows/ci.yml, active, 50ms)\n\n", output.String())
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
	assert.Equal(t, "geoffreywiseman/Moo (0 workflows; 0ms)\n\n", output.String())
}
