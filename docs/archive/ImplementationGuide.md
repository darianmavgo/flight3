# Quick Implementation Guide

## Overview

This is the step-by-step guide to implement the Flight3 ↔ SQLiter refactoring.

**Estimated Time:** 7 hours  
**Difficulty:** Medium  
**Risk:** Low (can rollback easily)

---

## Pre-Implementation Checklist

- [ ] Read `ResponsibilityBoundary.md` (understand the boundary)
- [ ] Read `ArchitectureSummary.md` (understand the flow)
- [ ] Read `CleanUpTodo.md` (know what to remove)
- [ ] Backup current code (`git commit`)
- [ ] Ensure tests pass currently
- [ ] Have SQLiter v1.4.2+ available

---

## Step 1: Backup and Branch (5 minutes)

```bash
cd /Users/darianhickman/Documents/flight3

# Ensure clean state
git status

# Create backup branch
git checkout -b backup-before-sqliter-refactor
git add .
git commit -m "Backup before SQLiter refactoring"

# Create working branch
git checkout -b refactor-sqliter-integration
```

**Verify:**
- [ ] Clean git status
- [ ] On new branch

---

## Step 2: Update Dependencies (5 minutes)

```bash
# Update sqliter to latest
go get github.com/darianmavgo/sqliter@latest

# Verify version
go list -m github.com/darianmavgo/sqliter

# Tidy dependencies
go mod tidy
```

**Verify:**
- [ ] SQLiter v1.4.2 or higher
- [ ] No dependency errors

---

## Step 3: Modify `flight.go` (30 minutes)

### 3.1: Add SQLiter Import

**File:** `internal/flight/flight.go`

**Add import:**
```go
import (
    // ... existing imports ...
    "github.com/darianmavgo/sqliter/sqliter"
)
```

### 3.2: Remove Template Initialization

**Find and DELETE (lines ~88-96):**
```go
// REMOVE THIS BLOCK
// Initialize SQLiter with Embedded Templates
// We use the templates embedded in the sqliter library.
tpl, err := sqliter.GetEmbeddedTemplates()
if err != nil {
    log.Fatal("Failed to load embedded templates from sqliter:", err)
}

// Initialize TableWriter with embedded templates
tw := sqliter.NewTableWriter(tpl, sqliter.DefaultConfig())
```

### 3.3: Add SQLiter Server Initialization

**Add in its place:**
```go
// Initialize SQLiter server
sqliterConfig := sqliter.DefaultConfig()
sqliterConfig.ServeFolder = filepath.Join(app.DataDir(), "cache")
sqliterConfig.Verbose = true
sqliterServer := sqliter.NewServer(sqliterConfig)

log.Printf("[FLIGHT] SQLiter server initialized, serving from: %s", sqliterConfig.ServeFolder)
```

### 3.4: Mount SQLiter Routes

**Find the `app.OnServe().BindFunc` section and add BEFORE the banquet handler:**

```go
app.OnServe().BindFunc(func(se *core.ServeEvent) error {
    // ... existing setup ...
    
    // Mount SQLiter for data rendering
    // SQLiter handles everything from ColumnPath → Query
    se.Router.Any("/_/data", func(e *core.RequestEvent) error {
        sqliterServer.ServeHTTP(e.Response, e.Request)
        return nil
    })
    se.Router.Any("/_/data/*", func(e *core.RequestEvent) error {
        sqliterServer.ServeHTTP(e.Response, e.Request)
        return nil
    })
    
    log.Printf("[FLIGHT] SQLiter mounted at /_/data/")
    
    // ... rest of routes ...
})
```

### 3.5: Update Banquet Handler Call

**Find the banquet handler call (line ~134) and change:**

```go
// BEFORE
err := HandleBanquet(e, tw, tpl, true)

// AFTER
err := HandleBanquet(e, true)
```

**Verify:**
- [ ] SQLiter imported
- [ ] Template code removed
- [ ] SQLiter server created
- [ ] Routes mounted
- [ ] Handler call updated

---

## Step 4: Modify `banquethandler.go` (45 minutes)

### 4.1: Remove Template Import

**File:** `internal/flight/banquethandler.go`

**Remove:**
```go
// DELETE THIS LINE
import "html/template"
```

### 4.2: Update Function Signatures

**Change `HandleBanquet` (line ~35):**
```go
// BEFORE
func HandleBanquet(e *core.RequestEvent, tw *sqliter.TableWriter, tpl *template.Template, verbose bool) error

// AFTER
func HandleBanquet(e *core.RequestEvent, verbose bool) error
```

**Change `HandleLocalDataset` (line ~150):**
```go
// BEFORE
func HandleLocalDataset(e *core.RequestEvent, b *banquet.Banquet, tw *sqliter.TableWriter, tpl *template.Template, verbose bool) error

// AFTER
func HandleLocalDataset(e *core.RequestEvent, b *banquet.Banquet, verbose bool) error
```

