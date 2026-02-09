# Conversion Workflow Documentation

## System Architecture

```
┌────────────────────────────────────────────────────────────────┐
│                         SQLiter Flutter App                     │
│                                                                 │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐    │
│  │ File Browser │    │ Recent Files │    │  DB Viewer   │    │
│  └──────┬───────┘    └──────────────┘    └──────────────┘    │
│         │                                                       │
│         │ User opens file.csv                                  │
│         ▼                                                       │
│  ┌────────────────────────────────────────┐                   │
│  │   ConversionService (Phase 1.2)        │                   │
│  │                                         │                   │
│  │  1. Detect file type                   │                   │
│  │  2. Check local cache                  │                   │
│  │  3. If not cached, call Flight3        │                   │
│  │  4. Cache result                       │                   │
│  │  5. Open in DB Viewer                  │                   │
│  └────────────┬───────────────────────────┘                   │
└───────────────┼───────────────────────────────────────────────┘
                │
                │ HTTP POST /api/convert
                │ multipart/form-data
                │
┌───────────────▼───────────────────────────────────────────────┐
│                      Flight3 Server                            │
│                                                                 │
│  ┌────────────────────────────────────────────────────────┐   │
│  │ POST /api/convert                                      │   │
│  │ HandleConvertToSQLite()                                │   │
│  │                                                        │   │
│  │  ┌─────────────────────────────────────────────────┐  │   │
│  │  │ 1. Receive file upload                          │  │   │
│  │  │ 2. Validate (size, type)                        │  │   │
│  │  │ 3. Save to /tmp/flight3-convert/                │  │   │
│  │  │ 4. Call ConvertToSQLite()                       │  │   │
│  │  │    → Uses mksqlite converters                   │  │   │
│  │  │    → CSV → SQLite                               │  │   │
│  │  │    → Excel → SQLite                             │  │   │
│  │  │    → JSON → SQLite                              │  │   │
│  │  │ 5. Serve .db file back to client                │  │   │
│  │  └─────────────────────────────────────────────────┘  │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌────────────────────────────────────────────────────────┐   │
│  │ ConvertToSQLite() - converter.go                       │   │
│  │                                                        │   │
│  │  Uses mksqlite library converters:                    │   │
│  │  • converters/csv                                     │   │
│  │  • converters/excel                                   │   │
│  │  • converters/json                                    │   │
│  │  • converters/html                                    │   │
│  │  • converters/markdown                                │   │
│  │  • converters/txt                                     │   │
│  │  • converters/zip                                     │   │
│  │  • converters/filesystem                              │   │
│  └────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

## User Flow

### Scenario 1: Opening a CSV file

```
1. User clicks "Open File" in SQLiter
   ↓
2. User selects customers.csv
   ↓
3. ConversionService detects .csv extension
   ↓
4. Checks cache: ~/.sqliter/cache/customers_csv.db
   ↓
5. If cache miss:
   ├─→ Send to Flight3 /api/convert
   │   ├─→ Flight3 converts CSV → SQLite
   │   └─→ Returns .db file
   └─→ Save to cache
   ↓
6. Load SQLite file into TrinaGrid
   ↓
7. User sees data in table view
```

### Scenario 2: Opening an Excel file

```
1. User selects sales_report.xlsx
   ↓
2. ConversionService detects .xlsx extension
   ↓
3. Cache miss
   ↓
4. Upload to Flight3 /api/convert
   ↓
5. Flight3 uses excel converter
   ├─→ Reads all sheets
   ├─→ Creates table for each sheet
   └─→ Returns SQLite database
   ↓
6. SQLiter caches locally
   ↓
7. User sees sheet selector + data grid
```

### Scenario 3: Flight3 offline (graceful degradation)

```
1. User selects data.csv
   ↓
2. ConversionService detects .csv extension
   ↓
3. Cache miss
   ↓
4. Try Flight3 /api/convert
   ↓
5. Network error (Flight3 offline)
   ↓
6. Show user-friendly error:
   "Cannot open CSV files directly.
   
   To view this file:
   • Start Flight3 server for automatic conversion
   • Or manually convert to SQLite first"
   ↓
7. User can choose to:
   ├─→ Start Flight3
   └─→ Use external tool to convert
