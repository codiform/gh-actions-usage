![Demo](doc/demo.gif)

# GH Actions Usage
GitHub CLI extension for measuring the usage of GitHub Actions workflows in the *selected billing period*, which on
GitHub's billing platform is a calendar month in UTC. The current month is reported by default; `--month` selects a
past one.

Usage is computed from the durations of the jobs that ran in the period, so it works for any repository you can
see, including public repositories and other people's repositories, without any billing permissions. Because it
counts every job, it also includes time on self-hosted runners.

This is a change made in 2026. Earlier versions read the per-workflow billable minutes from GitHub's Actions billing
API, which GitHub [closed down](https://github.blog/changelog/2025-02-02-actions-get-workflow-usage-and-get-workflow-run-usage-endpoints-closing-down/)
along with the move to its new billing platform; that endpoint now always reports zero. The replacement
[billing usage report](https://docs.github.com/en/rest/billing/enhanced-billing) only breaks usage down by
repository and runner type, not by workflow, and requires owner or billing-manager access, so this extension
switched to computing usage from job durations instead.

How it works, and what that means for the numbers:
- Workflow runs created in the period are listed, and the jobs of each commit they ran on are fetched from the check
  runs API. Each completed job adds its elapsed time to its workflow; skipped jobs count for nothing.
- A run belongs to the period in which it was created, and its jobs count in full: a run that starts on the last
  day of a month and finishes on the first of the next is counted entirely in the month it started.
- Only the latest attempt of a re-run job is counted, since GitHub replaces a job's check run when it is re-run.
- Runs of workflows that have since been deleted are still counted, and are listed with the state `deleted`.
- The result is elapsed job time, not billed minutes: GitHub rounds each job up to the minute and applies runner
  multipliers when billing, and neither adjustment is applied here.
- Busy repositories need one API call per commit with runs in the period, plus a few for listing. The extension paces
  its requests to stay within GitHub's rate limits, waits and retries when GitHub asks it to, and refuses up front if
  the remaining API budget looks too small for the repositories requested. That check is an estimate, since the
  number of jobs per commit is not known until they are fetched.

I wrote a version of this extension before the Golang support was available for `gh`, which is still available [here](https://github.com/geoffreywiseman/gh-actuse).

## 📦 Installation

1. Install the GitHub CLI - see the [installation instructions](https://github.com/cli/cli#installation).
2. Installation requires a minimum version (2.0.0) of the GitHub CLI that supports extensions.
3. Install this extension: `gh extension install codiform/gh-actions-usage`.

## Usage

Display the usage for the current repository:
```
gh-actions-usage on  main [!+] via 🐹 v1.19.4
❯ gh actions-usage
GitHub Actions Usage

codiform/gh-actions-usage (2 workflows; 4h 5m):
- CI (.github/workflows/ci.yml, active, 4h 3m)
- release (.github/workflows/release.yml, active, 2m 348ms)
```

Display the usage for a specified repository:
```
gh-actions-usage on  main [!+] via 🐹 v1.19.4
❯ gh actions-usage codiform/gh-actions-usage
GitHub Actions Usage

codiform/gh-actions-usage (2 workflows; 1h 1s):
- CI (.github/workflows/ci.yml, active, 59m 20s)
- release (.github/workflows/release.yml, active, 39s 980ms)
```

Display the usage for multiple specified repositories. When more than one repository is shown, the output also includes totals by owner and for all targets:
```
gh-actions-usage on  main [!+] via 🐹 v1.19.4
❯ gh actions-usage geoffreywiseman/gh-actuse codiform/gh-actions-usage
GitHub Actions Usage

codiform/gh-actions-usage (2 workflows; 0ms):
- CI (.github/workflows/ci.yml, active, 0ms)
- release (.github/workflows/release.yml, active, 0ms)

geoffreywiseman/gh-actuse (0 workflows)

Totals:
- codiform (1 repositories; 2 workflows; 0ms)
- geoffreywiseman (1 repositories; 0 workflows; 0ms)
- all repositories (2 repositories; 2 workflows; 0ms)
```

Display the usage for all repos of an organization:
```
gh-actions-usage on  main [!⇡] via 🐹 v1.19.4 took 2s
❯ gh actions-usage codiform
GitHub Actions Usage

codiform/gh-actions-usage (2 workflows; 0ms):
- CI (.github/workflows/ci.yml, active, 0ms)
- release (.github/workflows/release.yml, active, 0ms)
```

Display the usage for all repos of a user:
```shell
❯ gh actions-usage kkruszewska
GitHub Actions Usage

kkruszewska/data_polishers_titanic (0 workflows)

kkruszewska/hello-world (0 workflows)

Totals:
- kkruszewska (2 repositories; 0 workflows; 0ms)
- all repositories (2 repositories; 0 workflows; 0ms)
```

Display the usage for a mix of repos, organizations and users:
```shell
❯ gh actions-usage codiform geoffreywiseman/gh-actuse misaha
GitHub Actions Usage

codiform/gh-actions-usage (2 workflows; 0ms):
- CI (.github/workflows/ci.yml, active, 0ms)
- release (.github/workflows/release.yml, active, 0ms)

geoffreywiseman/gh-actuse (0 workflows)

misaha/curly-octo-tribble (0 workflows)

Totals:
- codiform (1 repositories; 2 workflows; 0ms)
- geoffreywiseman (1 repositories; 0 workflows; 0ms)
- misaha (1 repositories; 0 workflows; 0ms)
- all repositories (3 repositories; 2 workflows; 0ms)
```

Display the usage for a completed month instead of the current one:
```shell
❯ gh actions-usage --month=2026-08 codiform/gh-actions-usage
GitHub Actions Usage

codiform/gh-actions-usage (2 workflows; 4h 5m):
- CI (.github/workflows/ci.yml, active, 4h 3m)
- release (.github/workflows/release.yml, active, 2m 348ms)
```

Display the usage for a mix of repos using a tab-separated value format (TSV):

```shell
gh-actions-usage on  feature/formatters [!] via 🐹 v1.21.1 took 2s
❯ gh actions-usage --output=tsv --skip codiform geoffreywiseman/gh-actuse kim0
GitHub Actions Usage (3a7cfc0)

Repo	Workflow	Milliseconds
codiform/gh-actions-usage	.github/workflows/ci.yml	350000
codiform/gh-actions-usage	.github/workflows/release.yml	2500
kim0/brave-core	.github/workflows/pull_request.yml	0
kim0/brave-core	.github/workflows/require-checklist.yml	0
kim0/brave-core	.github/workflows/set-milestone-from-base-branch.yml	0
kim0/brave-core	.github/workflows/alert_unsigned_commits.yml	0
kim0/brave-core	.github/workflows/codeql-analysis.yml	0
kim0/haven-main	.github/workflows/linux-227.yml	0
kim0/haven-main	.github/workflows/linux-229.yml	0
kim0/haven-main	.github/workflows/macos.yml	0
kim0/haven-main	.github/workflows/windows.yml	0
kim0/haven-main	.github/workflows/docker-build-push.yml	0
kim0/haven-offshore	.github/workflows/main.yml	75035
kim0/terraform-switcher	.github/workflows/release.yml	1239
```

## Flags

- `--month=YYYY-MM` selects the billing period to report, a calendar month in UTC; defaults to the current month. The
  current month is reported up to now, and a past month in full. Future months are rejected.
- `--output=human|tsv` selects the output format; `tsv` is machine-readable.
- `--skip` omits repositories that have no workflows.
- `--verbose` prints full error details instead of the short message.

# References
- GitHub [REST OpenAPI](https://raw.githubusercontent.com/github/rest-api-description/main/descriptions/api.github.com/api.github.com.yaml)
- GitHub [Rest Docs](https://docs.github.com/en/rest/reference)
- [gh-actuse](https://github.com/geoffreywiseman/gh-actuse/blob/main/gh-actuse), the original / bash implementation
