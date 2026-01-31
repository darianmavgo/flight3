# Flight3 ↔ SQLiter Architecture Summary

## The Boundary (Visual)

```
┌─────────────────────────────────────────────────────────────────┐
│                         BANQUET URL                              │
│  s3://user@host/data/sales.csv;tb0/name,amount;+date?limit=100  │
└─────────────────────────────────────────────────────────────────┘
         │                                    │
         ▼                                    ▼
┌──────────────────────────┐      ┌──────────────────────────────┐
│      FLIGHT3             │      │         SQLITER              │
│  (Resource Acquisition)  │      │     (Data Querying)          │
├──────────────────────────┤      ├──────────────────────────────┤
│ • Scheme                 │      │ • ColumnSetPath              │
│ • User/Auth              │      │ • Table name                 │
│ • Host/Remote            │      │ • Select columns             │
│ • DataSetPath            │      │ • Where clause               │
│                          │      │ • GroupBy                    │
│ Actions:                 │      │ • Having                     │
│ ✓ Authenticate           │      │ • OrderBy                    │
│ ✓ Connect rclone         │      │ • SortDirection              │
│ ✓ Fetch file             │      │ • Limit                      │
│ ✓ Convert to SQLite      │      │ • Offset                     │
│ ✓ Cache .db file         │      │                              │
│ ✓ Redirect to SQLiter    │      │ Actions:                     │
│                          │      │ ✓ Build SQL query            │
│ Output:                  │      │ ✓ Execute query              │
│ → /cache/sales.csv.db    │      │ ✓ Render React UI            │
└──────────────────────────┘      │ ✓ Serve AG-Grid              │
         │                        │ ✓ Return JSON                │
         │                        └──────────────────────────────┘
         │                                    ▲
         │                                    │
         └────────── Redirect ────────────────┘
           /_/data/sales.csv.db;tb0/name,amount;+date?limit=100
```

---

## Data Flow

```
1. User Request
   ↓
   GET s3://mybucket@aws/data/sales.csv;tb0/name,amount;+date?limit=100

2. Flight3 Receives
   ↓
   Parse: Scheme=s3, Host=mybucket@aws, DataSetPath=data/sales.csv
   ↓
   Authenticate with AWS
   ↓
   Connect rclone VFS
   ↓
   Fetch data/sales.csv
   ↓
   Convert to /cache/sales.csv.db (mksqlite)
   ↓
   Redirect to: /_/data/sales.csv.db;tb0/name,amount;+date?limit=100

3. SQLiter Receives
   ↓
   Parse: DataSetPath=sales.csv.db, ColumnSetPath=tb0/name,amount;+date
   ↓
   Open /cache/sales.csv.db
   ↓
   Build SQL: SELECT "name", "amount" FROM "tb0" ORDER BY "date" ASC LIMIT 100
   ↓
   Execute query
   ↓
   Serve React UI
   ↓
   Return JSON to AG-Grid

4. User Sees
   ↓
   Interactive data table with sorting, filtering, pagination
```

---

## Responsibility Matrix

| Task | Flight3 | SQLiter |
|------|---------|---------|
| **Parse URL** | Scheme → DataSetPath | ColumnSetPath → Query |
| **Authenticate** | ✅ | ❌ |
| **Connect Remote** | ✅ | ❌ |
| **Fetch File** | ✅ | ❌ |
| **Convert Format** | ✅ (mksqlite) | ❌ |
| **Cache SQLite** | ✅ | ❌ |
| **Build SQL** | ❌ | ✅ (sqlite.Compose) |
| **Execute Query** | ❌ | ✅ |
| **Render UI** | ❌ | ✅ (React + AG-Grid) |
| **Handle Sorting** | ❌ | ✅ |
| **Handle Filtering** | ❌ | ✅ |
| **Pagination** | ❌ | ✅ |

---

## Code Changes Summary

### Flight3 Changes

**DELETE:**
- ❌ `internal/flight/server.go` (173 lines)

**MODIFY:**
- `internal/flight/flight.go`
  - Remove: Template initialization
  - Add: SQLiter server setup
  - Add: Mount `/_/data/` routes

- `internal/flight/banquethandler.go`
  - Remove: `html/template` import
  - Remove: `tw`, `tpl` parameters
  - Add: Redirect to SQLiter

**RESULT:**
- ~193 lines removed
- ~21 lines added
- **Net: -172 lines**

---

### SQLiter Changes

**NO CHANGES NEEDED! ✅**

