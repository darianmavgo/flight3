# Debug Error Response - Quick Reference

## Enable Debug Mode
```bash
DEBUG=true ./flight serve
```

## Error Response Comparison

### Production (Default - No Debug)
```json
{
  "data": {},
  "message": "Query execution failed",
  "status": 400
}
```

### Development (With Debug)
```json
{
  "data": {},
  "message": "Query execution failed",
  "status": 400,
  "debug": {
    "banquet": "Banquet{...}",
    "query": "SELECT * FROM ...",
    "error": "original error message"
  }
}
```

## Environment Variables
- `DEBUG=true` or `DEBUG=1` → Debug mode ON
- `VERBOSE=true` or `VERBOSE=1` → Debug mode ON
- Any other value or unset → Debug mode OFF (default)

## ⚠️ Security Warning
**NEVER use DEBUG=true in production!**

Debug mode exposes:
- SQL queries
- Database schema
- Internal file paths
- Detailed error messages

## Testing
```bash
# Run tests
go test -v -run TestError ./internal/flight/

# Build
go build ./cmd/flight/
```

## Files
- `internal/flight/errors.go` - Implementation
- `internal/flight/errors_test.go` - Tests
- `DEBUG_ERRORS.md` - Full documentation
- `DEBUG_ERRORS_EXAMPLE.md` - Usage examples
