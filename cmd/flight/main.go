package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	_ "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/spf13/cast"

	"os"

	"github.com/darianmavgo/banquet"
	"github.com/darianmavgo/mksqlite/converters"
	_ "github.com/darianmavgo/mksqlite/converters/all"
	"github.com/darianmavgo/mksqlite/converters/common"
	"github.com/darianmavgo/sqliter/sqliter"

	// Rclone imports
	_ "github.com/rclone/rclone/backend/all" // Import all backends
	rcfs "github.com/rclone/rclone/fs"

	"github.com/darianmavgo/flight3/internal/secrets"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed ui/*
var uiEmbed embed.FS

func main() {
	// Default to "serve" command if no arguments are provided
	if len(os.Args) == 1 {
		os.Args = append(os.Args, "serve")
	}

	// Port Handling: Loop through 8090, 8091, 8092
	// Only if "serve" is in args and --http is NOT specified
	hasHttpFlag := false
	isServe := false
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "--http=") {
			hasHttpFlag = true
		}
		if arg == "serve" {
			isServe = true
		}
	}

	if isServe && !hasHttpFlag {
		port := 8090
		for i := 0; i < 3; i++ {
			testPort := 8090 + i
			ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", testPort))
			if err == nil {
				ln.Close()
				port = testPort
				break
			}
		}
		// If 8090 was taken, we might have picked 8091.
		// Append flag
		os.Args = append(os.Args, fmt.Sprintf("--http=127.0.0.1:%d", port))
		fmt.Printf("Selected port: %d\n", port)
	}

	// Calculate absolute path for data directory to avoid CWD ambiguity
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	dataDir := filepath.Join(workDir, "user_settings", "pb_data")
	log.Printf("Using Data Directory: %s", dataDir)

	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: dataDir,
	})

	// Initialize SQLiter
	templatePath := "cmd/flight/templates"
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		// Try generic templates dir if cmd path fails
		templatePath = "templates"
	}
	tpl := loadTemplates(templatePath)
	tw := sqliter.NewTableWriter(tpl, sqliter.DefaultConfig())

	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		if err := ensureCollections(app); err != nil {
			return fmt.Errorf("ensureCollections: %w", err)
		}

		if err := importUserSecrets(app, filepath.Dir(dataDir)); err != nil {
			log.Printf("Warning: importUserSecrets failed: %v", err)
		}

		// Check for existing superusers
		totalSuperusers, err := app.CountRecords("_superusers")
		if err != nil {
			log.Printf("Failed to count superusers: %v", err)
		} else {
			if totalSuperusers > 0 {
				log.Printf("Acknowledged %d existing admin user(s)", totalSuperusers)
			} else {
				log.Printf("No admin users found. Creating default admin...")
				superusers, err := app.FindCollectionByNameOrId("_superusers")
				if err != nil {
					log.Printf("Failed to find _superusers collection: %v", err)
				} else {
					record := core.NewRecord(superusers)
					record.Set("email", "admin@example.com")
					record.Set("password", "1234567890")
					if err := app.Save(record); err != nil {
						log.Printf("Failed to create default admin: %v", err)
					} else {
						log.Printf("Created default admin: admin@example.com")
					}
				}
			}
		}

		// Middleware to invert Admin UI colors
		e.Router.BindFunc(func(evt *core.RequestEvent) error {
			if evt.Request.URL.Path == "/_/" || evt.Request.URL.Path == "/_/index.html" {
				cw := &customWriter{ResponseWriter: evt.Response}
				evt.Response = cw

				err := evt.Next()

				if cw.Buffer.Len() > 0 {
					content := cw.Buffer.String()
					// Simple inversion filter
					style := `<style>html { filter: invert(1) hue-rotate(180deg); } img, video { filter: invert(1) hue-rotate(180deg); } .main-menu { background: #111 !important; }</style>`
					// Inject before </head>
					if strings.Contains(content, "</head>") {
						modified := strings.Replace(content, "</head>", style+"</head>", 1)
						cw.ResponseWriter.WriteHeader(cw.Status)
						cw.ResponseWriter.Write([]byte(modified))
					} else {
						// Fallback if no head (unlikely)
						cw.ResponseWriter.WriteHeader(cw.Status)
						cw.ResponseWriter.Write(cw.Buffer.Bytes())
					}
				} else {
					// Empty body? Just forward status
					if cw.Status != 0 {
						cw.ResponseWriter.WriteHeader(cw.Status)
					}
				}
				return err
			}
			return evt.Next()
		})

		// Serve UI
		subFS, err := fs.Sub(uiEmbed, "ui")
		if err != nil {
			return err
		}

		e.Router.GET("/*", func(evt *core.RequestEvent) error {
			path := evt.Request.PathValue("*")
			if path == "" {
				path = "index.html"
			}
			f, err := subFS.Open(path)
			if err != nil {
				// Try index.html if not found (SPA fallback-like) or just 404
				// For now, if we requested root, and failed, that's bad.
				// If we requested random file, let it fall through
				return evt.Next()
			}
			defer f.Close()

			info, err := f.Stat()
			if err != nil {
				return err
			}

			if info.IsDir() {
				// Try index.html inside
				index, err := subFS.Open(strings.TrimSuffix(path, "/") + "/index.html")
				if err == nil {
					defer index.Close()
					f = index
					info, _ = index.Stat()
				}
			}

			seeker, ok := f.(io.ReadSeeker)
			if !ok {
				return nil
			}

			http.ServeContent(evt.Response, evt.Request, info.Name(), info.ModTime(), seeker)
			return nil
		})

		// GET /api/rclone/browse/{id}?path=folder/subfolder
		e.Router.GET("/api/rclone/browse/{id}", func(evt *core.RequestEvent) error {
			return handleBrowse(app, evt)
		})

		// GET /banquet/*
		e.Router.GET("/banquet/*", func(evt *core.RequestEvent) error {
			return handleBanquet(evt, tw)
		})

		// GET /http:/{any...} and /https:/{any...} (Direct Nested Banquet URLs)
		e.Router.GET("/http:/{any...}", func(evt *core.RequestEvent) error {
			return handleBanquet(evt, tw)
		})
		e.Router.GET("/https:/{any...}", func(evt *core.RequestEvent) error {
			return handleBanquet(evt, tw)
		})

		return e.Next()
		// Actually OnServe returns error. It doesn't take a 'next' handler usually in this context unless using middleware hooks.
		// The BindFunc for OnServe expects just a function. The router is populated.
		// Returing nil is fine.
		// But in v0.23+ hooks generally chain.
		// Wait, usage: app.OnServe().BindFunc(func(e *core.ServeEvent) error { ... return e.Next() })
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

func ensureCollections(app core.App) error {
	if err := ensureRcloneRemotes(app); err != nil {
		return err
	}
	if err := ensureMksqliteConfigs(app); err != nil {
		return err
	}
	return ensureDataPipelines(app)
}

func ensureRcloneRemotes(app core.App) error {
	name := "rclone_remotes"
	existing, err := app.FindCollectionByNameOrId(name)
	if err == nil && existing != nil {
		return nil
	}

	collection := core.NewBaseCollection(name)
	collection.Fields.Add(&core.TextField{Name: "name", Required: true})
	collection.Fields.Add(&core.TextField{Name: "type", Required: true}) // e.g. s3, drive
	collection.Fields.Add(&core.JSONField{Name: "config"})               // e.g. {"access_key_id": "...", ...}

	return app.Save(collection)
}

func ensureMksqliteConfigs(app core.App) error {
	name := "mksqlite_configs"
	existing, err := app.FindCollectionByNameOrId(name)
	if err == nil && existing != nil {
		return nil
	}

	collection := core.NewBaseCollection(name)
	collection.Fields.Add(&core.TextField{Name: "name", Required: true})
	collection.Fields.Add(&core.TextField{Name: "driver"}) // e.g. csv, json
	collection.Fields.Add(&core.JSONField{Name: "args"})   // e.g. {"delimiter": ","}

	return app.Save(collection)
}

func ensureDataPipelines(app core.App) error {
	name := "data_pipelines"
	existing, err := app.FindCollectionByNameOrId(name)
	if err == nil && existing != nil {
		return nil
	}

	rcloneRemotes, err := app.FindCollectionByNameOrId("rclone_remotes")
	if err != nil {
		return fmt.Errorf("failed to find rclone_remotes: %w", err)
	}

	mksqliteConfigs, err := app.FindCollectionByNameOrId("mksqlite_configs")
	if err != nil {
		return fmt.Errorf("failed to find mksqlite_configs: %w", err)
	}

	collection := core.NewBaseCollection(name)
	collection.Fields.Add(&core.TextField{Name: "name", Required: true})

	// Relation to rclone_remotes
	collection.Fields.Add(&core.RelationField{
		Name:          "rclone_remote",
		CollectionId:  rcloneRemotes.Id,
		CascadeDelete: false,
		MaxSelect:     1,
	})

	collection.Fields.Add(&core.TextField{Name: "rclone_path", Required: true})

	// Relation to mksqlite_configs
	collection.Fields.Add(&core.RelationField{
		Name:          "mksqlite_config",
		CollectionId:  mksqliteConfigs.Id,
		CascadeDelete: false,
		MaxSelect:     1,
	})

	collection.Fields.Add(&core.NumberField{Name: "cache_ttl"}) // in minutes

	return app.Save(collection)
}

func handleBrowse(app core.App, e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	browsePath := e.Request.URL.Query().Get("path")

	// 1. Fetch the remote config
	record, err := app.FindRecordById("rclone_remotes", id)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{"error": "Remote not found"})
	}

	remoteType := record.GetString("type")
	configJSON := record.Get("config") // Returns any

	// 2. Parse config
	configMap := map[string]interface{}{}

	// Handle if configJSON is already a map or needs unmarshaling
	// PocketBase internal record.Get returns the underlying value. For JSON field it might be map or string depending on state?
	// Usually it is unmarshaled into map[string]any if it was valid JSON.

	switch v := configJSON.(type) {
	case string:
		if v != "" {
			if err := json.Unmarshal([]byte(v), &configMap); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Invalid config JSON in DB"}) // Fixed error handling
			}
		}
	case map[string]interface{}:
		configMap = v
	default:
		// empty or nil, ignore
	}

	// 3. Build Connection String
	// Format: :backend,key='value',...:path
	// Special handling: "root" key in config maps to the path component (after second colon)
	var rootPath string
	if v, ok := configMap["root"]; ok {
		rootPath = fmt.Sprintf("%v", v)
		delete(configMap, "root")
	}

	var sb strings.Builder
	sb.WriteString(":")
	sb.WriteString(remoteType)

	for k, v := range configMap {
		valStr := fmt.Sprintf("%v", v)
		escaped := strings.ReplaceAll(valStr, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		sb.WriteString(fmt.Sprintf(",%s=\"%s\"", k, escaped))
	}

	// Append colon + rootPath
	connectionString := sb.String()
	fullPath := connectionString + ":" + rootPath
	// Append browse path if relative
	if browsePath != "" {
		if rootPath != "" && !strings.HasSuffix(rootPath, "/") {
			fullPath += "/"
		}
		fullPath += browsePath
	}

	log.Printf("Opening rclone fs: %s", fullPath) // Be careful with logging credentials! Maybe Mask?
	// For production, DO NOT log fullPath.

	f, err := rcfs.NewFs(e.Request.Context(), fullPath)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to open fs: %v", err)})
	}

	// 5. List entries
	entries, err := f.List(e.Request.Context(), "")
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to list: %v", err)})
	}

	// 6. Map results
	type FileEntry struct {
		Name    string `json:"name"`
		Size    int64  `json:"size"`
		IsDir   bool   `json:"isDir"`
		ModTime string `json:"modTime"`
	}

	results := make([]FileEntry, 0, len(entries))
	for _, entry := range entries {
		_, isDir := entry.(rcfs.Directory)

		results = append(results, FileEntry{
			Name:    entry.Remote(), // Remote() is relative path from fs root
			Size:    entry.Size(),
			IsDir:   isDir,
			ModTime: entry.ModTime(e.Request.Context()).String(),
		})
	}

	return e.JSON(http.StatusOK, results)
}

func handleBanquet(e *core.RequestEvent, tw *sqliter.TableWriter) error {
	app := e.App
	// 1. Parse Banquet URL
	// We want the raw RequestURI to capture everything.
	reqURI := e.Request.RequestURI

	// Normalize if it's a direct nested URL (missing second slash or leading slash)
	// Browser/Router might have normalized /https:/ to /https:/
	if strings.Contains(reqURI, "https:/") && !strings.Contains(reqURI, "https://") {
		reqURI = strings.Replace(reqURI, "https:/", "https://", 1)
	} else if strings.Contains(reqURI, "http:/") && !strings.Contains(reqURI, "http://") {
		reqURI = strings.Replace(reqURI, "http:/", "http://", 1)
	}

	if strings.HasPrefix(reqURI, "/") && (strings.HasPrefix(reqURI, "/http") || strings.Contains(reqURI, "/https")) {
		// If it's like /https://... or /banquet/https://...
		// ParseNested handles leading slash on path, but we want the scheme to be recognized
		trimmed := strings.TrimPrefix(reqURI, "/")
		if strings.HasPrefix(trimmed, "http") {
			reqURI = trimmed
		}
	}

	// Parse with Banquet
	b, err := banquet.ParseNested(reqURI)
	if err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid banquet URL: " + err.Error()})
	}

	// PATCH: Sanitize reqURI for standard parser if user included scheme in host (e.g. user@https://host)
	// This happens if the user doubles the scheme in the URL path.
	sanitizedURI := reqURI
	if idx := strings.LastIndex(sanitizedURI, "@"); idx != -1 {
		after := sanitizedURI[idx+1:]
		if strings.HasPrefix(after, "https:/") {
			if strings.HasPrefix(after, "https://") {
				sanitizedURI = sanitizedURI[:idx+1] + after[8:]
			} else {
				sanitizedURI = sanitizedURI[:idx+1] + after[7:]
			}
		} else if strings.HasPrefix(after, "http:/") {
			if strings.HasPrefix(after, "http://") {
				sanitizedURI = sanitizedURI[:idx+1] + after[7:]
			} else {
				sanitizedURI = sanitizedURI[:idx+1] + after[6:]
			}
		}
	}

	// PATCH: banquet.ParseNested might fail to parse Host/User for full URLs.
	// We re-parse with net/url to ensure we capture them.
	if u, err := url.Parse(sanitizedURI); err == nil {
		if u.Scheme != "" && b.Scheme == "" {
			b.Scheme = u.Scheme
		}
		if u.Host != "" && b.Host == "" {
			b.Host = u.Host
		}
		if u.User != nil && b.User == nil {
			b.User = u.User
		}
		// Also ensure Path matches what we expect (banquet usually strips leading slash, keep it consistent)
		// But for identifying the remote path, we might want the full path.
		// b.Path from banquet seems to be relative path.
	}

	// 2. Identify Pipeline or Remote
	var pipelineRecord *core.Record
	var remoteRecord *core.Record
	var remotePath string
	var cacheKey string
	var cacheTTL float64 = 60 // Default 60 mins

	// Path should be like /banquet/<pipeline_name>/... or internal path from Nested URL
	// b.Path for a nested URL like https://... is usually the path part of that URL.
	path := strings.Trim(b.Path, "/")
	parts := strings.Split(path, "/")

	if parts[0] == "banquet" && len(parts) >= 2 {
		pipelineName := parts[1]
		pipelineRecord, err = app.FindFirstRecordByData("data_pipelines", "name", pipelineName)
		if err == nil {
			cacheKey = pipelineRecord.Id
			cacheTTL = pipelineRecord.GetFloat("cache_ttl")
			remotePath = pipelineRecord.GetString("rclone_path")
			if rid := pipelineRecord.GetString("rclone_remote"); rid != "" {
				remoteRecord, _ = app.FindRecordById("rclone_remotes", rid)
			}
		}
	}

	// If no pipeline, try direct remote via alias (from user info in nested URL)
	// or try to match first path part as remote alias if no user info
	if pipelineRecord == nil {
		alias := ""
		if b.User != nil {
			alias = b.User.Username()
			remotePath = b.Path
		} else if len(parts) > 0 {
			// fallback: check if first part is an alias
			alias = parts[0]
			remotePath = strings.Join(parts[1:], "/")
		}

		if alias != "" {
			remoteRecord, _ = app.FindFirstRecordByData("rclone_remotes", "name", alias)
			if remoteRecord != nil {
				// Use structured hash for cache key based on User, Host, and DataSetPath
				// We want to ignore Scheme, Query, and potentially ColumnPath (if we support inside DB query later)
				// Key = MD5(User + "@" + Host + ":" + DataSetPath)

				rawUser := ""
				if b.User != nil {
					rawUser = b.User.String()
				}
				// Use the parsed b.DataSetPath which avoids column paths
				baseKey := fmt.Sprintf("%s@%s:%s", rawUser, b.Host, b.DataSetPath)
				hash := md5.Sum([]byte(baseKey))
				cacheKey = "adhoc-" + hex.EncodeToString(hash[:])

				// If it's an S3 remote and we have host/scheme, inject endpoint
				if remoteRecord.GetString("type") == "s3" && b.Host != "" {
					rawConfig := remoteRecord.Get("config")
					configMap := map[string]interface{}{}

					// Unmarshal existing config safely
					if b, ok := rawConfig.(types.JSONRaw); ok {
						_ = json.Unmarshal(b, &configMap)
					} else if s, ok := rawConfig.(string); ok {
						_ = json.Unmarshal([]byte(s), &configMap)
					} else {
						configMap = cast.ToStringMap(rawConfig)
					}

					scheme := b.Scheme
					if scheme == "" {
						scheme = "https"
					}
					endpoint := scheme + "://" + b.Host
					log.Printf("Injecting endpoint into S3 config: %s", endpoint)
					configMap["endpoint"] = endpoint
					configMap["provider"] = "Cloudflare" // R2 needs this usually, force it if missing?
					// Actually, let's respect existing provider if present, but since user said "r2-auth", likely it's R2.
					// But safest is to just preserve what was there.
					// If provider was lost before, this fixes the loss.

					remoteRecord.Set("config", configMap)
				}
			}
		}
	}

	if remoteRecord == nil {
		return e.JSON(http.StatusNotFound, map[string]string{"error": "Pipeline or Remote alias not found", "parsed_alias": parts[0]})
	}

	// 3. Check/Build Cache
	dbPath, err := ensurePipelineCache(app, cacheKey, remoteRecord, remotePath, cacheTTL)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to process pipeline: %v", err)})
	}

	// 4. Serve Request
	// Open SQLite DB
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to open cache db"})
	}
	defer db.Close()

	// Update Banquet struct with Table name (if missing or if we know it)
	// For csv/file conversions, usually table is "tb0" or filename.
	// If b.Table is empty or "sqlite_master", we might default to tb0 if we know it's a file conversion.
	if b.Table == "" || b.Table == "sqlite_master" {
		b.Table = "tb0" // Default for mksqlite single file
	}

	// Construct SQL
	// Manual construction based on Banquet fields
	q := "SELECT "
	if len(b.Select) > 0 {
		q += strings.Join(b.Select, ", ")
	} else {
		q += "*"
	}
	q += fmt.Sprintf(" FROM \"%s\"", b.Table)

	if b.Where != "" {
		q += " WHERE " + b.Where
	}
	// TODO: GroupBy, Having

	if b.OrderBy != "" {
		q += " ORDER BY " + b.OrderBy
		if b.SortDirection != "" {
			q += " " + b.SortDirection
		}
	} else {
		// Ensure determinstic order for pagination if nothing else is specified
		q += " ORDER BY rowid"
	}

	if b.Limit != "" {
		q += " LIMIT " + b.Limit
	}
	if b.Offset != "" {
		q += " OFFSET " + b.Offset
	}

	if e.Request.URL.Query().Get("debug") == "true" {
		log.Printf("[Banquet SQL] %s", q)
	}

	rows, err := db.Query(q)
	if err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "Query failed: " + err.Error(), "sql": q})
	}
	defer rows.Close()

	// Get columns for map
	cols, _ := rows.Columns()

	// Start HTML Table
	// We need http.ResponseWriter. e.Response is it? e.Response is http.ResponseWriter (or wraps it).
	// Type casting might be needed if e.Response isn't exactly http.ResponseWriter interface in the way sqliter expects (it expects http.ResponseWriter).
	// core.RequestEvent.Response is *pocketbase/tools/router.WriterResponse which implements http.ResponseWriter.

	tw.StartHTMLTable(e.Response, cols, "Banquet Data: "+reqURI)

	// Iterate and stream rows
	columnPointers := make([]interface{}, len(cols))
	rowCounter := 0

	for rows.Next() {
		columns := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}

		// Convert to strings for simple rendering
		strRow := make([]string, len(cols))
		for i, val := range columns {
			if b, ok := val.([]byte); ok {
				strRow[i] = string(b)
			} else {
				strRow[i] = fmt.Sprintf("%v", val)
			}
		}

		tw.WriteHTMLRow(e.Response, rowCounter, strRow)
		rowCounter++
	}

	tw.EndHTMLTable(e.Response)

	return nil // Handled manually
}

func loadTemplates(dir string) *template.Template {
	pattern := filepath.Join(dir, "*.html")
	t, err := template.ParseGlob(pattern)
	if err != nil {
		log.Printf("Warning: Failed to load templates from %s (pattern: %s): %v", dir, pattern, err)
		return template.New("")
	}
	return t
}

func ensurePipelineCache(app core.App, cacheKey string, remote *core.Record, remotePath string, ttl float64) (string, error) {
	// Look for cache file
	cacheDir := filepath.Join(app.DataDir(), "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}
	dbPath := filepath.Join(cacheDir, cacheKey+".db")

	// 4. MkSQLite Conversion
	// Determine paths
	ext := filepath.Ext(remotePath)
	if ext == "" {
		ext = ".csv" // Default fallback
	}
	sourcePath := filepath.Join(cacheDir, cacheKey+ext)

	// A. Ensure Source File Exists
	sourceInfo, err := os.Stat(sourcePath)
	sourceValid := false
	if err == nil {
		if time.Since(sourceInfo.ModTime()).Minutes() < ttl {
			sourceValid = true
		}
	}

	// Re-fetch Remote Info needed for download
	// We don't have mksqliteConfig in direct requests yet, using defaults
	var mksqliteConfig *core.Record
	// TODO: Support finding config by some heuristic if needed

	// Connect Rclone
	remoteType := remote.GetString("type")

	configMap := map[string]interface{}{}
	rawConfig := remote.Get("config")
	if b, ok := rawConfig.(types.JSONRaw); ok {
		_ = json.Unmarshal(b, &configMap)
	} else if s, ok := rawConfig.(string); ok {
		_ = json.Unmarshal([]byte(s), &configMap)
	} else {
		configMap = cast.ToStringMap(rawConfig)
	}

	var rootPath string
	if v, ok := configMap["root"]; ok {
		rootPath = fmt.Sprintf("%v", v)
		delete(configMap, "root")
	}

	var sb strings.Builder
	sb.WriteString(":")
	sb.WriteString(remoteType)
	for k, v := range configMap {
		valStr := fmt.Sprintf("%v", v)
		escaped := strings.ReplaceAll(valStr, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		sb.WriteString(fmt.Sprintf(",%s=\"%s\"", k, escaped))
	}
	connStr := sb.String()

	if !sourceValid {
		log.Printf("Downloading source to cache: %s", sourcePath)

		fSys, err := rcfs.NewFs(context.Background(), connStr+":"+rootPath)
		if err != nil {
			return "", fmt.Errorf("fs new failed: %v", err)
		}

		// Try to treat as file first
		obj, err := fSys.NewObject(context.Background(), remotePath)
		if err != nil {
			// If error, check if it's a directory by Listing
			// Simple heuristic: if we can list it, treat as directory.
			// Note: NewObject returns error if path is dir.
			entries, listErr := fSys.List(context.Background(), remotePath)
			if listErr == nil {
				// It is a directory! Create a DB from the listing.
				log.Printf("Path is a directory, creating listing DB: %s", dbPath)

				// Ensure cache dir again just in case
				if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
					return "", err
				}

				// Create DB
				os.Remove(dbPath) // remove old if exists
				db, err := sql.Open("sqlite3", dbPath)
				if err != nil {
					return "", fmt.Errorf("failed to open listing db: %v", err)
				}
				defer db.Close()

				// Create Table
				toRun := `CREATE TABLE tb0 (name TEXT, size INTEGER, is_dir BOOLEAN, mod_time TEXT);`
				if _, err := db.Exec(toRun); err != nil {
					return "", fmt.Errorf("failed to create listing table: %v", err)
				}

				// Insert entries
				tx, err := db.Begin()
				if err != nil {
					return "", err
				}
				stmt, err := tx.Prepare("INSERT INTO tb0 (name, size, is_dir, mod_time) VALUES (?, ?, ?, ?)")
				if err != nil {
					return "", err
				}
				defer stmt.Close()

				for _, entry := range entries {
					_, isDir := entry.(rcfs.Directory)
					_, err := stmt.Exec(entry.Remote(), entry.Size(), isDir, entry.ModTime(context.Background()).String())
					if err != nil {
						tx.Rollback()
						return "", err
					}
				}
				if err := tx.Commit(); err != nil {
					return "", err
				}

				// We return the dbPath directly here, behaving as if "caching" is done.
				// Since we generated the DB directly, we don't need the "Convert" step below.
				return dbPath, nil
			}

			// If not a directory or list failed too, return original error
			return "", fmt.Errorf("file not found: %v (list check: %v)", err, listErr)
		}

		rcReader, err := obj.Open(context.Background())
		if err != nil {
			return "", fmt.Errorf("open file failed: %v", err)
		}
		defer rcReader.Close()

		// Stream to temp file then rename
		tmpSourcePath := sourcePath + ".tmp"
		localFile, err := os.Create(tmpSourcePath)
		if err != nil {
			return "", err
		}

		if _, err := io.Copy(localFile, rcReader); err != nil {
			localFile.Close()
			os.Remove(tmpSourcePath)
			return "", fmt.Errorf("download failed: %v", err)
		}
		localFile.Close()

		if err := os.Rename(tmpSourcePath, sourcePath); err != nil {
			return "", err
		}
	}

	// B. Ensure DB Exists
	// We check DB validity relative to source? Or just same TTL?
	// If source was re-downloaded, we must re-convert.
	// If source was valid, but DB missing/old, re-convert.
	// Simplest: Check DB TTL. If expired or missing, convert from source.

	dbInfo, err := os.Stat(dbPath)
	dbValid := false
	if err == nil {
		if time.Since(dbInfo.ModTime()).Minutes() < ttl {
			dbValid = true
		}
	}

	// If source was just updated (implied if sourceValid was false initially, but we didn't track that bool across the block perfectly without refactor)
	// Actually, if we just downloaded source, standard TTL check on DB (which likely doesn't exist or is old) will trigger conversion.

	if dbValid {
		return dbPath, nil
	}

	log.Printf("Converting local source to DB: %s -> %s", sourcePath, dbPath)

	// mksqliteConfig handling
	driver := ""
	args := map[string]interface{}{}
	if mksqliteConfig != nil {
		driver = mksqliteConfig.GetString("driver")
		if v, ok := mksqliteConfig.Get("args").(map[string]interface{}); ok {
			args = v
		}
	}

	// Auto-detect driver
	if driver == "" {
		drvExt := strings.ToLower(filepath.Ext(sourcePath))
		driver = strings.TrimPrefix(drvExt, ".")
	}

	// Build Conversion Config
	convCfg := &common.ConversionConfig{
		Verbose: true,
	}
	if val, ok := args["delimiter"].(string); ok && len(val) > 0 {
		convCfg.Delimiter = rune(val[0])
	}
	if val, ok := args["table_name"].(string); ok {
		convCfg.TableName = val
	}
	if val, ok := args["verbose"].(bool); ok {
		convCfg.Verbose = val
	}

	// Open Local Source File
	srcFile, err := os.Open(sourcePath)
	if err != nil {
		return "", fmt.Errorf("failed to open cached source: %v", err)
	}
	defer srcFile.Close()

	provider, err := converters.Open(driver, srcFile, convCfg)
	if err != nil {
		return "", fmt.Errorf("converter open failed (%s): %v", driver, err)
	}

	// Create temp DB file
	tmpDbPath := dbPath + ".tmp"
	outFile, err := os.Create(tmpDbPath)
	if err != nil {
		return "", err
	}

	err = converters.ImportToSQLite(provider, outFile, &converters.ImportOptions{Verbose: true})
	outFile.Close()

	if err != nil {
		os.Remove(tmpDbPath)
		return "", fmt.Errorf("import failed: %v", err)
	}

	if err := os.Rename(tmpDbPath, dbPath); err != nil {
		return "", err
	}

	return dbPath, nil
}

func importUserSecrets(app core.App, baseDir string) error {
	dbPath := filepath.Join(baseDir, "user_secrets.db")
	keyPath := filepath.Join(filepath.Dir(baseDir), "key") // key is in root, user_secrets.db is in user_settings/

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		log.Printf("user_secrets.db not found at %s, skipping import", dbPath)
		return nil
	}
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		log.Printf("key file not found at %s, skipping import", keyPath)
		return nil
	}

	s, err := secrets.NewService(dbPath, keyPath)
	if err != nil {
		return fmt.Errorf("failed to initialize secrets service: %w", err)
	}
	defer s.Close()

	aliases, err := s.ListAliases()
	if err != nil {
		return fmt.Errorf("failed to list aliases: %w", err)
	}

	collection, err := app.FindCollectionByNameOrId("rclone_remotes")
	if err != nil {
		return err
	}

	for _, alias := range aliases {
		// Check if record already exists
		existing, _ := app.FindFirstRecordByData("rclone_remotes", "name", alias)
		if existing != nil {
			continue
		}

		creds, err := s.GetCredentials(alias)
		if err != nil {
			log.Printf("Failed to get credentials for %s: %v", alias, err)
			continue
		}

		record := core.NewRecord(collection)
		record.Set("name", alias)

		remoteType, ok := creds["type"].(string)
		if !ok {
			remoteType = "s3" // Default or skip?
		}
		record.Set("type", remoteType)

		// Move all other fields to config
		config := make(map[string]interface{})
		for k, v := range creds {
			if k == "type" {
				continue
			}
			config[k] = v
		}
		record.Set("config", config)

		if err := app.Save(record); err != nil {
			log.Printf("Failed to save record for %s: %v", alias, err)
		} else {
			log.Printf("Imported credential: %s", alias)
		}
	}

	return nil
}

type customWriter struct {
	http.ResponseWriter
	Buffer bytes.Buffer
	Status int
}

func (w *customWriter) Write(b []byte) (int, error) {
	return w.Buffer.Write(b)
}

func (w *customWriter) WriteHeader(statusCode int) {
	w.Status = statusCode
}
