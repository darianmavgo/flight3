package flight

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/darianmavgo/banquet"
	_ "github.com/darianmavgo/mksqlite/converters/all"
	"github.com/pocketbase/pocketbase/core"
	_ "modernc.org/sqlite" // Register driver
)

var extensionMap = map[string]string{
	".csv":      "csv",
	".xlsx":     "excel",
	".xls":      "excel",
	".tbc":      "excel", // old tbc support? just copy map
	".zip":      "zip",
	".html":     "html",
	".htm":      "html",
	".json":     "json",
	".txt":      "txt",
	".md":       "markdown",
	".markdown": "markdown",
	".db":       "sqlite",
	".sqlite":   "sqlite",
	".sqlite3":  "sqlite",
}

// EnsureBanquetDataset resolves the Banquet URL, ensures the local cache is fresh (fetching from remote if needed),
// and returns the Banquet object and the absolute path to the cached SQLite file.
func EnsureBanquetDataset(e *core.RequestEvent, reqURI string, verbose bool) (*banquet.Banquet, string, error) {
	// 1. Parse Banquet URL
	reqURI = strings.TrimPrefix(reqURI, "/")

	b, err := banquet.ParseNested(reqURI)
	if err != nil {
		if verbose {
			log.Printf("[BANQUET] Invalid banquet URL: %s", reqURI)
		}
		return nil, "", NewBanquetError(err, "Invalid banquet URL format", 400, nil, "", "")
	}

	if verbose {
		banquet.FmtPrintln(b)
	}

	// 2. Handle Local Dataset
	if b.Scheme == "" && b.Hostname() == "" {
		cachePath, err := EnsureLocalDataset(e, b, verbose)
		if err != nil {
			return nil, "", err
		}
		return b, cachePath, nil
	}

	// 3. Lookup Remote Configuration
	remoteRecord, err := LookupRemote(e.App, b.Hostname())
	if err != nil {
		// Check for ad-hoc HTTP/HTTPS support
		isHTTP := strings.HasPrefix(reqURI, "http:")
		isHTTPS := strings.HasPrefix(reqURI, "https:")

		if isHTTP || isHTTPS {
			if verbose {
				log.Printf("[BANQUET] Remote '%s' not found, attempting ad-hoc HTTP remote", b.Hostname())
			}

			collection, errCol := e.App.FindCollectionByNameOrId("rclone_remotes")
			if errCol != nil {
				return nil, "", NewBanquetError(errCol, "Failed to find rclone_remotes collection", 500, b, "", "")
			}

			// Create temporary in-memory record
			remoteRecord = core.NewRecord(collection)
			remoteRecord.Set("type", "http")

			scheme := "http"
			if isHTTPS {
				scheme = "https"
			}

			// Configure rclone http backend
			remoteRecord.Set("config", map[string]interface{}{
				"url": fmt.Sprintf("%s://%s", scheme, b.Hostname()),
			})
		} else {
			return nil, "", NewBanquetError(err, fmt.Sprintf("Remote '%s' not found", b.Hostname()), 404, b, "", "")
		}
	}

	// 4. Initialize VFS
	rcloneManager := GetRcloneManager()
	if rcloneManager == nil {
		return nil, "", NewBanquetError(nil, "Rclone manager not initialized", 500, b, "", "")
	}

	vfs, err := rcloneManager.GetVFS(remoteRecord)
	if err != nil {
		return nil, "", NewBanquetError(err, "Failed to initialize VFS", 500, b, "", "")
	}

	// 5. Generate Cache Key
	cacheKey := GenCacheKey(b)
	cachePath := GetCachePath(e.App.DataDir(), cacheKey)

	if verbose {
		log.Printf("[BANQUET] Cache key: %s", cacheKey)
		log.Printf("[BANQUET] Cache path: %s", cachePath)
	}

	// 6. Check Cache Validity
	ttl := 1440.0 // 24 hours
	valid, err := ValidateCache(cachePath, ttl)
	if err != nil {
		log.Printf("[BANQUET] Cache validation error: %v", err)
		valid = false
	}

	// 7. Fetch and Convert if Cache Miss
	if !valid {
		if verbose {
			log.Printf("[BANQUET] Cache miss or expired, fetching and converting...")
		}

		// Check if it's a directory or a file
		node, err := rcloneManager.Stat(vfs, b.DataSetPath)
		if err != nil {
			return nil, "", NewBanquetError(err, fmt.Sprintf("Failed to access remote path: %s", b.DataSetPath), 404, b, "", "")
		}

		if node.IsDir() {
			// Remote directory - index it
			if err := rcloneManager.IndexDirectory(vfs, b.DataSetPath, cachePath); err != nil {
				return nil, "", NewBanquetError(err, "Failed to index remote directory", 500, b, "", cachePath)
			}
			b.Table = "tb0"
		} else {
			// Remote file - fetch and convert
			tempDir := filepath.Join(e.App.DataDir(), "temp")
			if err := os.MkdirAll(tempDir, 0755); err != nil {
				return nil, "", NewBanquetError(err, "Failed to create temp directory", 500, b, "", cachePath)
			}

			rawFilePath := filepath.Join(tempDir, cacheKey+filepath.Ext(b.DataSetPath))

			// Construct fetch path with query parameters if present
			fetchPath := b.DataSetPath
			if b.URL != nil && b.URL.RawQuery != "" {
				fetchPath += "?" + b.URL.RawQuery
			}

			if err := rcloneManager.FetchFile(vfs, fetchPath, rawFilePath); err != nil {
				return nil, "", NewBanquetError(err, fmt.Sprintf("Failed to fetch file: %s", b.DataSetPath), 500, b, "", cachePath)
			}

			// Convert to SQLite
			if err := ConvertToSQLite(rawFilePath, cachePath); err != nil {
				os.Remove(rawFilePath) // Cleanup on error
				return nil, "", NewBanquetError(err, "Failed to convert file to SQLite", 500, b, "", cachePath)
			}

			// Cleanup temp file
			if err := os.Remove(rawFilePath); err != nil {
				log.Printf("[BANQUET] Warning: failed to cleanup temp file: %v", err)
			}
		}

		if verbose {
			log.Printf("[BANQUET] Data processed successfully")
		}
	} else {
		if verbose {
			log.Printf("[BANQUET] Cache hit, serving from cache")
		}
	}

	return b, cachePath, nil
}

