# GH Actions Usage
GitHub CLI extension for measuring the billable usage of GitHub Actions in the current billing period.

*Note that this doesn't include self-hosted runners, which don't incur billable time on GitHub Actions.*

I wrote a version of this extension before the Golang support was available for `gh`, which is still available [here](https://github.com/geoffreywiseman/gh-actuse).

## Features

Display the usage for the current repository:
```
GitHub Actions Usage

MyOrg/MyRepo (2 workflows; 10h 41m):
- CI (.github/workflows/ci.yaml, active, 10h 29m)
- Infrastructure (.github/workflows/infra.yaml, active, 3h 10m)
```       

Display the usage for a specified repository:
```
❯ gh actions-usage MyOrg/myrepo
GitHub Actions Usage

MyOrg/MyRepo (3 workflows; 8h 39m):
- Build (.github/workflows/build.yaml, active, 8h 33m)
- Deploy (.github/workflows/deploy.yaml, active, 0ms)
- Push (.github/workflows/push.yaml, active, 6m 0s)
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
