# GH Actions Usage
GitHub CLI extension for measuring the *billable usage* of GitHub Actions in the *current billing period*.

This is all the information that's available through the API currently:
- I can't go beyond the current billing period
- I can't see usage minutes that aren't billable, like self-hosted runners, which don't incur billable time on GitHub Actions

I wrote a version of this extension before the Golang support was available for `gh`, which is still available [here](https://github.com/geoffreywiseman/gh-actuse).

## 📦 Installation

1. Install the GitHub CLI - see the [installation instructions](https://github.com/cli/cli#installation).
1. Installation requires a minimum version (2.0.0) of the the GitHub CLI that supports extensions.
1. Install this extension: `gh extension install codiform/gh-actions-usage`.

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
```
❯ gh actions-usage kkruszewska
GitHub Actions Usage

kkruszewska/data_polishers_titanic (0 workflows)

kkruszewska/hello-world (0 workflows)
```

Display the usage for a mix of repos, organizations and users:
```
❯ gh actions-usage codiform geoffreywiseman/gh-actuse misaha
GitHub Actions Usage

codiform/gh-actions-usage (2 workflows; 0ms):
- CI (.github/workflows/ci.yml, active, 0ms)
- release (.github/workflows/release.yml, active, 0ms)

geoffreywiseman/gh-actuse (0 workflows)

misaha/curly-octo-tribble (0 workflows)
```
 
# References
- GitHub [REST OpenAPI](https://raw.githubusercontent.com/github/rest-api-description/main/descriptions/api.github.com/api.github.com.yaml)
- GitHub [Rest Docs](https://docs.github.com/en/rest/reference)
- [gh-actuse](https://github.com/geoffreywiseman/gh-actuse/blob/main/gh-actuse), the original / bash implementation
