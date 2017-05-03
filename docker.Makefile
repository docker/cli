#
# github.com/docker/cli
#
# Makefile for developing using Docker
#

+.PHONY: build_docker_image build_linter_image build clean test cross dev lint

DEV_DOCKER_IMAGE_NAME = docker-cli-dev
LINTER_IMAGE_NAME = docker-cli-lint
MOUNTS = -v `pwd`:/go/src/github.com/docker/cli

# build docker image (dockerfiles/Dockerfile.build)
build_docker_image:
	@docker build -q -t $(DEV_DOCKER_IMAGE_NAME) -f ./dockerfiles/Dockerfile.build .

.PHONY: builder_linter_image
build_linter_image:
	@docker build -q -t $(LINTER_IMAGE_NAME) -f ./dockerfiles/Dockerfile.lint .

# build executable using a container
build: build_docker_image
	@echo "WARNING: this will drop a Linux executable on your host (not a macOS of Windows one)"
	@docker run --rm $(MOUNTS) $(DEV_DOCKER_IMAGE_NAME) make build

# clean build artifacts using a container
clean: build_docker_image
	@docker run --rm $(MOUNTS) $(DEV_DOCKER_IMAGE_NAME) make clean

# run go test
test: build_docker_image
	@docker run --rm $(MOUNTS) $(DEV_DOCKER_IMAGE_NAME) make test

# build the CLI for multiple architectures using a container
cross: build_docker_image
	@docker run --rm $(MOUNTS) $(DEV_DOCKER_IMAGE_NAME) make cross

# start container in interactive mode for in-container development
dev: build_docker_image
	@docker run -ti $(MOUNTS) -v /var/run/docker.sock:/var/run/docker.sock $(DEV_DOCKER_IMAGE_NAME) ash

# run linters in a container
lint: build_linter_image
	@docker run -ti $(MOUNTS) $(LINTER_IMAGE_NAME)
