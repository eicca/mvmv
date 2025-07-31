.PHONY: build test clean install fmt vet

# Build binary
build:
	go build -o mvmv .

# Run tests
test:
	go test -v ./...

# Run tests with race detector
test-race:
	go test -race -v ./...

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Install binary to GOPATH/bin
install:
	go install .

# Clean build artifacts
clean:
	rm -f mvmv
	rm -rf test_data
	rm -rf build

# Run all checks
check: fmt vet test

# Build for multiple platforms
build-all:
	mkdir -p build
	GOOS=linux GOARCH=amd64 go build -o build/mvmv-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -o build/mvmv-linux-arm64 .
	GOOS=darwin GOARCH=amd64 go build -o build/mvmv-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -o build/mvmv-darwin-arm64 .

.DEFAULT_GOAL := build