GOPATH=$(shell git rev-parse --show-toplevel)
GO=GOPATH=$(GOPATH) go

.PHONY: test
test:
	$(GO) test børk.com/... github.com/jcorbin/anansi/...
