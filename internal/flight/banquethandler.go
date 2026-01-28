package flight

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/darianmavgo/banquet"
	_ "github.com/darianmavgo/mksqlite/converters/all"
	"github.com/darianmavgo/sqliter/sqliter"
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

func HandleBanquet(e *core.RequestEvent, tw *sqliter.TableWriter, tpl *template.Template, verbose bool) error {
	// 1. Parse Banquet URL
	reqURI := e.Request.RequestURI
	reqURI = strings.TrimPrefix(reqURI, "/")

	b, err := banquet.ParseNested(reqURI)
	if err != nil {
		log.Printf("[BANQUET] Invalid banquet URL: %s", reqURI)
		return NewBanquetError(err, "Invalid banquet URL format", 400, nil, "")
	}

	if verbose {
		banquet.FmtPrintln(b)
	}
	// 2. Handle Local Dataset
	if b.Scheme == "" && b.Hostname() == "" {
		return HandleLocalDataset(e, b, tw, tpl, verbose)
	}
	// 2. Lookup Remote Configuration
	remoteRecord, err := LookupRemote(e.App, b.Hostname())
	if err != nil {
		return NewBanquetError(err, fmt.Sprintf("Remote '%s' not found", b.Hostname()), 404, b, "")
	}

	// 3. Initialize VFS
	rcloneManager := GetRcloneManager()
	if rcloneManager == nil {
		return NewBanquetError(nil, "Rclone manager not initialized", 500, b, "")
	}

	vfs, err := rcloneManager.GetVFS(remoteRecord)
	if err != nil {
		return NewBanquetError(err, "Failed to initialize VFS", 500, b, "")
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
			return NewBanquetError(err, fmt.Sprintf("Failed to access remote path: %s", b.DataSetPath), 404, b, "")
		}

		if node.IsDir() {
			// Remote directory - index it
			if err := rcloneManager.IndexDirectory(vfs, b.DataSetPath, cachePath); err != nil {
				return NewBanquetError(err, "Failed to index remote directory", 500, b, "")
			}
			// When indexing a directory, the resulting table name in the cache is always 'tb0'
			b.Table = "tb0"
		} else {
			// Remote file - fetch and convert
			tempDir := filepath.Join(e.App.DataDir(), "temp")
			if err := os.MkdirAll(tempDir, 0755); err != nil {
				return NewBanquetError(err, "Failed to create temp directory", 500, b, "")
			}

			rawFilePath := filepath.Join(tempDir, cacheKey+filepath.Ext(b.DataSetPath))
			if err := rcloneManager.FetchFile(vfs, b.DataSetPath, rawFilePath); err != nil {
				return NewBanquetError(err, fmt.Sprintf("Failed to fetch file: %s", b.DataSetPath), 500, b, "")
			}

			// 6b. Convert to SQLite using mksqlite
			if err := ConvertToSQLite(rawFilePath, cachePath); err != nil {
				os.Remove(rawFilePath) // Cleanup on error
				return NewBanquetError(err, "Failed to convert file to SQLite", 500, b, "")
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

	// 7. Serve from Cache
	return ServeFromCache(cachePath, b, tw, tpl, e)
}

// HandleLocalDataset handles local file requests without rclone
// Still uses caching and serving infrastructure
func HandleLocalDataset(e *core.RequestEvent, b *banquet.Banquet, tw *sqliter.TableWriter, tpl *template.Template, verbose bool) error {
	if verbose {
		log.Printf("[LOCAL] Handling local dataset: %s", b.DataSetPath)
	}

	// 1. Resolve local file path
	// DataSetPath should be relative to pb_public or an absolute path
	var localFilePath string

	// Handle empty path (root request)
	if b.DataSetPath == "" || b.DataSetPath == "/" {
		localFilePath = filepath.Join(e.App.DataDir(), "..", "pb_public")
	} else if filepath.IsAbs(b.DataSetPath) {
		localFilePath = b.DataSetPath
	} else {
		// Relative to pb_public directory
		localFilePath = filepath.Join(e.App.DataDir(), "..", "pb_public", b.DataSetPath)
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
			return NewBanquetError(err, fmt.Sprintf("Local file not found: %s", b.DataSetPath), 404, b, "")
		}
		return NewBanquetError(err, "Error accessing local file", 500, b, "")
	}

	// 3. Generate Cache Key
	cacheKey := GenCacheKey(b)
	cachePath := GetCachePath(e.App.DataDir(), cacheKey)

	if verbose {
		log.Printf("[LOCAL] Cache key: %s", cacheKey)
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
				return NewBanquetError(err, "Failed to convert local file/directory to SQLite", 500, b, "")
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

	// 6. Serve from Cache
	// For directories, the table name in the cached DB is usually 'tb0' or 'data'
	// mksqlite/converters/filesystem uses 'tb0' by default if not specified.
	return ServeFromCache(cachePath, b, tw, tpl, e)
}
