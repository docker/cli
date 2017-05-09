#
# github.com/docker/cli 
#

# build the CLI
.PHONY: build
build: clean
	@go build -o ./build/docker github.com/docker/cli/cmd/docker

# remove build artifacts
.PHONY: clean
clean:
	@rm -rf ./build/*

# run go test
# the "-tags daemon" part is temporary
.PHONY: test
test:
	@go test -tags daemon -v $(shell go list ./... | grep -v /vendor/)

# run linters
.PHONY: lint
lint:
	@gometalinter --config gometalinter.json ./...

# build the CLI for multiple architectures
.PHONY: cross
cross: clean
	@gox -output build/docker-{{.OS}}-{{.Arch}} \
		 -osarch="linux/arm linux/amd64 darwin/amd64 windows/amd64" \
		 github.com/docker/cli/cmd/docker

# download dependencies (vendor/) listed in vendor.conf
.PHONY: vendor
vendor: vendor.conf
	@vndr 2> /dev/null
	@if [ "`git status --porcelain -- vendor 2>/dev/null`" ]; then \
		echo; echo "vendoring is wrong. These files were changed:"; \
		echo; git status --porcelain -- vendor 2>/dev/null; \
		echo; exit 1; \
	fi;
