# Onyx Framework Makefile
# Go framework development and build automation

# Variables
GO = go
GOCMD = $(GO)
GOBUILD = $(GOCMD) build
GOTEST = $(GOCMD) test
GOMOD = $(GOCMD) mod
GOCLEAN = $(GOCMD) clean
GOGET = $(GOCMD) get
GOINSTALL = $(GOCMD) install
GOFMT = $(GOCMD) fmt
GOVET = $(GOCMD) vet

# Project configuration
BINARY_NAME = onyx
BINARY_PATH = ./cmd/onyx
MAIN_PATH = ./cmd/onyx/main.go
BUILD_DIR = ./build
COVERAGE_FILE = coverage.out
COVERAGE_HTML = coverage.html

# Build flags
BUILD_FLAGS = -ldflags="-s -w"
TEST_FLAGS = -v -race -coverprofile=$(COVERAGE_FILE)
BENCH_FLAGS = -bench=. -benchmem

# Default target
.DEFAULT_GOAL := help

# Help target
.PHONY: help
help: ## Display this help message
	@echo "Onyx Framework - Available Commands:"
	@echo ""
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""

# Development commands
.PHONY: dev
dev: ## Start development server with auto-reload
	@echo "Starting Onyx development server..."
	@$(GOBUILD) -o $(BINARY_NAME) $(MAIN_PATH)
	@./$(BINARY_NAME) serve

.PHONY: serve
serve: build ## Build and start the server
	@echo "Starting Onyx server..."
	@./$(BINARY_NAME) serve

# Build commands
.PHONY: build
build: ## Build the Onyx CLI binary
	@echo "Building Onyx CLI..."
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@cp $(BUILD_DIR)/$(BINARY_NAME) ./$(BINARY_NAME)
	@echo "Build complete: $(BINARY_NAME)"

.PHONY: build-all
build-all: ## Build for all platforms
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	
	# Linux AMD64
	@GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	
	# Linux ARM64
	@GOOS=linux GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	
	# macOS AMD64
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	
	# macOS ARM64 (Apple Silicon)
	@GOOS=darwin GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	
	# Windows AMD64
	@GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	
	@echo "Cross-compilation complete. Binaries in $(BUILD_DIR)/"

.PHONY: install
install: ## Install the Onyx CLI globally
	@echo "Installing Onyx CLI..."
	@$(GOINSTALL) $(MAIN_PATH)
	@echo "Onyx CLI installed successfully"

# Testing commands
.PHONY: test
test: ## Run all tests
	@echo "Running tests..."
	@$(GOTEST) $(TEST_FLAGS) ./...
	@echo "All tests completed"

.PHONY: test-unit
test-unit: ## Run unit tests only
	@echo "Running unit tests..."
	@$(GOTEST) -v -short ./...

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@$(GOTEST) -v -run Integration ./...

.PHONY: test-coverage
test-coverage: test ## Generate and display test coverage
	@echo "Generating coverage report..."
	@$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@$(GOCMD) tool cover -func=$(COVERAGE_FILE)
	@echo "Coverage report generated: $(COVERAGE_HTML)"

.PHONY: test-watch
test-watch: ## Watch for changes and run tests automatically
	@echo "Watching for changes..."
	@which fswatch > /dev/null || (echo "fswatch not found. Install with: brew install fswatch" && exit 1)
	@fswatch -o . | xargs -n1 -I{} make test-unit

.PHONY: bench
bench: ## Run benchmarks
	@echo "Running benchmarks..."
	@$(GOTEST) $(BENCH_FLAGS) ./...

# Code quality commands
.PHONY: fmt
fmt: ## Format Go code
	@echo "Formatting code..."
	@$(GOFMT) ./...
	@echo "Code formatting complete"

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	@$(GOVET) ./...
	@echo "Vet check complete"

.PHONY: lint
lint: ## Run golangci-lint
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install from https://golangci-lint.run/usage/install/" && exit 1)
	@golangci-lint run ./...
	@echo "Linting complete"

.PHONY: staticcheck
staticcheck: ## Run staticcheck
	@echo "Running staticcheck..."
	@which staticcheck > /dev/null || (echo "staticcheck not found. Install with: go install honnef.co/go/tools/cmd/staticcheck@latest" && exit 1)
	@staticcheck ./...
	@echo "Static analysis complete"

.PHONY: check
check: fmt vet lint test ## Run all code quality checks

# Dependency management
.PHONY: deps
deps: ## Download and verify dependencies
	@echo "Downloading dependencies..."
	@$(GOMOD) download
	@$(GOMOD) verify
	@echo "Dependencies updated"

.PHONY: deps-update
deps-update: ## Update all dependencies
	@echo "Updating dependencies..."
	@$(GOMOD) tidy
	@$(GOGET) -u ./...
	@$(GOMOD) tidy
	@echo "Dependencies updated"

.PHONY: deps-vendor
deps-vendor: ## Create vendor directory
	@echo "Creating vendor directory..."
	@$(GOMOD) vendor
	@echo "Vendor directory created"

