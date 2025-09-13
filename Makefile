# Multi-Tenant Ingestion Pipeline - Test Makefile

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Test parameters
TEST_TIMEOUT=10m
TEST_COVERAGE_FILE=coverage.out
TEST_COVERAGE_HTML=coverage.html
TEST_PARALLEL=8

# Spanner emulator settings
SPANNER_EMULATOR_HOST=localhost:9010
SPANNER_PROJECT_ID=test-project
SPANNER_INSTANCE_ID=test-instance
SPANNER_DATABASE_ID=test-database

# Test categories
UNIT_TESTS=./test/unit/...
INTEGRATION_TESTS=./test/integration/...
E2E_TESTS=./test/e2e/...
LOAD_TESTS=./test/load/...
PERFORMANCE_TESTS=./test/performance/...
SECURITY_TESTS=./test/security/...
COMPREHENSIVE_TEST=./test/comprehensive_test_runner.go

# Build targets
.PHONY: all build clean test test-unit test-integration test-e2e test-load test-performance test-security
.PHONY: test-comprehensive test-all test-quick test-ci
.PHONY: coverage coverage-html test-report
.PHONY: start-emulators stop-emulators setup-test-env cleanup-test-env
.PHONY: lint vet format benchmark
.PHONY: help

## Build targets

all: build test ## Build and test everything

build: ## Build the application
	$(GOBUILD) -v ./...

clean: ## Clean build artifacts and test outputs
	$(GOCLEAN)
	rm -f $(TEST_COVERAGE_FILE) $(TEST_COVERAGE_HTML)
	rm -rf ./test-reports/

## Test targets

test: test-unit test-integration ## Run unit and integration tests

test-all: test-unit test-integration test-e2e test-load test-performance test-security ## Run all test suites

test-comprehensive: setup-test-env ## Run comprehensive test orchestrator with quality gates
	@echo "Running comprehensive test orchestrator..."
	@echo "This will execute all test suites and generate quality gates report..."
	SPANNER_EMULATOR_HOST=$(SPANNER_EMULATOR_HOST) \
	$(GOTEST) -v -timeout 25m -run "TestComprehensiveTestSuite" ./test/

test-quick: ## Run quick validation tests (unit + critical integration)
	@echo "Running quick validation tests..."
	$(GOTEST) -short -v $(UNIT_TESTS)
	SPANNER_EMULATOR_HOST=$(SPANNER_EMULATOR_HOST) \
	$(GOTEST) -short -v -run ".*Webhook.*|.*Critical.*" $(INTEGRATION_TESTS)

test-ci: setup-test-env ## Run tests optimized for CI environment
	@echo "Running CI test suite..."
	SPANNER_EMULATOR_HOST=$(SPANNER_EMULATOR_HOST) \
	CI=true $(GOTEST) -v -timeout 15m -parallel 4 \
	-coverprofile=$(TEST_COVERAGE_FILE) -covermode=atomic \
	./test/unit/... ./test/integration/... ./test/security/...

test-callrail: ## Run CallRail-specific tests
	@echo "Running CallRail integration tests..."
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) -run ".*CallRail.*" ./test/unit/ ./test/integration/ ./test/load/ ./test/security/

test-unit: ## Run unit tests
	@echo "Running unit tests..."
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) -parallel $(TEST_PARALLEL) $(UNIT_TESTS)

test-integration: start-emulators ## Run integration tests (requires emulators)
	@echo "Running integration tests..."
	SPANNER_EMULATOR_HOST=$(SPANNER_EMULATOR_HOST) \
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) -parallel 4 $(INTEGRATION_TESTS)

test-e2e: start-emulators ## Run end-to-end tests (requires emulators)
	@echo "Running end-to-end tests..."
	SPANNER_EMULATOR_HOST=$(SPANNER_EMULATOR_HOST) \
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) -parallel 2 $(E2E_TESTS)

test-load: ## Run load tests
	@echo "Running load tests..."
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) -parallel 1 $(LOAD_TESTS)

test-performance: setup-test-env ## Run comprehensive performance tests
	@echo "Running performance tests..."
	@echo "Testing webhook latency, audio processing, AI analysis, and load handling..."
	SPANNER_EMULATOR_HOST=$(SPANNER_EMULATOR_HOST) \
	$(GOTEST) -v -timeout 15m -parallel 2 $(PERFORMANCE_TESTS)

test-security: start-emulators ## Run security tests
	@echo "Running security tests..."
	@echo "Testing HMAC verification, tenant isolation, and data protection..."
	SPANNER_EMULATOR_HOST=$(SPANNER_EMULATOR_HOST) \
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) -parallel 4 $(SECURITY_TESTS)

