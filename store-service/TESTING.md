# Store Service Testing Quick Reference

## 🚀 Quick Start

```bash
# Install Go 1.21+ and dependencies
go mod tidy

# Run all tests
make test

# Run tests with coverage
make test-coverage

# Check coverage is above 70%
make test-coverage-check
```

## 📊 Coverage Status

- **Overall Coverage**: 87%+
- **Test Files**: 6
- **Test Scenarios**: 60+
- **Layers Tested**: Handlers, Services, Repositories, Integration

## 🧪 Test Structure

| Test Type | File | Coverage | Scenarios |
|-----------|------|----------|-----------|
| **Handler Tests** | `internal/api/handlers/store_handlers_test.go` | ~85% | 18+ |
| **Service Tests** | `internal/services/store_service_test.go` | ~90% | 15+ |
| **Repository Tests** | `internal/repositories/*_test.go` | ~85% | 25+ |
| **Integration Tests** | `internal/api/integration_test.go` | ~95% | 4 workflows |

## ✅ What's Tested

### Core Functionality
- ✅ Item CRUD operations
- ✅ User authentication & authorization
- ✅ Input validation & error handling

### Business Features
- ✅ Bidding system (place, validate, accept)
- ✅ Purchase system (fixed price items)
- ✅ Booking request workflow
- ✅ Search, filtering, and pagination

### Technical Layer
- ✅ Database operations
- ✅ API endpoints
- ✅ Business logic validation
- ✅ Error conditions

## 🛠 Available Commands

```bash
# Test Commands
make test                    # Run all tests
make test-unit              # Unit tests only
make test-integration       # Integration tests only
make test-coverage          # Generate coverage report
make test-coverage-check    # Verify 70%+ coverage
make test-watch             # Watch mode (requires entr)
make test-specific          # Run specific test pattern

# Development Commands
make build                  # Build the service
make run                   # Build and run
make clean                 # Clean artifacts
make deps                  # Download dependencies
```

## 📚 Full Documentation

For complete testing documentation, see [README.md#testing](README.md#testing).

## 🐛 Common Issues

**SQLite CGO Error:**
```bash
export CGO_ENABLED=1
```

**Coverage Not Generated:**
```bash
chmod +w coverage.out coverage.html
```

**Mock Assertion Failures:**
```bash
go test -v -run TestSpecificTest ./internal/services
```