# Database commands
.PHONY: migrate
migrate: build ## Run database migrations
	@echo "Running migrations..."
	@./$(BINARY_NAME) migrate

.PHONY: migrate-rollback
migrate-rollback: build ## Rollback last migration
	@echo "Rolling back migration..."
	@./$(BINARY_NAME) migrate:rollback

.PHONY: migrate-status
migrate-status: build ## Show migration status
	@echo "Migration status:"
	@./$(BINARY_NAME) migrate:status

.PHONY: seed
seed: build ## Run database seeds
	@echo "Running database seeds..."
	@./$(BINARY_NAME) db:seed

# Documentation commands
.PHONY: docs
docs: build ## Generate API documentation
	@echo "Generating API documentation..."
	@./$(BINARY_NAME) docs:generate
	@echo "Documentation generated"

.PHONY: docs-serve
docs-serve: docs ## Serve documentation locally
	@echo "Starting documentation server..."
	@./$(BINARY_NAME) docs:serve

# Code generation commands
.PHONY: generate
generate: ## Run go generate
	@echo "Running code generation..."
	@$(GOCMD) generate ./...
	@echo "Code generation complete"

.PHONY: make-controller
make-controller: build ## Generate a new controller (usage: make make-controller NAME=UserController)
	@test -n "$(NAME)" || (echo "Error: NAME is required. Usage: make make-controller NAME=UserController" && exit 1)
	@./$(BINARY_NAME) make:controller $(NAME)

.PHONY: make-model
make-model: build ## Generate a new model (usage: make make-model NAME=User)
	@test -n "$(NAME)" || (echo "Error: NAME is required. Usage: make make-model NAME=User" && exit 1)
	@./$(BINARY_NAME) make:model $(NAME)

.PHONY: make-middleware
make-middleware: build ## Generate a new middleware (usage: make make-middleware NAME=AuthMiddleware)
	@test -n "$(NAME)" || (echo "Error: NAME is required. Usage: make make-middleware NAME=AuthMiddleware" && exit 1)
	@./$(BINARY_NAME) make:middleware $(NAME)

.PHONY: make-migration
make-migration: build ## Generate a new migration (usage: make make-migration NAME=create_users_table)
	@test -n "$(NAME)" || (echo "Error: NAME is required. Usage: make make-migration NAME=create_users_table" && exit 1)
	@./$(BINARY_NAME) make:migration $(NAME)

# Development tools
.PHONY: routes
routes: build ## List all registered routes
	@echo "Application routes:"
	@./$(BINARY_NAME) route:list

.PHONY: cache-clear
cache-clear: build ## Clear application cache
	@echo "Clearing cache..."
	@./$(BINARY_NAME) cache:clear
	@echo "Cache cleared"

# Cleanup commands
.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@$(GOCLEAN)
	@rm -f $(BINARY_NAME)
	@rm -rf $(BUILD_DIR)
	@rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)
	@echo "Clean complete"

.PHONY: clean-deps
clean-deps: ## Clean module cache
	@echo "Cleaning module cache..."
	@$(GOCMD) clean -modcache
	@echo "Module cache cleaned"

.PHONY: clean-all
clean-all: clean clean-deps ## Clean everything

# Docker commands (if Dockerfile exists)
.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t onyx-framework .
	@echo "Docker image built: onyx-framework"

.PHONY: docker-run
docker-run: ## Run Docker container
	@echo "Running Docker container..."
	@docker run -p 8080:8080 onyx-framework

# Release commands
.PHONY: release-check
release-check: check test ## Run all checks before release
	@echo "Release checks passed ✅"

.PHONY: version
version: build ## Show version information
	@./$(BINARY_NAME) --version

# Security scanning
.PHONY: security-scan
security-scan: ## Run security scan with gosec
	@echo "Running security scan..."
	@which gosec > /dev/null || (echo "gosec not found. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest" && exit 1)
	@gosec ./...
	@echo "Security scan complete"

# Performance profiling
.PHONY: profile-cpu
profile-cpu: build ## Run CPU profiling
	@echo "Running CPU profile..."
	@$(GOTEST) -cpuprofile=cpu.prof -bench=. ./...
	@echo "CPU profile saved to cpu.prof"

.PHONY: profile-mem
profile-mem: build ## Run memory profiling  
	@echo "Running memory profile..."
	@$(GOTEST) -memprofile=mem.prof -bench=. ./...
	@echo "Memory profile saved to mem.prof"

# Quick development shortcuts
.PHONY: quick-test
quick-test: ## Quick test (short tests only)
	@$(GOTEST) -short ./...

.PHONY: quick-check
quick-check: fmt vet quick-test ## Quick quality check

# All-in-one commands
.PHONY: setup
setup: deps build ## Setup development environment
	@echo "Development environment setup complete ✅"

.PHONY: ci
ci: deps check test build ## Run CI pipeline locally
	@echo "CI pipeline completed successfully ✅"