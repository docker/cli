#
# github.com/docker/cli
#

all: binary

# remove build artifacts
.PHONY: clean
clean:
	@rm -rf ./build/*

# run go test
# the "-tags daemon" part is temporary
.PHONY: test
test:
	@go test -tags daemon -v $(shell go list ./... | grep -v /vendor/)

.PHONY: lint
lint:
	@gometalinter --config gometalinter.json ./...


.PHONY: binary
binary:
	@./scripts/build/binary

# build the CLI for multiple architectures
.PHONY: cross
cross:
	@./scripts/build/cross

.PHONY: dynbinary
dynbinary:
	@./scripts/build/dynbinary

# download dependencies (vendor/) listed in vendor.conf
.PHONY: vendor
vendor: vendor.conf
	@vndr 2> /dev/null
	@scripts/validate/check-git-diff vendor

cli/compose/schema/bindata.go: cli/compose/schema/data/*.json
	go generate github.com/docker/cli/cli/compose/schema

compose-jsonschema: cli/compose/schema/bindata.go
	@scripts/validate/check-git-diff cli/compose/schema/bindata.go
