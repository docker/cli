#
# github.com/docker/cli
#
# Makefile for developing using Docker
#

DEV_DOCKER_IMAGE_NAME = docker-cli-dev
LINTER_IMAGE_NAME = docker-cli-lint
MOUNTS = -v `pwd`:/go/src/github.com/docker/cli

# build docker image (dockerfiles/Dockerfile.build)
.PHONY: build_docker_image
build_docker_image:
	@docker build -q -t $(DEV_DOCKER_IMAGE_NAME) -f ./dockerfiles/Dockerfile.build .

# build docker image having the linting tools (dockerfiles/Dockerfile.lint)
.PHONY: build_linter_image
build_linter_image:
	@docker build -q -t $(LINTER_IMAGE_NAME) -f ./dockerfiles/Dockerfile.lint .

# build executable using a container
.PHONY: build
build: build_docker_image
	@echo "WARNING: this will drop a Linux executable on your host (not a macOS of Windows one)"
	@docker run --rm $(MOUNTS) $(DEV_DOCKER_IMAGE_NAME) make build

# clean build artifacts using a container
.PHONY: clean
clean: build_docker_image
	@docker run --rm $(MOUNTS) $(DEV_DOCKER_IMAGE_NAME) make clean

# run go test
.PHONY: test
test: build_docker_image
	@docker run --rm $(MOUNTS) $(DEV_DOCKER_IMAGE_NAME) make test

# build the CLI for multiple architectures using a container
.PHONY: cross
cross: build_docker_image
	@docker run --rm $(MOUNTS) $(DEV_DOCKER_IMAGE_NAME) make cross

# start container in interactive mode for in-container development
.PHONY: dev
dev: build_docker_image
	@docker run -ti $(MOUNTS) -v /var/run/docker.sock:/var/run/docker.sock $(DEV_DOCKER_IMAGE_NAME) ash

# run linters in a container
.PHONY: lint
lint: build_linter_image
	@docker run -ti $(MOUNTS) $(LINTER_IMAGE_NAME)

# download dependencies (vendor/) listed in vendor.conf, using a container
.PHONY: vendor
vendor: build_docker_image vendor.conf
	@docker run -ti --rm $(MOUNTS) $(DEV_DOCKER_IMAGE_NAME) make vendor
