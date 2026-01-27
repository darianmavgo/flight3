# Rclone + PocketBase Integration Implementation Plan

**Goal**: Integrate rclone's file fetching and VFS caching capabilities with PocketBase-managed configuration to enable dynamic, multi-cloud data access in Flight3.

**Based on**: `RCLONE_POCKETBASE.md` and existing `managepocketbase.go`

---

## Phase 1: Core Infrastructure

### 1.1 Create Rclone Manager Module
**File**: `internal/flight/rclone_manager.go`

**Purpose**: Centralize rclone VFS initialization and management.

**Key Components**:
- `RcloneManager` struct to maintain VFS cache map
- `InitRclone(cacheDir string)` - Initialize global rclone cache directory
- `GetVFS(remoteRecord *core.Record) (*vfs.VFS, error)` - Get or create VFS instance
- `generateVFSHash(remoteConfig map[string]interface{}) string` - Create unique hash for VFS caching

**Implementation Details**:
```go
type RcloneManager struct {
    vfsCache    map[string]*vfs.VFS
    cacheDir    string
    mu          sync.RWMutex
}

// VFS Configuration (from RCLONE.md):
// - CacheMode: vfscommon.CacheModeFull (for random access)
// - DirCacheTime: 10 minutes
// - CacheMaxAge: 24 hours
// - CachePollInterval: 1 minute
// - ChunkSize: 128 MB
```

**Dependencies**:
- `github.com/rclone/rclone/fs`
- `github.com/rclone/rclone/vfs`
- `github.com/rclone/rclone/vfs/vfscommon`

---

### 1.2 Enhance PocketBase Collections
**File**: `internal/flight/managepocketbase.go`

**Modifications to `EnsureRcloneRemotes`**:
- Add `vfs_settings` JSON field (optional) for per-remote VFS tuning
- Add `enabled` boolean field (default: true)
- Add `description` text field for documentation

**Example Schema**:
```go
collection.Fields.Add(&core.TextField{Name: "name", Required: true})
collection.Fields.Add(&core.TextField{Name: "type", Required: true})
collection.Fields.Add(&core.JSONField{Name: "config"})
collection.Fields.Add(&core.JSONField{Name: "vfs_settings"}) // NEW
collection.Fields.Add(&core.BoolField{Name: "enabled"})      // NEW
collection.Fields.Add(&core.TextField{Name: "description"})  // NEW
```

---

## Phase 2: Data Fetching Integration

### 2.1 Remote Lookup Function
**File**: `internal/flight/rclone_manager.go`

**Function**: `LookupRemote(app core.App, hostname string) (*core.Record, error)`

**Purpose**: Query PocketBase for remote configuration by hostname.

**Logic**:
1. Query `rclone_remotes` collection where `name = hostname`
2. Check `enabled` field
3. Return record or error if not found/disabled

---

### 2.2 File Fetcher
**File**: `internal/flight/rclone_manager.go`

**Function**: `FetchFile(vfs *vfs.VFS, remotePath string, localCachePath string) error`

**Purpose**: Download file from remote to local cache using VFS.

**Logic**:
1. Open file via `vfs.OpenFile(remotePath, os.O_RDONLY, 0)`
2. Create local cache file
3. Copy contents (VFS handles caching internally with CacheModeFull)
4. Return path to cached file

---

## Phase 3: Cache Management

### 3.1 Update Cache Key Generation
**File**: `internal/flight/cache.go`

**Current**: `GenCacheKey(b *banquet.Banquet) string`

**Enhancement**: Add MD5 hash of remote config to ensure cache isolation per credential set.

**New Signature**:
```go
func GenCacheKey(b *banquet.Banquet, remoteConfigHash string) string {
    userInfo := ""
    if b.User != nil {
        userInfo = b.User.String()
    }
    parts := []string{userInfo, b.Hostname(), b.DataSetPath, remoteConfigHash}
    return strings.Join(parts, "-")
}
```

---

### 3.2 Cache Validation
**File**: `internal/flight/cache.go`

**Function**: `ValidateCache(cachePath string, ttl float64) (bool, error)`

**Purpose**: Check if cached SQLite file is still valid based on TTL.

**Logic**:
1. Check file existence
2. Get file modification time
3. Compare with current time and TTL
4. Return true if valid, false if expired

---

## Phase 4: Pipeline Orchestration

### 4.1 Update HandleBanquet
**File**: `internal/flight/banquethandler.go`

**Current State**: Placeholder with cache key generation.

**New Implementation Flow**:

