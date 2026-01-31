# Flight3 Refactoring Analysis: Eliminate Redundant Code

## Overview
This document identifies code in `flight3` that duplicates functionality now available in the upgraded `banquet`, `mksqlite`, and `sqliter` packages.

## Current Package Versions
- `github.com/darianmavgo/banquet v1.1.0`
- `github.com/darianmavgo/mksqlite v1.3.1`
- `github.com/darianmavgo/sqliter v1.4.2`

## Redundant Code Identified

### 1. SQL Query Composition (HIGH PRIORITY)

**Location:** `/internal/flight/server.go`
- `buildSQLQuery()` function (lines 207-263)
- `quoteIdentifier()` function (lines 266-273)

**Replacement:** `github.com/darianmavgo/banquet/sqlite`
- `sqlite.Compose(bq *banquet.Banquet) string` - Builds complete SQL query
- `sqlite.QuoteIdentifier(s string) string` - Quotes identifiers safely
- `sqlite.InferTable(bq *banquet.Banquet) string` - Infers table name

**Benefits:**
- Centralized SQL composition logic
- Better tested and maintained
- Consistent behavior across all projects using banquet
- Already handles edge cases

**Impact:**
- Affects: `ServeFromCache()` function in `server.go`
- Risk: Low - direct replacement with identical functionality

---

### 2. Table Name Inference Logic

**Location:** `/internal/flight/server.go`
- Lines 29-64 in `ServeFromCache()` - Manual table inference

**Replacement:** `sqlite.InferTable()` from banquet package
- Already handles `tb0` preference
- Falls back to `sqlite_master` when appropriate
- Considers dataset path and column path

**Benefits:**
- Removes ~35 lines of code
- Consistent table inference logic
- Better handling of edge cases

**Impact:**
- Affects: `ServeFromCache()` function
- Risk: Low - logic is equivalent

---

### 3. Extension Mapping (MEDIUM PRIORITY)

**Location:** `/internal/flight/banquethandler.go`
- `extensionMap` variable (lines 18-33)

**Replacement:** This mapping is duplicated in:
- `/internal/flight/converter.go` (lines 60-86 - switch statement)
- `mksqlite/converters` package already knows these mappings

**Benefits:**
- Single source of truth
- Automatic support for new converters
- Less maintenance burden

**Recommendation:**
- Use `converters.Drivers()` to get available converters
- Let mksqlite handle extension-to-driver mapping internally
- Remove the hardcoded map

**Impact:**
- Affects: `HandleBanquet()` and `HandleLocalDataset()`
- Risk: Low - mksqlite already handles this

---

### 4. Cache Key Generation (LOW PRIORITY - Keep for now)

**Location:** `/internal/flight/cache.go`
- `GenCacheKey()` function

**Status:** **KEEP THIS**
- Flight3-specific caching strategy
- Includes auth alias and config hash
- Not generic enough for banquet package
- Well-encapsulated and working

---

### 5. Cache Validation (LOW PRIORITY - Keep for now)

**Location:** `/internal/flight/cache.go`
- `ValidateCache()` function
- `GetCachePath()` function

**Status:** **KEEP THIS**
- Flight3-specific TTL and caching logic
- Tied to PocketBase data directory structure
- Not generic enough for shared packages

---

### 6. Converter Wrapper (MEDIUM PRIORITY)

**Location:** `/internal/flight/converter.go`
- `ConvertToSQLite()` function (lines 25-127)

**Current Issues:**
- Duplicates extension-to-driver mapping
- Could be simplified

**Recommendation:**
- Simplify to use mksqlite more directly
- Remove hardcoded extension switch
- Let mksqlite auto-detect format when possible

**Benefits:**
- Less code to maintain
- Automatic support for new converters
- More robust error handling

**Impact:**
- Affects: `HandleBanquet()` and `HandleLocalDataset()`
- Risk: Medium - needs testing with all file types

---

## Recommended Refactoring Order

### Phase 1: SQL Composition (Immediate - Low Risk)
1. Replace `buildSQLQuery()` with `sqlite.Compose()`
2. Replace `quoteIdentifier()` with `sqlite.QuoteIdentifier()`
3. Replace table inference logic with `sqlite.InferTable()`
4. Update imports in `server.go`

**Files to modify:**
- `/internal/flight/server.go`

**Estimated effort:** 30 minutes
**Risk:** Low
**Testing:** Run existing query tests

---

### Phase 2: Simplify Converter (Medium Risk)
1. Remove hardcoded extension map from `banquethandler.go`
2. Simplify `ConvertToSQLite()` to rely on mksqlite auto-detection
3. Remove duplicate extension switch in `converter.go`

**Files to modify:**
- `/internal/flight/converter.go`
- `/internal/flight/banquethandler.go`

**Estimated effort:** 1-2 hours
**Risk:** Medium
**Testing:** Test with all supported file types (CSV, Excel, JSON, etc.)

---

### Phase 3: Future Considerations
1. Monitor banquet/sqliter packages for additional utilities
2. Consider contributing flight3-specific features back to shared packages
3. Keep cache management in flight3 (it's app-specific)

---

## Code Examples

### Before (server.go):
```go
query := buildSQLQuery(b)
```

### After (server.go):
```go
import "github.com/darianmavgo/banquet/sqlite"

query := sqlite.Compose(b)
```

---

### Before (server.go - table inference):
```go
if b.Table == "" {
    func() {
        rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
        // ... 30+ lines of inference logic
    }()
}
```

### After (server.go):
```go
import "github.com/darianmavgo/banquet/sqlite"

if b.Table == "" {
    b.Table = sqlite.InferTable(b)
}
```

---

## Testing Checklist

### Phase 1 Testing:
- [ ] Test basic SELECT queries
- [ ] Test queries with WHERE clauses
- [ ] Test queries with ORDER BY
- [ ] Test queries with LIMIT/OFFSET
- [ ] Test queries with GROUP BY/HAVING
- [ ] Test table name inference with tb0
- [ ] Test table name inference with sqlite_master
- [ ] Test identifier quoting with special characters
- [ ] Test directory listings
- [ ] Test file queries

### Phase 2 Testing:
- [ ] Test CSV file conversion
- [ ] Test Excel file conversion
- [ ] Test JSON file conversion
- [ ] Test HTML file conversion
- [ ] Test Markdown file conversion
- [ ] Test ZIP file conversion
- [ ] Test directory indexing
- [ ] Test already-SQLite files
- [ ] Test local files
- [ ] Test remote files via rclone

---

## Dependencies to Add

None - all required packages are already in `go.mod`:
- ✅ `github.com/darianmavgo/banquet v1.1.0`
- ✅ `github.com/darianmavgo/mksqlite v1.3.1`
- ✅ `github.com/darianmavgo/sqliter v1.4.2`

Just need to import the `sqlite` subpackage:
```go
import "github.com/darianmavgo/banquet/sqlite"
```

---

## Summary

**Total Lines to Remove:** ~100-150 lines
**Total Files to Modify:** 2-3 files
**Risk Level:** Low to Medium
**Estimated Total Effort:** 2-3 hours including testing

**Key Benefits:**
1. Less code to maintain
2. Better tested SQL composition
3. Consistent behavior across projects
4. Automatic support for new features in shared packages
5. Clearer separation of concerns
