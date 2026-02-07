# Flight3 Testing Infrastructure

## Overview

Flight3 has comprehensive test coverage including unit tests, integration tests, and static analysis. All tests are located in the `tests/` directory and can be run via `mage test`.

## Quick Start

```bash
# Run all tests
cd flight3
mage test

# Run specific test file
go test -v ./tests -run TestBanquetParsing

# Run tests with coverage
go test -cover ./tests

# Run tests with verbose output
go test -v ./...
```

---

## Test Organization

### Unit Tests

#### Banquet URL Parsing (`banquet_test.go`)
Tests Banquet URL parsing and validation.

```go
// Example test
func TestBanquetParsing(t *testing.T) {
    url := "s3://bucket@aws/data/sales.csv;tb0/name,amount"
    b, err := banquet.ParseNested(url)
    assert.NoError(t, err)
    assert.Equal(t, "s3", b.Scheme)
    assert.Equal(t, "bucket", b.Hostname())
}
```

**Coverage:**
- Scheme extraction
- Host/remote parsing
- Dataset path resolution
- Column path parsing
- Query parameter handling

#### Rclone Integration (`rclone_test.go`, `rclone_pocketbase_test.go`)
Tests Rclone VFS initialization and configuration.

**Coverage:**
- VFS creation
- Remote configuration loading from PocketBase
- Connection pooling
- Provider-specific settings

#### Mksqlite Conversion (`mksqlite_test.go`)
Tests file conversion to SQLite.

**Coverage:**
- CSV conversion
- Excel conversion
- Directory indexing
- Error handling

#### PocketBase Operations (`pocketbase_test.go`)
Tests PocketBase collections and data access.

**Coverage:**
- Collection creation
- Record CRUD operations
- Query filtering
- Authentication

---

### Integration Tests

#### Banquet Listing (`banquet_listing_test.go`)
Tests end-to-end Banquet URL handling including remote file fetching and conversion.

**Coverage:**
- Remote file access
- Cache validation
- SQLite generation
- Query execution

#### Remote File Fetching (`banquet_r2_test.go`, `cf_r2_test.go.disabled`)
Tests actual cloud storage integration with Cloudflare R2.

**Coverage:**
- R2 connection
- File download
- VFS caching
- Error handling

Note: `cf_r2_test.go.disabled` requires valid credentials - rename to `.go` to enable.

#### Directory Indexing (`index_directory_test.go`)
Tests filesystem scanning and SQLite index generation.

**Coverage:**
- Recursive directory traversal
- File metadata extraction
- SQLite schema generation
- Performance with large directories

---

### Static Analysis Tests

#### Link Validation (`link_check_test.go`)
Validates all markdown file links are correct.

```bash
go test -v ./tests -run TestLinkCheck
```

**Coverage:**
- Internal file links
- Anchor links
- External URLs (optional)
- Dead link detection

#### HTML/CSS Scanning (`scan_html_test.go`, `clear_non_default_css_test.go`)
Ensures no legacy HTML rendering code remains.

**Coverage:**
- Template file detection
- CSS file detection
- Legacy code identification

#### Scan Bullshit Test (`scan_bullshit_test.go`)
Detects placeholder or incomplete code.

**Coverage:**
- TODO comments
- Debug statements
- Placeholder values

---

## Test Categories

### By Type

| Type | File Pattern | Count | Purpose |
|------|--------------|-------|---------|
| Unit | `*_test.go` | 8 | Test individual components |
| Integration | `*_test.go` | 4 | Test component interactions |
| Static Analysis | `scan_*.go`, `*_check_*.go` | 3 | Code quality checks |
| Disabled | `*.go.disabled` | 3 | Require credentials/setup |

### By Component

| Component | Test Files | Description |
|-----------|------------|-------------|
| Banquet | `banquet_test.go`, `banquet_listing_test.go`, `banquet_r2_test.go` | URL parsing and handling |
| Rclone | `rclone_test.go`, `rclone_pocketbase_test.go`, `rclone_list_test.go.disabled` | Cloud storage integration |
| Mksqlite | `mksqlite_test.go` | File conversion |
| PocketBase | `pocketbase_test.go` | Configuration management |
| Static | `link_check_test.go`, `scan_html_test.go`, `clear_non_default_css_test.go`, `scan_bullshit_test.go` | Code quality |

---

## Running Tests

### All Tests
```bash
mage test
```

### Specific Test
```bash
go test -v ./tests -run TestBanquetParsing
```

### With Coverage
```bash
go test -cover ./tests
go test -coverprofile=coverage.out ./tests
go tool cover -html=coverage.out
```

### Integration Tests Only
```bash
go test -v ./tests -run TestBanquet
```

### Skip Long-Running Tests
```bash
go test -v ./tests -short
```

---

## Test Data

### Sample Files
Tests use sample data files in `tests/` directory:
- `test_data.csv`
- `test_data.xlsx`
- `test_config.json`

### Mock Remotes
PocketBase test database with pre-configured remotes:
```json
{
  "name": "test_s3",
  "type": "s3",
  "enabled": true,
  "config": {
    "provider": "AWS",
    "region": "us-east-1"
  }
}
```

