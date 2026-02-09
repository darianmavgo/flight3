# SQLiter + Flight3 Design Proposal

## Core Principles

### 1. Clean Separation of Concerns
- **SQLiter**: Pure Flutter app - UI, local file access, grid display
- **Flight3**: Pure Go/PocketBase app - remote sync, conversion, caching

### 2. Communication Protocol
Use simple HTTP API between Sqliter and Flight3 (when Flight3 is available)

---

## Feature 1: Smart File Conversion Workflow

### Architecture

```
User opens non-SQLite file in Sqliter
    ↓
Sqliter detects non-sqlite file type (CSV, Excel, JSON, etc.)
    ↓
Sqliter sends file to Flight3 at /api/convertRE
    ↓
Flight3 uses mksqlite converters
    ↓
Flight3 returns SQLite database (or error)
    ↓
Sqliter caches locally and displays
```

### Implementation Design

#### **Sqliter Side** (Flutter)
```dart
class ConversionService {
  final String? flight3Url;
  
  Future<File> ensureSqlite(File file) async {
    if (isSqliteFile(file)) return file;
    
    // Check local cache first
    final cached = await getCachedConversion(file);
    if (cached != null && cached.existsSync()) {
      return cached;
    }
    
    // Try Flight3 conversion if available
    if (flight3Url != null) {
      try {
        return await convertViaFlight3(file);
      } catch (e) {
        print("Flight3 conversion failed: $e");
      }
    }
    
    // Fallback: show error with helpful message
    throw ConversionException(
      "Cannot open ${fileExtension(file)} files directly.\n\n"
      "To view this file:\n"
      "• Start Flight3 server for automatic conversion\n"
      "• Or manually convert to SQLite first"
    );
  }
  
  Future<File> convertViaFlight3(File file) async {
    final uri = Uri.parse('$flight3Url/api/convert');
    final request = http.MultipartRequest('POST', uri);
    request.files.add(await http.MultipartFile.fromPath('file', file.path));
    
    final response = await request.send();
    if (response.statusCode == 200) {
      // Save to cache
      final cacheFile = await getCacheFileFor(file);
      await response.stream.pipe(cacheFile.openWrite());
      return cacheFile;
    }
    throw Exception('Conversion failed: ${response.reasonPhrase}');
  }
}
```

#### **Flight3 Side** (Go)
```go
// In router.go
se.Router.POST("/api/convert", HandleConvertToSQLite)

// In handlers.go
func HandleConvertToSQLite(e *core.RequestEvent) error {
    file, header, err := e.Request.FormFile("file")
    if err != nil {
        return e.JSON(400, map[string]string{"error": "No file provided"})
    }
    defer file.Close()
    
    // Save to temp
    tempPath := filepath.Join(os.TempDir(), header.Filename)
    out, _ := os.Create(tempPath)
    io.Copy(out, file)
    out.Close()
    
    // Convert using mksqlite
    sqlitePath := strings.TrimSuffix(tempPath, filepath.Ext(tempPath)) + ".db"
    if err := convertToSQLite(tempPath, sqlitePath); err != nil {
        return e.JSON(500, map[string]string{"error": err.Error()})
    }
    
    // Return SQLite file
    return e.ServeFile(sqlitePath)
}
```

### Benefits
✅ Sqliter stays pure Flutter - no converter dependencies  
✅ Flight3 handles heavy lifting - reuses existing mksqlite  
✅ Graceful degradation - works offline if file is already SQLite  
✅ Caching - converted files are cached locally  

---

## Feature 2: Enhanced Start View ("Home Dashboard")

### Design Vision
A modern dashboard showing recent activity, quick access, and helpful actions.

### Layout Concept

