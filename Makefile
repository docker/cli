#
# github.com/docker/cli 
#

.PHONY: build clean cross

# build the CLI
build: clean
	@go build -o ./build/docker ./cmd/docker

# remove build artifacts
clean:
	@rm -rf ./build

# run go test
# the "-tags daemon" part is temporary
test:
	@go test -tags daemon -v $(shell go list ./... | grep -v /vendor/)

# build the CLI for multiple architectures
cross: clean
	@gox -output build/docker-{{.OS}}-{{.Arch}} \
		 -osarch="linux/arm linux/amd64 darwin/amd64 windows/amd64" \
		 ./cmd/docker