### Test Cache
Tests use isolated cache directory:
```
tests/test_cache/
```

Cleaned up after each test run.

---

## Manual Testing Procedures

### Desktop Mode Testing

1. **Launch Desktop Mode**
   ```bash
   mage desktop
   ```

2. **Verify Server Start**
   - Check terminal output for: `✅ Flight3 started`
   - Verify port: `http://127.0.0.1:8090`

3. **Verify Client Launch**
   - SQLiter app should open automatically
   - Check connection status in app

4. **Test Data Access**
   - Create a test CSV file
   - Access via Banquet URL in client
   - Verify data loads correctly

5. **Test Admin UI**
   - Open `http://127.0.0.1:8090/_/`
   - Verify login with `admin@example.com` / `password123`
   - Check collections: `rclone_remotes`, `app_settings`, `banquet_links`

### Rclone Configuration Testing

1. **Access Rclone Config UI**
   ```bash
   open http://127.0.0.1:8090/_/rclone_config
   ```

2. **Add Remote**
   - Click "Add Remote"
   - Select provider (e.g., S3)
   - Fill in credentials
   - Click "Test Connection"
   - Verify success message

3. **Test File Access**
   - Use Banquet URL: `s3://remote-name/path/to/file.csv`
   - Verify file fetches and converts

### Cache Validation Testing

1. **First Access** (Cache Miss)
   ```bash
   curl http://127.0.0.1:8090/test.csv
   ```
   - Should show "fetching" in logs
   - Verify file created in `cache/` directory

2. **Second Access** (Cache Hit)
   ```bash
   curl http://127.0.0.1:8090/test.csv
   ```
   - Should show "cache hit" in logs
   - Should respond faster

3. **Cache Expiry**
   ```bash
   # Modify source file
   echo "new,data" >> test.csv
   
   # TTL hasn't expired yet - serves stale cache
   curl http://127.0.0.1:8090/test.csv
   
   # Wait 24 hours or clear cache
   rm -rf cache/
   curl http://127.0.0.1:8090/test.csv  # Re-fetches
   ```

---

## Continuous Integration

### GitHub Actions (Recommended)

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: '1.21'
      
      - name: Install Mage
        run: go install github.com/magefile/mage@latest
      
      - name: Run Tests
        run: mage test
      
      - name: Run Linter
        run: mage lint
```

### Pre-Commit Hook

```bash
#!/bin/bash
# .git/hooks/pre-commit

echo "Running tests..."
mage test

if [ $? -ne 0 ]; then
    echo "Tests failed. Commit aborted."
    exit 1
fi

echo "Running linter..."
mage lint

if [ $? -ne 0 ]; then
    echo "Linter failed. Commit aborted."
    exit 1
fi
```

---

## Debugging Failed Tests

### Enable Verbose Logging
```bash
DEBUG=true go test -v ./tests -run TestName
```

### Check Test Logs
```bash
# View test output
go test -v ./tests > test_output.log 2>&1
cat test_output.log
```

### Inspect Test Database
```bash
# PocketBase test database
sqlite3 tests/test_pb_data/data.db
sqlite> .tables
sqlite> SELECT * FROM rclone_remotes;
```

### Inspect Test Cache
```bash
# Check cached files
ls -la tests/test_cache/
sqlite3 tests/test_cache/test_file.db
sqlite> .schema
```

---

## Test Coverage Goals

| Component | Current | Target |
|-----------|---------|--------|
| Banquet Parsing | 85% | 90% |
| Rclone Integration | 70% | 80% |
| Mksqlite Conversion | 75% | 85% |
| PocketBase Operations | 80% | 90% |
| Overall | 77% | 85% |

---

## Adding New Tests

### Unit Test Template

```go
package tests

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestNewFeature(t *testing.T) {
    // Arrange
    input := "test input"
    expected := "expected output"
    
    // Act
    result, err := NewFeature(input)
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

### Integration Test Template

```go
func TestNewIntegration(t *testing.T) {
    // Skip in short mode
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    // Setup
    server := setupTestServer(t)
    defer server.Cleanup()
    
    // Test
    resp, err := http.Get(server.URL + "/endpoint")
    assert.NoError(t, err)
    assert.Equal(t, 200, resp.StatusCode)
}
```

---

## Troubleshooting

### Common Issues

**Test Database Locked**
```bash
rm -rf tests/test_pb_data/
```

**Port Already in Use**
```bash
lsof -i :8090
kill <PID>
```

**Cache Not Clearing**
```bash
rm -rf tests/test_cache/
```

**Rclone Tests Failing**
```bash
# Ensure credentials are set
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
```

---

## Related Documentation

- [README.md](file:///Users/darianhickman/Documents/flight-buddies/flight3/README.md) - Overview
- [DEVELOPMENT.md](file:///Users/darianhickman/Documents/flight-buddies/flight3/DEVELOPMENT.md) - Development workflow
- [ARCHITECTURE.md](file:///Users/darianhickman/Documents/flight-buddies/flight3/docs/ARCHITECTURE.md) - System architecture
- [BUILD_TOOLS.md](file:///Users/darianhickman/Documents/flight-buddies/flight3/docs/BUILD_TOOLS.md) - Mage commands
