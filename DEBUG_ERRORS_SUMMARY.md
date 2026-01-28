# Debug Error Response Implementation Summary

## What Was Implemented

Added conditional debug information to error responses in Flight3. When `DEBUG` or `VERBOSE` environment variables are set, error responses include detailed information about the Banquet URL structure and SQL query that was attempted.

## Files Created/Modified

### Created Files:
1. **`internal/flight/errors.go`** - Core error handling system
   - `ErrorResponse` struct matching PocketBase format
   - `DebugInfo` struct for debug data
   - `IsDebugMode()` function
   - `SendBanquetError()` function
   - `BanquetError` type for wrapping errors with context
   - `HandleBanquetError()` function

2. **`internal/flight/errors_test.go`** - Comprehensive test suite
   - Tests for error response format with/without debug mode
   - Tests for `IsDebugMode()` function
   - Tests for `BanquetError` type

3. **`DEBUG_ERRORS.md`** - Full documentation
   - Overview and security warnings
   - Usage examples
   - Implementation details

4. **`DEBUG_ERRORS_EXAMPLE.md`** - Quick start guide
   - curl command examples
   - Expected responses

### Modified Files:
1. **`internal/flight/server.go`**
   - Updated error returns to use `NewBanquetError()`
   - Captures query context for error responses

2. **`internal/flight/banquethandler.go`**
   - Updated all error returns to use `NewBanquetError()`
   - Provides better error messages and context

3. **`internal/flight/flight.go`**
   - Updated `banquetHandler` to use `HandleBanquetError()`
   - Properly formats error responses

## How It Works

### Without Debug Mode (Production - Default)
```json
{
  "data": {},
  "message": "Query execution failed",
  "status": 400
}
```

### With Debug Mode (Development)
```bash
DEBUG=true ./flight serve
```

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

## Security Considerations

⚠️ **WARNING: This customizes PocketBase's default error handling behavior**

### What's Exposed in Debug Mode:
- SQL queries (reveals database schema)
- Banquet URL structure (internal routing)
- Underlying error messages (may contain sensitive paths)

### Best Practices:
1. **Never enable debug mode in production**
2. Use environment-specific configuration
3. Review logs before sharing
4. Use secure channels when sharing debug output

## Testing

All tests pass:
```bash
go test -v -run TestError ./internal/flight/
```

Results:
- ✅ TestErrorResponseFormat (all 3 sub-tests pass)
- ✅ TestIsDebugMode (all 6 sub-tests pass)
- ✅ TestBanquetError (passes)

## Usage

### Enable Debug Mode:
```bash
# Option 1
DEBUG=true ./flight serve

# Option 2
VERBOSE=true ./flight serve

# Option 3
DEBUG=1 ./flight serve
```

### Disable Debug Mode (Default):
```bash
./flight serve
```

## Benefits

1. **Better Development Experience**: Developers can see exactly what went wrong
2. **Faster Debugging**: SQL queries and Banquet structures are immediately visible
3. **Production Safe**: Debug info is only shown when explicitly enabled
4. **PocketBase Compatible**: Maintains standard error format, just adds optional field
5. **Backward Compatible**: Existing error handling still works

## Answer to Original Question

**Q: Can the banquet and query also be embedded in error responses?**

**A: Yes!** ✅

**Q: Warn me if this means customizing PocketBase default behavior**

**A: ⚠️ WARNING - This DOES customize PocketBase's default error handling**

However, the customization is:
- **Safe**: Only active when DEBUG mode is explicitly enabled
- **Minimal**: Adds an optional `debug` field, doesn't change existing fields
- **Secure**: Disabled by default to prevent information leakage
- **Reversible**: Can be completely disabled via environment variable

The implementation follows Option 1 (Environment Variable Control) as requested, providing the best balance between developer experience and production security.
