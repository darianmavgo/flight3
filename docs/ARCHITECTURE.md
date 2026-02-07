# Flight3 Architecture

## System Overview

Flight3 is a hyper-local data server that bridges cloud storage and local clients through a unified API. The architecture follows a clear separation of concerns with well-defined component responsibilities.

## Component Responsibilities

### Flight3 Server (Resource Acquisition)
**Scope:** Scheme → DataSetPath

Responsibilities:
- Parse Banquet URLs (scheme, host, auth, dataset path)
- Authenticate with remote storage providers
- Connect to cloud storage via Rclone VFS
- Fetch remote files with intelligent caching
- Convert non-SQLite files using mksqlite
- Maintain PocketBase admin interface
- Serve REST API for data access

**Does NOT:**
- Build SQL queries (delegated to client)
- Render UI (delegated to client)
- Execute queries (delegated to client)

### PocketBase (Configuration)  
**Embedded database and admin framework**

Responsibilities:
- Store remote configurations (`rclone_remotes`)
- Manage application settings (`app_settings`)
- Store banquet link bookmarks (`banquet_links`)
- Provide admin UI at `/_/`
- Handle authentication

### Rclone VFS (Cloud Storage)
**Embedded cloud storage client**

Responsibilities:
- Connect to 40+ storage providers (S3, GCS, R2, Drive, etc.)
- Implement VFS layer with caching
- Handle partial reads and streaming
- Manage connection pooling
- Index remote directories

Configuration stored in PocketBase, VFS instances cached per remote.

### Banquet (URL Parser)
**Library for parsing data query URLs**

Responsibilities:
- Parse complex URL notation
- Extract scheme, host, dataset path
- Extract table name and column selections
- Parse filter, sort, and pagination parameters

Does NOT execute queries - only parses URLs.

### Mksqlite (File Converter)
**Library for converting files to SQLite**

Supported formats:
- CSV → SQLite
- Excel (.xlsx, .xls) → SQLite
- JSON → SQLite
- HTML tables → SQLite
- Markdown tables → SQLite
- Text files → SQLite
- ZIP archives → SQLite
- Directories → SQLite (with file listing)

### SQLiter API (Data Querying)
**Internal REST API for client access**

Endpoints:
- `GET /sqliter/file/{banquet-path}` - Download cached SQLite file
- `GET /sqliter/sync/{banquet-path}` - Get sync metadata
- `GET /sqliter/rows?path={path}&start={offset}&end={limit}` - Query rows

**Scope:** ColumnSetPath → Query execution

---

## Data Flow

### Request Lifecycle

```
1. Client Request
   ↓
   GET s3://bucket@aws/data/sales.csv;tb0/name,amount;+date?limit=100

2. Flight3 Parses URL
   ↓
   Scheme: s3
   User: bucket (remote alias/credential ID in PocketBase)
   Host: aws
   DataSetPath: data/sales.csv
   Table: tb0
   ColumnPath: name,amount;+date (column selection and sort syntax)
   Query: limit=100

3. Flight3 Processes Resource
   ↓
   - Lookup remote "bucket" in PocketBase
   - Get/create Rclone VFS for AWS S3
   - Check cache: /cache/sales.csv.db
   - If miss: fetch data/sales.csv via VFS
   - Convert CSV to SQLite using mksqlite
   - Store at /cache/sales.csv.db

4. Client Queries Data
   ↓
   GET /sqliter/rows?path=sales.csv;tb0/name,amount;+date&start=0&end=100
   
5. SQLiter API Returns JSON
   ↓
   {
     "columns": ["name", "amount"],
     "rows": [...],
     "totalCount": 1000
   }
```

### Architecture Diagram

```
┌───────────────────────────────────────────────────────────┐
│                  SQLiter-Dart Client                      │
│                  (Flutter Desktop App)                    │
│                                                            │
│  ┌──────────────┐  ┌──────────────┐                      │
│  │  PlutoGrid   │  │ File Browser │                      │
│  │  (Table UI)  │  │    (UI)      │                      │
│  └──────────────┘  └──────────────┘                      │
└──────────────────────────┬────────────────────────────────┘
                           │ HTTP REST API
                           ↓
┌───────────────────────────────────────────────────────────┐
│                    Flight3 Server                         │
│                                                            │
│  ┌────────────────────── Router ──────────────────────┐  │
│  │  /              → Banquet Handler                   │  │
│  │  /_/            → PocketBase Admin                  │  │
│  │  /api/          → PocketBase API                    │  │
│  │  /sqliter/*     → SQLiter API                       │  │
│  │  /_/rclone_config → Rclone Config UI                │  │
│  └─────────────────────────────────────────────────────┘  │
│                                                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │  PocketBase  │  │  Rclone VFS  │  │   Mksqlite   │   │
│  │   (Config)   │  │  (Storage)   │  │ (Converter)  │   │
│  │              │  │              │  │              │   │
│  │ • Remotes    │  │ • S3         │  │ • CSV        │   │
│  │ • Settings   │  │ • GCS        │  │ • Excel      │   │
│  │ • Links      │  │ • R2         │  │ • JSON       │   │
│  │              │  │ • Drive      │  │ • Directory  │   │
│  └──────────────┘  └──────────────┘  └──────────────┘   │
│                                                            │
│  ┌─────────────────────────────────────────────────────┐  │
│  │                Cache Directory                      │  │
│  │  /cache/                                            │  │
│  │    sales.csv.db                                     │  │
│  │    inventory.xlsx.db                                │  │
│  │    Users_Documents_.db (directory listing)          │  │
│  └─────────────────────────────────────────────────────┘  │
└───────────────────────────┬───────────────────────────────┘
                            │
                            ↓
                   Cloud Storage Providers
                   (S3, GCS, R2, Drive, etc.)
```

