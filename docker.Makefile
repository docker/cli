#
# github.com/docker/cli
#
# Makefile for developing using Docker
#

+.PHONY: build_docker_image build clean cross dev

DEV_DOCKER_IMAGE_NAME = docker_cli_dev

# build docker image (dockerfiles/Dockerfile.build)
build_docker_image:
	@docker build -t $(DEV_DOCKER_IMAGE_NAME) -f ./dockerfiles/Dockerfile.build . > /dev/null

# build executable using a container
build: build_docker_image
	@echo "WARNING: this will drop a Linux executable on your host (not a macOS of Windows one)"
	@docker run --rm -v `pwd`:/go/src/github.com/docker/cli $(DEV_DOCKER_IMAGE_NAME) make build

# clean build artifacts using a container
clean: build_docker_image
	@docker run --rm -v `pwd`:/go/src/github.com/docker/cli $(DEV_DOCKER_IMAGE_NAME) make clean

# build the CLI for multiple architectures using a container
cross: build_docker_image
	@docker run --rm -v `pwd`:/go/src/github.com/docker/cli $(DEV_DOCKER_IMAGE_NAME) make cross

# start container in interactive mode for in-container development
dev: build_docker_image
	@docker run -ti -v `pwd`:/go/src/github.com/docker/cli $(DEV_DOCKER_IMAGE_NAME) ash