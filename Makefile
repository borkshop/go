GOPATH=$(shell git rev-parse --show-toplevel)

PACKAGES=borkshop/...
PACKAGES+=deathroom/...
PACKAGES+=github.com/jcorbin/anansi/...

.PHONY: test
test:
	export GOPATH=$(GOPATH)
	go test $(PACKAGES)
