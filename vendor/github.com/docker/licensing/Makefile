LINT_EXCLUDES := vendor|mocks

# Help
.PHONY: default
default:
	@echo "Please specify a build target. The choices are:"
	@echo "		test: run go test
	@echo "		fmt: run go fmt
	@echo "		vet: run go vet
	@echo "		lint: run golint
	@echo "		check: run go fmt, go vet, golint
	@echo "		mocks: generate mocks with mockery"
	@false

.PHONY: check
check: fmt vet lint

.PHONY: test
test:
	@echo "============= Go test ==============="
	go test --cover -v ./...

.PHONY: fmt
fmt:
	@echo "============= Go fmt ==============="
	@test -z "$$(gofmt -s -l . 2>&1 | grep -vE "$(LINT_EXCLUDES)" | tee /dev/stderr)"

.PHONY: vet
vet:
	@echo "============= Go vet ==============="
	@test -z "$$(go list ./... | xargs go vet | tee /dev/stderr)"

.PHONY: lint
lint:
	@echo "============= Go lint ==============="
	@go get golang.org/x/lint/golint
	@test -z "$$(go list ./... | xargs -L1 "$(GOPATH)"/bin/golint | tee /dev/stderr)"

.PHONY: mocks
mocks:
	@echo "=== Generating mocks ===="
	@go get github.com/vektra/mockery/...
	"$(GOPATH)"/bin/mockery -name=Client

