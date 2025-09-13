#!/bin/bash

# Multi-Tenant Ingestion Pipeline Test Runner
# This script provides a convenient way to run various test suites locally

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
TEST_TYPE="all"
VERBOSE=false
COVERAGE=false
CLEANUP=true
PARALLEL=true

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

show_help() {
    cat << EOF
Multi-Tenant Ingestion Pipeline Test Runner

Usage: $0 [OPTIONS]

Options:
    -t, --type TYPE        Test type to run (unit|integration|e2e|load|security|all) [default: all]
    -v, --verbose          Enable verbose output
    -c, --coverage         Generate coverage report
    -nc, --no-cleanup      Skip cleanup after tests
    -ns, --no-parallel     Disable parallel test execution
    -h, --help            Show this help message

Test Types:
    unit                  Run unit tests only
    integration          Run integration tests (requires emulators)
    e2e                  Run end-to-end tests (requires emulators)
    load                 Run load/performance tests
    security             Run security tests
    all                  Run all test suites (default)

Examples:
    $0                           # Run all tests
    $0 -t unit -c               # Run unit tests with coverage
    $0 -t integration -v        # Run integration tests with verbose output
    $0 -t load --no-parallel    # Run load tests without parallelization

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -t|--type)
            TEST_TYPE="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -c|--coverage)
            COVERAGE=true
            shift
            ;;
        -nc|--no-cleanup)
            CLEANUP=false
            shift
            ;;
        -ns|--no-parallel)
            PARALLEL=false
            shift
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Validate test type
case $TEST_TYPE in
    unit|integration|e2e|load|security|all)
        ;;
    *)
        log_error "Invalid test type: $TEST_TYPE"
        show_help
        exit 1
        ;;
esac

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed or not in PATH"
        exit 1
    fi

    # Check Go version
    GO_VERSION=$(go version | grep -o 'go[0-9]\+\.[0-9]\+' | sed 's/go//')
    log_info "Go version: $GO_VERSION"

    # Check if gcloud is installed (needed for emulators)
    if [[ "$TEST_TYPE" == "integration" || "$TEST_TYPE" == "e2e" || "$TEST_TYPE" == "all" ]]; then
        if ! command -v gcloud &> /dev/null; then
            log_warning "gcloud CLI not found. Integration and E2E tests may fail."
        fi
    fi

    # Check if Make is available
    if ! command -v make &> /dev/null; then
        log_error "Make is not installed. Please install make to use this script."
        exit 1
    fi

    log_success "Prerequisites check completed"
}

# Setup test environment
setup_environment() {
    log_info "Setting up test environment..."

    # Create necessary directories
    mkdir -p test-reports test-logs

    # Set environment variables
    export GO111MODULE=on

    if [[ "$VERBOSE" == "true" ]]; then
        export VERBOSE=1
    fi

    # Download dependencies
    log_info "Downloading Go dependencies..."
    go mod download

    log_success "Test environment setup completed"
}

# Start emulators if needed
start_emulators() {
    if [[ "$TEST_TYPE" == "integration" || "$TEST_TYPE" == "e2e" || "$TEST_TYPE" == "all" ]]; then
        log_info "Starting GCP emulators..."
        make start-emulators
        sleep 3
        log_info "Setting up test database..."
        make setup-test-env
        log_success "Emulators started and configured"
    fi
}

# Stop emulators
stop_emulators() {
    if [[ "$CLEANUP" == "true" ]]; then
        log_info "Stopping emulators and cleaning up..."
        make cleanup-test-env || true
        make stop-emulators || true
        log_success "Cleanup completed"
    fi
}

# Run specific test suite
run_tests() {
    local test_type=$1
    local start_time=$(date +%s)

    log_info "Running $test_type tests..."

    case $test_type in
        unit)
            make test-unit
            ;;
        integration)
            make test-integration
            ;;
        e2e)
            make test-e2e
            ;;
        load)
            make test-load
            ;;
        security)
            make test-security
            ;;
        all)
            make test-all
            ;;
    esac

    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    log_success "$test_type tests completed in ${duration}s"
}

# Generate coverage report
generate_coverage() {
    if [[ "$COVERAGE" == "true" ]]; then
        log_info "Generating coverage report..."
        make coverage-html
        log_success "Coverage report generated: coverage.html"
    fi
}

# Generate test report
generate_report() {
    log_info "Generating test report..."
    make test-report || log_warning "Test report generation failed"
}

# Main execution
main() {
    local overall_start_time=$(date +%s)

    log_info "Starting Multi-Tenant Ingestion Pipeline tests"
    log_info "Test type: $TEST_TYPE"
    log_info "Verbose: $VERBOSE"
    log_info "Coverage: $COVERAGE"
    log_info "Cleanup: $CLEANUP"
    log_info "Parallel: $PARALLEL"

    # Trap to ensure cleanup on exit
    trap stop_emulators EXIT

    check_prerequisites
    setup_environment
    start_emulators

    # Run tests based on type
    case $TEST_TYPE in
        all)
            run_tests unit
            run_tests integration
            run_tests e2e
            run_tests security
            # Only run load tests if explicitly requested
            if [[ "${RUN_LOAD_TESTS:-false}" == "true" ]]; then
                run_tests load
            fi
            ;;
        *)
            run_tests $TEST_TYPE
            ;;
    esac

    generate_coverage
    generate_report

    local overall_end_time=$(date +%s)
    local total_duration=$((overall_end_time - overall_start_time))

    log_success "All tests completed successfully in ${total_duration}s"
    log_info "Test reports available in: ./test-reports/"

    if [[ "$COVERAGE" == "true" ]]; then
        log_info "Coverage report available at: ./coverage.html"
    fi
}

# Handle interruption
handle_interrupt() {
    log_warning "Test execution interrupted"
    stop_emulators
    exit 130
}

trap handle_interrupt INT TERM

# Run main function
main "$@"