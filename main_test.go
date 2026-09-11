package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/geoffreywiseman/gh-actions-usage/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errGeneric = errors.New("something went wrong")

// cfgVerbose returns a config with verbose enabled, writing to w.
func cfgVerbose(w io.Writer) config {
	return config{verbose: true, w: w}
}

// cfgQuiet returns a config with verbose disabled, writing to w.
func cfgQuiet(w io.Writer) config {
	return config{verbose: false, w: w}
}

func TestPrintError_Verbose_GenericError(t *testing.T) {
	// Given
	var out bytes.Buffer
	err := errGeneric

	// When
	printError(cfgVerbose(&out), "No current repository", err)

	// Then
	assert.Equal(t, "No current repository: something went wrong\n\n", out.String())
}

func TestPrintError_Verbose_HTTPError(t *testing.T) {
	// Given
	var out bytes.Buffer
	err := fmt.Errorf("could not get repository: %w", &api.HTTPError{StatusCode: 403, Message: "Forbidden"})

	// When
	printError(cfgVerbose(&out), "No current repository", err)

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
	printError(cfgQuiet(&out), "Error getting targets", err)

	// Then
	assert.Equal(t, "Unknown repository: codiform/missing\n\n", out.String())
}

func TestPrintError_UnknownRepo_Wrapped(t *testing.T) {
	// Given
	var out bytes.Buffer
	err := fmt.Errorf("outer: %w", UnknownRepoError("codiform/missing"))

	// When
	printError(cfgQuiet(&out), "Error getting targets", err)

	// Then
	assert.Equal(t, "Unknown repository: codiform/missing\n\n", out.String())
}

func TestPrintError_UnknownUser(t *testing.T) {
	// Given
	var out bytes.Buffer
	err := UnknownUserError("johndoe")

	// When
	printError(cfgQuiet(&out), "Error getting targets", err)

	// Then
	assert.Equal(t, "Unknown user: johndoe\n\n", out.String())
}

func TestPrintError_InvalidMonth(t *testing.T) {
	// Given
	var out bytes.Buffer
	err := InvalidMonthError{Value: "2027-01", Reason: "the month has not started yet"}

	// When
	printError(cfgQuiet(&out), "Invalid option", err)

	// Then
	assert.Equal(t, "Invalid month \"2027-01\": the month has not started yet\n\n", out.String())
}

func TestParseMonth(t *testing.T) {
	now := time.Date(2026, time.September, 9, 12, 34, 56, 789, time.UTC)
	tests := []struct {
		name  string
		value string
		now   time.Time
		month time.Time
	}{
		{name: "current month", value: "2026-09", now: now, month: time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)},
		{name: "past month", value: "2025-12", now: now, month: time.Date(2025, time.December, 1, 0, 0, 0, 0, time.UTC)},
		{
			name:  "first instant of the month",
			value: "2026-09",
			now:   time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
			month: time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "local evening is next month in UTC",
			value: "2026-09",
			now:   time.Date(2026, time.August, 31, 22, 0, 0, 0, time.FixedZone("EST", -5*3600)),
			month: time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			month, err := parseMonth(tc.value, tc.now)
			require.NoError(t, err)
			assert.Equal(t, tc.month, month)
			assert.Equal(t, time.UTC, month.Location())
		})
	}
}

func TestParseMonth_Invalid(t *testing.T) {
	now := time.Date(2026, time.September, 9, 12, 34, 56, 789, time.UTC)
	tests := []struct {
		name   string
		value  string
		reason string
	}{
		{name: "next month", value: "2026-10", reason: "the month has not started yet"},
		{name: "next year", value: "2027-01", reason: "the month has not started yet"},
		{name: "empty", value: "", reason: "expected a year and month such as 2026-09"},
		{name: "unpadded month", value: "2026-9", reason: "expected a year and month such as 2026-09"},
		{name: "month out of range", value: "2026-13", reason: "expected a year and month such as 2026-09"},
		{name: "full date", value: "2026-09-01", reason: "expected a year and month such as 2026-09"},
		{name: "month name", value: "September 2026", reason: "expected a year and month such as 2026-09"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseMonth(tc.value, now)
			var invalid InvalidMonthError
			require.ErrorAs(t, err, &invalid)
			assert.Equal(t, InvalidMonthError{Value: tc.value, Reason: tc.reason}, invalid)
		})
	}
}

func TestPrintError_UnexpectedHost(t *testing.T) {
	// Given
	var out bytes.Buffer
	err := client.UnexpectedHostError("gitlab.com")

	// When
	printError(cfgQuiet(&out), "No current repository", err)

	// Then
	assert.Equal(t, "Unexpected host: gitlab.com\n\n", out.String())
}

func TestPrintError_UnexpectedUserType(t *testing.T) {
	// Given
	var out bytes.Buffer
	err := client.UnexpectedUserTypeError("Bot")

	// When
	printError(cfgQuiet(&out), "Error getting targets", err)

	// Then
	assert.Equal(t, "Unexpected user type: Bot\n\n", out.String())
}

func TestPrintError_HTTPError(t *testing.T) {
	// Given
	var out bytes.Buffer
	err := &api.HTTPError{StatusCode: 403, Message: "Resource not accessible by integration"}

	// When
	printError(cfgQuiet(&out), "No current repository", err)

	// Then
	assert.Equal(t, "No current repository: HTTP 403: Resource not accessible by integration\n\n", out.String())
}

func TestPrintError_HTTPError_Wrapped(t *testing.T) {
	// Given
	var out bytes.Buffer
	err := fmt.Errorf("could not get current repository: %w", &api.HTTPError{StatusCode: 401, Message: "Unauthorized"})

	// When
	printError(cfgQuiet(&out), "No current repository", err)

	// Then
	assert.Equal(t, "No current repository: HTTP 401: Unauthorized\n\n", out.String())
}

func TestPrintError_GenericError(t *testing.T) {
	// Given
	var out bytes.Buffer
	err := errGeneric

	// When
	printError(cfgQuiet(&out), "No current repository", err)

	// Then
	assert.Equal(t, "No current repository (use --verbose for details)\n\n", out.String())
}
