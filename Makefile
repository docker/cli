#
# github.com/docker/cli
#

# build the CLI
.PHONY: build
build: clean
	@./scripts/build/binary

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
	@./scripts/build/cross

# download dependencies (vendor/) listed in vendor.conf
.PHONY: vendor
vendor: vendor.conf
	@vndr 2> /dev/null
	@if [ "`git status --porcelain -- vendor 2>/dev/null`" ]; then \
		echo; echo "vendoring is wrong. These files were changed:"; \
		echo; git status --porcelain -- vendor 2>/dev/null; \
		echo; exit 1; \
	fi;
