#
# github.com/docker/cli
#

# Sets the name of the company that produced the windows binary.
PACKAGER_NAME ?=

# The repository doesn't have a go.mod, but "go list", and "gotestsum"
# expect to be run from a module.
GO111MODULE=auto
export GO111MODULE

all: binary

_:=$(shell ./scripts/warn-outside-container $(MAKECMDGOALS))

.PHONY: dev
dev: ## start a build container in interactive mode for in-container development
	@if [ -n "${DISABLE_WARN_OUTSIDE_CONTAINER}" ]; then \
		echo "you are already in the dev container"; \
	else \
		$(MAKE) -f docker.Makefile dev; \
	fi

.PHONY: shell
shell: dev ## alias for dev

.PHONY: clean
clean: ## remove build artifacts
	rm -rf ./build/* man/man[1-9] docs/yaml

.PHONY: test
test: test-unit ## run tests

.PHONY: test-unit
test-unit: ## run unit tests, to change the output format use: GOTESTSUM_FORMAT=(dots|short|standard-quiet|short-verbose|standard-verbose) make test-unit
	gotestsum -- $${TESTDIRS:-$(shell go list ./... | grep -vE '/vendor/|/e2e/')} $(TESTFLAGS)

.PHONY: test-coverage
test-coverage: ## run test coverage
	mkdir -p $(CURDIR)/build/coverage
	gotestsum -- $(shell go list ./... | grep -vE '/vendor/|/e2e/') -coverprofile=$(CURDIR)/build/coverage/coverage.txt

.PHONY: lint
lint: ## run all the lint tools
	golangci-lint run

.PHONY: shellcheck
shellcheck: ## run shellcheck validation
	find scripts/ contrib/completion/bash -type f | grep -v scripts/winresources | grep -v '.*.ps1' | xargs shellcheck

.PHONY: fmt
fmt: ## run gofumpt (if present) or gofmt
	@if command -v gofumpt > /dev/null; then \
		gofumpt -w -d -lang=1.21 . ; \
	else \
		go list -f {{.Dir}} ./... | xargs gofmt -w -s -d ; \
	fi

.PHONY: binary
binary: ## build executable for Linux
	./scripts/build/binary

.PHONY: dynbinary
dynbinary: ## build dynamically linked binary
	GO_LINKMODE=dynamic ./scripts/build/binary

.PHONY: plugins
plugins: ## build example CLI plugins
	./scripts/build/plugins

.PHONY: vendor
vendor: ## update vendor with go modules
	rm -rf vendor
	./scripts/vendor update

.PHONY: validate-vendor
validate-vendor: ## validate vendor
	./scripts/vendor validate

.PHONY: mod-outdated
mod-outdated: ## check outdated dependencies
	./scripts/vendor outdated

.PHONY: authors
authors: ## generate AUTHORS file from git history
	scripts/docs/generate-authors.sh

.PHONY: manpages
manpages: ## generate man pages from go source and markdown
	scripts/docs/generate-man.sh

.PHONY: mddocs
mddocs: ## generate markdown files from go source
	scripts/docs/generate-md.sh

.PHONY: yamldocs
yamldocs: ## generate documentation YAML files consumed by docs repo
	scripts/docs/generate-yaml.sh

.PHONY: help
help: ## print this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {gsub("\\\\n",sprintf("\n%22c",""), $$2);printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
