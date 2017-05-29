PACKAGES:=$(shell go list ./... | grep -v /vendor/)

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
	go test -tags daemon -v ${PACKAGES}

.PHONY: test-coverage
test-coverage:
	( for pkg in ${PACKAGES}; do \
		go test -tags daemon \
			-cover \
			-coverprofile=profile.out \
			-covermode=atomic \
			$$pkg || exit;\
		\
		if test -f profile.out; then \
			cat profile.out >> coverage.txt; \
			rm profile.out; \
		fi; \
	done )

.PHONY: lint
lint:
	gometalinter --config gometalinter.json ./...

.PHONY: binary
binary:
	@echo "WARNING: binary creates a Linux executable. Use cross for macOS or Windows."
	# TODO(stevvooe): Remove the use of this script. It does nothing. It is
	# more complicated because the variables need to be converted back from
	# shell.
	./scripts/build/binary

.PHONY: cross
cross:
	# TODO(stevvooe): Fix the cross build so this isn't a separate target.
	# Should simply be able to do `make GOOS=windows` to get a cross-compiled
	# binary.
	./scripts/build/cross

.PHONY: dynbinary
dynbinary:
	# TODO(stevvooe): Change this to target binary but enable cgo, as that is
	# the only difference.
	./scripts/build/dynbinary

# Check vendor matches vendor.conf
vendor: vendor.conf
	vndr 2> /dev/null
	if [ "`git status --porcelain -- vendor/ 2>/dev/null`" ]; then \
		echo; \
		echo "These files were changed:"; \
		echo; \
		git status --porcelain -- vendor/ 2>/dev/null; \
		echo; \
		exit 1; \
	else \
		echo "vendor/ is correct"; \
	fi;

cli/compose/schema/bindata.go: cli/compose/schema/data/*.json
	go generate github.com/docker/cli/cli/compose/schema

compose-jsonschema: cli/compose/schema/bindata.go
	scripts/validate/check-git-diff cli/compose/schema/bindata.go
