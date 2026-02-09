# Client-Side Logic That Should Be Server-Side

Analysis of SQLiter (Dart) and Flight3 (Go) codebases to identify opportunities for moving logic server-side to reduce client-side complexity and async state management issues.

---

## **1. RecentFilesService → PocketBase Collection** ⭐️ HIGHEST PRIORITY

**Current Implementation:**
- **Location:** `sqliter/lib/services/recent_files_service.dart`
- **Storage:** `SharedPreferences` (local JSON)
- **Complexity:** 158 lines of Dart code with async state management
- **Operations:** CRUD on recent files, sorting, cleanup, validation

**Why Move Server-Side:**
- Recent files should be **per-user, cross-device**
- Currently stored locally, lost if you switch machines
- Validation logic (checking if files still exist) should be server-side
- PocketBase already has auth → perfect for user-scoped data

**Server-Side Design:**
```go
// New collection: recent_files
type RecentFile struct {
    ID            string    `json:"id"`
    UserID        string    `json:"user"`           // Relation to _superusers
    Path          string    `json:"path"`
    Name          string    `json:"name"`
    LastOpened    time.Time `json:"last_opened"`
    WasConverted  bool      `json:"was_converted"`
    OriginalFormat string   `json:"original_format"`
    Created       time.Time `json:"created"`
    Updated       time.Time `json:"updated"`
}
```

**Client Becomes:**
```dart
// GET /api/collections/recent_files/records?filter=(user='USER_ID')&sort=-last_opened
Future<List<RecentFile>> getRecentFiles() async {
  final response = await _flightService.get('/api/collections/recent_files/records');
  return response.map((json) => RecentFile.fromJson(json)).toList();
}

// POST /api/collections/recent_files/records
Future<void> addRecentFile(String path) async {
  await _flightService.post('/api/collections/recent_files/records', {
    'path': path,
    'name': basename(path),
    'last_opened': DateTime.now().toIso8601String(),
  });
}
```

**Lines Removed:** ~158 Dart → ~30 Dart
**State Complexity:** Eliminated (no more SharedPreferences, JSON serialization, cleanup logic)

---

## **2. CacheService → Flight3 Server-Side Cache** ⭐️ HIGH PRIORITY

**Current Implementation:**
- **Location:** `sqliter/lib/cache_service.dart`
- **Storage:** Local filesystem (`~/Library/Application Support/...`)
- **Complexity:** 208 lines with cache key generation, expiration, cleanup
- **Problem:** Each client has its own cache, wastes space, no sharing

**Why Move Server-Side:**
```
User A converts file.csv → Flight3 caches result
User B opens same file.csv → Flight3 serves cached version instantly
```

**Server-Side Design:**
Flight3 already has rclone cache directory:
```go
// In internal/flight/cache.go (already exists!)
type ConversionCache struct {
    SourcePath     string
    SourceModTime  time.Time
    CachedDBPath   string
    ExpiresAt      time.Time
}

// Just need to expose management endpoints
// GET /api/cache/stats
// DELETE /api/cache/clear
// DELETE /api/cache/entry?path=...
```

**Client Becomes:**
```dart
// Just call Flight3's /api/convert
// Server handles caching automatically
Future<File> convertFile(File file) async {
  final response = await _flightService.uploadForConversion(file);
  return response; // Server already checked cache, did conversion if needed
}
```

**Lines Removed:** ~208 Dart → ~20 Dart
**Benefits:** 
- Shared cache across all clients
- Server can use better cache invalidation strategies
- Client doesn't manage filesystem state

---

## **3. PathValidator → Flight3 Validation API** ⭐️ MEDIUM PRIORITY

**Current Implementation:**
- **Location:** `sqliter/lib/utils/path_validator.dart`
- **Complexity:** 83 lines of path parsing, tilde expansion, backwards validation
- **Problem:** Duplicates logic that should be standardized server-side

**Why Move Server-Side:**
- Path validation rules should be consistent across all clients
- Server has canonical view of what paths are accessible
- Tilde expansion is environment-specific (server knows user context)

**Server-Side Design:**
```go
// GET /api/validate-path?path=/foo/bar
type PathValidationResponse struct {
    Valid         bool     `json:"valid"`
    ExpandedPath  string   `json:"expanded_path"`
    Segments      []string `json:"segments"`
    ValidSegments []string `json:"valid_segments"`
    BreakPoint    string   `json:"break_point,omitempty"`
    ErrorMessage  string   `json:"error_message,omitempty"`
}
```

**Client Becomes:**
```dart
Future<PathValidationResult> validatePath(String path) async {
  final response = await _flightService.get('/api/validate-path?path=${Uri.encodeComponent(path)}');
  return PathValidationResult.fromJson(response);
}
```

**Lines Removed:** ~83 Dart → ~15 Dart
**Benefits:**
- Server can apply security policies (e.g., prevent access outside certain dirs)
- Consistent path handling across Web/Desktop/Mobile clients

