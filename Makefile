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
	@rm -rf ./man/man[1-9]

# run go test
# the "-tags daemon" part is temporary
.PHONY: test
test:
	@go test -tags daemon -v $(shell go list ./... | grep -vE '/vendor/|github.com/docker/cli/man$$')

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
	@scripts/validate/check-git-diff vendor

## Generate man pages from go source and markdown
.PHONY: manpages
manpages:
	@man/generate.sh

cli/compose/schema/bindata.go: cli/compose/schema/data/*.json
	go generate github.com/docker/cli/cli/compose/schema

compose-jsonschema: cli/compose/schema/bindata.go
	@scripts/validate/check-git-diff cli/compose/schema/bindata.go
