# GH Actions Usage
GitHub CLI extension for measuring the billable usage of GitHub Actions in the current billing period.

*Note that this doesn't include self-hosted runners, which don't incur billable time on GitHub Actions.*

I wrote a version of this extension before the Golang support was available for `gh`, which is still available [here](https://github.com/geoffreywiseman/gh-actuse).

## Features

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

# Tasks

## To Do
- Writeup
  - Go implementation
  - Lessons learned
  - Speed comparison

## Done
- Project Skeleton
  - using `gh extension create --precompiled=go`
  - editing in GoLand
- PoC
  - Print all the workflows in a repo
  - JSON Unmarshalling with Struct
  - Print usage for all workflows in a repo
- Restructuring
  - Added client package, moved in client code
  - Added repository check
- Added Test
  
# References
- GitHub [REST OpenAPI](https://raw.githubusercontent.com/github/rest-api-description/main/descriptions/api.github.com/api.github.com.yaml)
- GitHub [Rest Docs](https://docs.github.com/en/rest/reference)
- [gh-actuse](https://github.com/geoffreywiseman/gh-actuse/blob/main/gh-actuse), the original / bash implementation
