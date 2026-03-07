# Copilot Instructions

## Project Overview

`gh-actions-usage` is a [GitHub CLI](https://cli.github.com/) extension written in Go that measures the billable usage of GitHub Actions in the current billing period. It is installed and run as `gh actions-usage`.

## Architecture

- **`main.go`** — Entry point; parses CLI flags (`--output`, `--skip`) and dispatches to per-target or current-repo logic.
- **`client/`** — GitHub API client wrapping `github.com/cli/go-gh`. Provides `GetCurrentRepository`, `GetRepository`, `GetUser`, `GetAllRepositories`, `GetWorkflows`, and `GetWorkflowUsage`.
- **`format/`** — Output formatters: `human` (default, readable) and `tsv` (machine-readable). `formatters.go` registers formatters; `usage_summary.go` computes owner/total rollups shared by both formatters.
- **`mock/`** — Testify-based mock for `client.Client`, used in unit tests.

## Coding Conventions

- **Language & toolchain**: Go; see `go.mod` for the required Go version and toolchain.
- **Module path**: `github.com/geoffreywiseman/gh-actions-usage`
- **Linting**: `golangci-lint` v2 with `default: all`; see `.golangci.yml` for the list of disabled linters.
- **Tests**: Use `github.com/stretchr/testify` for assertions; mocks live in `mock/`. Run with `go test -race --vet=off ./...`.
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