SQLiter already:
- ✅ Implements `http.Handler`
- ✅ Parses Banquet URLs
- ✅ Builds SQL with `sqlite.Compose()`
- ✅ Serves React UI
- ✅ Returns JSON

---

## File Organization

```
flight3/
├── internal/flight/
│   ├── flight.go              ← Mount SQLiter server
│   ├── banquethandler.go      ← Redirect to SQLiter
│   ├── converter.go           ← mksqlite integration
│   ├── rclone_manager.go      ← Remote file fetching
│   ├── cache.go               ← Cache management
│   ├── handlers_auth.go       ← PocketBase auth UI (keep)
│   ├── handlers_rclone_config.go ← PocketBase config UI (keep)
│   └── server.go              ← DELETE THIS FILE
└── templates/
    └── rclone_config.html     ← PocketBase template (keep)

sqliter/
└── sqliter/
    ├── server.go              ← HTTP handler (no changes)
    ├── config.go              ← Configuration (no changes)
    └── ui/                    ← React app (no changes)
```

---

## Example Scenarios

### Scenario 1: Local CSV File

**User Request:**
```
GET /data/sales.csv
```

**Flight3:**
1. Parse: DataSetPath = "data/sales.csv"
2. Check cache: `/cache/sales.csv.db`
3. If missing: Convert with mksqlite
4. Redirect: `/_/data/sales.csv.db;tb0`

**SQLiter:**
1. Receive: `;tb0` (ColumnSetPath with table only = SELECT *)
2. Query: `SELECT * FROM "tb0" LIMIT 100`
3. Render: React UI with data

---

### Scenario 2: Remote Excel with Query

**User Request:**
```
GET s3://reports@aws/2024/Q1.xlsx;Sheet1/revenue,region;revenue>1000;+revenue?limit=50
```

**Flight3:**
1. Parse: Scheme=s3, Host=reports@aws, DataSetPath=2024/Q1.xlsx
2. Authenticate with AWS
3. Connect rclone
4. Fetch 2024/Q1.xlsx
5. Convert to `/cache/Q1.xlsx.db`
6. Redirect: `/_/data/Q1.xlsx.db;Sheet1/revenue,region;revenue>1000;+revenue?limit=50`

**SQLiter:**
1. Parse: ColumnSetPath=Sheet1/revenue,region;revenue>1000;+revenue
2. Build SQL: `SELECT "revenue", "region" FROM "Sheet1" WHERE revenue>1000 ORDER BY "revenue" ASC LIMIT 50`
3. Execute query
4. Render: React UI with filtered, sorted data

---

### Scenario 3: Directory Listing

**User Request:**
```
GET /Users/me/Documents/;name,size;is_dir=1
```

**Flight3:**
1. Parse: DataSetPath = "/Users/me/Documents/"
2. Index directory to `/cache/Documents_.db` (table: tb0)
3. Redirect: `/_/data/Documents_.db;tb0/name,size;is_dir=1`

**SQLiter:**
1. Parse: ColumnSetPath=tb0/name,size;is_dir=1
2. Build SQL: `SELECT "name", "size" FROM "tb0" WHERE is_dir=1`
3. Execute query
4. Render: React UI showing only directories

---

## Benefits

### Clean Architecture
- ✅ Single Responsibility Principle
- ✅ Clear boundaries
- ✅ Easy to understand

### Easy Testing
- ✅ Test Flight3: Mock file fetching
- ✅ Test SQLiter: Use pre-made .db files
- ✅ Independent test suites

### Independent Evolution
- ✅ Flight3 can add new remote types
- ✅ SQLiter can add new query features
- ✅ No coordination needed

### Simplified Code
- ✅ Flight3: -172 lines
- ✅ SQLiter: No changes
- ✅ Better maintainability

---

## Next Steps

1. ✅ Review `ResponsibilityBoundary.md` (detailed boundary)
2. ✅ Review `RefactorSQLiter.md` (integration plan)
3. ✅ Review `CleanUpTodo.md` (HTML removal checklist)
4. ⏳ Implement Flight3 changes
5. ⏳ Test integration
6. ⏳ Deploy

---

## Success Criteria

- [ ] Flight3 has zero HTML rendering code
- [ ] All data queries go through SQLiter
- [ ] PocketBase admin UI still works
- [ ] Banquet URLs work correctly
- [ ] Local files work
- [ ] Remote files work
- [ ] Directory listings work
- [ ] All tests pass

**Timeline: ~7 hours**

🎯 Clear boundary = Clean code = Happy developers!