```go
func HandleBanquet(e *core.RequestEvent, tw *sqliter.TableWriter, tpl *template.Template, verbose bool) error {
    // 1. Parse Banquet URL
    b, err := banquet.ParseNested(e.Request.RequestURI)
    if err != nil {
        return err
    }

    // 2. Lookup Remote Configuration
    remoteRecord, err := LookupRemote(e.App, b.Hostname())
    if err != nil {
        return fmt.Errorf("remote not found: %w", err)
    }

    // 3. Initialize VFS
    vfs, err := rcloneManager.GetVFS(remoteRecord)
    if err != nil {
        return fmt.Errorf("failed to init VFS: %w", err)
    }

    // 4. Generate Cache Key
    remoteConfigHash := generateVFSHash(remoteRecord.Get("config"))
    cacheKey := GenCacheKey(b, remoteConfigHash)
    cachePath := filepath.Join(e.App.DataDir(), "cache", cacheKey+".db")

    // 5. Check Cache Validity
    valid, _ := ValidateCache(cachePath, 1440) // 24 hours default
    
    // 6. Fetch and Convert if Cache Miss
    if !valid {
        // 6a. Fetch raw file via rclone VFS
        rawFilePath := filepath.Join(e.App.DataDir(), "temp", cacheKey+filepath.Ext(b.DataSetPath))
        if err := FetchFile(vfs, b.DataSetPath, rawFilePath); err != nil {
            return err
        }

        // 6b. Convert to SQLite using mksqlite
        if err := ConvertToSQLite(rawFilePath, cachePath); err != nil {
            return err
        }

        // 6c. Cleanup temp file
        os.Remove(rawFilePath)
    }

    // 7. Serve from Cache
    return ServeFromCache(cachePath, b, tw, tpl)
}
```

---

### 4.2 SQLite Conversion Function
**File**: `internal/flight/converter.go` (NEW)

**Function**: `ConvertToSQLite(sourcePath, destPath string) error`

**Purpose**: Wrapper around mksqlite to convert files to SQLite.

**Logic**:
1. Detect file type from extension
2. Call appropriate mksqlite converter
3. Write output to destPath

---

### 4.3 Data Serving Function
**File**: `internal/flight/server.go` (NEW)

**Function**: `ServeFromCache(cachePath string, b *banquet.Banquet, tw *sqliter.TableWriter, tpl *template.Template) error`

**Purpose**: Open cached SQLite DB and serve query results.

**Logic**:
1. Open SQLite database at cachePath
2. Build SQL query from banquet fields (Table, Select, Where, OrderBy, Limit, Offset)
3. Execute query
4. Stream results via TableWriter

---

## Phase 5: Error Handling & Logging

### 5.1 Structured Logging
- Add verbose logging for each phase (fetch, convert, serve)
- Log VFS cache hits/misses
- Log remote lookup failures with helpful messages

### 5.2 Error Recovery
- Implement retry logic for transient rclone errors
- Fallback to re-fetch if cache file is corrupted
- Clear cache entry on conversion failure

---

## Phase 6: Testing & Validation

### 6.1 Unit Tests
**File**: `internal/flight/rclone_manager_test.go`

**Test Cases**:
- VFS initialization with different remote types (S3, GCS, R2)
- Cache key generation consistency
- Cache validation with various TTLs

### 6.2 Integration Tests
**File**: `tests/rclone_pocketbase_test.go`

**Test Scenarios**:
1. Fetch CSV from R2, convert, serve
2. Fetch Excel from GCS, convert, serve
3. Cache hit scenario (second request)
4. Cache expiration and refresh
5. Invalid remote handling

---

## Phase 7: Documentation & Migration

### 7.1 Update Existing Docs
- Update `RCLONE.md` with PocketBase integration details
- Add examples to `README.md`

### 7.2 Migration Guide
**File**: `docs/MIGRATION_TO_POCKETBASE_RCLONE.md`

**Contents**:
- How to populate `rclone_remotes` collection
- Example configurations for common providers
- How to migrate from static rclone.conf

---

## Implementation Order

1. **Week 1**: Phase 1 (Core Infrastructure)
   - Create `rclone_manager.go` with VFS caching
   - Enhance PocketBase collections

2. **Week 2**: Phase 2 & 3 (Fetching & Caching)
   - Implement remote lookup
   - Implement file fetcher
   - Update cache key generation

3. **Week 3**: Phase 4 (Pipeline Orchestration)
   - Update `HandleBanquet` with full flow
   - Create converter and server modules

4. **Week 4**: Phase 5 & 6 (Error Handling & Testing)
   - Add logging and error recovery
   - Write comprehensive tests

5. **Week 5**: Phase 7 (Documentation)
   - Update all docs
   - Create migration guide

---

## Dependencies to Add

```go
require (
    github.com/rclone/rclone v1.72.1  // Already present
    // Ensure these rclone sub-packages are accessible:
    // - github.com/rclone/rclone/fs
    // - github.com/rclone/rclone/vfs
    // - github.com/rclone/rclone/vfs/vfscommon
)
```

---

## Success Criteria

- [ ] Can fetch files from S3, GCS, and R2 using PocketBase-stored credentials
- [ ] VFS cache reuses connections for same remote config
- [ ] Cache invalidation works correctly based on TTL
- [ ] All tests pass
- [ ] Documentation is complete and accurate
- [ ] No hardcoded credentials in codebase
