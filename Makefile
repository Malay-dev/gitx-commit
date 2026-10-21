BINARY := git-xcommit

.PHONY: build clean fmt vet test cover cover-html cover-func all

## Build the binary
build:
	go build -o $(BINARY) .

## Remove build artifacts
clean:
	go clean
	rm -f $(BINARY) coverage.out

## Format all Go files
fmt:
	gofmt -w .

## Run linter
vet:
	go vet ./...

## Run tests
test:
	go test ./...

## Run tests with coverage summary
cover:
	go test ./... -cover

## Generate detailed per-function coverage report
cover-func:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

## Open coverage report in browser (color-coded)
cover-html:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

## Format + lint + test + build
all: fmt vet test build

## Scan dependencies for known vulnerabilities
vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

## Verify dependency checksums
verify:
	go mod verify

## Full security check
security: verify vuln
