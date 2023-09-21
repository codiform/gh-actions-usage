package format

import (
	"bytes"
	"github.com/geoffreywiseman/gh-actions-usage/client"
	"github.com/stretchr/testify/assert"
	"testing"
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
	assert.Equal(t, "Repo\tWorkflow\tMilliseconds\ncodiform/gh-actions-usage\t.github/workflows/DevSecOps.yaml\t2500\n", output.String())
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
	assert.Equal(t, "Repo\tWorkflow\tMilliseconds\nkim0/salt-states\tn/a\t0\n", output.String())
}
