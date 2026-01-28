# Debug Error Response Example

## Testing the Debug Error Feature

### 1. Start the server with DEBUG mode enabled:

```bash
DEBUG=true ./flight serve
```

### 2. Test with an invalid request:

Try accessing a non-existent file:
```bash
curl http://localhost:PORT/nonexistent.csv
```

**Expected Response (with DEBUG=true)**:
```json
{
  "data": {},
  "message": "Local file not found: /nonexistent.csv",
  "status": 404,
  "debug": {
    "banquet": "Banquet{Scheme:\"\" Host:\"\" Path:\"/nonexistent.csv\" Table:\"tb0\" Where:\"\" Limit:\"\" Offset:\"\"}",
    "query": "",
    "error": "stat /path/to/pb_public/nonexistent.csv: no such file or directory"
  }
}
```

### 3. Test with an invalid SQL query:

Try accessing a non-existent table:
```bash
curl http://localhost:PORT/sample.csv/invalid_table
```

**Expected Response (with DEBUG=true)**:
```json
{
  "data": {},
  "message": "Query execution failed",
  "status": 400,
  "debug": {
    "banquet": "Banquet{Scheme:\"\" Host:\"\" Path:\"/sample.csv\" Table:\"invalid_table\" Where:\"\" Limit:\"\" Offset:\"\"}",
    "query": "SELECT * FROM \"invalid_table\"",
    "error": "no such table: invalid_table"
  }
}
```

### 4. Test WITHOUT debug mode:

Restart the server without DEBUG:
```bash
./flight serve
```

Try the same request:
```bash
curl http://localhost:PORT/nonexistent.csv
```

**Expected Response (without DEBUG)**:
```json
{
  "data": {},
  "message": "Local file not found: /nonexistent.csv",
  "status": 404
}
```

Notice: No `debug` field is included!

## Environment Variables

You can enable debug mode with either:
- `DEBUG=true`
- `DEBUG=1`
- `VERBOSE=true`
- `VERBOSE=1`

Any other value (including no value) will disable debug mode.

## Security Note

⚠️ **NEVER run with DEBUG=true in production!**

The debug information exposes:
- SQL queries (reveals database schema)
- Internal file paths
- Error details that could aid attackers

Use debug mode only in development environments.
