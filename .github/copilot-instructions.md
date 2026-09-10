# Copilot Instructions

## Project Overview

`gh-actions-usage` is a [GitHub CLI](https://cli.github.com/) extension written in Go that measures the usage of GitHub Actions workflows in the current billing period (the calendar month, UTC). Usage is computed from job durations via the check runs API, so it works for any visible repository without billing permissions. (Until 2026 it read billable minutes from the per-workflow timing endpoint, which GitHub closed down with its new billing platform; the replacement billing usage report has no per-workflow breakdown.) It is installed and run as `gh actions-usage`.

## Architecture

- **`main.go`** — Entry point; parses CLI flags (`--output`, `--skip`, `--verbose`), resolves targets to repositories, runs the collector, and prints results or errors.
- **`client/`** — GitHub API client wrapping `github.com/cli/go-gh/v2`. `client.go` provides `GetCurrentRepository`, `GetRepository`, `GetUser`, `GetAllRepositories`, and `GetWorkflows`; `actions.go` provides `GetWorkflowRuns` (splits windows that exceed GitHub's 1,000-result cap by day), `GetCheckRuns`, and `GetRateLimit`; `transport.go` is an `http.RoundTripper` that paces requests and retries rate-limited ones.
- **`usage/`** — `Collector` computes `client.RepoUsage` for a period: seeds every workflow at zero, lists runs, joins each commit's check runs to workflows by check suite id, and sums completed non-skipped job durations. It preflights the rate limit before fetching check runs and returns `RateLimitError` if the budget is short.
- **`format/`** — Output formatters: `human` (default, readable) and `tsv` (machine-readable). `formatters.go` registers formatters; `usage_summary.go` computes owner/total rollups shared by both formatters.
- **`mock/`** — Testify-based mock of the `client.REST` interface (the one-method seam over go-gh), used by `client` tests. `usage` tests mock the `usage.API` interface locally.

## Coding Conventions

- **Language & toolchain**: Go; see `go.mod` for the required Go version and toolchain.
- **Module path**: `github.com/geoffreywiseman/gh-actions-usage`
- **Linting**: `golangci-lint` v2 with `default: all`; see `.golangci.yml` for the list of disabled linters.
- **Tests**: Use `github.com/stretchr/testify` for assertions; mocks live in `mock/`. Run with `go test -race --vet=off ./...`.
- **Rate limits**: Every request goes through `client/transport.go`; do not add ad-hoc sleeps or retries elsewhere. Progress and backoff messages go to stderr so stdout stays machine-readable.
- **Error handling**: Errors propagate via `fmt.Errorf` with `%w`; nil-not-found is the documented pattern for "not found" vs "error" in client functions.
- **Nil-not-found pattern**: Client functions return `(nil, nil)` when a resource is not found and `(nil, err)` on a real error. Callers check for `nil` result before checking the error.
- **Globals**: Package-level globals are intentional for the CLI client and formatter map (`gochecknoglobals` is disabled).
- **Comments**: Exported types and functions have doc comments; no period required at end of comment.

## Build & Test Commands

```sh
# Build
go build -v ./...

# Run tests
go test -race --vet=off ./...

# Lint (requires golangci-lint v2 installed)
golangci-lint run
```

Or use the `justfile` targets: `just lint`, `just test`, `just build`.

## Output Formats

- **human** (default): Formatted for readability; includes a `Totals:` section when multiple repositories are displayed.
- **tsv**: Tab-separated values; columns are `Repo`, `Workflow`, `Milliseconds`. No aggregate totals row in TSV output.

## Key Patterns

- New output formats should implement the `format.Formatter` interface and register via `format.GetFormatter`.
- `format/usage_summary.go` (`summarizeUsage`) provides owner-level and all-repos rollups for formatters that need them.
- The `--skip` flag omits repositories with no workflows from output.
- The period is always the current calendar month in UTC (`usage.CurrentPeriod`); the collector takes explicit `From`/`To` bounds so a period flag could be added without changing it.
