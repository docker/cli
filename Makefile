#
# github.com/docker/cli 
#

.PHONY: build clean test lint cross

# build the CLI
build: clean
	@go build -o ./build/docker github.com/docker/cli/cmd/docker

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
	@gox -output build/docker-{{.OS}}-{{.Arch}} \
		 -osarch="linux/arm linux/amd64 darwin/amd64 windows/amd64" \
		 github.com/docker/cli/cmd/docker

vendor: vendor.conf
	@vndr 2> /dev/null
	@script/validate/check-git-diff vendor

cli/compose/schema/bindata.go: cli/compose/schema/data/*.json
	go generate github.com/docker/cli/cli/compose/schema

compose-jsonschema: cli/compose/schema/bindata.go
	@script/validate/check-git-diff cli/compose/schema/bindata.go
