.PHONY: build test test-race vet staticcheck run clean cross-windows cross-linux cross-darwin deps

# Default target
all: build

# Build the project
build:
	go build -o bin/termvu ./cmd/termvu

# Run tests
test:
	go test ./...

# Run tests with race detector
test-race:
	go test -race ./...

# Run go vet
vet:
	go vet ./...

# Run staticcheck
staticcheck:
	staticcheck ./...

# Run the application
run: build
	./bin/termvu

# Cross-compile for Windows
cross-windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 go build -o bin/termvu.exe ./cmd/termvu

# Cross-compile for Linux
cross-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o bin/termvu-linux ./cmd/termvu

# Cross-compile for macOS
cross-darwin:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -o bin/termvu-darwin ./cmd/termvu

# Cross-compile all platforms
cross-all: cross-windows cross-linux cross-darwin

# Download dependencies
deps:
	go mod download
	go mod tidy

# Clean build artifacts
clean:
	rm -rf bin/

# Install staticcheck if not present
install-staticcheck:
	go install honnef.co/go/tools/cmd/staticcheck@latest

# Run all checks (CI pipeline)
ci: deps vet staticcheck test-race cross-windows
	@echo "CI checks passed"

# Development workflow: build and run
dev: build run

# Format code
fmt:
	go fmt ./...