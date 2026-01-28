# Debug Error Responses

## Overview

Flight3 now supports **conditional debug information** in error responses. When debug mode is enabled, error responses include detailed information about the Banquet URL structure and the SQL query that was attempted.

## Error Response Format

### Standard Response (Production)

When debug mode is **disabled** (default), errors follow PocketBase's standard format:

```json
{
  "data": {},
  "message": "Something went wrong while processing your request.",
  "status": 400
}
```

### Debug Response (Development)

When debug mode is **enabled**, errors include additional debug information:

```json
{
  "data": {},
  "message": "Query execution failed",
  "status": 400,
  "debug": {
    "banquet": "Banquet{Scheme:\"\" Host:\"myremote\" Path:\"/data.csv\" Table:\"tb0\" Where:\"id > 10\" Limit:\"100\" Offset:\"\"}",
    "query": "SELECT * FROM \"tb0\" WHERE id > 10 LIMIT 100",
    "error": "no such column: id"
  }
}
```

## Enabling Debug Mode

Set either of these environment variables to enable debug mode:

```bash
# Option 1: Using DEBUG
DEBUG=true ./flight serve

# Option 2: Using VERBOSE
VERBOSE=true ./flight serve

# Option 3: Using numeric values
DEBUG=1 ./flight serve
```

## Security Considerations

⚠️ **WARNING: This customizes PocketBase's default error handling behavior**

### Why This Matters

PocketBase's default error handler intentionally provides generic error messages to prevent leaking sensitive information. When debug mode is enabled, the following information is exposed:

1. **SQL Queries**: Reveals your database schema and query structure
2. **Banquet Structure**: Shows internal routing and data source paths
3. **Error Details**: Exposes underlying error messages that might contain sensitive info

### Best Practices

1. **Never enable debug mode in production**
2. **Use environment-specific configuration**:
   ```bash
   # Development
   DEBUG=true ./flight serve
   
   # Production (default - no debug)
   ./flight serve
   ```
3. **Review logs before sharing** - debug info may also appear in server logs
4. **Use secure channels** when sharing debug output with team members

## Implementation Details

### Error Types

The system uses a custom `BanquetError` type that wraps errors with context:

```go
type BanquetError struct {
    Err     error              // Original error
    Message string             // User-friendly message
    Status  int                // HTTP status code
    Banquet *banquet.Banquet   // Banquet URL context
    Query   string             // SQL query that was attempted
}
```

### Usage in Code

```go
// Create a BanquetError with context
return NewBanquetError(
    err,                    // original error
    "Query execution failed", // user message
    400,                    // HTTP status
    b,                      // banquet context
    query,                  // SQL query
)
```

### Error Handler

The `HandleBanquetError` function automatically:
1. Checks if debug mode is enabled
2. Formats the error response appropriately
3. Includes debug info only when safe to do so

## Testing

Run the error handling tests:

```bash
cd internal/flight
go test -v -run TestError
```

## Examples

### Example 1: Invalid Table Name (Debug Mode)

**Request**: `GET /myremote/data.csv/nonexistent_table`

**Response** (with `DEBUG=true`):
```json
{
  "data": {},
  "message": "Query execution failed",
  "status": 400,
  "debug": {
    "banquet": "Banquet{Scheme:\"\" Host:\"myremote\" Path:\"/data.csv\" Table:\"nonexistent_table\" Where:\"\" Limit:\"\" Offset:\"\"}",
    "query": "SELECT * FROM \"nonexistent_table\"",
    "error": "no such table: nonexistent_table"
  }
}
```

### Example 2: File Not Found (Debug Mode)

**Request**: `GET /local/missing.csv`

**Response** (with `DEBUG=true`):
```json
{
  "data": {},
  "message": "Local file not found: /local/missing.csv",
  "status": 404,
  "debug": {
    "banquet": "Banquet{Scheme:\"\" Host:\"\" Path:\"/local/missing.csv\" Table:\"\" Where:\"\" Limit:\"\" Offset:\"\"}",
    "query": "",
    "error": "stat /path/to/pb_public/local/missing.csv: no such file or directory"
  }
}
```

### Example 3: Same Error Without Debug Mode

**Request**: `GET /local/missing.csv`

**Response** (without debug mode):
```json
{
  "data": {},
  "message": "Local file not found: /local/missing.csv",
  "status": 404
}
```

## Related Files

- `internal/flight/errors.go` - Error handling implementation
- `internal/flight/errors_test.go` - Test suite
- `internal/flight/banquethandler.go` - Uses BanquetError for context
- `internal/flight/server.go` - Query execution error handling
- `internal/flight/flight.go` - Main error handler integration
