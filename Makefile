#
# github.com/docker/cli
#

# Sets the name of the company that produced the windows binary.
COMPANY_NAME ?=

all: binary

_:=$(shell ./scripts/warn-outside-container $(MAKECMDGOALS))

.PHONY: clean
clean: ## remove build artifacts
	rm -rf ./build/* cli/winresources/rsrc_* ./man/man[1-9] docs/yaml/gen

.PHONY: test-unit
test-unit: ## run unit tests, to change the output format use: GOTESTSUM_FORMAT=(dots|short|standard-quiet|short-verbose|standard-verbose) make test-unit 
	gotestsum $(TESTFLAGS) -- $${TESTDIRS:-$(shell go list ./... | grep -vE '/vendor/|/e2e/')}

.PHONY: test
test: test-unit ## run tests

.PHONY: test-coverage
test-coverage: ## run test coverage
	gotestsum -- -coverprofile=coverage.txt $(shell go list ./... | grep -vE '/vendor/|/e2e/')

.PHONY: lint
lint: ## run all the lint tools
	golangci-lint run

.PHONY: shellcheck
shellcheck: ## run shellcheck validation
	find scripts/ contrib/completion/bash -type f | grep -v scripts/winresources | grep -v '.*.ps1' | xargs shellcheck

.PHONY: fmt
fmt:
	go list -f {{.Dir}} ./... | xargs gofmt -w -s -d

.PHONY: binary
binary: ## build executable for Linux
	./scripts/build/binary

.PHONY: dynbinary
dynbinary: ## build dynamically linked binary
	GO_LINKMODE=dynamic ./scripts/build/binary

.PHONY: plugins
plugins: ## build example CLI plugins
	./scripts/build/plugins

vendor: vendor.conf ## check that vendor matches vendor.conf
	rm -rf vendor
	bash -c 'vndr |& grep -v -i clone | tee ./vndr.log'
	scripts/validate/check-git-diff vendor
	scripts/validate/check-all-packages-vendored

.PHONY: authors
authors: ## generate AUTHORS file from git history
	scripts/docs/generate-authors.sh

.PHONY: manpages
manpages: ## generate man pages from go source and markdown
	scripts/docs/generate-man.sh

.PHONY: yamldocs
yamldocs: ## generate documentation YAML files consumed by docs repo
	scripts/docs/generate-yaml.sh

.PHONY: help
help: ## print this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {gsub("\\\\n",sprintf("\n%22c",""), $$2);printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: ci-validate
ci-validate:
	time make -B vendor
	time make manpages
	time make yamldocs
