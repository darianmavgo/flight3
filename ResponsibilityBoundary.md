# Flight3 ↔ SQLiter Responsibility Boundary

## The Clear Boundary

For any Banquet URL, responsibilities are split at the **ColumnPath**:

```
[Scheme]://[User]@[Host]/[DataSetPath]/[Table];[ColumnPath]?[Query]
└────────────────────────────────────┘ └──────────────────────┘
         FLIGHT3 TERRITORY                  SQLITER TERRITORY
```

---

## Banquet Structure

```go
type Banquet struct {
    // FLIGHT3 HANDLES (Resource Location)
    *url.URL        // Scheme, User, Host, Path
    DataSetPath     // Path to source file (e.g., "mydata.csv")
    Table           // Table name (e.g., "tb0", "users")
    
    // SQLITER HANDLES (Query Execution)
    ColumnPath      // Column selections, sort, conditions
    Select          // Columns to select
    Where           // Filter conditions
    GroupBy         // Grouping
    Having          // Group filters
    OrderBy         // Sort column
    SortDirection   // ASC/DESC
    Limit           // Row limit
    Offset          // Row offset
}
```

---

## Flight3 Responsibilities

### What Flight3 Handles

**From Scheme to DataSetPath:**
1. **Scheme** - Protocol (http, https, s3, etc.)
2. **User** - Authentication credentials
3. **Host** - Remote hostname/alias
4. **DataSetPath** - Path to the source file

**Actions:**
- ✅ Parse authentication from URL
- ✅ Lookup rclone remote configuration
- ✅ Connect to remote storage
- ✅ Fetch the file
- ✅ Convert to SQLite (if needed)
- ✅ Cache the SQLite database
- ✅ Determine final SQLite file path

**Output:**
```go
// Flight3 produces:
sqlitePath := "/path/to/cache/mydata.csv.db"
tableName := "tb0" // or inferred table name
```

---

## SQLiter Responsibilities

### What SQLiter Handles

**From ColumnPath to Query:**
1. **ColumnPath** - Column selections and operations
2. **Select** - Which columns to return
3. **Where** - Filter conditions
4. **GroupBy** - Aggregation grouping
5. **Having** - Group filtering
6. **OrderBy** - Sort column
7. **SortDirection** - Sort direction
8. **Limit** - Row limit
9. **Offset** - Pagination offset

**Actions:**
- ✅ Parse ColumnPath into SQL clauses
- ✅ Build SQL query using `sqlite.Compose()`
- ✅ Execute query on SQLite database
- ✅ Render results (React UI + AG-Grid)
- ✅ Handle sorting, filtering, pagination
- ✅ Return JSON to client

**Input:**
```go
// SQLiter receives:
dbPath := "/path/to/cache/mydata.csv.db"
banquet := &banquet.Banquet{
    Table: "tb0",
    ColumnPath: "name,size;+modified",
    // ... all query parameters
}
```

---

## Example URL Breakdown

### Example 1: Simple Remote File

```
s3://mybucket@aws/data/sales.csv/tb0;name,amount;+date?limit=100
└──────────────────────────────┘└────────────────────────────┘
      FLIGHT3                           SQLITER
```

**Flight3 Handles:**
- `s3://` - Scheme
- `mybucket@aws` - Remote alias
- `/data/sales.csv` - DataSetPath
- Converts to: `/cache/sales.csv.db`

**SQLiter Handles:**
- `/tb0` - Table name
- `;name,amount` - Select columns
- `;+date` - Order by date ASC
- `?limit=100` - Limit 100 rows
- Executes: `SELECT "name", "amount" FROM "tb0" ORDER BY "date" ASC LIMIT 100`

---

### Example 2: Local Directory

```
/Users/me/Documents/;name,size;is_dir=1
└────────────────┘└──────────────────┘
    FLIGHT3            SQLITER
```

**Flight3 Handles:**
- `/Users/me/Documents/` - DataSetPath
- Indexes directory to: `/cache/Documents_.db`
- Table: `tb0` (filesystem listing)

**SQLiter Handles:**
- `;name,size` - Select columns
- `;is_dir=1` - Where clause (directories only)
- Executes: `SELECT "name", "size" FROM "tb0" WHERE is_dir=1`

---

