![Demo](doc/demo.gif)

# GH Actions Usage
GitHub CLI extension for measuring the *billable usage* of GitHub Actions in the *current billing period*.

This is all the information that's available through the API currently:
- I can't go beyond the current billing period
- I can't see usage minutes that aren't billable, like self-hosted runners, which don't incur billable time on GitHub Actions

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

Display the usage for multiple specified repositories:
```
gh-actions-usage on  main [!+] via 🐹 v1.19.4
❯ gh actions-usage geoffreywiseman/gh-actuse codiform/gh-actions-usage
GitHub Actions Usage

geoffreywiseman/gh-actuse (0 workflows)

codiform/gh-actions-usage (2 workflows; 0ms):
- CI (.github/workflows/ci.yml, active, 0ms)
- release (.github/workflows/release.yml, active, 0ms)
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

# References
- GitHub [REST OpenAPI](https://raw.githubusercontent.com/github/rest-api-description/main/descriptions/api.github.com/api.github.com.yaml)
- GitHub [Rest Docs](https://docs.github.com/en/rest/reference)
- [gh-actuse](https://github.com/geoffreywiseman/gh-actuse/blob/main/gh-actuse), the original / bash implementation
