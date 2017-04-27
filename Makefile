#
# github.com/docker/cli 
#

.PHONY: build clean cross

# build the CLI
build: clean
	@go build -o ./build/docker github.com/docker/cli/cmd/docker

# remove build artifacts
clean:
	@rm -rf ./build

# build the CLI for multiple architectures
cross: clean
	@gox -output build/docker-{{.OS}}-{{.Arch}} \
		 -osarch="linux/arm linux/amd64 darwin/amd64 windows/amd64" \
		 github.com/docker/cli/cmd/docker
