# Refactor SQLiter Integration Plan (CORRECTED)

## Architecture Understanding

### Flight3's Role (NO HTML RENDERING)
Flight3 is a **data orchestration layer** that:
1. **Authenticates users** via PocketBase
2. **Manages rclone connections** to remote storage
3. **Caches files** from any source (local/remote)
4. **Converts files to SQLite** using mksqlite
5. **Passes SQLite files to sqliter** for rendering

**Flight3 does NOT render HTML tables** - it delegates that to sqliter!

### SQLiter's Role (RENDERS TO REACT/AG-GRID)
SQLiter:
1. **Receives SQLite database path** from flight3
2. **Executes SQL queries** using banquet
3. **Serves React UI** with AG-Grid
4. **Returns JSON data** to React client
5. **Handles all HTML/UI rendering**

### PocketBase's Role
- Admin UI (HTML)
- Authentication
- Database management

---

## Current Problem

Flight3 has **HTML rendering code that shouldn't exist**:

```go
// internal/flight/server.go - SHOULD NOT EXIST
tw.StartHTMLTableWithDebug(e.Response, columns, title, banquetDebug, query)
tw.WriteHTMLRow(e.Response, rowIndex, cells)
tw.EndHTMLTable(e.Response)
```

This violates the architecture where:
- ✅ PocketBase handles HTML (admin UI)
- ✅ SQLiter handles HTML (data rendering)
- ❌ Flight3 should NOT handle HTML (orchestration only)

---

## Correct Architecture

```
User Request
    ↓
[PocketBase] ← Flight3 uses for auth/admin
    ↓
[Flight3] ← Orchestrates everything
    ├─→ Authenticates user
    ├─→ Connects to rclone
    ├─→ Fetches file
    ├─→ Converts to SQLite (mksqlite)
    ├─→ Passes SQLite path to SQLiter
    ↓
[SQLiter Server] ← Renders UI
    ├─→ Receives SQLite DB path
    ├─→ Executes SQL queries
    ├─→ Serves React UI
    └─→ Returns JSON to AG-Grid
```

---

## What Needs to Change

### Flight3 Should:
1. **Prepare SQLite database** (already does this ✅)
2. **Pass database path to sqliter** (needs implementation)
3. **Proxy/redirect to sqliter** (needs implementation)
4. **Remove all HTML rendering** (needs cleanup)

### SQLiter Should:
1. **Accept database path** (already does via file serving ✅)
2. **Render React UI** (already does ✅)
3. **Serve JSON to AG-Grid** (already does ✅)
4. **Export reusable server** (needs minor changes)

---

## Integration Pattern

### Option 1: HTTP Proxy (RECOMMENDED)

Flight3 proxies requests to embedded sqliter server:

```go
// internal/flight/flight.go
import "github.com/darianmavgo/sqliter/sqliter"

// Start sqliter server
sqliterConfig := sqliter.DefaultConfig()
sqliterConfig.ServeFolder = filepath.Join(app.DataDir(), "cache")
sqliterServer := sqliter.NewServer(sqliterConfig)

// Mount sqliter at /_/data/
se.Router.Any("/_/data/*", func(e *core.RequestEvent) error {
    // Proxy to sqliter server
    sqliterServer.ServeHTTP(e.Response, e.Request)
    return nil
})
```

**Flow:**
1. User requests `/myfile.csv`
2. Flight3 converts to `/cache/myfile.csv.db`
3. Flight3 redirects to `/_/data/myfile.csv.db`
4. SQLiter serves React UI
5. React fetches data via SQLiter's JSON API

**Benefits:**
- ✅ Clean separation
- ✅ No HTML in flight3
- ✅ SQLiter handles all rendering
- ✅ Easy to test independently

---

### Option 2: Embedded Server (Alternative)

Flight3 embeds sqliter as a library:

```go
// internal/flight/banquethandler.go
func HandleBanquet(e *core.RequestEvent, sqliterServer *sqliter.Server) error {
    // 1. Convert file to SQLite (already does this)
    cachePath := GetCachePath(...)
    
    // 2. Redirect to sqliter
    redirectURL := fmt.Sprintf("/_/data/%s", filepath.Base(cachePath))
    return e.Redirect(302, redirectURL)
}
```

**Benefits:**
- ✅ Single binary
- ✅ Shared cache directory
- ✅ No separate ports

**Drawbacks:**
- ⚠️ Tighter coupling
- ⚠️ Harder to update sqliter independently

---

## Implementation Plan

### Phase 1: Prepare SQLiter for Embedding

**File: `sqliter/sqliter/server.go`**

Already done! ✅ The server implements `http.Handler`:

```go
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Already implemented
}
```

**No changes needed** - sqliter is already embeddable.

---

### Phase 2: Update Flight3 to Use SQLiter

**File: `flight3/internal/flight/flight.go`**