### Example 3: Complex Query

```
https://user:pass@myserver/reports/2024.xlsx/Sheet1;revenue,region;region;revenue>1000;+revenue?limit=50&offset=100
└────────────────────────────────────────────────┘└─────────────────────────────────────────────────────────┘
                FLIGHT3                                              SQLITER
```

**Flight3 Handles:**
- `https://` - Scheme
- `user:pass@myserver` - Auth + host
- `/reports/2024.xlsx` - DataSetPath
- Converts to: `/cache/2024.xlsx.db`
- Table: `Sheet1`

**SQLiter Handles:**
- `;revenue,region` - Select columns
- `;region` - Group by region
- `;revenue>1000` - Having clause
- `;+revenue` - Order by revenue ASC
- `?limit=50&offset=100` - Pagination
- Executes: `SELECT "revenue", "region" FROM "Sheet1" GROUP BY "region" HAVING revenue>1000 ORDER BY "revenue" ASC LIMIT 50 OFFSET 100`

---

## Communication Protocol

### Flight3 → SQLiter

**Step 1: Flight3 Processes Resource**
```go
// Parse Banquet URL
b, _ := banquet.ParseNested(requestURL)

// Flight3 handles: Scheme → DataSetPath
sqlitePath := convertToSQLite(b.Scheme, b.User, b.Host, b.DataSetPath)
// Result: "/cache/myfile.db"
```

**Step 2: Flight3 Passes to SQLiter**
```go
// Construct SQLiter URL with ONLY query parts
sqliterURL := fmt.Sprintf("/_/data/%s/%s%s",
    filepath.Base(sqlitePath),  // Database file
    b.Table,                     // Table name
    b.ColumnPath,                // Everything from ColumnPath onward
)

// Add query parameters
if b.RawQuery != "" {
    sqliterURL += "?" + b.RawQuery
}

// Redirect to SQLiter
return e.Redirect(302, sqliterURL)
```

**Example:**
```
Original:  s3://bucket@aws/data/sales.csv/tb0;name,amount;+date?limit=100
           └─────────────────────────────┘└──────────────────────────┘
                   Flight3                        SQLiter

Redirect:  /_/data/sales.csv.db/tb0;name,amount;+date?limit=100
                  └──────────┘└──────────────────────────┘
                   DB File         Query Parts
```

---

### SQLiter Receives

**Step 1: SQLiter Parses URL**
```go
// Receives: /_/data/sales.csv.db/tb0;name,amount;+date?limit=100
path := r.URL.Path
// Extract: sales.csv.db/tb0;name,amount;+date

// Parse as Banquet
b, _ := banquet.ParseNested(path)
```

**Step 2: SQLiter Executes**
```go
// Open database
dbPath := filepath.Join(config.ServeFolder, b.DataSetPath)
db, _ := sql.Open("sqlite", dbPath)

// Build query (handles ColumnPath → Query)
query := sqlite.Compose(b)
// Result: SELECT "name", "amount" FROM "tb0" ORDER BY "date" ASC LIMIT 100

// Execute and render
rows, _ := db.Query(query)
renderToReact(rows)
```

---

## Responsibility Matrix

| Component | Flight3 | SQLiter |
|-----------|---------|---------|
| **URL Parsing** | Scheme, User, Host, DataSetPath | ColumnPath, Query params |
| **Authentication** | ✅ Validate credentials | ❌ |
| **Remote Access** | ✅ Connect via rclone | ❌ |
| **File Fetching** | ✅ Download/cache | ❌ |
| **Format Conversion** | ✅ mksqlite | ❌ |
| **SQLite Creation** | ✅ Create .db file | ❌ |
| **Table Inference** | ✅ Determine table name | ❌ |
| **SQL Composition** | ❌ | ✅ Build SELECT query |
| **Query Execution** | ❌ | ✅ Execute on SQLite |
| **Result Rendering** | ❌ | ✅ React UI + AG-Grid |
| **Sorting/Filtering** | ❌ | ✅ Handle in SQL |
| **Pagination** | ❌ | ✅ LIMIT/OFFSET |

---

## Code Examples

### Flight3 Implementation