```

## Cache Strategy

### Cache Key Generation

```dart
String getCacheKey(File file) {
  // Use file path + modification time for cache key
  final stat = file.statSync();
  final modTime = stat.modified.millisecondsSinceEpoch;
  final path = file.path.replaceAll('/', '_');
  return '${path}_${modTime}';
}
```

### Cache Location

```
~/.sqliter/cache/
├── _Users_user_data_customers_csv_1707357600000.db
├── _Users_user_data_sales_xlsx_1707357601000.db
└── _tmp_report_json_1707357602000.db
```

### Cache Invalidation

- **Time-based**: 7 days (configurable)
- **Source file modified**: Regenerate if source is newer
- **Manual**: User can clear cache from settings

## API Reference

### POST /api/convert

**Request:**
```http
POST /api/convert HTTP/1.1
Host: localhost:8095
Content-Type: multipart/form-data; boundary=----WebKitFormBoundary

------WebKitFormBoundary
Content-Disposition: form-data; name="file"; filename="data.csv"
Content-Type: text/csv

name,age,city
Alice,30,NYC
------WebKitFormBoundary--
```

**Response (Success):**
```http
HTTP/1.1 200 OK
Content-Type: application/octet-stream
Content-Disposition: attachment; filename="data.db"
Content-Length: 8192

<SQLite binary data>
```

**Response (Error - Unsupported Format):**
```http
HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "error": "Unsupported file type: .xyz",
  "code": "unsupported_format",
  "supported_formats": [".csv", ".xlsx", ".xls", ".json", ...]
}
```

**Response (Error - File Too Large):**
```http
HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "error": "File too large. Maximum size is 100 MB",
  "code": "file_too_large"
}
```

**Response (Error - Conversion Failed):**
```http
HTTP/1.1 500 Internal Server Error
Content-Type: application/json

{
  "error": "Conversion failed",
  "code": "conversion_error",
  "detail": "CSV parse error at line 42: unexpected quote character"
}
```

## Performance Considerations

### File Size Limits

| File Type | Recommended Max | Hard Limit |
|-----------|----------------|------------|
| CSV       | 50 MB          | 100 MB     |
| Excel     | 25 MB          | 100 MB     |
| JSON      | 50 MB          | 100 MB     |
| ZIP       | 100 MB         | 100 MB     |

### Conversion Times (approx)

| File Type | Size  | Time         |
|-----------|-------|--------------|
| CSV       | 1 MB  | ~1 second    |
| CSV       | 10 MB | ~5 seconds   |
| Excel     | 1 MB  | ~2 seconds   |
| Excel     | 10 MB | ~10 seconds  |
| JSON      | 1 MB  | ~1.5 seconds |

### Network Considerations

- **Upload**: Limited by user's upload speed
- **Download**: SQLite files are typically larger than source (indexes, metadata)
- **Caching**: Reduces network calls after first conversion

## Security Notes

### Current Implementation

- ✅ File size validation (100MB limit)
- ✅ Extension-based type checking
- ✅ Temporary file isolation
- ⚠️ No authentication required
- ⚠️ No rate limiting
- ⚠️ No MIME type validation

### Production Recommendations

1. **Add Authentication**: Require API key or user auth
2. **Implement Rate Limiting**: Prevent abuse (e.g., 10 conversions/minute)
3. **Add MIME Type Validation**: Don't trust file extensions alone
4. **Scan for Malicious Content**: Especially for ZIP files
5. **Implement Cleanup Job**: Remove old temp files automatically
6. **Add Request Logging**: Track who converts what
7. **Add Timeout**: Kill long-running conversions

## Known Limitations

1. **Temp File Cleanup**: Currently no automatic cleanup of /tmp/flight3-convert/
2. **Concurrent Requests**: No limit on simultaneous conversions
3. **Memory Usage**: Large Excel files can use significant memory
4. **ZIP File Security**: No validation of ZIP contents before extraction
5. **Error Details**: Some mksqlite errors may not be user-friendly

## Future Enhancements (Not in Phase 1)

- **Streaming Conversion**: For very large files
- **Progress Reporting**: WebSocket for conversion progress
- **Batch Conversion**: Upload multiple files at once
- **Format Options**: Allow user to specify conversion parameters
- **Preview**: Return sample data before full conversion
- **Async Jobs**: Queue large conversions, poll for completion