### 4.3: Update Function Calls

**In `HandleBanquet`, find call to `HandleLocalDataset` (line ~51):**
```go
// BEFORE
return HandleLocalDataset(e, b, tw, tpl, verbose)

// AFTER
return HandleLocalDataset(e, b, verbose)
```

### 4.4: Replace ServeFromCache with Redirect (HandleBanquet)

**Find the call to `ServeFromCache` (line ~145) and replace:**

```go
// BEFORE
return ServeFromCache(cachePath, b, tw, tpl, e)

// AFTER
// Redirect to SQLiter for rendering
// SQLiter handles: ColumnPath → Query
relPath := strings.TrimPrefix(cachePath, e.App.DataDir()+"/cache/")
sqliterURL := fmt.Sprintf("/_/data/%s", relPath)

// Append table and query parts if present
if b.Table != "" {
    sqliterURL += "/" + b.Table
}
if b.ColumnPath != "" {
    sqliterURL += ";" + b.ColumnPath
}
if b.RawQuery != "" {
    sqliterURL += "?" + b.RawQuery
}

if verbose {
    log.Printf("[BANQUET] Redirecting to SQLiter: %s", sqliterURL)
}

return e.Redirect(302, sqliterURL)
```

### 4.5: Replace ServeFromCache with Redirect (HandleLocalDataset)

**Find the call to `ServeFromCache` (line ~294) and replace with same code:**

```go
// BEFORE
return ServeFromCache(cachePath, b, tw, tpl, e)

// AFTER
// Redirect to SQLiter for rendering
relPath := strings.TrimPrefix(cachePath, e.App.DataDir()+"/cache/")
sqliterURL := fmt.Sprintf("/_/data/%s", relPath)

if b.Table != "" {
    sqliterURL += "/" + b.Table
}
if b.ColumnPath != "" {
    sqliterURL += ";" + b.ColumnPath
}
if b.RawQuery != "" {
    sqliterURL += "?" + b.RawQuery
}

if verbose {
    log.Printf("[LOCAL] Redirecting to SQLiter: %s", sqliterURL)
}

return e.Redirect(302, sqliterURL)
```

**Verify:**
- [ ] Template import removed
- [ ] Function signatures updated
- [ ] Function calls updated
- [ ] Both redirects implemented

---

## Step 5: Delete `server.go` (2 minutes)

```bash
# Delete the file
rm internal/flight/server.go

# Verify it's gone
ls internal/flight/server.go
# Should show: No such file or directory
```

**Verify:**
- [ ] File deleted
- [ ] No references to `ServeFromCache` remain

---

## Step 6: Clean Up Imports (5 minutes)

**Run:**
```bash
# Auto-remove unused imports
goimports -w internal/flight/

# Or use gofmt
go fmt ./internal/flight/...
```

**Manually verify no unused imports in:**
- `internal/flight/flight.go`
- `internal/flight/banquethandler.go`

**Verify:**
- [ ] No unused imports
- [ ] Code formatted

---

## Step 7: Build and Test (30 minutes)

### 7.1: Build

```bash
go build -o flight3 .
```

**Expected:** No errors

**If errors:**
- Check for missing `sqliter` references
- Check for undefined `tw` or `tpl` variables
- Check import paths

### 7.2: Run Server

```bash
./flight3 serve
```

**Expected output:**
```
[FLIGHT] SQLiter server initialized, serving from: /path/to/pb_data/cache
[FLIGHT] SQLiter mounted at /_/data/
...
Server started at http://127.0.0.1:8090
```

### 7.3: Test Scenarios

**Test 1: Local CSV File**
```bash
# Create test file
echo "name,age\nAlice,30\nBob,25" > test.csv

# Request it
curl http://localhost:8090/test.csv
```

**Expected:**
- Flight3 converts to SQLite
- Redirects to `/_/data/test.csv.db/tb0`
- SQLiter serves React UI

**Test 2: Directory Listing**
```bash
curl http://localhost:8090/
```

**Expected:**
- Flight3 indexes directory
- Redirects to SQLiter
- Shows file listing

**Test 3: PocketBase Admin**
```bash
open http://localhost:8090/_/
```

**Expected:**
- PocketBase admin UI loads
- Can log in
- No errors

**Test 4: Rclone Config**
```bash
open http://localhost:8090/_/rclone/config
```

**Expected:**
- Rclone config UI loads
- No errors

**Verify:**
- [ ] Local files work
- [ ] Directory listing works
- [ ] PocketBase admin works
- [ ] Rclone config works
- [ ] No HTML in Flight3 logs
- [ ] SQLiter renders data

---

## Step 8: Verify No HTML in Flight3 (10 minutes)

**Run these checks:**

