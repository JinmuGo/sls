.PHONY: build install test clean version help

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILT_BY ?= $(shell whoami)

LDFLAGS := -s -w \
	-X github.com/jinmugo/sls/cmd.version=$(VERSION) \
	-X github.com/jinmugo/sls/cmd.commit=$(COMMIT) \
	-X github.com/jinmugo/sls/cmd.date=$(DATE) \
	-X github.com/jinmugo/sls/cmd.builtBy=$(BUILT_BY)

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	@echo "Building sls $(VERSION)..."
	go build -ldflags="$(LDFLAGS)" -o sls .

install: ## Install the binary to $GOPATH/bin
	@echo "Installing sls $(VERSION)..."
	go install -ldflags="$(LDFLAGS)" .

test: ## Run tests
	go test -v ./...

clean: ## Clean build artifacts
	rm -f sls

version: ## Show version information
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"
	@echo "Date:    $(DATE)"
	@echo "Built:   $(BUILT_BY)"

run: build ## Build and run the binary
	./sls
