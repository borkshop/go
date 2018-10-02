GOPATH=$(shell git rev-parse --show-toplevel)
GO=GOPATH=$(GOPATH) go

run: game
	./$<

.PHONY: game
game:
	$(GO) build børk.no/cmd/game

.PHONY: test
test:
	$(GO) test børk.no/... github.com/jcorbin/anansi/...
