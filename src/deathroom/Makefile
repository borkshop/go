sources=$(shell find . -maxdepth 1 -name '*.go' -not -name '*_test.go')
branch=$(shell git rev-parse --abbrev-ref HEAD)

all: $(branch)

.PHONY: $(branch)
$(branch):
	go build -o $@ $(sources)

run: $(branch)
	./$(branch) 2>$(branch)-$$(date '+%Y%m%dT%H%M%S%z').log

test:
	go test -v ./...
