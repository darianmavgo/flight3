# Flight3 HTML Cleanup TODO

## Overview
This document lists all HTML code in Flight3 that exists outside of PocketBase or SQLiter packages. These need to be removed or moved to appropriate packages.

**Goal:** Flight3 should have ZERO HTML rendering code. Only orchestration logic.

---

## HTML Code to Remove

### 1. ❌ `internal/flight/server.go` - ENTIRE FILE

**Current State:** 173 lines of HTML table rendering  
**Action:** DELETE entire file after migration to sqliter

**HTML Code:**
- Lines 6: `import "html/template"`
- Lines 18: Function signature with `*template.Template`
- Lines 50-60: HTML table header generation
- Lines 108-156: HTML link generation with icons (📁 📄 📊)
- Lines 160: `tw.WriteHTMLRow()`
- Lines 168: `tw.EndHTMLTable()`

**Replacement:**
```go
// DELETE THIS FILE
// Functionality moved to sqliter redirect in banquethandler.go
```

**Dependencies to Remove:**
- `html/template` import
- `sqliter.TableWriter` references
- All HTML string formatting

---

### 2. ❌ `internal/flight/banquethandler.go` - Remove HTML Dependencies

**Current State:** Uses `html/template` and `sqliter.TableWriter`  
**Action:** Remove template parameters, redirect to sqliter instead

**Lines to Change:**

**Line 5:**
```go
// REMOVE
import "html/template"
```

**Line 35:**
```go
// BEFORE
func HandleBanquet(e *core.RequestEvent, tw *sqliter.TableWriter, tpl *template.Template, verbose bool) error

// AFTER
func HandleBanquet(e *core.RequestEvent, verbose bool) error
```

**Line 51:**
```go
// BEFORE
return HandleLocalDataset(e, b, tw, tpl, verbose)

// AFTER
return HandleLocalDataset(e, b, verbose)
```

**Line 145:**
```go
// BEFORE
return ServeFromCache(cachePath, b, tw, tpl, e)

// AFTER
// Redirect to sqliter
relPath := strings.TrimPrefix(cachePath, e.App.DataDir()+"/cache/")
return e.Redirect(302, "/_/data/"+relPath)
```

**Line 150:**
```go
// BEFORE
func HandleLocalDataset(e *core.RequestEvent, b *banquet.Banquet, tw *sqliter.TableWriter, tpl *template.Template, verbose bool) error

// AFTER
func HandleLocalDataset(e *core.RequestEvent, b *banquet.Banquet, verbose bool) error
```

**Line 294:**
```go
// BEFORE
return ServeFromCache(cachePath, b, tw, tpl, e)

// AFTER
// Redirect to sqliter
relPath := strings.TrimPrefix(cachePath, e.App.DataDir()+"/cache/")
return e.Redirect(302, "/_/data/"+relPath)
```

**Extension Map (Lines 18-33):**
```go
// KEEP - This is for mksqlite converter detection, not HTML
var extensionMap = map[string]string{
    ".csv":      "csv",
    ".xlsx":     "excel",
    // ... etc
}
```

---

### 3. ❌ `internal/flight/flight.go` - Remove Template Initialization

**Current State:** Initializes sqliter templates  
**Action:** Remove template code, add sqliter server instead

**Lines 88-96:**
```go
// REMOVE
// Initialize SQLiter with Embedded Templates
// We use the templates embedded in the sqliter library.
tpl, err := sqliter.GetEmbeddedTemplates()
if err != nil {
    log.Fatal("Failed to load embedded templates from sqliter:", err)
}

// Initialize TableWriter with embedded templates
tw := sqliter.NewTableWriter(tpl, sqliter.DefaultConfig())
```

**REPLACE WITH:**
```go
// Initialize SQLiter server
sqliterConfig := sqliter.DefaultConfig()
sqliterConfig.ServeFolder = filepath.Join(app.DataDir(), "cache")
sqliterConfig.Verbose = true
sqliterServer := sqliter.NewServer(sqliterConfig)
```

**Line 134:**
```go
// BEFORE
err := HandleBanquet(e, tw, tpl, true)

// AFTER
err := HandleBanquet(e, true)
```

**Add after line 150 (after rclone routes):**
```go
// Mount SQLiter for data rendering
se.Router.Any("/_/data", func(e *core.RequestEvent) error {
    sqliterServer.ServeHTTP(e.Response, e.Request)
    return nil
})
se.Router.Any("/_/data/*", func(e *core.RequestEvent) error {
    sqliterServer.ServeHTTP(e.Response, e.Request)
    return nil
})
```

---

### 4. ⚠️ `internal/flight/handlers_auth.go` - KEEP (PocketBase UI)

**Current State:** 103 lines of HTML for auto-login  
**Action:** **KEEP** - This is PocketBase-related UI, not data rendering

**Justification:**
- This is authentication UI
- Part of PocketBase integration
- Not data table rendering
- Acceptable HTML in flight3

**Status:** ✅ NO CHANGES NEEDED

---

### 5. ⚠️ `internal/flight/handlers_rclone_config.go` - KEEP (PocketBase UI)

**Current State:** Serves `rclone_config.html` template  
**Action:** **KEEP** - This is configuration UI, not data rendering

**Lines 14-26:**
```go
// KEEP - This is config UI, not data rendering
func HandleRcloneConfigUI(e *core.RequestEvent) error {
    // Read template file
    templatePath := filepath.Join("templates", "rclone_config.html")
    htmlContent, err := os.ReadFile(templatePath)
    // ... serve HTML
}
```