// EnsureLocalDataset processes local file requests and returns the cache path
func EnsureLocalDataset(e *core.RequestEvent, b *banquet.Banquet, verbose bool) (string, error) {
	if verbose {
		log.Printf("[LOCAL] Handling local dataset: %s", b.DataSetPath)
	}

	// 1. Resolve local file path
	baseDir := filepath.Join(e.App.DataDir(), "..", "pb_public") // Default

	// Try to find serve_folder setting
	if record, err := e.App.FindFirstRecordByData("app_settings", "key", "serve_folder"); err == nil && record != nil {
		if val := record.GetString("value"); val != "" {
			if strings.HasPrefix(val, "~/") || val == "~" {
				if homeDir, err := os.UserHomeDir(); err == nil {
					if val == "~" {
						val = homeDir
					} else {
						val = filepath.Join(homeDir, val[2:])
					}
				}
			}

			if filepath.IsAbs(val) {
				baseDir = val
			} else {
				baseDir = filepath.Join(e.App.DataDir(), "..", val)
			}
			if verbose {
				log.Printf("[LOCAL] Using configured serve_folder: %s", baseDir)
			}
		}
	}

	var localFilePath string
	if b.DataSetPath == "" || b.DataSetPath == "/" {
		localFilePath = baseDir
	} else if filepath.IsAbs(b.DataSetPath) {
		localFilePath = b.DataSetPath
	} else {
		localFilePath = filepath.Join(baseDir, b.DataSetPath)
	}

	localFilePath = filepath.Clean(localFilePath)

	if verbose {
		log.Printf("[LOCAL] Resolved file path: %s", localFilePath)
	}

	// 2. Check if file exists
	fileInfo, err := os.Stat(localFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", NewBanquetError(err, fmt.Sprintf("Local file not found: %s", b.DataSetPath), 404, b, "", "")
		}
		return "", NewBanquetError(err, "Error accessing local file", 500, b, "", "")
	}

	// 3. Determine Cache Path
	var cachePath string
	flatPath := strings.ReplaceAll(localFilePath, "/", "_")
	flatPath = strings.ReplaceAll(flatPath, "\\", "_")

	if fileInfo.IsDir() {
		localIndexPath := filepath.Join(localFilePath, "index.sqlite")
		if isWritable(localFilePath) {
			cachePath = localIndexPath
		} else {
			cachePath = filepath.Join(e.App.DataDir(), "cache", flatPath+".db")
		}
		b.Table = "tb0"
	} else {
		cachePath = filepath.Join(e.App.DataDir(), "cache", flatPath+".db")
	}

	if verbose {
		log.Printf("[LOCAL] Cache path: %s", cachePath)
	}

	// 4. Check Cache Validity
	ttl := 1440.0
	valid, err := ValidateCache(cachePath, ttl)
	if err != nil {
		log.Printf("[LOCAL] Cache validation error: %v", err)
		valid = false
	}

	// 5. Convert if Cache Miss
	if !valid {
		if verbose {
			log.Printf("[LOCAL] Cache miss or expired, converting local file...")
		}

		sourceInfo, _ := os.Stat(localFilePath)
		if sourceInfo != nil {
			cacheInfo, err := os.Stat(cachePath)
			if err == nil && cacheInfo.Size() > 0 && cacheInfo.ModTime().After(sourceInfo.ModTime()) {
				valid = true
				if verbose {
					log.Printf("[LOCAL] Cache is newer than source file, using cache")
				}
			}
		}

		if !valid {
			if err := ConvertToSQLite(localFilePath, cachePath); err != nil {
				return "", NewBanquetError(err, "Failed to convert local file/directory to SQLite", 500, b, "", cachePath)
			}
			if verbose {
				log.Printf("[LOCAL] File/Directory converted successfully")
			}
		}

		if fileInfo.IsDir() {
			b.Table = "tb0"
		}
	} else {
		if fileInfo.IsDir() {
			b.Table = "tb0"
		}
		if verbose {
			log.Printf("[LOCAL] Cache hit, serving from cache")
		}
	}

	return cachePath, nil
}

