GO=go
FILES := $(shell go list ./.../)

.PHONY: all test clean dep

all: test
	$(GO) build -mod=vendor -o bin/http-bench-target

test:
	$(GO) test -mod=vendor -race -v $(FILES) -cover -coverprofile=coverage.out

clean:
	rm -vrf bin

dep:
	$(GO) get
	$(GO) mod tidy
	$(GO) mod vendor
