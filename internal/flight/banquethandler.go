package flight

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/darianmavgo/banquet"
	_ "github.com/darianmavgo/mksqlite/converters/all"
	"github.com/pocketbase/pocketbase/core"
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

func HandleBanquet(e *core.RequestEvent, verbose bool) error {
	// 1. Parse Banquet URL
	reqURI := e.Request.RequestURI
	reqURI = strings.TrimPrefix(reqURI, "/")

	b, err := banquet.ParseNested(reqURI)
	if err != nil {
		log.Printf("[BANQUET] Invalid banquet URL: %s", reqURI)
		return NewBanquetError(err, "Invalid banquet URL format", 400, nil, "", "")
	}

	if verbose {
		banquet.FmtPrintln(b)
	}
	// 2. Handle Local Dataset
	if b.Scheme == "" && b.Hostname() == "" {
		return HandleLocalDataset(e, b, verbose)
	}
	// 2. Lookup Remote Configuration
	remoteRecord, err := LookupRemote(e.App, b.Hostname())
	if err != nil {
		return NewBanquetError(err, fmt.Sprintf("Remote '%s' not found", b.Hostname()), 404, b, "", "")
	}

	// 3. Initialize VFS
	rcloneManager := GetRcloneManager()
	if rcloneManager == nil {
		return NewBanquetError(nil, "Rclone manager not initialized", 500, b, "", "")
	}

	vfs, err := rcloneManager.GetVFS(remoteRecord)
	if err != nil {
		return NewBanquetError(err, "Failed to initialize VFS", 500, b, "", "")
	}

	// 4. Generate Cache Key
	cacheKey := GenCacheKey(b)
	cachePath := GetCachePath(e.App.DataDir(), cacheKey)

	if verbose {
		log.Printf("[BANQUET] Cache key: %s", cacheKey)
		log.Printf("[BANQUET] Cache path: %s", cachePath)
	}

	// 5. Check Cache Validity
	// Get TTL from data_pipelines or use default (24 hours = 1440 minutes)
	ttl := 1440.0
	valid, err := ValidateCache(cachePath, ttl)
	if err != nil {
		log.Printf("[BANQUET] Cache validation error: %v", err)
		valid = false
	}

	// 6. Fetch and Convert if Cache Miss
	if !valid {
		if verbose {
			log.Printf("[BANQUET] Cache miss or expired, fetching and converting...")
		}

		// Check if it's a directory or a file
		node, err := rcloneManager.Stat(vfs, b.DataSetPath)
		if err != nil {
			return NewBanquetError(err, fmt.Sprintf("Failed to access remote path: %s", b.DataSetPath), 404, b, "", "")
		}

		if node.IsDir() {
			// Remote directory - index it
			if err := rcloneManager.IndexDirectory(vfs, b.DataSetPath, cachePath); err != nil {
				return NewBanquetError(err, "Failed to index remote directory", 500, b, "", cachePath)
			}
			// When indexing a directory, the resulting table name in the cache is always 'tb0'
			b.Table = "tb0"
		} else {
			// Remote file - fetch and convert
			tempDir := filepath.Join(e.App.DataDir(), "temp")
			if err := os.MkdirAll(tempDir, 0755); err != nil {
				return NewBanquetError(err, "Failed to create temp directory", 500, b, "", cachePath)
			}

			rawFilePath := filepath.Join(tempDir, cacheKey+filepath.Ext(b.DataSetPath))
			if err := rcloneManager.FetchFile(vfs, b.DataSetPath, rawFilePath); err != nil {
				return NewBanquetError(err, fmt.Sprintf("Failed to fetch file: %s", b.DataSetPath), 500, b, "", cachePath)
			}

			// 6b. Convert to SQLite using mksqlite
			if err := ConvertToSQLite(rawFilePath, cachePath); err != nil {
				os.Remove(rawFilePath) // Cleanup on error
				return NewBanquetError(err, "Failed to convert file to SQLite", 500, b, "", cachePath)
			}

			// 6c. Cleanup temp file
			if err := os.Remove(rawFilePath); err != nil {
				log.Printf("[BANQUET] Warning: failed to cleanup temp file: %v", err)
			}
		}

		if verbose {
			log.Printf("[BANQUET] Data processed successfully")
		}

		if verbose {
			log.Printf("[BANQUET] File fetched and converted successfully")
		}
	} else {
		if verbose {
			log.Printf("[BANQUET] Cache hit, serving from cache")
		}
	}

	// 7. Serve SQLiter UI (keeps Banquet URL in browser)
	// SQLiter handles: ColumnSetPath → Query
	// The React UI will make API calls to /sqliter/ internally

	// Store the database path and banquet info for SQLiter to access
	// SQLiter's React app will parse the current URL and make API calls

	if verbose {
		log.Printf("[BANQUET] Serving SQLiter UI for: %s", cachePath)
		log.Printf("[BANQUET] Banquet URL preserved in browser")
	}

	// Get the sqliter server from the app context
	// We need to pass it through the request context or use a global
	// For now, we'll use the global pattern
	sqliterServer := GetSQLiterServer()
	if sqliterServer == nil {
		return NewBanquetError(nil, "SQLiter server not initialized", 500, b, "", cachePath)
	}

	// Serve SQLiter's React UI directly (no redirect)
	// This keeps the Banquet URL in the browser
	sqliterServer.ServeHTTP(e.Response, e.Request)
	return nil
}