func HandleBanquet(e *core.RequestEvent, verbose bool) error {
	_, cachePath, err := EnsureBanquetDataset(e, e.Request.RequestURI, verbose)
	if err != nil {
		if be, ok := err.(*BanquetError); ok {
			return e.Error(be.Status, be.Message, be.Unwrap())
		}
		return err
	}

	// Serve SQLiter UI (keeps Banquet URL in browser)
	if verbose {
		log.Printf("[BANQUET] Dataset cached at: %s", cachePath)
	}

	// Previously served SQLiter UI. Now we just confirm caching.
	// Users should use the SQLiter-Dart client or API.
	return e.String(200, "Dataset synced and ready. Use SQLiter-Dart to access.")
}

// HandleBanquetDownload serves the raw SQLite file for delivery to clients.
func HandleBanquetDownload(e *core.RequestEvent) error {
	prefix := "/sqliter/file/"
	reqURI := e.Request.URL.Path
	if !strings.HasPrefix(reqURI, prefix) {
		return e.Error(400, "Invalid download path", nil)
	}

	banquetPath := strings.TrimPrefix(reqURI, prefix)

	// Delegate to Ensure logic
	_, cachePath, err := EnsureBanquetDataset(e, banquetPath, false)
	if err != nil {
		return e.Error(404, "File not found or failed to process: "+err.Error(), nil)
	}

	// Serve the file
	http.ServeFile(e.Response, e.Request, cachePath)
	return nil
}

