# Testing Guide

## Test Organization

This project follows Go's standard testing conventions:

### File Placement

- **Test files are placed alongside source code** - Each `*_test.go` file is in the same directory as the code it tests
- **Package-level tests** - Tests in `internal/` test the `internal` package
- **Subpackage tests** - Tests in `internal/export/` test the `export` subpackage

### Current Structure

```
internal/
├── bubble_map.go          # Source code
├── bubble_map_test.go     # Tests for bubble_map.go
├── cache.go
├── cache_test.go
├── export/
│   ├── json.go
│   ├── json_test.go       # Tests for json.go
│   └── ...
└── ...
```

### Why This Structure?

1. **Go Convention**: The Go community standard is to place tests alongside source files
2. **Package Access**: Tests in the same package can test both exported and unexported functions
3. **Discoverability**: Easy to find tests for any given file
4. **Tooling Support**: Go tools (`go test`, `go cover`) work seamlessly with this structure

### Test Coverage

Current coverage: **54.0%** (internal package), **98.1%** (export package)

To check coverage:
```bash
make test-coverage          # Generate coverage profile
make test-coverage-html     # Generate HTML report
make test-coverage-check    # Check against 80% threshold
```

### Test Helpers

Common test utilities are in:
- `internal/testhelpers.go` - Helper functions for creating test data
- `testutil/` - Mock database and fixture utilities

### Best Practices

1. **Table-driven tests** - Use table-driven tests for multiple scenarios
2. **Test isolation** - Each test should be independent and not rely on shared state
3. **No real databases** - Tests use in-memory SQLite databases, not real Cursor DB files
4. **Descriptive names** - Test names should clearly describe what they test
5. **Error cases** - Test both success and error paths

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
make test-coverage

# Run tests for a specific package
go test ./internal/export

# Run a specific test
go test ./internal -run TestDeduplicator
```