---

## **4. ConversionService Logic → Pure HTTP Client** ⭐️ MEDIUM PRIORITY

**Current Implementation:**
- **Location:** `sqliter/lib/conversion_service.dart`
- **Complexity:** 202 lines including format detection, caching, error handling
- **Problem:** Too much business logic in the client

**Already Partially Server-Side:**
- Flight3 already has `/api/convert` endpoint
- Client just needs to upload and receive result

**What Can Be Simplified:**
```dart
// Current: Complex logic about supported formats, cache checking, etc.
// Future: Just send file, server handles everything

class ConversionService {
  Future<File> ensureSqlite(File file) async {
    // Check if already SQLite
    if (file.path.endsWith('.db')) return file;
    
    // Send to server, it handles:
    // - Format detection
    // - Cache checking
    // - Conversion
    // - Error messages
    return await _flightService.uploadForConversion(file);
  }
}
```

**Lines Removed:** ~202 → ~50 (just HTTP client calls)

---

## **5. DatabaseService → Query Proxy Through Flight3** ⭐️ LOW PRIORITY (Keep Local)

**Current Implementation:**
- **Location:** `sqliter/lib/db_service.dart`
- **Complexity:** 65 lines of SQLite FFI direct access
- **Decision:** **KEEP LOCAL FOR NOW**

**Why Keep Client-Side:**
- Local DB access is fast and doesn't require network
- Offline mode support
- Grid scrolling needs low-latency data fetching

**Hybrid Approach:**
```dart
// Local mode: Direct SQLite FFI (current)
// Remote mode: Proxy through Flight3 Banquet API
class DatabaseService {
  Future<List<Map>> fetchRows(String table, {int limit, int offset}) async {
    if (_isLocal) {
      return _db!.rawQuery('SELECT * FROM "$table" LIMIT $limit OFFSET $offset');
    } else {
      // Use Flight3 Banquet API
      return _flightService.banquetQuery('/$dbPath;$table[$offset:$offset+$limit]');
    }
  }
}
```

---

## **Summary Table**

| Component | Current Lines | After Migration | State Complexity | Priority |
|-----------|--------------|-----------------|------------------|----------|
| RecentFilesService | 158 | 30 | High → None | ⭐️⭐️⭐️ |
| CacheService | 208 | 20 | High → None | ⭐️⭐️⭐️ |
| PathValidator | 83 | 15 | Medium → None | ⭐️⭐️ |
| ConversionService | 202 | 50 | Medium → Low | ⭐️⭐️ |
| DatabaseService | 65 | 65 (keep) | Low | N/A |

**Total Lines Removed:** ~651 Dart lines → ~180 lines
**Net Reduction:** ~470 lines of client-side async state hell

---

## **Additional Opportunities**

### **6. Home Dashboard State → Server-Side Rendering**
The `HomeDashboard` widget currently assembles UI from multiple sources:
- Recent files
- Cache stats
- Flight connection status

**Could become:**
```go
// GET /api/dashboard
{
  "recent_files": [...],
  "cache_stats": {...},
  "connected": true,
  "recommended_actions": [...]
}
```

Client just renders JSON.

---

### **7. Breadcrumb Navigation → Server-Side Path Parsing**
Currently, breadcrumb parsing happens in `BreadcrumbPathField`:
- Tilde expansion
- Banquet URL parsing (`;table` suffix)
- Segment existence checking

**Move to Flight3:**
```go
// GET /api/parse-path?url=~/foo/bar.db;table
{
  "segments": [
    {"text": "~", "path": "/Users/darian", "exists": true},
    {"text": "foo", "path": "/Users/darian/foo", "exists": true},
    {"text": "bar.db", "path": "/Users/darian/foo/bar.db", "exists": true},
    {"text": "table", "type": "banquet_table", "exists": true}
  ]
}
```

---

## **Implementation Order**

1. **RecentFilesService** → PocketBase (Immediate win, high user value)
2. **CacheService** → Flight3 server cache (Performance + storage savings)
3. **PathValidator** → Flight3 API (Consistency + security)
4. **ConversionService** → Simplify to pure HTTP (Maintainability)
5. **Breadcrumb parsing** → Flight3 API (Polish)
6. **Dashboard state** → Server-rendered (If doing HTMX pivot)

---

## **The Nuclear Option: HTMX**

If you truly want to eliminate client-side state management:

```html
<!-- Flight3 serves this HTML -->
<div hx-get="/api/recent-files" hx-trigger="load">
  <!-- Server renders the list -->
</div>

<div hx-post="/api/open-file" hx-include="#path-input">
  <!-- Form submission handled by server -->
</div>
```

The client becomes a dumb terminal. All state lives in Go.

---

**Want me to start with #1 (RecentFilesService → PocketBase)?**
