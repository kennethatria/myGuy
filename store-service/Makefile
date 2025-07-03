# Store Service Makefile

.PHONY: test test-unit test-integration test-coverage clean build run

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=store-service
BINARY_UNIX=$(BINARY_NAME)_unix

# Test parameters
TEST_PATTERN=./...
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

all: test build

# Build the application
build:
	$(GOBUILD) -o $(BINARY_NAME) -v ./cmd/api/

# Build for unix
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v ./cmd/api/

# Run the application
run:
	$(GOBUILD) -o $(BINARY_NAME) -v ./cmd/api/
	./$(BINARY_NAME)

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -f $(COVERAGE_FILE)
	rm -f $(COVERAGE_HTML)

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Run all tests
test: test-unit test-integration

# Run unit tests only
test-unit:
	@echo "Running unit tests..."
	$(GOTEST) -v -race -short $(TEST_PATTERN)

# Run integration tests only
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -v -race -run Integration $(TEST_PATTERN)

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic $(TEST_PATTERN)
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	$(GOCMD) tool cover -func=$(COVERAGE_FILE)

# Run tests with coverage and open HTML report
test-coverage-html: test-coverage
	@echo "Opening coverage report in browser..."
	open $(COVERAGE_HTML) || xdg-open $(COVERAGE_HTML) || echo "Please open $(COVERAGE_HTML) manually"

# Run tests and check coverage percentage
test-coverage-check:
	@echo "Running tests and checking coverage..."
	$(GOTEST) -v -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic $(TEST_PATTERN)
	@total=$$($(GOCMD) tool cover -func=$(COVERAGE_FILE) | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo "Total coverage: $$total%"; \
	if [ $$(echo "$$total >= 70" | bc -l) -eq 1 ]; then \
		echo "✅ Coverage is above 70% ($$total%)"; \
	else \
		echo "❌ Coverage is below 70% ($$total%)"; \
		exit 1; \
	fi

# Run tests in watch mode (requires entr: brew install entr)
test-watch:
	find . -name "*.go" | entr -c make test-unit

# Run specific test
test-specific:
	@read -p "Enter test pattern (e.g., TestCreateItem): " pattern; \
	$(GOTEST) -v -race -run $$pattern $(TEST_PATTERN)

# Run benchmarks
bench:
	$(GOTEST) -bench=. -benchmem $(TEST_PATTERN)

# Lint the code (requires golangci-lint)
lint:
	golangci-lint run

# Format the code
fmt:
	$(GOCMD) fmt $(TEST_PATTERN)

# Vet the code
vet:
	$(GOCMD) vet $(TEST_PATTERN)

# Security scan (requires gosec)
security:
	gosec $(TEST_PATTERN)

# Run all checks
check: fmt vet lint security test-coverage-check

# Docker commands
docker-build:
	docker build -t $(BINARY_NAME) .

docker-run:
	docker run -p 8081:8081 $(BINARY_NAME)

# Development setup
dev-setup:
	$(GOMOD) download
	$(GOGET) github.com/cosmtrek/air@latest

# Run in development mode with hot reload (requires air)
dev:
	air

# Database commands
db-migrate:
	$(GOCMD) run migrations/*.go

# Help
help:
	@echo "Available commands:"
	@echo "  build              Build the application"
	@echo "  run                Build and run the application"
	@echo "  test               Run all tests"
	@echo "  test-unit          Run unit tests only"
	@echo "  test-integration   Run integration tests only"
	@echo "  test-coverage      Run tests with coverage report"
	@echo "  test-coverage-html Run tests with coverage and open HTML report"
	@echo "  test-coverage-check Run tests and check if coverage is above 70%"
	@echo "  test-watch         Run tests in watch mode"
	@echo "  test-specific      Run specific test pattern"
	@echo "  bench              Run benchmarks"
	@echo "  clean              Clean build artifacts"
	@echo "  deps               Download dependencies"
	@echo "  fmt                Format code"
	@echo "  vet                Vet code"
	@echo "  lint               Lint code"
	@echo "  security           Run security scan"
	@echo "  check              Run all checks (fmt, vet, lint, security, coverage)"
	@echo "  docker-build       Build Docker image"
	@echo "  docker-run         Run Docker container"
	@echo "  dev-setup          Setup development environment"
	@echo "  dev                Run in development mode with hot reload"
	@echo "  help               Show this help message"