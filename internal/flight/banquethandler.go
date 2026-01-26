package flight

import (
	"html/template"
	"log"
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
	// We want the raw RequestURI to capture everything.
	reqURI := e.Request.RequestURI
	reqURI = strings.TrimPrefix(reqURI, "/")

	// Parse with Banquet
	b, err := banquet.ParseNested(reqURI)
	if err != nil {
		log.Printf("Invalid banquet URL: %s", reqURI)
		return err
	}
	banquet.FmtPrintln(b)

	// 2. Check Cache
	cacheKey := GenCacheKey(b)
	if verbose {
		log.Printf("Cache key: %s", cacheKey)
	}

	// 3. If cache miss, fetch using rclone.
	// 4. If cache miss, generate using mksqlite.
	// 5. Write to cache.
	// 6. Serve via tw

	return nil
}
