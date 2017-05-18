#
# github.com/docker/cli
#
all: binary

# remove build artifacts
.PHONY: clean
clean:
	rm -rf ./build/* cli/winresources/rsrc_*

# run go test
# the "-tags daemon" part is temporary
.PHONY: test
test:
	go test -tags daemon -v $(shell go list ./... | grep -v /vendor/)

.PHONY: test-coverage
test-coverage:
	for pkg in $(shell go list ./... | grep -v /vendor/); do \
		go test -tags daemon -v -cover -parallel 8 -coverprofile=profile.out -covermode=atomic $${pkg} || exit 1; \
		if test -f profile.out; then \
			cat profile.out >> coverage.txt && rm profile.out; \
		fi; \
	done

.PHONY: codecov
codecov:
	$(shell curl -s https://codecov.io/bash | bash)

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

cli/compose/schema/bindata.go: cli/compose/schema/data/*.json
	go generate github.com/docker/cli/cli/compose/schema

compose-jsonschema: cli/compose/schema/bindata.go
	scripts/validate/check-git-diff cli/compose/schema/bindata.go