// HandleBanquetSync returns JSON metadata about the dataset, useful for "Same Server" checks.
func HandleBanquetSync(e *core.RequestEvent) error {
	prefix := "/sqliter/sync/"
	reqURI := e.Request.URL.Path
	if !strings.HasPrefix(reqURI, prefix) {
		return e.Error(400, "Invalid sync path", nil)
	}

	banquetPath := strings.TrimPrefix(reqURI, prefix)

	// Delegate to Ensure logic
	_, cachePath, err := EnsureBanquetDataset(e, banquetPath, false)
	if err != nil {
		return e.Error(500, "Failed to sync: "+err.Error(), nil)
	}

	// Construct JSON response
	// Note: We need to know the download URL. It corresponds to the input path.
	// We might need to encode it properly.
	downloadURL := "/sqliter/file/" + banquetPath

	return e.JSON(200, map[string]string{
		"status":       "ready",
		"server_path":  cachePath,
		"download_url": downloadURL,
	})
}

// isWritable checks if a directory is writable by attempting to create a temp file

// QueryResult represents the JSON response for data rows
type QueryResult struct {
	Columns    []string                 `json:"columns"`
	Rows       []map[string]interface{} `json:"rows"`
	TotalCount int64                    `json:"totalCount"`
}

// HandleBanquetRows executes a query against the cached SQLite file and returns JSON rows.
// Replacement for sqliter.Server.
func HandleBanquetRows(e *core.RequestEvent) error {
	pathParam := e.Request.URL.Query().Get("path")
	if pathParam == "" {
		return e.Error(400, "Missing path parameter", nil)
	}

	startStr := e.Request.URL.Query().Get("start")
	endStr := e.Request.URL.Query().Get("end")

	limit := 100
	offset := 0

	if startStr != "" {
		fmt.Sscanf(startStr, "%d", &offset)
	}
	if endStr != "" {
		var end int
		fmt.Sscanf(endStr, "%d", &end)
		if end > offset {
			limit = end - offset
		}
	}

	b, cachePath, err := EnsureBanquetDataset(e, pathParam, false)
	if err != nil {
		return e.Error(404, "Dataset not found: "+err.Error(), nil)
	}

	db, err := sql.Open("sqlite", cachePath)
	if err != nil {
		return e.Error(500, "Failed to open database: "+err.Error(), nil)
	}
	defer db.Close()

	tableName := b.Table
	if tableName == "" {
		row := db.QueryRow("SELECT name FROM sqlite_schema WHERE type='table' AND name NOT LIKE 'sqlite_%' LIMIT 1")
		if err := row.Scan(&tableName); err != nil {
			return e.Error(404, "No tables found in database", nil)
		}
	}

	var totalCount int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM \"%s\"", strings.ReplaceAll(tableName, "\"", "\"\""))
	if err := db.QueryRow(countQuery).Scan(&totalCount); err != nil {
		return e.Error(500, "Count query failed: "+err.Error(), nil)
	}

	query := fmt.Sprintf("SELECT * FROM \"%s\" LIMIT %d OFFSET %d", strings.ReplaceAll(tableName, "\"", "\"\""), limit, offset)
	rows, err := db.Query(query)
	if err != nil {
		return e.Error(500, "Query failed: "+err.Error(), nil)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return e.Error(500, "Failed to get columns: "+err.Error(), nil)
	}

	var resultRows []map[string]interface{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		rowMap := make(map[string]interface{})
		for i, x := range values {
			colName := columns[i]
			if b, ok := x.([]byte); ok {
				rowMap[colName] = string(b)
			} else {
				rowMap[colName] = x
			}
		}
		resultRows = append(resultRows, rowMap)
	}

	if resultRows == nil {
		resultRows = []map[string]interface{}{}
	}

	res := QueryResult{
		Columns:    columns,
		Rows:       resultRows,
		TotalCount: totalCount,
	}

	return e.JSON(200, res)
}

// isWritable checks if a directory is writable by attempting to create a temp file
func isWritable(path string) bool {
	testFile := filepath.Join(path, ".perm_test_"+fmt.Sprintf("%d", time.Now().UnixNano()))
	f, err := os.Create(testFile)
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(testFile)
	return true
}
