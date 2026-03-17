package main

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/cli/go-gh/pkg/api"
	"github.com/geoffreywiseman/gh-actions-usage/client"
	"github.com/stretchr/testify/assert"
)

var errGeneric = errors.New("something went wrong")

// cfgVerbose returns a config with verbose enabled.
func cfgVerbose() config {
	return config{verbose: true}
}

// cfgQuiet returns a config with verbose disabled.
func cfgQuiet() config {
	return config{verbose: false}
}

func TestPrintError_Verbose_GenericError(t *testing.T) {
	// Given
	var out bytes.Buffer
	err := errGeneric

	// When
	printError(cfgVerbose(), "No current repository", err, &out)

	// Then
	assert.Equal(t, "No current repository: something went wrong\n\n", out.String())
}

func TestPrintError_Verbose_HTTPError(t *testing.T) {
	// Given
	var out bytes.Buffer
	err := fmt.Errorf("could not get repository: %w", api.HTTPError{StatusCode: 403, Message: "Forbidden"})

	// When
	printError(cfgVerbose(), "No current repository", err, &out)

	// Then
	// Verbose always prints the full error chain, including the URL placeholder.
	assert.Contains(t, out.String(), "No current repository: ")
	assert.Contains(t, out.String(), "Forbidden")
}

func TestPrintError_UnknownRepo(t *testing.T) {
	// Given
	var out bytes.Buffer
	err := UnknownRepoError("codiform/missing")

	// When
	printError(cfgQuiet(), "Error getting targets", err, &out)

	// Then
	assert.Equal(t, "Unknown repository: codiform/missing\n\n", out.String())
}

func TestPrintError_UnknownRepo_Wrapped(t *testing.T) {
	// Given
	var out bytes.Buffer
	err := fmt.Errorf("outer: %w", UnknownRepoError("codiform/missing"))

	// When
	printError(cfgQuiet(), "Error getting targets", err, &out)

	// Then
	assert.Equal(t, "Unknown repository: codiform/missing\n\n", out.String())
}

func TestPrintError_UnknownUser(t *testing.T) {
	// Given
	var out bytes.Buffer
	err := UnknownUserError("johndoe")

	// When
	printError(cfgQuiet(), "Error getting targets", err, &out)

	// Then
	assert.Equal(t, "Unknown user: johndoe\n\n", out.String())
}

func TestPrintError_UnexpectedHost(t *testing.T) {
	// Given
	var out bytes.Buffer
	err := client.UnexpectedHostError("gitlab.com")

	// When
	printError(cfgQuiet(), "No current repository", err, &out)

	// Then
	assert.Equal(t, "Unexpected host: gitlab.com\n\n", out.String())
}

func TestPrintError_UnexpectedUserType(t *testing.T) {
	// Given
	var out bytes.Buffer
	err := client.UnexpectedUserTypeError("Bot")

	// When
	printError(cfgQuiet(), "Error getting targets", err, &out)

	// Then
	assert.Equal(t, "Unexpected user type: Bot\n\n", out.String())
}

func TestPrintError_HTTPError(t *testing.T) {
	// Given
	var out bytes.Buffer
	err := api.HTTPError{StatusCode: 403, Message: "Resource not accessible by integration"}

	// When
	printError(cfgQuiet(), "No current repository", err, &out)

	// Then
	assert.Equal(t, "No current repository: HTTP 403: Resource not accessible by integration\n\n", out.String())
}

func TestPrintError_HTTPError_Wrapped(t *testing.T) {
	// Given
	var out bytes.Buffer
	err := fmt.Errorf("could not get current repository: %w", api.HTTPError{StatusCode: 401, Message: "Unauthorized"})

	// When
	printError(cfgQuiet(), "No current repository", err, &out)

	// Then
	assert.Equal(t, "No current repository: HTTP 401: Unauthorized\n\n", out.String())
}

func TestPrintError_GenericError(t *testing.T) {
	// Given
	var out bytes.Buffer
	err := errGeneric

	// When
	printError(cfgQuiet(), "No current repository", err, &out)

	// Then
	assert.Equal(t, "No current repository (use --verbose for details)\n\n", out.String())
}
