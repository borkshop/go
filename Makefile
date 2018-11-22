GOPATH=$(shell git rev-parse --show-toplevel)

PACKAGES=borkshop/...
PACKAGES+=deathroom/...
PACKAGES+=github.com/jcorbin/anansi/...

.PHONY: test
test: lint
	export GOPATH=$(GOPATH)
	go test $(PACKAGES)

.PHONY: lint
lint:
	export GOPATH=$(GOPATH)
	./bin/go_list_sources.sh $(PACKAGES) | xargs gofmt -e -d
	golint $(PACKAGES)

.PHONY: fmt
fmt:
	export GOPATH=$(GOPATH)
	./bin/go_list_sources.sh $(PACKAGES) | xargs gofmt -w
