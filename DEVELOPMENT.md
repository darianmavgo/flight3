# Development Workflow Guide

## Quick Start

### Start Development Environment

**Terminal 1: Go Server with Hot Reload**
```bash
cd flight3
air
```

The server will automatically rebuild on code changes using Air.

**Terminal 2: Flutter Client**
```bash
cd sqliter-dart
flutter run  -d chrome
```

---

## PocketBase Backup Automation

### Automatic Backups

The `mage clean` command now **automatically backs up PocketBase data** before cleaning:

```bash
cd flight3
mage clean
```

This creates a timestamped backup in `pb_data/backups/pb_backup_flight3_YYYYMMDD_HHMMSS.zip` containing:
- `data.db` (collection schemas and data)
- `auxiliary.db` (system data)
- WAL files (write-ahead logs)

### Manual Backups

Create a backup anytime:

```bash
cd flight3
mage backuppocketbase
```

### Restore from Backup

```bash
cd flight3/pb_data
# Stop the server first!
unzip backups/pb_backup_flight3_20260207_120000.zip
# Start the server
```

---

## Unified Logging

### Server-Side Logging (Go)

Use the `devlog` package for structured logging:

```go
import "github.com/darianmavgo/flight3/pkg/devlog"

logger := devlog.New("flight3")
logger.Info("Server started", map[string]interface{}{
    "port": 8090,
    "mode": "development",
})

logger.Error("Database error", err, map[string]interface{}{
    "table": "users",
})
```

Output format:
```json
{"timestamp":"2026-02-07T12:00:00.123456Z","level":"INFO","source":"flight3","message":"Server started","context":{"port":8090,"mode":"development"}}
```

### Client-Side Logging (Dart/Flutter)

Add timestamp-prefixed logs:

```dart
void log(String message, {String level = 'INFO'}) {
  final timestamp = DateTime.now().toUtc().toIso8601String();
  print('{"timestamp":"$timestamp","level":"$level","source":"flutter","message":"$message"}');
}
```

### Merge Server and Client Logs

```bash
cd flight3
mage mergelogs server.log ../sqliter-dart/logs/app.log
```

View merged logs:
```bash
cat merged_logs.json | jq
```

Filter by source:
```bash
cat merged_logs.json | jq 'select(.source=="flutter")'
```

Filter by time range:
```bash
cat merged_logs.json | jq 'select(.timestamp > "2026-02-07T12:00:00")'
```

---

## Hot Reload Configuration

### Go (Air)

Air configuration is in `flight3/.air.toml`. It watches for changes and rebuilds automatically.

**Customization:**
- Edit `.air.toml` to change watched directories
- Logs are in `build-errors.log`

### Flutter

Flutter hot reload works automatically with `flutter run`. Press `r` to reload, `R` to restart.

---

## Best Practices

### 1. Always Backup Before Major Changes
```bash
mage backuppocketbase  # Manual backup
# Make your changes
# Test
```

### 2. Use Structured Logging
- **Do:** `logger.Info("User login", map[string]interface{}{"user_id": 123})`
- **Don't:** `fmt.Println("User 123 logged in")`

This makes logs mergeable and searchable.

### 3. Correlate Logs by Request ID

Add request IDs to both server and client logs:

**Server:**
```go
requestID := uuid.New().String()
logger.Info("API request", map[string]interface{}{
    "request_id": requestID,
    "endpoint": "/api/data",
})
```

**Client:**
```dart
final requestID = Uuid().v4();
log('Making API request', context:{'request_id': requestID});
```

Then filter merged logs:
```bash
cat merged_logs.json | jq 'select(.context.request_id=="abc-123")'
```

---

## Troubleshooting

### Air Not Rebuilding

1. Check `.air.toml` excluded directories
2. Verify Go files have correct extensions (`.go`)
3. Check `build-errors.log`

### Backup Failed

Backups fail gracefully. `mage clean` will ask for confirmation if backup fails.

To force backup:
```bash
mage backuppocketbase
```

### Logs Not Merging

Ensure both log files use JSON format with matching timestamp format. Non-JSON logs will be treated as plaintext with current timestamp.
