default: testacc

# Format all Go source files
.PHONY: fmt
fmt:
	find . -name "*.go" -exec gofmt -s -w {} \;

# Install golangci-lint v2 if missing, then lint
.PHONY: lint
lint:
	@which golangci-lint > /dev/null 2>&1 || \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	golangci-lint run --timeout 10m ./...

# Run unit tests
.PHONE: test
test:
	go test ./...

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m
