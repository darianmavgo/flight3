# Flight3 Improvement Roadmap

Based on user feedback from ARCHITECTURE_Fix_List.md, this document outlines planned improvements to simplify architecture and enhance the feedback loop.

## Core Philosophy

> **Flight3 should primarily just make SQLite available to clients.**
> 
> Minimize custom API routes. In the future, admin routes should be routes to SQLite tables that affect settings and operation.

## Immediate Fixes (Documentation)

### ✅ Completed
- [x] Fixed Banquet URL terminology (User vs Host)
- [x] Removed non-existent Settings UI from diagrams  
- [x] Clarified ColumnPath as selection+sort syntax
- [x] Added notes about transitional API routes
- [x] Documented planned routing changes

## Code Improvements

### Phase 1: Route Simplification

#### 1.1 Reroute Rclone ConfigChanged from: `/_/rclone_config`
To: `/_/rclone/`

**Rationale:** Consistency with PocketBase admin routes pattern.

```go
// router.go
- se.Router.GET("/_/rclone_config", HandleRcloneConfigUI)
+ se.Router.GET("/_/rclone/", HandleRcloneConfigUI)

// Update all API subroutes
- se.Router.GET("/_/rclone_config/api/...")
+ se.Router.GET("/_/rclone/api/...")
```

#### 1.2 Fix Root Path Behavior

**Current:** Redirects to `/_/` (PocketBase admin)
**Desired:** Serve Banquet URL for configured home directory

```go
// router.go
se.Router.Any("/", func(e *core.RequestEvent) error {
    // Get serve_folder from app_settings PocketBase table
    settings, err := GetAppSettings(e.App)
    if err != nil || settings.ServeFolder == "" {
        // Fallback to PocketBase admin if not configured
        return e.Redirect(302, "/_/")
    }
    
    // Construct Banquet URL for serve_folder
    // e.g., file://./data or local path
    banquetURL := fmt.Sprintf("file://%s", settings.ServeFolder)
    
    // Handle as standard Banquet request
    return HandleBanquetWithRedirect(e, banquetURL)
})
```

**User Experience:**
- Root URL becomes user-visible Banquet URL
- Displays directory listing of configured folder
- API routes remain at `/_/` and `/api/`

#### 1.3 Review and Remove Failed API Routes

**Investigate `/sqliter/sync`:**
```bash
# Find all usages
grep -r "HandleBanquetSync" internal/
grep -r "/sqliter/sync" ../sqliter/
```

**Questions:**
- What does HandleBanquetSync do?
- Is it used by SQLiter-Dart client?
- Can functionality beintegrated into main Banquet handler?

**Remove `/sqliter/rows`** (confirmed failed pagination attempt):
```go
// router.go
- se.Router.GET("/sqliter/rows", HandleBanquetRows)
```

**Migration Path:**
- Client uses `#anchor` for page numbers instead
- Example: `http://server/data.csv;tb0#page=5`
- Client parses anchor, server just serves SQLite file

### Phase 2: Testing Infrastructure

#### 2.1 Banquet URL Parsing Tests

**Goal:** Verify correct parsing of User@Host pattern

```go
// tests/banquet_url_test.go
package tests

import (
    "testing"
    "github.com/darianmavgo/banquet"
    "github.com/stretchr/testify/assert"
)

func TestBanquetURLUserInfoParsing(t *testing.T) {
    tests := []struct {
        name     string
        url      string
        wantUser string
        wantHost string
    }{
        {
            name:     "S3 with remote alias",
            url:      "s3://bucket@aws/data/file.csv",
            wantUser: "bucket",
            wantHost: "aws",
        },
        {
            name:     "GCS with remote alias",
            url:      "gs://mybucket@gcloud/data/file.csv",
            wantUser: "mybucket",
            wantHost: "gcloud",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            b, err := banquet.ParseNested(tt.url)
            assert.NoError(t, err)
            assert.Equal(t, tt.wantUser, b.User.Username())
            assert.Equal(t, tt.wantHost, b.Host)
        })
    }
}

func TestBanquetColumnPathParsing(t *testing.T) {
    tests := []struct {
        name         string
        url          string
        wantSelect   []string
        wantOrderBy  string
        wantWhere    string
    }{
        {
            name:       "Columns with sort",
            url:        "data.csv;tb0;name,amount;+date",
            wantSelect: []string{"name", "amount"},
            wantOrderBy: "date",
        },
        {
            name:       "Columns with condition",
            url:        "data.csv;tb0;name,amount;status!=active",
            wantSelect: []string{"name", "amount"},
            wantWhere:  "status != 'active'",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            b, err := banquet.ParseBanquet(tt.url)
            assert.NoError(t, err)
            assert.Equal(t, tt.wantSelect, b.Select)
            if tt.wantOrderBy != "" {
                assert.Equal(t, tt.wantOrderBy, b.OrderBy)
            }
            if tt.wantWhere != "" {
                assert.Contains(t, b.Where, tt.wantWhere)
            }
        })
    }
}
```

