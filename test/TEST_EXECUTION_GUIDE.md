# Multi-Tenant CallRail Ingestion Pipeline - Test Execution Guide

## Overview

This guide provides comprehensive instructions for executing and validating the complete test suite for the multi-tenant CallRail ingestion pipeline. The test strategy is designed to validate all requirements from the Implementation Guide (lines 506-531) and ensure system reliability, performance, and security.

## Test Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Test Orchestrator                        │
│              (test/test_orchestrator.go)                   │
├─────────────────────────────────────────────────────────────┤
│  Unit Tests     │ Integration │  E2E Tests  │ Load Tests     │
│  (90% coverage) │    Tests    │ (Critical   │ (Performance   │
│                 │ (Spanner,   │  Journeys)  │  Validation)   │
│                 │  AI, CRM)   │             │                │
├─────────────────┼─────────────┼─────────────┼────────────────┤
│           Security Tests (Authentication, Authorization,     │
│              Data Protection, Tenant Isolation)             │
└─────────────────────────────────────────────────────────────┘
```

## Performance Targets (from Implementation Guide)

- **Webhook Latency**: <200ms for forms processing
- **AI Analysis**: <1s for information extraction
- **Throughput**: 1,000+ requests/minute per tenant
- **Audio Processing**: <5s transcription latency
- **Availability**: 99.9% SLA target
- **Coverage**: >90% unit test coverage

## Quick Start

### Prerequisites

```bash
# Install Go dependencies
go mod download

# Install required tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install golang.org/x/perf/cmd/benchstat@latest

# Install GCP emulators
gcloud components install cloud-spanner-emulator
```

### Run All Tests

```bash
# Complete test suite with orchestrator
make test-comprehensive

# Quick validation (unit + integration)
make test

# All test suites individually
make test-all
```

## Test Categories

### 1. Unit Tests (`test/unit/`)

**Purpose**: Test individual components in isolation with >90% coverage

**Key Test Files**:
- `callrail_webhook_test.go` - CallRail webhook processing logic
- `workflow_test.go` - Core workflow orchestration
- Additional unit tests for business logic components

**Execution**:
```bash
# Run all unit tests
make test-unit

# Run specific CallRail tests
go test -v -run "TestCallRailWebhook.*" ./test/unit/

# Generate coverage report
make coverage-html
```

**Coverage Requirements**:
- Statement coverage: >90%
- Branch coverage: >85%
- Function coverage: >95%

### 2. Integration Tests (`test/integration/`)

**Purpose**: Test service interactions and database operations

**Key Test Files**:
- `callrail_end_to_end_test.go` - Complete CallRail workflow integration
- `spanner_integration_test.go` - Database operations and tenant isolation

**Execution**:
```bash
# Setup test environment
make setup-test-env

# Run integration tests
make test-integration

# Cleanup
make cleanup-test-env
```

**Test Scenarios**:
- CallRail webhook → Audio processing → AI extraction → CRM integration
- Multi-tenant data isolation verification
- Database transaction handling
- External service integration

### 3. End-to-End Tests (`test/e2e/`)

**Purpose**: Test complete user journeys and critical workflows

**Key Test Files**:
- `ingestion_flow_test.go` - Complete ingestion workflows
- Additional E2E scenarios for different call types

**Execution**:
```bash
# Run E2E tests
make test-e2e

# Test specific scenarios
go test -v -run "TestCompleteCallRailWorkflow.*" ./test/e2e/
```

**Test Scenarios**:
- High-value kitchen remodeling lead processing
- Emergency call prioritization and routing
- Abandoned call handling and analytics
- Large file processing and timeout handling

### 4. Load Tests (`test/load/`)

**Purpose**: Validate performance under realistic and peak load conditions

**Key Test Files**:
- `callrail_performance_test.go` - CallRail-specific load testing
- `tenant_isolation_test.go` - Multi-tenant performance isolation

**Execution**:
```bash
# Standard load tests
make test-load

# Specific load test types
make load-test-light     # Quick performance validation
make load-test-stress    # Peak load testing
make load-test-endurance # Memory leak detection
```

**Performance Validation**:
- Webhook latency: <200ms (P95)
- Throughput: 1,000+ req/min per tenant
- Multi-tenant isolation under load
- Auto-scaling behavior validation

### 5. Security Tests (`test/security/`)

**Purpose**: Ensure robust security, authentication, and tenant isolation

**Key Test Files**:
- `callrail_security_test.go` - CallRail webhook security validation
- `multi_tenant_security_test.go` - General multi-tenant security

**Execution**:
```bash
# All security tests
make test-security

# Specific security categories
make security-test-auth      # Authentication tests
make security-test-isolation # Tenant isolation tests
make security-test-injection # Injection attack tests
```

**Security Validation**:
- Webhook signature validation
- Rate limiting enforcement
- Tenant data isolation
- Input sanitization and validation
- Audit logging and monitoring

## Test Environment Configuration

### Local Development

```bash
# Start required services
make start-emulators

# Run tests with emulator
SPANNER_EMULATOR_HOST=localhost:9010 make test-integration

# Stop services
make stop-emulators
```

### CI/CD Pipeline

```bash
# Complete CI test suite
make ci-test

# Quick CI validation
make ci-quick

# Generate reports for CI
make test-report
```

### Environment Variables

```bash
# Test configuration
export TEST_TIMEOUT=10m
export PERFORMANCE_MODE=standard  # quick, standard, comprehensive
export SPANNER_EMULATOR_HOST=localhost:9010

