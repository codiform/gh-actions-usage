package main

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// clock is a fake time source that only moves when told to.
type clock struct{ now time.Time }

func (c *clock) tick(d time.Duration) { c.now = c.now.Add(d) }

func newClockedReporter(out *bytes.Buffer, terminal bool) (*progressReporter, *clock) {
	c := &clock{now: time.Date(2026, time.September, 13, 9, 0, 0, 0, time.UTC)}
	reporter := newProgressReporter(out, terminal, "commits fetched")
	reporter.now = func() time.Time { return c.now }
	return reporter, c
}

func TestProgressReporter_Terminal_RewritesInPlaceAFewTimesASecond(t *testing.T) {
	// Given
	var out bytes.Buffer
	reporter, clock := newClockedReporter(&out, true)

	// When: four commits complete in quick succession, then one after a pause
	reporter.update(1, 200)
	clock.tick(progressInterval / 2)
	reporter.update(2, 200)
	reporter.update(3, 200)
	clock.tick(progressInterval / 2)
	reporter.update(4, 200)
	clock.tick(progressInterval)
	reporter.update(5, 200)

	// Then: the first and the ones after each interval are drawn, the ones in between are dropped
	assert.Equal(t, "\r1 of 200 commits fetched (0%)...\r4 of 200 commits fetched (2%)...\r5 of 200 commits fetched (2%)...", out.String())
}

func TestProgressReporter_Terminal_AlwaysEndsTheLine(t *testing.T) {
	// Given
	var out bytes.Buffer
	reporter, _ := newClockedReporter(&out, true)

	// When: the last commit completes within the interval of the previous rewrite
	reporter.update(1, 2)
	reporter.update(2, 2)

	// Then: the final count is drawn regardless, and the line is ended
	assert.Equal(t, "\r1 of 2 commits fetched (50%)...\r2 of 2 commits fetched (100%)...\n", out.String())
}

func TestProgressReporter_Log_PrintsALineEveryTenth(t *testing.T) {
	// Given
	var out bytes.Buffer
	reporter, _ := newClockedReporter(&out, false)

	// When
	for done := uint(1); done <= 25; done++ {
		reporter.update(done, 25)
	}

	// Then: nothing at zero, a line at each tenth, and the last line is the total
	assert.Equal(t, "3 of 25 commits fetched (12%)...\n"+
		"5 of 25 commits fetched (20%)...\n"+
		"8 of 25 commits fetched (32%)...\n"+
		"10 of 25 commits fetched (40%)...\n"+
		"13 of 25 commits fetched (52%)...\n"+
		"15 of 25 commits fetched (60%)...\n"+
		"18 of 25 commits fetched (72%)...\n"+
		"20 of 25 commits fetched (80%)...\n"+
		"23 of 25 commits fetched (92%)...\n"+
		"25 of 25 commits fetched (100%)...\n", out.String())
}

func TestProgressReporter_Log_SkipsTenthsAtOnceWhenUpdatesAreSparse(t *testing.T) {
	// Given
	var out bytes.Buffer
	reporter, _ := newClockedReporter(&out, false)

	// When
	reporter.update(7, 10)
	reporter.update(8, 10)

	// Then: one line for the jump, then one for the next tenth
	assert.Equal(t, "7 of 10 commits fetched (70%)...\n8 of 10 commits fetched (80%)...\n", out.String())
}

func TestProgressReporter_IgnoresAnEmptyTotal(t *testing.T) {
	// Given
	var out bytes.Buffer
	reporter, _ := newClockedReporter(&out, false)

	// When
	reporter.update(0, 0)

	// Then
	assert.Empty(t, out.String())
}

func TestStatusWriter_SeparatesOnlyAfterOutput(t *testing.T) {
	// Given
	var out bytes.Buffer
	status := &statusWriter{Writer: &out}

	// When: nothing has been written
	status.separate()

	// Then
	assert.Empty(t, out.String())

	// When: something has been written, and then separated twice
	_, _ = status.Write([]byte("Listing...\n"))
	status.separate()
	status.separate()

	// Then: one blank line follows it
	assert.Equal(t, "Listing...\n\n", out.String())
}