#### 2.2 Root Path Integration Test

```go
// tests/root_path_test.go
func TestRootPathServeFolder(t *testing.T) {
    // Setup test Flight3 server
    app := setupTestApp(t)
    
    // Configure serve_folder in app_settings
    settings := &AppSettings{
        ServeFolder: "./testdata/sample_folder",
    }
    err := SaveAppSettings(app, settings)
    require.NoError(t, err)
    
    // Request root path
    req := httptest.NewRequest("GET", "/", nil)
    resp := httptest.NewRecorder()
    
    app.OnBeforeServe().Trigger(&core.ServeEvent{
        Router: echo.New(),
    })
    
    // Verify response
    assert.Equal(t, 200, resp.Code)
    assert.NotContains(t, resp.Header().Get("Location"), "/_/")
    
    // Should contain directory listing data
    body := resp.Body.String()
    assert.Contains(t, body, "Dataset cached")
}
```

#### 2.3 Client Pagination Test (Design Validation)

```go
// tests/client_pagination_test.go
func TestClientSidePaginationViaAnchor(t *testing.T) {
    // This test validates the DESIGN, not implementation
    
    // URL pattern: data.csv;tb0#page=5
    // Flight3 should:
    // 1. Ignore #anchor (client-side only)
    // 2. Serve full SQLite file
    // 3. Client handles pagination
    
    url := "data.csv;tb0#page=5"
    b, err := banquet.ParseNested(url)
    assert.NoError(t, err)
    
    // Banquet should strip anchor
    assert.Equal(t, "data.csv", b.DataSetPath)
    assert.Equal(t, "tb0", b.Table)
    assert.Empty(t, b.Fragment) // URL fragment not in Banquet
    
    // Client responsibility to parse #page=5
}
```

### Phase 3: Enhanced Feedback Loop

#### 3.1 Banquet URL Validation Endpoint

Add a validation endpoint for development:

```go
// router.go
se.Router.GET("/_/debug/banquet", func(e *core.RequestEvent) error {
    rawURL := e.QueryParam("url")
    if rawURL == "" {
        return e.JSON(400, map[string]string{
            "error": "Missing 'url' parameter",
        })
    }
    
    b, err := banquet.ParseNested(rawURL)
    if err != nil {
        return e.JSON(400, map[string]interface{}{
            "error": err.Error(),
            "url":   rawURL,
        })
    }
    
    return e.JSON(200, map[string]interface{}{
        "rawURL":      rawURL,
        "scheme":      b.Scheme,
        "user":        b.User.Username(),
        "host":        b.Host,
        "dataSetPath": b.DataSetPath,
        "table":       b.Table,
        "columnPath":  b.ColumnPath,
        "select":      b.Select,
        "where":       b.Where,
        "orderBy":     b.OrderBy,
        "limit":       b.Limit,
        "offset":      b.Offset,
    })
})
```

**Usage:**
```bash
curl "http://localhost:8090/_/debug/banquet?url=s3://bucket@aws/data.csv;tb0;name,+date"
```

#### 3.2 Verbose Logging for Development

```go
// flight.go - Add flag
var debugBanquet = flag.Bool("debug-banquet", false, "Enable verbose Banquet URL parsing")

func init() {
    if *debugBanquet {
        banquet.SetVerbose(true)
    }
}
```

