package flight

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/pocketbase/pocketbase/core"
	_ "modernc.org/sqlite" // Ensure sqlite driver is available
)

// HandleHomeSQLite ensures home.sqlite exists and returns its absolute path
func HandleHomeSQLite(e *core.RequestEvent) error {
	// 1. Ensure home.sqlite exists and is up to date
	homePath, err := EnsureHomeSQLite(e.App)
	if err != nil {
		return e.Error(500, "Failed to generate home.sqlite: "+err.Error(), err)
	}

	// 2. Return the absolute path as JSON
	// The client (Sqliter) running on the SAME machine will read this path
	// and open the file directly via FFI.
	return e.JSON(http.StatusOK, map[string]string{
		"path": homePath,
	})
}

// EnsureHomeSQLite creates/updates the home.sqlite database in the root data directory
func EnsureHomeSQLite(app core.App) (string, error) {
	dataDir := GetAppDataDirectory()
	homePath := filepath.Join(dataDir, "home.sqlite")

	// 1. Init DB if needed
	db, err := sql.Open("sqlite", homePath)
	if err != nil {
		return "", fmt.Errorf("failed to open home.sqlite: %w", err)
	}
	defer db.Close()

	// 2. Enable WAL mode
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return "", fmt.Errorf("failed to set WAL mode: %w", err)
	}

	// 3. Create Tables
	schema := `
	CREATE TABLE IF NOT EXISTS "0_quick_links" (
		label TEXT, 
		target TEXT, 
		icon TEXT, 
		action TEXT, 
		description TEXT
	);
	CREATE TABLE IF NOT EXISTS "1_recent_files" (
		filename TEXT, 
		path TEXT, 
		last_opened DATETIME,
		size_mb REAL,
		PRIMARY KEY (path)
	);
	CREATE TABLE IF NOT EXISTS "2_banquet_links" (
		name TEXT, 
		original_url TEXT, 
		description TEXT,
		PRIMARY KEY (original_url)
	);
	CREATE TABLE IF NOT EXISTS "3_query_styles" (
		name TEXT,
		sql TEXT,
		description TEXT,
		is_dangerous BOOLEAN,
		PRIMARY KEY (name)
	);
	CREATE TABLE IF NOT EXISTS "9_system_messages" (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP, 
		level TEXT, 
		message TEXT
	);
	`
	if _, err := db.Exec(schema); err != nil {
		return "", fmt.Errorf("failed to create schema: %w", err)
	}

	// 4. Populate "0_quick_links" (Always refresh)
	if _, err := db.Exec("DELETE FROM \"0_quick_links\""); err != nil {
		return "", err
	}

	quickLinks := []struct {
		Label, Target, Icon, Action, Desc string
	}{
		{"📂 Open Local File", "", "folder_open", "open_file", "Pick a SQLite database from your computer"},
		{"☁️ Connect to Remote", "", "cloud", "connect_remote", "Enter a Flight URL or S3 bucket"},
		{"📝 New Query", "", "edit", "new_query", "Start a scratchpad query"},
	}

	tx, err := db.Begin()
	if err != nil {
		return "", err
	}
	stmt, err := tx.Prepare("INSERT INTO \"0_quick_links\" (label, target, icon, action, description) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return "", err
	}
	defer stmt.Close()

	for _, link := range quickLinks {
		if _, err := stmt.Exec(link.Label, link.Target, link.Icon, link.Action, link.Desc); err != nil {
			tx.Rollback()
			return "", err
		}
	}

	// 5. Populate "2_banquet_links" from PocketBase
	if _, err := tx.Exec("DELETE FROM \"2_banquet_links\""); err != nil {
		tx.Rollback()
		return "", err
	}

	records, err := app.FindRecordsByFilter("banquet_links", "created > '2000-01-01'", "-created", 100, 0)
	if err == nil {
		stmtLink, err := tx.Prepare("INSERT INTO \"2_banquet_links\" (name, original_url, description) VALUES (?, ?, ?)")
		if err == nil {
			defer stmtLink.Close()
			for _, r := range records {
				name := r.GetString("name")
				if name == "" {
					name = filepath.Base(r.GetString("original_url"))
				}
				stmtLink.Exec(name, r.GetString("original_url"), r.GetString("description"))
			}
		}
	}

	// 6. Populate "3_query_styles" from PocketBase
	if _, err := tx.Exec("DELETE FROM \"3_query_styles\""); err != nil {
		tx.Rollback()
		return "", err
	}

	styleRecords, err := app.FindRecordsByFilter("query_style", "created > '2000-01-01'", "-created", 100, 0)
	if err == nil {
		stmtStyle, err := tx.Prepare("INSERT INTO \"3_query_styles\" (name, sql, description, is_dangerous) VALUES (?, ?, ?, ?)")
		if err == nil {
			defer stmtStyle.Close()
			for _, r := range styleRecords {
				isDangerous := r.GetBool("is_dangerous")
				stmtStyle.Exec(r.GetString("name"), r.GetString("sql"), r.GetString("description"), isDangerous)
			}
		}
	}

	// 6. Add System Message
	if _, err := tx.Exec("INSERT INTO \"9_system_messages\" (level, message) VALUES (?, ?)", "INFO", fmt.Sprintf("Home refreshed at %s", time.Now().Format(time.RFC3339))); err != nil {
		// ignore error
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	log.Printf("[FLIGHT] Refreshed home.sqlite at %s", homePath)
	return homePath, nil
}