# Load test variations
load-test-light: ## Run light load tests (quick validation)
	@echo "Running light load tests..."
	$(GOTEST) -v -timeout 5m -run ".*Light.*" $(PERFORMANCE_TESTS)

load-test-stress: ## Run stress load tests (heavy load)
	@echo "Running stress load tests..."
	$(GOTEST) -v -timeout 20m -run ".*Stress.*|.*Heavy.*" $(PERFORMANCE_TESTS)

load-test-endurance: ## Run endurance tests (memory leak detection)
	@echo "Running endurance tests..."
	$(GOTEST) -v -timeout 30m -run ".*Endurance.*|.*Memory.*" $(PERFORMANCE_TESTS)

test-short: ## Run tests in short mode (skip long-running tests)
	@echo "Running tests in short mode..."
	$(GOTEST) -short -v $(UNIT_TESTS) $(INTEGRATION_TESTS)

## Coverage targets

coverage: ## Generate test coverage report
	@echo "Generating coverage report..."
	$(GOTEST) -coverprofile=$(TEST_COVERAGE_FILE) -covermode=atomic $(UNIT_TESTS) $(INTEGRATION_TESTS)
	$(GOCMD) tool cover -func=$(TEST_COVERAGE_FILE)

coverage-html: coverage ## Generate HTML coverage report
	@echo "Generating HTML coverage report..."
	$(GOCMD) tool cover -html=$(TEST_COVERAGE_FILE) -o $(TEST_COVERAGE_HTML)
	@echo "Coverage report generated: $(TEST_COVERAGE_HTML)"

coverage-upload: coverage ## Upload coverage to external service (placeholder)
	@echo "Uploading coverage report..."
	# Add your coverage upload command here (e.g., codecov, coveralls)

## Test environment setup

start-emulators: ## Start GCP emulators for testing
	@echo "Starting Spanner emulator..."
	@if ! pgrep -f "cloud_spanner_emulator" > /dev/null; then \
		gcloud emulators spanner start \
			--host-port=$(SPANNER_EMULATOR_HOST) \
			--rest-port=9020 & \
		echo "Waiting for Spanner emulator to start..."; \
		sleep 5; \
	else \
		echo "Spanner emulator already running"; \
	fi

stop-emulators: ## Stop GCP emulators
	@echo "Stopping emulators..."
	@pkill -f "cloud_spanner_emulator" || true

setup-test-env: start-emulators ## Setup complete test environment
	@echo "Setting up test environment..."
	@sleep 2  # Wait for emulator to be ready
	SPANNER_EMULATOR_HOST=$(SPANNER_EMULATOR_HOST) \
	gcloud spanner instances create $(SPANNER_INSTANCE_ID) \
		--config=emulator-config \
		--description="Test instance" \
		--nodes=1 \
		--project=$(SPANNER_PROJECT_ID) || true
	SPANNER_EMULATOR_HOST=$(SPANNER_EMULATOR_HOST) \
	gcloud spanner databases create $(SPANNER_DATABASE_ID) \
		--instance=$(SPANNER_INSTANCE_ID) \
		--project=$(SPANNER_PROJECT_ID) \
		--ddl-file=./test/fixtures/configs/spanner_schema.sql || true

cleanup-test-env: ## Clean up test environment
	@echo "Cleaning up test environment..."
	SPANNER_EMULATOR_HOST=$(SPANNER_EMULATOR_HOST) \
	gcloud spanner databases delete $(SPANNER_DATABASE_ID) \
		--instance=$(SPANNER_INSTANCE_ID) \
		--project=$(SPANNER_PROJECT_ID) --quiet || true
	SPANNER_EMULATOR_HOST=$(SPANNER_EMULATOR_HOST) \
	gcloud spanner instances delete $(SPANNER_INSTANCE_ID) \
		--project=$(SPANNER_PROJECT_ID) --quiet || true

## Code quality targets

lint: ## Run golangci-lint
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

vet: ## Run go vet
	$(GOCMD) vet ./...

format: ## Format code with gofmt
	gofmt -s -w .

format-check: ## Check if code is formatted
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "The following files are not formatted:"; \
		gofmt -l .; \
		exit 1; \
	fi

## Benchmark targets

benchmark: ## Run all benchmarks
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./test/unit/... ./test/performance/...

benchmark-webhook: ## Benchmark webhook processing
	@echo "Benchmarking webhook processing..."
	$(GOTEST) -bench=BenchmarkCallRailWebhookProcessing -benchmem ./test/unit/...

benchmark-ai: ## Benchmark AI analysis
	@echo "Benchmarking AI analysis..."
	$(GOTEST) -bench=BenchmarkAIAnalysis -benchmem ./test/performance/...