// HandleLocalDataset handles local file requests without rclone
// Still uses caching and serving infrastructure
func HandleLocalDataset(e *core.RequestEvent, b *banquet.Banquet, verbose bool) error {
	if verbose {
		log.Printf("[LOCAL] Handling local dataset: %s", b.DataSetPath)
	}

	// 1. Resolve local file path
	// Look up serve_folder in app_settings
	baseDir := filepath.Join(e.App.DataDir(), "..", "pb_public") // Default

	// Try to find serve_folder setting - dynamic lookup prevents restart requirement
	if record, err := e.App.FindFirstRecordByData("app_settings", "key", "serve_folder"); err == nil && record != nil {
		if val := record.GetString("value"); val != "" {
			// Expand home directory ~
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
				// Treat relative paths as relative to the application root (parent of pb_data)
				baseDir = filepath.Join(e.App.DataDir(), "..", val)
			}
			if verbose {
				log.Printf("[LOCAL] Using configured serve_folder: %s", baseDir)
			}
		}
	}

	// DataSetPath should be relative to baseDir or an absolute path
	var localFilePath string

	// Handle empty path (root request)
	if b.DataSetPath == "" || b.DataSetPath == "/" {
		localFilePath = baseDir
	} else if filepath.IsAbs(b.DataSetPath) {
		localFilePath = b.DataSetPath
	} else {
		// Relative to base directory
		localFilePath = filepath.Join(baseDir, b.DataSetPath)
	}

	// Clean the path
	localFilePath = filepath.Clean(localFilePath)

	if verbose {
		log.Printf("[LOCAL] Resolved file path: %s", localFilePath)
	}

	// 2. Check if file exists
	fileInfo, err := os.Stat(localFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewBanquetError(err, fmt.Sprintf("Local file not found: %s", b.DataSetPath), 404, b, "", "")
		}
		return NewBanquetError(err, "Error accessing local file", 500, b, "", "")
	}

	// 3. Determine Cache Path
	var cachePath string
	flatPath := strings.ReplaceAll(localFilePath, "/", "_")
	flatPath = strings.ReplaceAll(flatPath, "\\", "_")

	if fileInfo.IsDir() {
		// Try index.sqlite in the folder
		localIndexPath := filepath.Join(localFilePath, "index.sqlite")
		if isWritable(localFilePath) {
			cachePath = localIndexPath
		} else {
			// Fallback to global cache with flattened name
			cachePath = filepath.Join(e.App.DataDir(), "cache", flatPath+".db")
		}
		// Ensure table assumption for directories
		b.Table = "tb0"
	} else {
		// File: use global cache with flattened path
		cachePath = filepath.Join(e.App.DataDir(), "cache", flatPath+".db")
	}

	if verbose {
		log.Printf("[LOCAL] Cache path: %s", cachePath)
	}

	// 4. Check Cache Validity
	ttl := 1440.0 // 24 hours default
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

		// Check if source file is newer than cache
		sourceInfo, _ := os.Stat(localFilePath)
		if sourceInfo != nil {
			cacheInfo, err := os.Stat(cachePath)
			if err == nil && cacheInfo.Size() > 0 && cacheInfo.ModTime().After(sourceInfo.ModTime()) {
				// Cache is newer than source and not empty, use it
				valid = true
				if verbose {
					log.Printf("[LOCAL] Cache is newer than source file, using cache")
				}
			}
		}

		if !valid {
			// Convert to SQLite (File or Directory)
			if err := ConvertToSQLite(localFilePath, cachePath); err != nil {
				return NewBanquetError(err, "Failed to convert local file/directory to SQLite", 500, b, "", cachePath)
			}

			if verbose {
				log.Printf("[LOCAL] File/Directory converted successfully")
			}
		}

		// If it's a directory, we must use 'tb0' for the listing table
		if fileInfo.IsDir() {
			b.Table = "tb0"
		}
	} else {
		// Cache hit, but ensure table is correct if it's a directory
		if fileInfo.IsDir() {
			b.Table = "tb0"
		}
		if verbose {
			log.Printf("[LOCAL] Cache hit, serving from cache")
		}
	}

	// 6. Serve SQLiter UI (keeps Banquet URL in browser)
	if verbose {
		log.Printf("[LOCAL] Serving SQLiter UI for: %s", cachePath)
		log.Printf("[LOCAL] Banquet URL preserved in browser")
	}

	sqliterServer := GetSQLiterServer()
	if sqliterServer == nil {
		return NewBanquetError(nil, "SQLiter server not initialized", 500, b, "", cachePath)
	}

	// Serve SQLiter's React UI directly (no redirect)
	sqliterServer.ServeHTTP(e.Response, e.Request)
	return nil
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
