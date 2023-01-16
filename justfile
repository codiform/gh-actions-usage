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
