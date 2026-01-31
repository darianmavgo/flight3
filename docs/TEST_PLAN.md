# Flight3 Test Plan

This document outlines the assessment of the current Flight3 test suite and the roadmap for achieving robust test coverage.

## 1. Current State Assessment

### 1.1 Existing Coverage
The current test suite provides good coverage for:
- **PocketBase Integration**: Bootstrapping, collection creation, and record synchronization.
- **Local Directory Listing**: Validating that local directories are indexed into SQLite.
- **Asset Compliance**: Lint-style tests for CSS/HTML paths and file output locations.
- **Banquet URL Parsing**: Basic validation of Banquet URL structures.

### 1.2 Identified Gaps
- **Remote Directory Indexing**: No automated test verifies that `rclone` remotes are correctly walked and indexed into the standard `tb0` schema.
- **File Type Diversification**: Sparse testing for specific `mksqlite` converters (Excel, JSON, Markdown) using representative data.
- **Cache TTL Boundary Conditions**: Tests for cache expiry are present but don't deeply exercise edge cases (e.g., partial file corruption, clock skew).
- **Security & Access Control**: The `security_test.go` is currently disabled and lacks rigorous checks for path traversal.
- **Mocked Remotes**: Most tests rely on real local filesystem "remotes" rather than mocked rclone backend behaviors.

---

## 2. Recommended Updates

### 2.1 Refactor Existing Tests
- **Unify Setup**: Create a standardized `test_util.go` setup that initializes a clean `pb_data` and `pb_public` structure for all integration tests.
- **Parameterize Conversion Tests**: Update `rclone_pocketbase_test.go` to iterate over multiple file types in `pb_public/sample_data`.

### 2.2 Re-enable & Fix
- **Security Test**: Re-enable `security_test.go` and add cases for `../` injection in Banquet URLs (both local and remote).
- **Rclone List Test**: Re-enable and adapt `rclone_list_test.go` to use the new `IndexDirectory` method.

---

## 3. New Test Requirements

### 3.1 Unit Tests (Internal Package)
- **Converter Suite**:
  - Test CSV with headers vs. without headers.
  - Test Excel multi-sheet handling (standardizing on the first sheet).
  - Test JSON array vs. object-root conversion.
- **Rclone Manager Suite**:
  - Mock `vfs.Dir` to test `IndexDirectory` without actual network/remote calls.
  - Verify `Stat()` behavior on missing vs. existing remote paths.

### 3.2 Integration Tests
- **Remote Directory Crawl**:
  - Simulate a multi-level remote directory structure.
  - Verify that clicking a link in the generated HTML correctly navigates to the sub-resource.
- **Cache Invalidation**:
  - Test that modifying a local source file immediately invalidates the cache if the source timestamp is newer than the cache.

### 3.3 End-to-End Tests
- **Full Banquet Request Cycle**:
  - A test that uses `httptest.NewRecorder` to call the `banquetHandler`.
  - Verify headers, status codes, and the presence of expected table rows in the response body.

---

## 4. Test Infrastructure Updates

- **Coverage Implementation**: Add `go test -coverprofile=coverage.out ./...` to the build pipeline.
- **Standardized Fixtures**: Formalize `pb_public/sample_data` as the source of truth for all test fixtures.
- **Cleanup Automation**: Update `CleanTestArtifacts` in `util.go` to more aggressively clean `test_output` after each run to prevent state leakage.

---

## 5. Priorities

1. **✅ COMPLETED**: `IndexDirectory` verification for remote remotes.
   - `tests/index_directory_test.go` created with comprehensive tests
   - `TestIndexDirectoryLocal`: Validates local directory indexing via mksqlite filesystem converter
   - `TestIndexDirectoryRemote`: Validates remote S3 directory indexing (skips if no S3 remote available)
   - Both tests verify tb0 schema compliance and data integrity
2. **High**: Path traversal security audit tests.
3. **Medium**: Specific converter unit tests (Excel/JSON).
4. **Low**: Performance/Concurrency tests for VFS cache.
