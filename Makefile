#
# github.com/docker/cli
#
all: binary


_:=$(shell ./scripts/warn-outside-container $(MAKECMDGOALS))

.PHONY: clean
clean: ## remove build artifacts
	rm -rf ./build/* cli/winresources/rsrc_* ./man/man[1-9] docs/yaml/gen

.PHONY: test-unit
test-unit: ## run unit tests, to change the output format use: GOTESTSUM_FORMAT=(dots|short|standard-quiet|short-verbose|standard-verbose) make test-unit 
	gotestsum $(TESTFLAGS) -- $(shell go list ./... | grep -vE '/vendor/|/e2e/')

.PHONY: test
test: test-unit ## run tests

.PHONY: test-coverage
test-coverage: ## run test coverage
	gotestsum -- -coverprofile=coverage.txt $(shell go list ./... | grep -vE '/vendor/|/e2e/')

.PHONY: fmt
fmt:
	go list -f {{.Dir}} ./... | xargs gofmt -w -s -d

.PHONY: lint
lint: ## run all the lint tools
	gometalinter --config gometalinter.json ./...

.PHONY: binary
binary: ## build executable for Linux
	@echo "WARNING: binary creates a Linux executable. Use cross for macOS or Windows."
	./scripts/build/binary

.PHONY: plugins
plugins: ## build example CLI plugins
	./scripts/build/plugins

.PHONY: cross
cross: ## build executable for macOS and Windows
	./scripts/build/cross

.PHONY: binary-windows
binary-windows: ## build executable for Windows
	./scripts/build/windows

.PHONY: plugins-windows
plugins-windows: ## build example CLI plugins for Windows
	./scripts/build/plugins-windows

.PHONY: binary-osx
binary-osx: ## build executable for macOS
	./scripts/build/osx

.PHONY: plugins-osx
plugins-osx: ## build example CLI plugins for macOS
	./scripts/build/plugins-osx

.PHONY: dynbinary
dynbinary: ## build dynamically linked binary
	./scripts/build/dynbinary

vendor: vendor.conf ## check that vendor matches vendor.conf
	rm -rf vendor
	bash -c 'vndr |& grep -v -i clone'
	scripts/validate/check-git-diff vendor

.PHONY: authors
authors: ## generate AUTHORS file from git history
	scripts/docs/generate-authors.sh

.PHONY: manpages
manpages: ## generate man pages from go source and markdown
	scripts/docs/generate-man.sh

.PHONY: yamldocs
yamldocs: ## generate documentation YAML files consumed by docs repo
	scripts/docs/generate-yaml.sh

.PHONY: shellcheck
shellcheck: ## run shellcheck validation
	scripts/validate/shellcheck

.PHONY: help
help: ## print this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {gsub("\\\\n",sprintf("\n%22c",""), $$2);printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)


cli/compose/schema/bindata.go: cli/compose/schema/data/*.json
	go generate github.com/docker/cli/cli/compose/schema

compose-jsonschema: cli/compose/schema/bindata.go
	scripts/validate/check-git-diff cli/compose/schema/bindata.go

.PHONY: ci-validate
ci-validate:
	time make -B vendor
	time make -B compose-jsonschema
	time make manpages
	time make yamldocs
