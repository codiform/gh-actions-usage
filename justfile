default: lint test vuln

lint:
    go fmt
    go vet
    staticcheck ./...
    golangci-lint run

vuln:
    govulncheck ./...

test:
    go test -race --vet=off ./...

build: lint test
    go build

install-stable:
    gh extension remove codiform/gh-actions-usage
    gh extension install codiform/gh-actions-usage

install-dev: build
    gh extension remove codiform/gh-actions-usage
    gh extension install .