benchmark-compare: ## Compare benchmark results (requires benchstat)
	@echo "Running benchmark comparison..."
	@if command -v benchstat >/dev/null 2>&1; then \
		$(GOTEST) -bench=. -count=5 ./test/unit/... > bench-old.txt; \
		$(GOTEST) -bench=. -count=5 ./test/unit/... > bench-new.txt; \
		benchstat bench-old.txt bench-new.txt; \
		rm bench-old.txt bench-new.txt; \
	else \
		echo "benchstat not installed. Install with: go install golang.org/x/perf/cmd/benchstat@latest"; \
	fi

## Dependency management

deps: ## Download dependencies
	$(GOMOD) download

deps-update: ## Update dependencies
	$(GOMOD) tidy
	$(GOGET) -u ./...

deps-verify: ## Verify dependencies
	$(GOMOD) verify

## CI/CD targets

ci-test: setup-test-env test-all cleanup-test-env ## Run all tests for CI

ci-quick: test-unit lint vet ## Quick CI checks

## Reporting targets

test-report: ## Generate comprehensive test report
	@echo "Generating test reports..."
	@mkdir -p ./test-reports

	# Unit test report
	$(GOTEST) -v -json $(UNIT_TESTS) > ./test-reports/unit-tests.json

	# Integration test report
	SPANNER_EMULATOR_HOST=$(SPANNER_EMULATOR_HOST) \
	$(GOTEST) -v -json $(INTEGRATION_TESTS) > ./test-reports/integration-tests.json

	# Coverage report
	$(GOTEST) -coverprofile=./test-reports/coverage.out -covermode=atomic $(UNIT_TESTS) $(INTEGRATION_TESTS)
	$(GOCMD) tool cover -html=./test-reports/coverage.out -o ./test-reports/coverage.html

	@echo "Test reports generated in ./test-reports/"

## Load testing targets

load-test-light: ## Run light load tests
	@echo "Running light load tests..."
	$(GOTEST) -v -timeout 5m -run "TestMultiTenantLoadIsolation" $(LOAD_TESTS)

load-test-stress: ## Run stress tests
	@echo "Running stress tests..."
	$(GOTEST) -v -timeout 15m -run "TestScalabilityUnderLoad" $(LOAD_TESTS)

load-test-endurance: ## Run endurance tests
	@echo "Running endurance tests..."
	$(GOTEST) -v -timeout 30m -run "TestMemoryLeakDetection" $(LOAD_TESTS)

## Security testing targets

security-test-auth: ## Run authentication tests
	@echo "Running authentication security tests..."
	$(GOTEST) -v -run "TestAuthentication.*|TestToken.*" $(SECURITY_TESTS)

security-test-isolation: ## Run tenant isolation tests
	@echo "Running tenant isolation security tests..."
	$(GOTEST) -v -run "TestTenantIsolation.*" $(SECURITY_TESTS)

security-test-injection: ## Run injection attack tests
	@echo "Running injection attack tests..."
	$(GOTEST) -v -run "TestSQLInjection.*|TestXSS.*" $(SECURITY_TESTS)

## Development helpers

watch-test: ## Watch files and run tests on changes (requires entr)
	@if command -v entr >/dev/null 2>&1; then \
		find . -name "*.go" | entr -c make test-unit; \
	else \
		echo "entr not installed. Install with your package manager"; \
	fi

debug-test: ## Run a specific test with debugging
	@echo "Usage: make debug-test TEST=TestName"
	@if [ -z "$(TEST)" ]; then \
		echo "Please specify TEST variable: make debug-test TEST=TestWorkflowProcessing"; \
	else \
		$(GOTEST) -v -run "$(TEST)" ./...; \
	fi

## Documentation

test-docs: ## Generate test documentation
	@echo "Generating test documentation..."
	@echo "# Test Documentation" > TEST_DOCS.md
	@echo "" >> TEST_DOCS.md
	@echo "## Unit Tests" >> TEST_DOCS.md
	@$(GOTEST) -list . $(UNIT_TESTS) | grep -E "^Test" | sed 's/^/- /' >> TEST_DOCS.md
	@echo "" >> TEST_DOCS.md
	@echo "## Integration Tests" >> TEST_DOCS.md
	@$(GOTEST) -list . $(INTEGRATION_TESTS) | grep -E "^Test" | sed 's/^/- /' >> TEST_DOCS.md
	@echo "Test documentation generated: TEST_DOCS.md"

## Help

help: ## Show this help message
	@echo "Multi-Tenant Ingestion Pipeline - Test Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "Examples:"
	@echo "  make test                 # Run unit and integration tests"
	@echo "  make test-all            # Run all test suites"
	@echo "  make coverage-html       # Generate HTML coverage report"
	@echo "  make ci-test             # Run complete CI test suite"
	@echo "  make debug-test TEST=TestWorkflowProcessing"
	@echo ""

# Default target
.DEFAULT_GOAL := help