```bash
# Check for HTML imports
grep -r "html/template" internal/flight/*.go
# Expected: Only in handlers_auth.go and handlers_rclone_config.go

# Check for HTML tags
grep -r "<a href" internal/flight/*.go
# Expected: Only in handlers_auth.go

# Check for TableWriter
grep -r "TableWriter" internal/flight/*.go
# Expected: No results

# Check for template usage
grep -r "\.Execute" internal/flight/*.go
# Expected: Only in handlers_rclone_config.go (if any)
```

**Verify:**
- [ ] No HTML in core flight code
- [ ] Only PocketBase handlers have HTML
- [ ] No TableWriter references

---

## Step 9: Run Tests (if any) (15 minutes)

```bash
# Run all tests
go test ./...

# Run with verbose
go test -v ./internal/flight/...
```

**If tests fail:**
- Update test mocks to remove `tw`, `tpl` parameters
- Update test expectations for redirects

**Verify:**
- [ ] All tests pass
- [ ] No test failures

---

## Step 10: Commit Changes (5 minutes)

```bash
git add .
git status
# Review changes

git commit -m "Refactor: Delegate data rendering to SQLiter

- Remove HTML rendering from Flight3
- Mount SQLiter server at /_/data/
- Redirect to SQLiter after file conversion
- Delete server.go (173 lines)
- Update banquethandler.go to redirect
- Update flight.go to mount SQLiter
- Net: -172 lines

Flight3 now handles: Scheme → DataSetPath (resource acquisition)
SQLiter now handles: ColumnPath → Query (data querying)

See ResponsibilityBoundary.md for details."
```

**Verify:**
- [ ] Changes committed
- [ ] Good commit message

---

## Step 11: Final Verification (30 minutes)

### Checklist

**Architecture:**
- [ ] Flight3 has zero HTML rendering code
- [ ] SQLiter handles all data display
- [ ] Clear boundary: Scheme→DataSetPath vs ColumnPath→Query

**Functionality:**
- [ ] Local files convert and display
- [ ] Remote files (if configured) work
- [ ] Directory listings work
- [ ] Banquet queries work
- [ ] PocketBase admin works
- [ ] Rclone config works

**Code Quality:**
- [ ] No unused imports
- [ ] No compilation errors
- [ ] No runtime errors
- [ ] Logs are clean

**Documentation:**
- [ ] ResponsibilityBoundary.md is accurate
- [ ] ArchitectureSummary.md is accurate
- [ ] CleanUpTodo.md tasks completed

---

## Rollback Plan (if needed)

If something goes wrong:

```bash
# Switch back to backup
git checkout backup-before-sqliter-refactor

# Rebuild
go build -o flight3 .

# Run
./flight3 serve
```

---

## Success Criteria

✅ All items checked in Step 11  
✅ No errors in logs  
✅ Data displays correctly  
✅ ~172 lines removed  
✅ Clean architecture  

---

## Troubleshooting

### Issue: "undefined: sqliter.NewServer"

**Solution:**
```bash
go get github.com/darianmavgo/sqliter@latest
go mod tidy
```

### Issue: "404 Not Found" on data requests

**Solution:**
- Check SQLiter routes are mounted
- Check redirect URL format
- Check cache directory exists

### Issue: "Redirect loop"

**Solution:**
- Ensure redirect goes to `/_/data/` not back to Flight3
- Check SQLiter is handling `/_/data/*` routes

### Issue: PocketBase admin broken

**Solution:**
- Ensure PocketBase routes are BEFORE SQLiter routes
- Check `handlers_auth.go` is unchanged

---

## Post-Implementation

After successful implementation:

1. **Update documentation**
   - Update README.md
   - Document new architecture
   - Add examples

2. **Monitor production**
   - Watch logs for errors
   - Monitor performance
   - Collect user feedback

3. **Cleanup**
   - Delete backup branch (after confidence)
   - Archive old documentation

4. **Celebrate!** 🎉
   - Cleaner code
   - Better architecture
   - Easier maintenance

---

## Timeline Summary

| Step | Time | Cumulative |
|------|------|------------|
| 1. Backup | 5 min | 5 min |
| 2. Dependencies | 5 min | 10 min |
| 3. Modify flight.go | 30 min | 40 min |
| 4. Modify banquethandler.go | 45 min | 85 min |
| 5. Delete server.go | 2 min | 87 min |
| 6. Clean imports | 5 min | 92 min |
| 7. Build & Test | 30 min | 122 min |
| 8. Verify no HTML | 10 min | 132 min |
| 9. Run tests | 15 min | 147 min |
| 10. Commit | 5 min | 152 min |
| 11. Final verification | 30 min | 182 min |
| **Total** | **~3 hours** | **3 hours** |

**Note:** Original estimate was 7 hours, but with this guide it should be faster!

---

## Next Steps

Ready to start? Follow the steps in order. Good luck! 🚀