**Justification:**
- This is configuration/admin UI
- Part of PocketBase integration
- Not data table rendering
- Acceptable HTML in flight3

**Status:** ✅ NO CHANGES NEEDED

---

### 6. ⚠️ `templates/rclone_config.html` - KEEP (PocketBase UI)

**Current State:** HTML template for rclone configuration  
**Action:** **KEEP** - This is admin UI

**Status:** ✅ NO CHANGES NEEDED

---

### 7. ❌ `internal/flight/converter.go` - Minor Cleanup

**Current State:** References HTML converter  
**Action:** Keep references (they're for mksqlite, not rendering)

**Lines 17, 63-64:**
```go
// KEEP - This is for mksqlite HTML file conversion, not rendering
_ "github.com/darianmavgo/mksqlite/converters/html"

case ".html", ".htm":
    driverName = "html"
```

**Justification:**
- This converts HTML files TO SQLite
- Not rendering HTML
- Part of mksqlite integration

**Status:** ✅ NO CHANGES NEEDED

---

## Summary of Changes

### Files to DELETE
- ❌ `internal/flight/server.go` (entire file - 173 lines)

### Files to MODIFY
- ⚠️ `internal/flight/banquethandler.go`
  - Remove `html/template` import
  - Remove `tw` and `tpl` parameters
  - Replace `ServeFromCache()` calls with redirects
  
- ⚠️ `internal/flight/flight.go`
  - Remove template initialization
  - Add sqliter server initialization
  - Mount sqliter routes
  - Update `HandleBanquet()` call

### Files to KEEP (No Changes)
- ✅ `internal/flight/handlers_auth.go` (PocketBase auth UI)
- ✅ `internal/flight/handlers_rclone_config.go` (PocketBase config UI)
- ✅ `templates/rclone_config.html` (PocketBase config template)
- ✅ `internal/flight/converter.go` (mksqlite integration)

---

## Verification Checklist

After cleanup, verify:

### Code Checks
- [ ] No `import "html/template"` in flight package (except handlers_*)
- [ ] No `sqliter.TableWriter` references
- [ ] No `tw.StartHTMLTable*()` calls
- [ ] No `tw.WriteHTMLRow()` calls
- [ ] No `tw.EndHTMLTable()` calls
- [ ] No HTML string generation in flight package
- [ ] No `<a href=` or HTML tags in flight package (except handlers_*)

### Functional Checks
- [ ] PocketBase admin UI works
- [ ] Auto-login works
- [ ] Rclone config UI works
- [ ] Data queries redirect to sqliter
- [ ] SQLiter renders tables correctly
- [ ] Local files work
- [ ] Remote files work
- [ ] Directory listings work
- [ ] Banquet URLs work

### Architecture Checks
- [ ] Flight3 only orchestrates (auth, rclone, conversion)
- [ ] SQLiter handles all data rendering
- [ ] PocketBase handles admin UI
- [ ] Clear separation of concerns

---

## Detailed Change Plan

### Step 1: Backup Current Code
```bash
cd /Users/darianhickman/Documents/flight3
git checkout -b remove-html-rendering
git add .
git commit -m "Backup before removing HTML rendering"
```

### Step 2: Modify flight.go
1. Remove lines 88-96 (template initialization)
2. Add sqliter server initialization
3. Add sqliter route mounting
4. Update HandleBanquet call (remove tw, tpl)

### Step 3: Modify banquethandler.go
1. Remove `html/template` import
2. Update `HandleBanquet` signature
3. Update `HandleLocalDataset` signature
4. Replace `ServeFromCache` calls with redirects

### Step 4: Delete server.go
```bash
rm internal/flight/server.go
```

### Step 5: Test
1. Build: `go build`
2. Run: `./flight serve`
3. Test all endpoints
4. Verify no HTML rendering in flight3

### Step 6: Commit
```bash
git add .
git commit -m "Remove HTML rendering from flight3, delegate to sqliter"
```

---

## Lines of Code Removed

| File | Lines Removed | Purpose |
|------|--------------|---------|
| `server.go` | 173 | Entire file deleted |
| `banquethandler.go` | ~10 | Remove template params |
| `flight.go` | ~10 | Remove template init |
| **Total** | **~193** | **HTML rendering removed** |

---

## Lines of Code Added

| File | Lines Added | Purpose |
|------|------------|---------|
| `flight.go` | ~15 | SQLiter server setup |
| `banquethandler.go` | ~6 | Redirect logic |
| **Total** | **~21** | **SQLiter integration** |

---

## Net Result

- **Removed:** ~193 lines of HTML rendering
- **Added:** ~21 lines of orchestration
- **Net:** -172 lines
- **Complexity:** Significantly reduced
- **Maintainability:** Greatly improved

---

## Post-Cleanup Architecture

```
Flight3 (Orchestration Only)
├── Auth (PocketBase UI) ✅ Has HTML
├── Rclone Config (PocketBase UI) ✅ Has HTML
├── File Conversion (mksqlite) ✅ No HTML
├── Cache Management ✅ No HTML
└── Redirect to SQLiter ✅ No HTML

SQLiter (Rendering Only)
├── React UI ✅ Has HTML
├── AG-Grid ✅ Has HTML
├── JSON API ✅ No HTML
└── SQL Execution ✅ No HTML

PocketBase (Admin Only)
└── Admin UI ✅ Has HTML
```

**Result:** Clean separation of concerns! 🎉
