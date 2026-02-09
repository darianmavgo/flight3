# Client-Side Migration Implementation

## Phase 1: Server-Side (Flight3) ✅ COMPLETED

### Completed Changes:
1. ✅ Added `recent_files` PocketBase collection with user relation
2. ✅ Created `/api/validate-path` endpoint
3. ✅ Created `/api/parse-path` endpoint  
4. ✅ Created `/api/cache/stats`, `/api/cache/clear`, `/api/cache/entry` endpoints
5. ✅ Build successful

---

## Phase 2: Client-Side (SQLiter) - IN PROGRESS

### 1. RecentFilesService → Flight3 API

**Files to Modify:**
- `lib/services/recent_files_service.dart` - Replace with Flight API client
- `lib/main.dart` - Update initialization

**New Implementation:**
```dart
class RecentFilesService {
  final FlightService _flightService;
  
  Future<List<RecentFile>> getRecentFiles() async {
    final response = await _flightService.get(
      '/api/collections/recent_files/records',
      queryParams: {
        'sort': '-last_opened',
        'perPage': '15',
      },
    );
    return (response['items'] as List)
        .map((json) => RecentFile.fromJson(json))
        .toList();
  }
  
  Future<void> addRecentFile({
    required String path,
    bool wasConverted = false,
    String? originalFormat,
  }) async {
    await _flightService.post(
      '/api/collections/recent_files/records',
      body: {
        'user': _flightService.userId, // From auth
        'path': path,
        'name': basename(path),
        'last_opened': DateTime.now().toIso8601String(),
        'was_converted': wasConverted,
        'original_format': originalFormat,
      },
    );
  }
  
  Future<void> removeRecentFile(String id) async {
    await _flightService.delete(
      '/api/collections/recent_files/records/$id',
    );
  }
}
```

**Lines Reduced:** 158 → ~60

---

### 2. CacheService → Delegate to Flight3

**Files to Modify:**
- `lib/cache_service.dart` - Gut and delegate to server
- `lib/conversion_service.dart` - Remove local cache logic

**New Implementation:**
```dart
class CacheService {
  final FlightService _flightService;
  
  // Server handles caching automatically in /api/convert
  // Client just needs stats/management endpoints
  
  Future<Map<String, dynamic>> getCacheStats() async {
    return await _flightService.get('/api/cache/stats');
  }
  
  Future<void> clearCache() async {
    await _flightService.delete('/api/cache/clear');
  }
}
```

**Lines Reduced:** 208 → ~20

---

### 3. PathValidator → Flight3 API

**Files to Modify:**
- `lib/utils/path_validator.dart` - Replace with API calls
- `lib/main.dart` - Update error handling

**New Implementation:**
```dart
class PathValidator {
  static Future<Map<String, dynamic>> validatePathBackwards(
    FlightService flight,
    String fullPath,
  ) async {
    return await flight.get(
      '/api/validate-path',
      queryParams: {'path': fullPath},
    );
  }
  
  static Future<String> generateSmartErrorMessage(
    FlightService flight,
    String fullPath,
  ) async {
    final result = await validatePathBackwards(flight, fullPath);
    
    if (result['error_message'] != null) {
      return result['error_message'];
    }
    
    return 'Path not found:\n\n' +
           '✓ Valid: ${result["valid_segments"].join("/")}\n' +
           '✗ Not found: ${result["break_point"]}\n\n' +
           result['error_message'] ?? '';
  }
}
```

**Lines Reduced:** 83 → ~25

---

### 4. ConversionService → Pure HTTP Client

**Files to Modify:**
- `lib/conversion_service.dart` - Simplify to just HTTP upload

**New Implementation:**
```dart
class ConversionService {
  final FlightService _flightService;
  
  // Server now handles:
  // - Format detection
  // - Cache checking
  // - Conversion
  // - Error messages
  
  Future<File> ensureSqlite(File file) async {
    // Check if already SQLite
    if (isSqliteFile(file)) return file;
    
    // Upload to server, get converted result
    return await _flightService.uploadForConversion(file);
  }
  
  bool isSqliteFile(File file) {
    final ext = extension(file.path).toLowerCase();
    return ['.db', '.sqlite', '.sqlite3'].contains(ext);
  }
}
```

**Lines Reduced:** 202 → ~40

---

### 5. Breadcrumb Parsing → Flight3 API

**Files to Modify:**
- `lib/widgets/breadcrumb_path_field.dart` - Use `/api/parse-path`

**New Implementation:**
```dart
Future<List<PathSegment>> _parsePathSegments(String path) async {
  final response = await _flightService.get(
    '/api/parse-path',
    queryParams: {'url': path},
  );
  
  return (response['segments'] as List)
      .map((json) => PathSegment.fromJson(json))
      .toList();
}
```

**Current complex logic:** ~80 lines
**New logic:** ~10 lines

---

## Testing Plan

After each migration:
1. Run `mage testsqliter` (unit tests)
2. Run `mage testdesktop` (integration tests)
3. Manual smoke test:
   - Launch app
   - Open a file
   - Verify recent files sync
   - Check breadcrumb navigation
   - Test conversion

---

## Rollout Order

1. **PathValidator** (lowest risk, easiest to test)
2. **BreadcrumbParsing** (visual feedback, easy to verify)
3. **CacheService** (medium risk, performance win)
4. **ConversionService** (depends on cache)
5. **RecentFilesService** (highest value, cross-device sync)

---

## Expected Outcomes

- **Lines of Dart removed:** ~470 lines
- **Async complexity:** Dramatically reduced
- **Cross-device sync:** Enabled for recent files
- **Performance:** Better with shared server cache
- **Security:** Server-side path validation
- **Maintainability:** Single source of truth for business logic
