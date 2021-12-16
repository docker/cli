#
# github.com/docker/cli
#
# Makefile for developing using Docker
#

# Overridable env vars
DOCKER_CLI_MOUNTS ?= -v "$(CURDIR)":/go/src/github.com/docker/cli
DOCKER_CLI_CONTAINER_NAME ?=
DOCKER_CLI_GO_BUILD_CACHE ?= y

# Sets the name of the company that produced the windows binary.
COMPANY_NAME ?=

DEV_DOCKER_IMAGE_NAME = docker-cli-dev$(IMAGE_TAG)
E2E_IMAGE_NAME = docker-cli-e2e
E2E_ENGINE_VERSION ?=
CACHE_VOLUME_NAME := docker-cli-dev-cache
ifeq ($(DOCKER_CLI_GO_BUILD_CACHE),y)
DOCKER_CLI_MOUNTS += -v "$(CACHE_VOLUME_NAME):/root/.cache/go-build"
endif
VERSION = $(shell cat VERSION)
TIMESTAMP = $(shell date '+%s')
ENVVARS = -e VERSION=$(VERSION) -e GITCOMMIT -e PLATFORM -e TESTFLAGS -e TESTDIRS -e GOOS -e GOARCH -e GOARM -e TEST_ENGINE_VERSION=$(E2E_ENGINE_VERSION)

# Some Dockerfiles use features that are only supported with BuildKit enabled
export DOCKER_BUILDKIT=1

# build docker image (dockerfiles/Dockerfile.build)
.PHONY: build_docker_image
build_docker_image:
	# build dockerfile from stdin so that we don't send the build-context; source is bind-mounted in the development environment
	cat ./dockerfiles/Dockerfile.dev | docker build ${DOCKER_BUILD_ARGS} --build-arg=GO_VERSION -t $(DEV_DOCKER_IMAGE_NAME) -

DOCKER_RUN_NAME_OPTION := $(if $(DOCKER_CLI_CONTAINER_NAME),--name $(DOCKER_CLI_CONTAINER_NAME),)
DOCKER_RUN := docker run --rm $(ENVVARS) $(DOCKER_CLI_MOUNTS) $(DOCKER_RUN_NAME_OPTION)

.PHONY: binary
binary:
	COMPANY_NAME=$(COMPANY_NAME) docker buildx bake binary

build: binary ## alias for binary

plugins: ## build the CLI plugin examples
	docker buildx bake plugins

plugins-cross: ## build the CLI plugin examples for all platforms
	docker buildx bake plugins-cross

.PHONY: clean
clean: build_docker_image ## clean build artifacts
	$(DOCKER_RUN) $(DEV_DOCKER_IMAGE_NAME) make clean
	docker volume rm -f $(CACHE_VOLUME_NAME)

.PHONY: cross
cross:
	COMPANY_NAME=$(COMPANY_NAME) docker buildx bake cross

.PHONY: dynbinary
dynbinary: ## build dynamically linked binary
	USE_GLIBC=1 COMPANY_NAME=$(COMPANY_NAME)  docker buildx bake dynbinary

.PHONY: dev
dev: build_docker_image ## start a build container in interactive mode for in-container development
	$(DOCKER_RUN) -it \
		--mount type=bind,src=/var/run/docker.sock,dst=/var/run/docker.sock \
		$(DEV_DOCKER_IMAGE_NAME)

shell: dev ## alias for dev

.PHONY: lint
lint: ## run linters
	docker buildx bake lint

.PHONY: shellcheck
shellcheck: ## run shellcheck validation
	docker buildx bake shellcheck

.PHONY: fmt
fmt: ## run gofmt
	$(DOCKER_RUN) $(DEV_DOCKER_IMAGE_NAME) make fmt

.PHONY: vendor
vendor: ## update vendor with go modules
	$(eval $@_TMP_OUT := $(shell mktemp -d -t dockercli-output.XXXXXXXXXX))
	docker buildx bake --set "*.output=$($@_TMP_OUT)" update-vendor
	rm -rf ./vendor
	cp -R "$($@_TMP_OUT)"/out/* .
	rm -rf $($@_TMP_OUT)/*

.PHONY: validate-vendor
validate-vendor: ## validate vendor
	docker buildx bake validate-vendor

.PHONY: mod-outdated
mod-outdated: ## check outdated dependencies
	TIMESTAMP=$(TIMESTAMP) docker buildx bake mod-outdated

.PHONY: authors
authors: ## generate AUTHORS file from git history
	$(DOCKER_RUN) -it $(DEV_DOCKER_IMAGE_NAME) make authors

.PHONY: manpages
manpages: build_docker_image ## generate man pages from go source and markdown
	$(DOCKER_RUN) -it $(DEV_DOCKER_IMAGE_NAME) make manpages

.PHONY: yamldocs
yamldocs: build_docker_image ## generate documentation YAML files consumed by docs repo
	$(DOCKER_RUN) -it $(DEV_DOCKER_IMAGE_NAME) make yamldocs

.PHONY: test ## run unit and e2e tests
test: test-unit test-e2e

.PHONY: test-unit
test-unit: ## run unit tests
	docker buildx bake test

.PHONY: test-coverage
test-coverage: ## run test with coverage
	docker buildx bake test-coverage

.PHONY: build-e2e-image
build-e2e-image:
	mkdir -p $(CURDIR)/build/coverage
	IMAGE_NAME=$(E2E_IMAGE_NAME) VERSION=$(VERSION) docker buildx bake e2e-image

.PHONY: test-e2e
test-e2e: test-e2e-non-experimental test-e2e-experimental test-e2e-connhelper-ssh ## run all e2e tests

.PHONY: test-e2e-experimental
test-e2e-experimental: build-e2e-image # run experimental e2e tests
	docker run --rm $(ENVVARS) -e DOCKERD_EXPERIMENTAL=1 -e TEST_ENGINE_VERSION=$(E2E_ENGINE_VERSION) \
		--mount type=bind,src=$(CURDIR)/build/coverage,dst=/tmp/coverage \
		--mount type=bind,src=/var/run/docker.sock,dst=/var/run/docker.sock \
		$(E2E_IMAGE_NAME)

.PHONY: test-e2e-non-experimental
test-e2e-non-experimental: build-e2e-image # run non-experimental e2e tests
	docker run --rm $(ENVVARS) -e TEST_ENGINE_VERSION=$(E2E_ENGINE_VERSION) \
		--mount type=bind,src=$(CURDIR)/build/coverage,dst=/tmp/coverage \
		--mount type=bind,src=/var/run/docker.sock,dst=/var/run/docker.sock \
		$(E2E_IMAGE_NAME)

.PHONY: test-e2e-connhelper-ssh
test-e2e-connhelper-ssh: build-e2e-image # run experimental SSH-connection helper e2e tests
	docker run --rm $(ENVVARS) -e DOCKERD_EXPERIMENTAL=1 -e TEST_ENGINE_VERSION=$(E2E_ENGINE_VERSION) -e TEST_CONNHELPER=ssh \
		--mount type=bind,src=$(CURDIR)/build/coverage,dst=/tmp/coverage \
		--mount type=bind,src=/var/run/docker.sock,dst=/var/run/docker.sock \
		$(E2E_IMAGE_NAME)

.PHONY: help
help: ## print this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {gsub("\\\\n",sprintf("\n%22c",""), $$2);printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