# Skip specific tests
export SKIP_INTEGRATION_TESTS=true
export SKIP_LOAD_TESTS=true

# CI mode
export CI=true
```

## Test Data and Fixtures

### Test Audio Files (`test/fixtures/audio/`)

- `kitchen_remodel.wav` - High-value lead scenario
- `emergency_water_damage.wav` - Emergency call scenario
- `bathroom_renovation.wav` - Standard renovation inquiry
- `abandoned_call.wav` - Short abandoned call
- `large_file.wav` - Large file processing test

### Test Tenant Configurations (`test/fixtures/data/`)

- Multiple tenant configurations for isolation testing
- CRM integration settings (Salesforce, HubSpot)
- Rate limiting and security configurations

### Mock Services

All tests use comprehensive mocking for external dependencies:
- CallRail webhook simulation
- Audio transcription services
- AI/ML processing services
- CRM integration endpoints
- Email notification services

## Performance Monitoring and Validation

### Key Metrics Tracked

1. **Latency Metrics**:
   - Webhook response time (target: <200ms)
   - Audio processing time (target: <5s)
   - End-to-end workflow time (target: <30s)

2. **Throughput Metrics**:
   - Requests per second per tenant
   - Concurrent webhook processing
   - Database operation performance

3. **Resource Utilization**:
   - Memory usage patterns
   - CPU utilization under load
   - Database connection pooling

4. **Error Rates**:
   - Overall system error rate
   - Tenant-specific error isolation
   - Service degradation handling

### Validation Criteria

```bash
# Performance targets validation
Webhook Latency P95: <200ms ✓
Processing Time P95: <1s ✓
Throughput: >1000 req/min per tenant ✓
Error Rate: <1% ✓
```

## Troubleshooting and Debugging

### Common Issues

1. **Spanner Emulator Not Starting**:
   ```bash
   # Check if emulator is already running
   ps aux | grep spanner

   # Kill existing processes
   make stop-emulators

   # Restart emulator
   make start-emulators
   ```

2. **Test Timeouts**:
   ```bash
   # Increase timeout for load tests
   export TEST_TIMEOUT=20m
   make test-load
   ```

3. **Coverage Issues**:
   ```bash
   # Generate detailed coverage report
   make coverage-html

   # Open in browser
   open coverage.html
   ```

### Debug Specific Tests

```bash
# Run specific test with verbose output
make debug-test TEST=TestCallRailWebhookProcessing

# Run with race detection
go test -race -v ./test/unit/

# Enable profiling for performance tests
go test -cpuprofile=cpu.prof -memprofile=mem.prof ./test/load/
```

## Quality Gates and CI Integration

### Automated Quality Checks

The test orchestrator enforces quality gates:

1. **Unit Test Coverage**: ≥90%
2. **Integration Test Success**: 100% critical paths
3. **Security Test Success**: 100% (no failures allowed)
4. **Performance Targets**: Must meet all latency/throughput targets
5. **Zero Critical Security Vulnerabilities**

### CI/CD Integration

```yaml
# Example GitHub Actions integration
- name: Run Comprehensive Tests
  run: make test-comprehensive

- name: Validate Performance
  run: make test-load

- name: Security Validation
  run: make test-security

- name: Upload Coverage
  run: make coverage-upload
```

### Exit Codes and Results

- `0`: All tests passed, quality gates met
- `1`: Test failures or quality gate violations
- Environment variables set for CI consumption:
  - `QUALITY_GATE`: PASSED/WARNING/FAILED
  - `OVERALL_COVERAGE`: Coverage percentage
  - `FAILED_SUITES`: Number of failed test suites

## Test Maintenance

### Regular Maintenance Tasks

1. **Update Test Data**: Refresh fixtures monthly
2. **Performance Baselines**: Update targets quarterly
3. **Security Scenarios**: Add new attack vectors
4. **Dependency Updates**: Keep test tools current

### Adding New Tests

1. **Unit Tests**: Add to appropriate `test/unit/*.go` file
2. **Integration Tests**: Extend existing integration scenarios
3. **Load Tests**: Add performance scenarios to load tests
4. **Security Tests**: Add security validations

### Test Review Process

1. All new tests must have documentation
2. Performance tests must validate against targets
3. Security tests must cover new attack vectors
4. Integration tests must validate tenant isolation

## Reporting and Analytics

### Test Reports Generated

- HTML coverage report (`coverage.html`)
- JSON test results (`test-reports/*.json`)
- Performance benchmark results
- Security audit logs
- Quality gate status report

### Metrics Dashboard

Key metrics tracked over time:
- Test execution duration trends
- Coverage evolution
- Performance regression detection
- Flaky test identification

## Getting Help

### Documentation
- [Implementation Guide](../COMPLETE-IMPLEMENTATION-GUIDE.md)
- [CallRail Integration Flow](../callrail-integration-flow.md)
- [Database Schema](../database-schema-updates.sql)

### Commands Reference
```bash
# Get help with available commands
make help

# List all available tests
go test -list . ./test/...

# Check test dependencies
go mod verify
```

### Support
- Review test logs for detailed error information
- Check emulator status if integration tests fail
- Verify environment variables are set correctly
- Ensure all prerequisites are installed

This comprehensive test strategy ensures the CallRail integration meets all requirements with high confidence in system reliability, performance, and security.