**Usage:**
```bash
./flight serve --http=127.0.0.1:8090 --debug-banquet
```

#### 3.3 Test Data Generator

Create realistic test Banquet URLs:

```go
// tests/testdata/banquet_urls.txt
# Basic file access
file://./data/sample.csv

# S3 with remote
s3://mybucket@aws/reports/2024.csv

# Column selection
s3://mybucket@aws/reports/2024.csv;tb0;name,revenue

# Sorting
s3://mybucket@aws/reports/2024.csv;tb0;+date

# Filtering
s3://mybucket@aws/reports/2024.csv;tb0;status!=pending

# Everything combined
s3://mybucket@aws/reports/2024.csv;tb0;name,revenue;status!=pending;+date?limit=100
```

```bash
# Test all URLs
go test -v ./tests -run TestAllBanquetURLs
```

## Future Enhancements (From UPGRADE_IDEAS.md)

### Query Styles (Inspired by SwiftUI/Tailwind)

**Concept:** Default presentation rules for data, customizable per-table

```sql
-- CREATE TABLE query_styles (
--     table_name TEXT PRIMARY KEY,
--     auto_hide_null BOOLEAN DEFAULT true,
--     hide_columns TEXT,  -- JSON array: ["created_at", "updated_at"]
--     column_widths TEXT, -- JSON: {"name": 200, "amount": 100}
--     default_sort TEXT   -- "+date"
-- )
```

**Example:**
```javascript
// Client fetches query_style for table
const style = await getQueryStyle("sales");

// Apply to PlutoGrid
plutoGrid.hideColumns(style.hide_columns);
plutoGrid.setColumnWidths(style.column_widths);

// Auto-hide null columns
if (style.auto_hide_null) {
    columns.forEach(col => {
        if (isAllNull(col)) hideColumn(col);
    });
}
```

### Usage-Based Column Optimization

Track which columns users actually view:

```sql
-- CREATE TABLE column_usage (
--     table_name TEXT,
--     column_name TEXT,
--     view_count INTEGER DEFAULT 0,
--     last_viewed TIMESTAMP,
--     PRIMARY KEY (table_name, column_name)
-- )
```

**Auto-adjust defaults:**
- Frequently viewed columns → wider
- Never viewed columns → auto-hide
- Recently viewed → keep visible

## Migration Plan

### Week 1: Documentation & Testing
- [x] Fix ARCHITECTURE.md errors
- [ ] Add Banquet URL parsing tests
- [ ] Add root path integration test
- [ ] Document client pagination pattern

### Week 2: Route Simplification
- [ ] Implement rclone route migration (`/_/rclone/`)
- [ ] Remove `/sqliter/rows` endpoint
- [ ] Investigate and decide on `/sqliter/sync`
- [ ] Test API changes with SQLiter-Dart client

### Week 3: Root Path Behavior
- [ ] Add serve_folder to app_settings schema
- [ ] Implement root path Banquet URL handler
- [ ] Test directory listing rendering
- [ ] Update README with new root behavior

### Week 4: Feedback Loop
- [ ] Add `/_/debug/banquet` validation endpoint
- [ ] Add `--debug-banquet` flag
- [ ] Create test data generator
- [ ] Write iteration guide for developers

## Success Metrics

- **Reduced API surface:** 3 custom routes → 1 route (just `/sqliter/file`)
- **Improved testing:** 0 Banquet URL tests → 20+ test cases
- **Faster iteration:** Manual URL testing → automated validation endpoint
- **Clear architecture:** Custom pagination → client-side #anchors

## Related Documents

- [ARCHITECTURE_Fix_List.md](file:///Users/darianhickman/Documents/flight-buddies/flight3/docs/ARCHITECTURE_Fix_List.md) - User feedback
- [UPGRADE_IDEAS.md](file:///Users/darianhickman/Documents/flight-buddies/flight3/docs/UPGRAD E_IDEAS.md) - Future enhancements
- [ARCHITECTURE.md](file:///Users/darianhickman/Documents/flight-buddies/flight3/docs/ARCHITECTURE.md) - Current architecture (corrected)
- [TESTING.md](file:///Users/darianhickman/Documents/flight-buddies/flight3/docs/TESTING.md) - Testing guide
