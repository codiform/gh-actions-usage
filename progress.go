package main

import (
	"fmt"
	"io"
	"sync"
	"time"
)

const (
	// progressInterval is the shortest time between rewrites of the progress line on a terminal, so that the
	// counter never becomes the bottleneck of a collection.
	progressInterval = 250 * time.Millisecond
	// progressSteps is the number of lines a collection prints when stderr is not a terminal: one per tenth.
	progressSteps = 10
	percent       = 100
)

// statusWriter is the writer for everything said about a collection's progress on stderr: the collector's
// messages and counters, and the transport's rate-limit notices, which can arrive from any goroutine. It keeps
// a counter being redrawn in place from being overwritten by a message, by finishing the line first, and it
// remembers whether anything was said, so that what follows can be set apart by a blank line only when there
// is something to set it apart from.
type statusWriter struct {
	io.Writer

	mu      sync.Mutex
	written bool // anything at all since the last separation
	open    bool // the last write left its line unfinished, as an in-place counter does
}

func (s *statusWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.open && p[0] != '\r' {
		_, _ = s.Writer.Write([]byte{'\n'})
	}
	s.written = true
	s.open = p[len(p)-1] != '\n'
	return s.Writer.Write(p) //nolint:wrapcheck // a plain pass-through
}

// separate finishes any unfinished line and ends the status output with a blank line if there was any, then
// forgets it all, so that a later collection starts afresh.
func (s *statusWriter) separate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.open {
		_, _ = s.Writer.Write([]byte{'\n'})
	}
	if s.written {
		_, _ = s.Writer.Write([]byte{'\n'})
	}
	s.open, s.written = false, false
}

// progressReporter renders the collector's commit counter on stderr. On a terminal the line is rewritten in
// place a few times a second; anywhere else, such as a log or a pipe, a line is printed each time another tenth
// of the total completes, so the output stays readable without a carriage return.
type progressReporter struct {
	w        io.Writer
	terminal bool
	what     string // what is counted, past tense, such as "commits fetched"
	now      func() time.Time
	last     time.Time // when the line was last rewritten on a terminal
	shown    uint      // tenths of the total already printed off a terminal
}

func newProgressReporter(w io.Writer, terminal bool, what string) *progressReporter {
	return &progressReporter{w: w, terminal: terminal, what: what, now: time.Now}
}

// update reports done of total; the collector calls it once per item.
func (p *progressReporter) update(done, total uint) {
	if total == 0 {
		return
	}
	if p.terminal {
		p.rewrite(done, total)
	} else {
		p.step(done, total)
	}
}

func (p *progressReporter) rewrite(done, total uint) {
	now := p.now()
	if done < total && now.Sub(p.last) < progressInterval {
		return
	}
	p.last = now
	end := ""
	if done == total {
		end = "\n"
	}
	_, _ = fmt.Fprintf(p.w, "\r%s%s", p.line(done, total), end)
}

func (p *progressReporter) step(done, total uint) {
	tenths := done * progressSteps / total
	if tenths <= p.shown {
		return
	}
	p.shown = tenths
	_, _ = fmt.Fprintln(p.w, p.line(done, total))
}

// line describes the progress so far; its length never shrinks as done grows, so rewriting it in place leaves
// nothing behind.
func (p *progressReporter) line(done, total uint) string {
	return fmt.Sprintf("%d of %d %s (%d%%)...", done, total, p.what, done*percent/total)
}
