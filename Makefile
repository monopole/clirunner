GOBIN = $(shell go env GOBIN)
ifeq ($(GOBIN),)
GOBIN = $(shell go env GOPATH)/bin
endif

.PHONY: all
all: lint test $(GOBIN)/testcli

# Must explicitly descend into sub-modules to run their tests.
.PHONY: test
test: $(GOBIN)/testcli
	go test ./...

report: $(GOBIN)/goreportcard-cli
	$(GOBIN)/goreportcard-cli -v

.PHONY: lint
lint: $(GOBIN)/golangci-lint fix-imports
	$(GOBIN)/golangci-lint run ./...

.PHONY: fix-imports
fix-imports: $(GOBIN)/goimports
	./hack/fix_imports.sh

$(GOBIN)/goreportcard-cli: $(GOBIN)/misspell
	( \
		set -e; \
		d=$(shell mktemp -d); cd $$d; \
		git clone https://github.com/gojp/goreportcard.git; \
		cd goreportcard; \
		make install; \
		go install ./cmd/goreportcard-cli; \
		cd; rm -rf $$d \
	)

$(GOBIN)/goimports:
	./hack/go_tool_install.sh goimports

$(GOBIN)/golangci-lint:
	./hack/go_tool_install.sh golangci-lint

$(GOBIN)/misspell:
	./hack/go_tool_install.sh misspell

$(GOBIN)/testcli:
	./hack/go_tool_install.sh testcli

.PHONY: clean
clean:
	go clean -testcache
	rm -f $(GOBIN)/goimports
	rm -f $(GOBIN)/golangci-lint
	rm -f $(GOBIN)/goreportcard-cli
	rm -f $(GOBIN)/misspell
	rm -f $(GOBIN)/stringer
	rm -f $(GOBIN)/testcli