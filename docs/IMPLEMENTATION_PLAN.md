# Implementation Plan for Design Proposal

**Created**: 2026-02-07  
**Based on**: DESIGN_PROPOSAL.md  
**Status**: Ready to Execute

---

## Phase 1: Foundation (Current Focus)

### 1.1 Flight3: Add `/api/convert` Endpoint ✅ COMPLETED

**Files modified:**
- ✅ `flight3/internal/flight/handlers_convert.go` - **Created** handler function
- ✅ `flight3/internal/flight/router.go` - **Added** route
- ✅ Uses existing `flight3/internal/flight/converter.go` - Conversion logic with mksqlite

**Implementation:**
```go
// Added to router.go (line 37-38)
se.Router.POST("/api/convert", HandleConvertToSQLite)

// Created handlers_convert.go
func HandleConvertToSQLite(e *core.RequestEvent) error {
    // Accept file upload via multipart/form-data
    // Validate file size (100MB limit)
    // Save to temp directory
    // Detect file type from extension
    // Call ConvertToSQLite (existing function from converter.go)
    // Serve SQLite file back to client
    // Returns proper error codes for various failure scenarios
}
```

**Features:**
- ✅ Accepts file uploads via POST /api/convert
- ✅ Supports: CSV, Excel (.xlsx, .xls), JSON, HTML, Markdown, TXT, ZIP
- ✅ Returns SQLite files directly if already SQLite
- ✅ 100MB file size limit
- ✅ Proper error handling with JSON error responses
- ✅ Detailed logging for debugging
- ✅ Uses existing mksqlite converters

**Testing:**
- ✅ Created test CSV file: `/tmp/test_data.csv`
- ✅ Created test script: `test_conversion.sh`
- ✅ Build successful: `./flight3` binary created

**Next:** Run Flight3 and test the endpoint with the test script

### 1.2 SQLiter: Create ConversionService ✅ COMPLETED

**Files created:**
- ✅ `sqliter/lib/conversion_service.dart` - **Created** main conversion logic  
- ✅ `sqliter/lib/cache_service.dart` - **Created** cache management
- ✅ `sqliter/pubspec.yaml` - **Modified** added path_provider dependency

**Files modified:**
- ✅ `sqliter/lib/main.dart` - **Integrated** ConversionService into file handling

**Features:**
- ✅ Automatic file type detection
- ✅ Local cache management with expiration
- ✅ Flight3 API integration
- ✅ Graceful degradation when offline
- ✅ User-friendly error messages
- ✅ Seamless integration with existing file opening flow

**Cache Features:**
- Cache directory: Application Support/conversions/
- Cache keys: Based on file path + modification time
- Expiration: 7 days default
- Auto-cleanup: Removes files older than 30 days

**Next:** Test the conversion flow end-to-end

### 1.3 SQLiter: Enhanced Home Dashboard

**Files to create:**
- `sqliter/lib/pages/home_dashboard.dart` - New dashboard page
- `sqliter/lib/models/recent_file.dart` - Recent file model
- `sqliter/lib/services/recent_files_service.dart` - Track recent files

**Sections:**
1. Quick Actions (Browse, Flight Server, New Query, Import)
2. Recent Files (last 10-15 with timestamps)
3. Flight3 Connections (status, connect/disconnect)
4. Cached Datasets (view, manage)

### 1.4 SQLiter: Pagination Stats Header

**Files to modify:**
- `sqliter/lib/pages/db_viewer_page.dart` - Add stats header above grid

**Features:**
- Show: "📊 {table_name} • {total_rows} rows • Showing {start}-{end}"
- Add "Jump to Row" button
- Keep existing infinite scroll

---

## Phase 2: Grid Enhancements

### 2.1 Enable TrinaGrid Features

**Files to modify:**
- `sqliter/lib/pages/db_viewer_page.dart`

**Features to enable:**
```dart
// Column sorting
enableSorting: true

// Column resizing
resizeMode: TrinaResizeMode.normal

// Frozen first column
TrinaColumn(frozen: TrinaColumnFrozen.start)

// CSV Export
String csv = stateManager.exportToCsv()
```

### 2.2 Add Jump to Row Dialog

**Files to create:**
- `sqliter/lib/dialogs/jump_to_row_dialog.dart`

---

## Phase 3: Caching & Offline Support

### 3.1 Local Cache Management

**Files to create:**
- `sqliter/lib/services/cache_manager.dart`

**Features:**
- Cache converted files
- Cache Flight3 datasets
- Show cache size
- Clear cache option

### 3.2 Flight3 Connection State

**Files to modify:**
- `sqliter/lib/services/flight3_service.dart`

**Features:**
- Connection status (connected, disconnected, connecting)
- Auto-detect local server
- Save multiple server configurations

---

## Execution Order

### Step 1: Flight3 Conversion Endpoint ✓ START HERE
1. Create `flight3/handlers.go::HandleConvertToSQLite()`
2. Add route in `flight3/router.go`
3. Test with curl/Postman

### Step 2: SQLiter Conversion Service
1. Create `ConversionService`
2. Create `CacheService`
3. Integrate with file opening flow

### Step 3: Home Dashboard
1. Create dashboard UI
2. Implement recent files tracking
3. Add quick actions
4. Add Flight3 connection panel

### Step 4: Grid Improvements
1. Add stats header
2. Enable sorting/resizing
3. Add jump to row
4. Add CSV export

### Step 5: Polish
1. Add keyboard shortcuts
2. Performance testing
3. User testing

---

## Questions to Resolve

1. **Cache Location**: Where should converted files be cached?
   - Suggestion: `~/.sqliter/cache/` or app data directory

2. **Cache Expiration**: How long should cached conversions last?
   - Suggestion: 7 days, or based on source file modification time

3. **Multiple Flight3 Servers**: Support multiple server configs?
   - Suggestion: Yes, with primary/secondary designation

4. **Offline Mode**: How to handle when Flight3 is down?
   - Suggestion: Show helpful error, fall back to cached files

---

## Success Criteria

- [ ] Can convert CSV to SQLite via Flight3
- [ ] Converted files are cached locally
- [ ] Home dashboard shows recent files
- [ ] Can connect/disconnect from Flight3
- [ ] Grid shows pagination stats
- [ ] Can sort/resize columns
- [ ] Can export to CSV
- [ ] Can jump to specific row

---

## Next Actions

**Immediate:**
1. Review this plan with user
2. Start with Flight3 `/api/convert` endpoint
3. Create basic ConversionService in SQLiter
4. Test end-to-end conversion flow

**Then:**
5. Build Home Dashboard UI
6. Add Grid enhancements
7. Implement caching
8. Polish and test