---

## Responsibility Boundary

### The Banquet URL Split

For any Banquet URL, responsibilities split after the **Table** (or DataSetPath):

**Canonical Format:**
```
[Scheme]://[User]@[Host]/[DataSetPath];[Table];[ColumnPath]?[Query]
└─────────────────────────────────────────────┘ └────────────────┘
              FLIGHT3 HANDLES                    CLIENT HANDLES
```

**Familiar Syntax** (using `/` instead of `;`):
```
[Scheme]://[User]@[Host]/[DataSetPath]/[Table]/[ColumnPath]?[Query]
```

**Flight3 Handles:**
- Scheme (s3, https, file, etc.)
- User (remote alias/credential ID in PocketBase - e.g., "bucket")
- Host (provider identifier - e.g., "aws")
- DataSetPath (path to file)
- Table (inferred from file or explicit)

**Output:** Cached SQLite file at known path

**Client Handles:**
- ColumnPath (column selection, conditions, sort syntax)
  - Columns: `name,amount` (comma-separated)
  - Conditions: `status!=active` (inline filters)
  - Sort: `+date` (ASC), `-date` (DESC)
- Query parameters (limit=100, offset=0)

**Output:** SQL query execution and UI rendering

### Example URL Breakdown

```
s3://mybucket@aws/reports/2024.xlsx;Sheet1;revenue,region;revenue>1000;+revenue?limit=50
└────────────────────────────────────────────┘└─────────────────────────────────────────┘
              FLIGHT3                                      CLIENT

Flight3:
- Parse: User="mybucket" (PocketBase remote alias), Host="aws"
- Lookup remote "mybucket" configuration in PocketBase
- Connect to S3 with AWS credentials
- Fetch reports/2024.xlsx
- Convert to SQLite using mksqlite
- Cache as /cache/abc123.db (hash of scheme+user+host+path)
- Serve table "Sheet1" to client

Client:
- Parse ColumnPath: columns=[revenue, region], condition=revenue>1000, sort=+revenue
- Build query: SELECT revenue, region FROM Sheet1 WHERE revenue>1000 ORDER BY revenue ASC LIMIT 50
- Execute query against cached SQLite file
- Render results in PlutoGrid
```

> [!NOTE]
> **UserInfo vs Host**: In `s3://bucket@aws`, `bucket` is parsed as URL UserInfo (the remote alias stored in PocketBase), while `aws` is the Host (provider identifier). This differs from standard web URLs where `user@host` represents credentials and server.

---

## Caching Strategy

### Cache Key Generation

```go
func GenCacheKey(b *banquet.Banquet) string {
    parts := []string{
        b.Scheme,
        b.Hostname(),
        b.DataSetPath,
    }
    combined := strings.Join(parts, "|")
    hash := md5.Sum([]byte(combined))
    return fmt.Sprintf("%x", hash)
}
```

### Cache Validation

```go
func ValidateCache(cachePath string, ttlMinutes float64) (bool, error) {
    info, err := os.Stat(cachePath)
    if err != nil {
        return false, err
    }
    
    age := time.Since(info.ModTime())
    maxAge := time.Duration(ttlMinutes) * time.Minute
    
    return age < maxAge, nil
}
```

Default TTL: 24 hours (1440 minutes)

### Cache Directory Structure

```
~/Library/Application Support/Flight3/
├── pb_data/              # PocketBase database
│   ├── data.db
│   ├── auxiliary.db
│   └── backups/
├── cache/                # Converted SQLite files
│   ├── abc123def.db
│   ├── xyz789ghi.db
│   └── ...
├── temp/                 # Temporary downloads
└── pb_public/            # Static assets
```

---

## API Endpoints

### Public Routes

```
GET /                       → Serve folder configured in app_settings (planned)
GET /{banquet-path}         → Convert and cache dataset
```

> [!NOTE]
> **Root Path Behavior**: Currently redirects to `/_/` (PocketBase admin). Planned to serve Banquet URL for configurable home directory.

