GOPATH=$(shell git rev-parse --show-toplevel)
GO=GOPATH=$(GOPATH) go

run: game
	./$<

.PHONY: game
game:
	$(GO) build børk.com/cmd/game

.PHONY: test
test:
	$(GO) test børk.com/... github.com/jcorbin/anansi/...
