GOPATH=$(shell git rev-parse --show-toplevel)

.PHONY: test
test:
	export GOPATH=$(GOPATH)
	go test børk.com/... github.com/jcorbin/anansi/...