```go
func HandleBanquet(e *core.RequestEvent, verbose bool) error {
    // Parse full URL
    b, err := banquet.ParseNested(e.Request.RequestURI)
    if err != nil {
        return err
    }
    
    // FLIGHT3 RESPONSIBILITY: Scheme → DataSetPath
    // 1. Authenticate
    if b.User != nil {
        // Validate credentials
    }
    
    // 2. Connect to remote (if needed)
    var sqlitePath string
    if b.Host != "" {
        // Remote file
        remote := LookupRemote(e.App, b.Host)
        vfs := GetVFS(remote)
        
        // Fetch file
        tempPath := fetchFile(vfs, b.DataSetPath)
        
        // Convert to SQLite
        sqlitePath = convertToSQLite(tempPath)
    } else {
        // Local file
        sqlitePath = convertToSQLite(b.DataSetPath)
    }
    
    // 3. Redirect to SQLiter with query parts
    relPath := filepath.Base(sqlitePath)
    sqliterURL := fmt.Sprintf("/_/data/%s/%s%s",
        relPath,
        b.Table,
        b.ColumnPath, // Everything from here is SQLiter's job
    )
    if b.RawQuery != "" {
        sqliterURL += "?" + b.RawQuery
    }
    
    return e.Redirect(302, sqliterURL)
}
```

### SQLiter Implementation

```go
func (s *Server) apiQueryTable(w http.ResponseWriter, r *http.Request) {
    // Parse path as Banquet
    path := strings.TrimPrefix(r.URL.Path, "/sqliter/rows/")
    b, err := banquet.ParseNested(path)
    if err != nil {
        http.Error(w, "Invalid path", 400)
        return
    }
    
    // SQLITER RESPONSIBILITY: ColumnPath → Query
    // 1. Open database (already in ServeFolder)
    dbPath := filepath.Join(s.config.ServeFolder, b.DataSetPath)
    db, err := sql.Open("sqlite", dbPath)
    if err != nil {
        http.Error(w, "DB error", 500)
        return
    }
    defer db.Close()
    
    // 2. Build SQL query from ColumnPath onward
    query := sqlite.Compose(b)
    // Handles: Select, Where, GroupBy, Having, OrderBy, Limit, Offset
    
    // 3. Execute
    rows, err := db.Query(query)
    if err != nil {
        http.Error(w, "Query error", 400)
        return
    }
    defer rows.Close()
    
    // 4. Render to JSON for React
    renderJSON(w, rows)
}
```

---

## Benefits of This Boundary

### Clear Separation
- ✅ Flight3 focuses on **resource acquisition**
- ✅ SQLiter focuses on **data querying**
- ✅ No overlap in responsibilities

### Easy Testing
- ✅ Test Flight3 with mock files
- ✅ Test SQLiter with pre-made SQLite DBs
- ✅ Independent test suites

### Independent Evolution
- ✅ Flight3 can add new remote types
- ✅ SQLiter can add new query features
- ✅ No coordination needed

### Simple Interface
- ✅ Flight3 output: SQLite file path
- ✅ SQLiter input: SQLite file path + query
- ✅ Clean handoff

---

## Migration Impact

### What Changes in Flight3

**REMOVE:**
- ❌ SQL query building (now SQLiter's job)
- ❌ HTML rendering (now SQLiter's job)
- ❌ Table inference from query (now SQLiter's job)

**KEEP:**
- ✅ Authentication
- ✅ Remote connections
- ✅ File fetching
- ✅ Format conversion
- ✅ Table inference from file type

**ADD:**
- ✅ Redirect to SQLiter with query parts

### What Changes in SQLiter

**Already Has:**
- ✅ SQL query building (`sqlite.Compose()`)
- ✅ React UI rendering
- ✅ JSON API

**No Changes Needed:**
- ✅ SQLiter already handles ColumnPath → Query
- ✅ Already has `http.Handler` interface
- ✅ Already serves from folder

---

## Summary

**The Boundary:**
```
Flight3: [Scheme → DataSetPath] = Resource Acquisition
SQLiter: [ColumnPath → Query]   = Data Querying
```

**The Handoff:**
```
Flight3 produces: SQLite file path + table name
SQLiter receives: SQLite file path + query parameters
```

**The Result:**
- Clean separation of concerns
- Easy to test
- Easy to maintain
- Clear ownership

Perfect architecture! 🎯
