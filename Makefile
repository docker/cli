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
	gotestsum $(TESTFLAGS) -- $${TESTDIRS:-$(shell go list ./... | grep -vE '/vendor/|/e2e/')}

.PHONY: test
test: test-unit ## run tests

.PHONY: test-coverage
test-coverage: ## run test coverage
	gotestsum -- -coverprofile=coverage.txt $(shell go list ./... | grep -vE '/vendor/|/e2e/')

.PHONY: fmt
fmt:
	go list -f {{.Dir}} ./... | xargs gofmt -w -s -d

.PHONY: binary
binary:
	docker buildx bake binary

.PHONY: plugins
plugins: ## build example CLI plugins
	./scripts/build/plugins

.PHONY: cross
cross:
	docker buildx bake cross

.PHONY: plugins-windows
plugins-windows: ## build example CLI plugins for Windows
	./scripts/build/plugins-windows

.PHONY: plugins-osx
plugins-osx: ## build example CLI plugins for macOS
	./scripts/build/plugins-osx

.PHONY: dynbinary
dynbinary: ## build dynamically linked binary
	USE_GLIBC=1 docker buildx bake dynbinary

.PHONY: vendor
vendor: ## update vendor based on go mod/sum
	rm -rf vendor
	go mod tidy
	go mod vendor

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
	time make manpages
	time make yamldocs