### PocketBase Routes

```
GET  /_/                    → Admin dashboard
GET  /api/                  → REST API  
POST /api/collections/...   → Collection operations
GET  /api/auto_login        → Auto-login for desktop mode
```

### Rclone Configuration

```
GET  /_/rclone_config                  → Config UI
GET  /_/rclone_config/api/providers    → List providers
GET  /_/rclone_config/api/remotes      → List remotes
POST /_/rclone_config/api/remotes      → Create remote
PUT  /_/rclone_config/api/remotes/{id} → Update remote
DELETE /_/rclone_config/api/remotes/{id} → Delete remote
POST /_/rclone_config/api/test         → Test connection
```

### SQLiter Data API

> [!WARNING]
> **Transitional Routes**: These routes exist for legacy support but preference is for pure Banquet URL routing. `/sqliter/sync` and `/sqliter/rows` are candidates for removal.

```
GET /sqliter/file/{path}    → Download cached SQLite file
GET /sqliter/sync/{path}    → Sync metadata (purpose unclear, may be removed)
GET /sqliter/rows?path=...  → Query rows (failed pagination attempt, may be removed)
```

**Design Philosophy**: Flight3 should primarily serve SQLite files directly via Banquet URLs. Clients access data through standard SQLite queries, not custom API endpoints.

---

## Desktop Mode Integration

### Launch Sequence

```
1. mage desktop
   ↓
2. Build flight3 binary
   ↓
3. Find available port (8090-8099)
   ↓
4. Launch flight3 server
   ./flight serve --http=127.0.0.1:8090
   ↓
5. Wait for health check
   curl http://127.0.0.1:8090/api/health
   ↓
6. Build SQLiter-Dart client
   flutter build macos --dart-define=FLIGHT_URL=http://127.0.0.1:8090
   ↓
7. Launch SQLiter app
   open build/macos/Build/Products/Release/sqliter.app
   ↓
8. Client auto-connects to server
   GET /api/auto_login
   GET /api/collections/banquet_links/records
   ↓
9. Display home page with banquet links
```

### Environment Variables

Desktop mode passes server URL to client:
```bash
flutter build macos --dart-define=FLIGHT_URL=http://127.0.0.1:8090
```

Client reads this and auto-configures connection.

---

## Security Model

### Local Desktop Mode

- Auto-login enabled for single-user desktop usage
- Default admin credentials: `admin@example.com` / `password123`
- Server binds to `127.0.0.1` (localhost only)
- No external network exposure

### Production Deployment

For production use:
1. Change default admin password
2. Configure proper authentication
3. Use HTTPS with proper certificates
4. Consider network firewall rules
5. Regular PocketBase backups (`mage backuppocketbase`)

---

## Performance Considerations

### VFS Caching

Rclone VFS uses `CacheModeFull` for optimal performance:
- Entire files cached locally
- Random access support for SQLite
- Connection pooling per remote
- Chunk size: 128MB

### Database Optimization

Converted SQLite databases use:
- WAL mode for concurrent reads
- Appropriate indexes (converter-dependent)
- Pragmas for performance:
  - `journal_mode=WAL`
  - `synchronous=NORMAL`
  - `cache_size=-64000` (64MB)

### Concurrent Access

- PocketBase handles concurrent config reads
- Rclone VFS manages concurrent remote access
- SQLite WAL mode allows concurrent client queries
- Cache validation uses file modification time (fast)

---

## Scalability

### Current Limitations

Flight3 is designed for **single-user, local desktop** use:
- Single process server
- Local file caching
- Expected dataset size: Up to 10GB per file
- Expected concurrent clients: 1-5

### Not Designed For

- Multi-user server deployment
- High-concurrency web applications
- Real-time collaboration
- Datasets > 50GB

For those use cases, consider dedicated solutions like ClickHouse, BigQuery, or traditional database servers.

---

## Future Enhancements

Potential improvements:
1. **Streaming queries** for large datasets
2. **Incremental updates** for cached files
3. **Multi-user authentication** via PocketBase
4. **Webhook support** for cache invalidation
5. **Query result caching** at API level
6. **Compression** for cached SQLite files

---

## Related Documentation

- [README.md](file:///Users/darianhickman/Documents/flight-buddies/flight3/README.md) - Overview and quick start
- [DEVELOPMENT.md](file:///Users/darianhickman/Documents/flight-buddies/flight3/DEVELOPMENT.md) - Development workflow
- [Flight3Dart.md](file:///Users/darianhickman/Documents/flight-buddies/flight3/Flight3Dart.md) - Client integration guide
- [TESTING.md](file:///Users/darianhickman/Documents/flight-buddies/flight3/docs/TESTING.md) - Testing infrastructure
- [BUILD_TOOLS.md](file:///Users/darianhickman/Documents/flight-buddies/flight3/docs/BUILD_TOOLS.md) - Mage commands