```
┌─────────────────────────────────────────────────────────────┐
│  🔥 SQLiter                                    [Local] [⚙️]  │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  Quick Actions                                                │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │ 📂 Browse│  │ 🔗 Flight│  │ ➕ New   │  │ 📥 Import│   │
│  │  Local   │  │  Server  │  │  Query   │  │  SQL     │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
│                                                               │
│  ─────────────────────────────────────────────────────────  │
│                                                               │
│  Recent Files                              [Clear History]   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ 📊 employees.db                    2 hours ago  ✕   │   │
│  │ 📂 ~/Documents/data/                               │   │
│  │                                                     │   │
│  │ 📊 sales_2026.db                   Yesterday   ✕   │   │
│  │ 📂 ~/Desktop/                                      │   │
│  │                                                     │   │
│  │ 📊 inventory.csv → .db             3 days ago  ✕   │   │
│  │ 📂 ~/Downloads/  [Converted]                       │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                               │
│  ─────────────────────────────────────────────────────────  │
│                                                               │
│  Flight3 Connections                      [+ Add Server]     │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ 🟢 Local Dev Server                                 │   │
│  │    http://127.0.0.1:8095            [Disconnect]   │   │
│  │    Last sync: 5 mins ago                           │   │
│  │                                                     │   │
│  │ 🔴 Production Server                                │   │
│  │    https://data.example.com         [Connect]      │   │
│  │    admin@example.com                               │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                               │
│  ─────────────────────────────────────────────────────────  │
│                                                               │
│  Cached Datasets                          [Manage Cache]     │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ 📦 bucket/customers.db              45 MB  [View]   │   │
│  │    From: s3://data-prod                            │   │
│  │    Cached: 1 hour ago                              │   │
│  │                                                     │   │
│  │ 📦 remote:analytics/metrics.csv     12 MB  [View]   │   │
│  │    Converted to SQLite                             │   │
│  │    Cached: Today                                   │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### Sections to Include

#### **1. Quick Actions** (Top Priority)
- 📂 **Browse Local Files** - Opens file picker or recent folders
- 🔗 **Flight Server** - Quick access to saved Banquet links
- ➕ **New Query** - Start with empty SQL editor
- 📥 **Import SQL** - Load .sql file to execute

#### **2. Recent Files** (High Value)
Show last 10-15 files with:
- **Icon** - Different icons for .db, .csv, .xlsx, converted files
- **File name** - Click to reopen
- **Path** - Show parent directory
- **Timestamp** - Relative time (2 hours ago, Yesterday, etc.)
- **Type badge** - Show if converted file
- **Remove** - ✕ button to remove from history

**Storage**: Save to local preferences/SQLite
```dart
class RecentFile {
  String path;
  String name;
  DateTime lastOpened;
  bool wasConverted;
  String? originalFormat; // CSV, XLSX, etc.
}
```

#### **3. Flight3 Connections**
- **Status indicator** - 🟢 Connected, 🔴 Disconnected, 🟡 Connecting
- **Server URL** - Editable/clickable
- **Credentials** - Show username, hide password
- **Actions** - Connect/Disconnect buttons
- **Last sync** - Show when last activity occurred

#### **4. Cached Datasets**
Show Flight3-synced files that are cached locally:
- **Name** - From Banquet URL
- **Source** - Show remote URL (s3://, remote:, etc.)
- **Size** - File size
- **Cache time** - When downloaded
- **Quick actions** - [View] [Delete Cache]

#### **5. Additional Ideas**

**Saved Queries** (Optional)
```
┌─────────────────────────────────────────────────────────┐
│ 📝 Sales Report Query                    [Run] [Edit]   │
│    SELECT * FROM sales WHERE amount > 1000             │
│    Last run: 2 days ago                                │
└─────────────────────────────────────────────────────────┘
```

**Statistics** (Optional)
```
┌─────────────────────────────────────────────────────────┐
│ 📊 Quick Stats                                          │
│    • 12 files opened this week                         │
│    • 3 datasets cached (102 MB)                        │
│    • 2 Flight3 servers configured                      │
└─────────────────────────────────────────────────────────┘
```

**Empty State**
When no recent activity:
```
┌─────────────────────────────────────────────────────────┐
│                 👋 Welcome to SQLiter                    │
│                                                          │
│  Get started by:                                        │
│  • Opening a local SQLite database                     │
│  • Connecting to a Flight3 server                      │
│  • Importing a CSV/Excel file for conversion          │
│                                                          │
│  [Browse Files]  [Connect to Server]                   │
└─────────────────────────────────────────────────────────┘
```

---

## Feature 3: Better Pagination UI

### Current Problem
- Infinite scroll is invisible to users
- No indication of progress or total rows
- Can't jump to specific page
- No control over page size

### Proposed Solutions

#### **Option A: Status Bar at Bottom**
```
┌─────────────────────────────────────────────────────────┐
│  [Grid Content]                                          │
│                                                          │
└─────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────┐
│ Showing 1-100 of 45,231 rows  │ [◀ Prev] [Next ▶]      │
│ Loaded: 500 rows              │ Jump to: [____] [Go]   │
│ [⚙️ Page Size: 100 ▾]         │ [↻ Reload]             │
└─────────────────────────────────────────────────────────┘
```

**Features:**
- Shows current position (rows 1-100)
- Shows total count (if known)
- Shows how many rows loaded in memory
- Prev/Next buttons for explicit navigation
- Jump to page number
- Configurable page size (50, 100, 500, 1000)
- Reload button to refresh data

#### **Option B: Floating Pagination Widget**
```
┌─────────────────────────────────────────────────────────┐
│  [Grid Content]                                          │
│                                          ┌─────────────┐ │
│                                          │ Page 1 of 452│ │
│                                          │ ◀  1  ▶     │ │
│                                          └─────────────┘ │
└─────────────────────────────────────────────────────────┘
```

**Features:**
- Floating widget in bottom-right corner
- Minimal disruption
- Click to expand for more options

#### **Option C: Header Stats + Footer Controls**
```
┌─────────────────────────────────────────────────────────┐
│ 📊 45,231 rows  •  12 columns  •  2.3 MB                │
├─────────────────────────────────────────────────────────┤
│  [Grid Content]                                          │
│                                                          │
├─────────────────────────────────────────────────────────┤
│ ◀◀ ◀  Page [__5__] of 452  ▶ ▶▶    Size: [100 ▾]      │
└─────────────────────────────────────────────────────────┘
```

**Features:**
- Stats in header (always visible)
- Navigation controls in footer
- ◀◀ First page, ◀ Previous, ▶ Next, ▶▶ Last page

#### **My Recommendation: Hybrid Approach**

Combine header stats with smart infinite scroll:

```
┌─────────────────────────────────────────────────────────┐
│ 📊 employees.db  •  45,231 rows  •  Showing 1-500       │
├─────────────────────────────────────────────────────────┤
│  [Grid with infinite scroll]                            │
│                                                          │
│  [Auto-loads more as you scroll down]                   │
│                                                          │
│                         ⬇️ Loading more...               │
└─────────────────────────────────────────────────────────┘
```

**Plus a toolbar button:**
```
[...] [Jump to Row] -> Opens dialog:
┌──────────────────────────────┐
│  Jump to Row                 │
│  ┌────────────────────────┐  │
│  │ 1000                   │  │
│  └────────────────────────┘  │
│  [Cancel]  [Go]              │
└──────────────────────────────┘
```

### Implementation with TrinaGrid

```dart
Widget _buildGridWithPagination() {
  return Column(
    children: [
      // Stats header
      Container(
        padding: EdgeInsets.all(8),
        decoration: BoxDecoration(
          color: Color(0xFF2D2D2D),
          border: Border(bottom: BorderSide(color: Colors.white24)),
        ),
        child: Row(
          children: [
            Icon(CupertinoIcons.table, size: 16, color: Colors.white70),
            SizedBox(width: 8),
            Text('${_selectedTable ?? "No table"} ', 
              style: TextStyle(fontWeight: FontWeight.bold)),
            if (_totalRows != null) ...[
              Text('•  '),
              Text('${NumberFormat('#,###').format(_totalRows)} rows'),
            ],
            Text('•  '),
            Text('Showing 1-${_dbStateManager?.rows.length ?? 0}'),
            Spacer(),
            // Jump to row button
            MacosIconButton(
              icon: Icon(CupertinoIcons.arrow_down_to_line, size: 16),
              onPressed: _showJumpToRowDialog,
            ),
          ],
        ),
      ),
      
      // Grid with infinite scroll
      Expanded(
        child: TrinaGrid(
          columns: _dbColumns,
          rows: [],
          createFooter: (stateManager) {
            _dbStateManager = stateManager;
            return TrinaInfinityScrollRows(
              fetch: _fetchMoreRows,
              stateManager: stateManager,
            );
          },
          configuration: _getGridConfiguration(context),
        ),
      ),
    ],
  );
}
```

---

## Feature 4: Untapped TrinaGrid Features

### Features We Should Use

#### **1. Column Resizing & Reordering**
```dart
TrinaGridConfiguration.dark(
  columnSize: TrinaGridColumnSizeConfig(
    autoSizeMode: TrinaAutoSizeMode.none,
    resizeMode: TrinaResizeMode.normal, // ✅ Enable resize
  ),
)
```

#### **2. Column Sorting**
```dart
TrinaColumn(
  title: 'Name',
  field: 'name',
  type: TrinaColumnType.text(),
  enableSorting: true, // ✅ Enable sorting
  enableContextMenu: true, // ✅ Right-click menu
)
```

#### **3. Column Filtering** (Already using!)
```dart
event.stateManager.setShowColumnFilter(true); // ✅ We have this
```

#### **4. Row Selection & Actions**
```dart
TrinaGrid(
  mode: TrinaGridMode.selectWithOneTap, // ✅ Enable selection
  onSelected: (event) {
    // Show context menu or actions
  },
)
```

#### **5. Copy/Paste Support**
```dart
TrinaGridConfiguration(
  enableMoveDownAfterSelecting: true,
  enableMoveHorizontalInEditing: true,
  enterKeyAction: TrinaGridEnterKeyAction.moveDown,
)
```

#### **6. Excel-like Key Navigation**
- Arrow keys to navigate
- Tab to move right
- Enter to move down
- Built-in, just needs to be enabled

#### **7. Custom Cell Renderers** (Already using for dates!)
We're using this for the banquet links display - great!

#### **8. Frozen Columns**
```dart
TrinaColumn(
  title: 'ID',
  field: 'id',
  frozen: TrinaColumnFrozen.start, // ✅ Keep ID visible while scrolling
)
```

#### **9. Export Functionality**
TrinaGrid has built-in CSV export:
```dart
void _exportToCsv() {
  String csv = _dbStateManager?.exportToCsv();
  // Save to file
}
```

#### **10. Aggregation Footer**
Show SUM, COUNT, AVG in column footers:
```dart
TrinaColumn(
  title: 'Amount',
  field: 'amount',
  type: TrinaColumnType.number(),
  footerRenderer: (context) {
    final sum = calculateSum(context.stateManager.rows);
    return Text('Total: \$${sum.toStringAsFixed(2)}');
  },
)
```

### Recommended Feature Additions

**High Priority:**
1. ✅ Column sorting (essential for data exploration)
2. ✅ Column resizing (better fit for different data)

**Medium Priority:**
6. ✅ Row selection + context menu (for actions)
7. ✅ Keyboard navigation improvements
8. ✅ Column hiding (show/hide columns)

**Low Priority:**
10. Aggregation headers (useful for numerical data) Make the same line as the banquet bar.

---

## Implementation Roadmap

### Phase 1: Foundation (Week 1)
- [ ] Implement new Home Dashboard UI
- [ ] Add Recent Files tracking (local storage)
- [ ] Add Flight3 connection management UI
- [ ] Improve pagination stats header

### Phase 2: Smart Conversion (Week 2)
- [ ] Create ConversionService in Sqliter
- [ ] Implement `/api/convert` endpoint in Flight3
- [ ] Add local caching for converted files
- [ ] Update file browser to detect non-SQLite files

### Phase 3: Grid Enhancements (Week 3)
- [ ] Enable column sorting
- [ ] Enable column resizing
- [ ] Add frozen first column option

### Phase 4: Polish (Week 4)
- [ ] Add cached datasets view
- [ ] Implement cache management
- [ ] Add keyboard shortcuts help
- [ ] Performance optimizations
- [ ] User testing & refinements

---

## Success Metrics

**User Experience:**
- ✅ Users can open any file type (with Flight3)
- ✅ Recently used files are one click away
- ✅ Pagination is obvious and controllable
- ✅ Large datasets load smoothly (infinite scroll)

**Technical:**
- ✅ Clean separation: Flutter UI, Go backend
- ✅ Graceful degradation when Flight3 offline
- ✅ Local caching reduces network calls
- ✅ No embedded converters in Flutter app

**Performance:**
- ✅ Home dashboard loads in < 100ms
- ✅ Grid scrolling at 60fps
- ✅ File conversion < 5s for typical files
- ✅ Cache lookup < 50ms
