#
# github.com/docker/cli
#
all: binary

# remove build artifacts
.PHONY: clean
clean:
	rm -rf ./build/* cli/winresources/rsrc_* ./man/man[1-9] docs/yaml/gen

# run go test
# the "-tags daemon" part is temporary
.PHONY: test
test:
	./scripts/test/unit $(shell go list ./... | grep -v '/vendor/')

.PHONY: test-coverage
test-coverage:
	./scripts/test/unit-with-coverage $(shell go list ./... | grep -v '/vendor/')

.PHONY: lint
lint:
	gometalinter --config gometalinter.json ./...

.PHONY: binary
binary:
	@echo "WARNING: binary creates a Linux executable. Use cross for macOS or Windows."
	./scripts/build/binary

.PHONY: cross
cross:
	./scripts/build/cross

.PHONY: dynbinary
dynbinary:
	./scripts/build/dynbinary

.PHONY: watch
watch:
	./scripts/test/watch

# Check vendor matches vendor.conf
vendor: vendor.conf
	vndr 2> /dev/null
	scripts/validate/check-git-diff vendor

## generate man pages from go source and markdown
.PHONY: manpages
manpages:
	scripts/docs/generate-man.sh

## generate documentation YAML files consumed by docs repo
.PHONY: yamldocs
yamldocs:
	scripts/docs/generate-yaml.sh

## Shellcheck validation
.PHONY: shellcheck
shellcheck:
	scripts/validate/shellcheck

cli/compose/schema/bindata.go: cli/compose/schema/data/*.json
	go generate github.com/docker/cli/cli/compose/schema

compose-jsonschema: cli/compose/schema/bindata.go
	scripts/validate/check-git-diff cli/compose/schema/bindata.go
