default: lint test

lint:
    go fmt
    go vet
    golint ./...

test:
    go test -race --vet=off ./...

build: lint test
   go build
