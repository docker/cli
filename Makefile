#
# github.com/docker/cli
#

.PHONY: build clean test lint cross

# build the CLI
build: clean
	@./scripts/build/binary

# remove build artifacts
clean:
	@rm -rf ./build/*

# run go test
# the "-tags daemon" part is temporary
test:
	@go test -tags daemon -v $(shell go list ./... | grep -v /vendor/)

# run linters
lint:
	@gometalinter --config gometalinter.json ./...

# build the CLI for multiple architectures
cross: clean
	@./scripts/build/cross

vendor: vendor.conf
	@vndr 2> /dev/null
	@if [ "`git status --porcelain -- vendor 2>/dev/null`" ]; then \
		echo; echo "vendoring is wrong. These files were changed:"; \
		echo; git status --porcelain -- vendor 2>/dev/null; \
		echo; exit 1; \
	fi;
