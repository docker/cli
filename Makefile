#
# github.com/docker/cli 
#

.PHONY: build clean test lint cross

# build the CLI
build: clean
	@go build -o ./build/docker github.com/docker/cli/cmd/docker

# remove build artifacts
clean:
	@rm -rf ./build

# run go test
# the "-tags daemon" part is temporary
test:
	@go test -tags daemon -v $(shell go list ./... | grep -v /vendor/)

# run linters
lint:
	@gometalinter --config gometalinter.json ./...

# build the CLI for multiple architectures
cross: clean
	@gox -output build/docker-{{.OS}}-{{.Arch}} \
		 -osarch="linux/arm linux/amd64 darwin/amd64 windows/amd64" \
		 github.com/docker/cli/cmd/docker

vendor: vendor.conf
	@vndr 2> /dev/null
	@if [ "`git status --porcelain -- vendor 2>/dev/nul`" ]; then \
		echo; echo "vendoring is wrong. These files were changed:"; \
		echo; git status --porcelain -- vendor 2>/dev/nul; \
		echo; exit 1; \
	fi;

ci: cross test
