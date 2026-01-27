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
		return err
	}

	if verbose {
		banquet.FmtPrintln(b)
	}

	// 2. Lookup Remote Configuration
	remoteRecord, err := LookupRemote(e.App, b.Hostname())
	if err != nil {
		return fmt.Errorf("remote not found: %w", err)
	}

	// 3. Initialize VFS
	rcloneManager := GetRcloneManager()
	if rcloneManager == nil {
		return fmt.Errorf("rclone manager not initialized")
	}

	vfs, err := rcloneManager.GetVFS(remoteRecord)
	if err != nil {
		return fmt.Errorf("failed to init VFS: %w", err)
	}

	// 4. Generate Cache Key
	remoteConfigHash := rcloneManager.GetConfigHash(remoteRecord)
	cacheKey := GenCacheKey(b, remoteConfigHash)
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

		// 6a. Fetch raw file via rclone VFS
		tempDir := filepath.Join(e.App.DataDir(), "temp")
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			return fmt.Errorf("failed to create temp directory: %w", err)
		}

		rawFilePath := filepath.Join(tempDir, cacheKey+filepath.Ext(b.DataSetPath))
		if err := rcloneManager.FetchFile(vfs, b.DataSetPath, rawFilePath); err != nil {
			return fmt.Errorf("failed to fetch file: %w", err)
		}

		// 6b. Convert to SQLite using mksqlite
		if err := ConvertToSQLite(rawFilePath, cachePath); err != nil {
			os.Remove(rawFilePath) // Cleanup on error
			return fmt.Errorf("failed to convert to SQLite: %w", err)
		}

		// 6c. Cleanup temp file
		if err := os.Remove(rawFilePath); err != nil {
			log.Printf("[BANQUET] Warning: failed to cleanup temp file: %v", err)
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
