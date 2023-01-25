default: lint test

lint:
    go fmt
    go vet
    golint ./...
    staticcheck ./...

test:
    go test -race --vet=off ./...

build: lint test
    go build

install-stable:
    gh extension remove .
    gh extension install codiform/gh-actions-usage

install-dev:
    gh extension remove codiform/gh-actions-usage
    gh extension install .