```go
import "github.com/darianmavgo/sqliter/sqliter"

func Flight() {
    // ... existing PocketBase setup ...
    
    // Initialize SQLiter server
    sqliterConfig := sqliter.DefaultConfig()
    sqliterConfig.ServeFolder = filepath.Join(app.DataDir(), "cache")
    sqliterConfig.Verbose = true
    sqliterServer := sqliter.NewServer(sqliterConfig)
    
    app.OnServe().BindFunc(func(se *core.ServeEvent) error {
        // ... existing setup ...
        
        // Mount SQLiter at /_/data/
        se.Router.Any("/_/data", func(e *core.RequestEvent) error {
            sqliterServer.ServeHTTP(e.Response, e.Request)
            return nil
        })
        se.Router.Any("/_/data/*", func(e *core.RequestEvent) error {
            sqliterServer.ServeHTTP(e.Response, e.Request)
            return nil
        })
        
        // Banquet handler now redirects to sqliter
        banquetHandler := func(e *core.RequestEvent) error {
            // ... existing auth/conversion logic ...
            
            // After conversion, redirect to sqliter
            dbPath := GetCachePath(...)
            relPath := strings.TrimPrefix(dbPath, app.DataDir()+"/cache/")
            redirectURL := "/_/data/" + relPath
            return e.Redirect(302, redirectURL)
        }
        
        // ... rest of setup ...
    })
}
```

**Changes:**
1. ✅ Import sqliter
2. ✅ Create sqliter server instance
3. ✅ Mount at `/_/data/`
4. ✅ Redirect to sqliter after conversion
5. ❌ Remove all HTML rendering code

---

### Phase 3: Remove HTML from Flight3

See `CleanUpTodo.md` for detailed list.

---

## Communication Contract

### Flight3 → SQLiter

**What Flight3 Provides:**
```go
// 1. SQLite database in cache directory
cachePath := "/path/to/pb_data/cache/myfile.csv.db"

// 2. HTTP redirect to sqliter
redirectURL := "/_/data/myfile.csv.db"
e.Redirect(302, redirectURL)
```

**What SQLiter Receives:**
```
GET /_/data/myfile.csv.db
```

SQLiter automatically:
- Serves React UI
- Parses banquet URL from path
- Executes SQL queries
- Returns JSON to AG-Grid

---

### SQLiter → React Client

**SQLiter Provides:**
```json
{
  "columns": ["name", "size", "modified"],
  "rows": [
    {"name": "file1.txt", "size": "1024", "modified": "2024-01-30"},
    {"name": "file2.txt", "size": "2048", "modified": "2024-01-30"}
  ],
  "totalCount": 2,
  "sql": "SELECT * FROM tb0 LIMIT 100"
}
```

**React/AG-Grid Consumes:**
- Displays data in grid
- Handles sorting/filtering
- Implements infinite scroll

---

## Filesystem Converter - No Changes Needed

**Current Usage (CORRECT):**

1. **mksqlite converter** - Converts local files
   - Used by: `flight3/internal/flight/converter.go`
   - Purpose: Local filesystem → SQLite
   
2. **flight3 IndexDirectory()** - Indexes remote directories
   - Used by: `flight3/internal/flight/rclone_manager.go`
   - Purpose: Remote rclone VFS → SQLite
   - Creates compatible `tb0` schema

**No conflicts** - both produce compatible schemas.

---

## Migration Steps

### Step 1: Update flight3/go.mod
```bash
cd /Users/darianhickman/Documents/flight3
go get github.com/darianmavgo/sqliter@latest
```

### Step 2: Modify flight3/internal/flight/flight.go
- Import sqliter
- Create sqliter server
- Mount at `/_/data/`

### Step 3: Modify flight3/internal/flight/banquethandler.go
- Remove HTML rendering
- Add redirect to sqliter

### Step 4: Remove HTML code (see CleanUpTodo.md)
- Delete `server.go` HTML rendering
- Remove template dependencies
- Clean up imports

### Step 5: Test
- Test local files
- Test remote files
- Test directory listings
- Test banquet queries

---

## Benefits

### For Flight3
- ✅ Removes ~200 lines of HTML code
- ✅ Focuses on orchestration only
- ✅ Easier to maintain
- ✅ Clear separation of concerns

### For SQLiter
- ✅ Becomes the single source of truth for rendering
- ✅ Can evolve UI independently
- ✅ Better tested
- ✅ Reusable across projects

### For Users
- ✅ Better UI (React + AG-Grid)
- ✅ Faster rendering
- ✅ More features (sorting, filtering, etc.)
- ✅ Consistent experience

---

## Timeline

- **Step 1-2**: 1 hour (setup sqliter in flight3)
- **Step 3**: 2 hours (refactor banquet handler)
- **Step 4**: 2 hours (remove HTML code)
- **Step 5**: 2 hours (testing)
- **Total**: 7 hours

---

## Success Criteria

✅ Flight3 has ZERO HTML rendering code  
✅ All data rendering goes through sqliter  
✅ PocketBase admin UI still works  
✅ Banquet URLs work correctly  
✅ Local and remote files work  
✅ Directory listings work  
✅ Tests pass  

---

## Next Steps

1. Review this plan
2. Review `CleanUpTodo.md` for HTML cleanup
3. Implement Step 1 (update go.mod)
4. Implement Step 2 (mount sqliter)
5. Implement Step 3 (redirect logic)
6. Execute cleanup from `CleanUpTodo.md`
7. Test thoroughly
8. Deploy
