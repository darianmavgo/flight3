package flight

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/pocketbase/pocketbase/core"
	_ "modernc.org/sqlite"
)

// HandleHome ensures the home database exists and returns its path.
func HandleHome(e *core.RequestEvent) error {
	homePath, err := EnsureHomeSQLite(e.App)
	if err != nil {
		return e.Error(500, "Failed to ensure home database", err)
	}

	// Client expects a JSON response with path
	return e.JSON(200, map[string]string{"path": homePath})
}

// EnsureHomeSQLite creates a specialized SQLite database for the client,
// populating it with '0_quick_links' table from PocketBase 'banquet_links' collection.
func EnsureHomeSQLite(app core.App) (string, error) {
	dataDir := app.DataDir()
	homePath := filepath.Join(dataDir, "home.sqlite")

	// Always ensure schema and data is fresh
	db, err := sql.Open("sqlite", homePath)
	if err != nil {
		return "", fmt.Errorf("failed to open home database: %w", err)
	}
	defer db.Close()

	// Drop table to ensure schema is fresh
	_, err = db.Exec(`DROP TABLE IF EXISTS "0_quick_links"`)
	if err != nil {
		return "", fmt.Errorf("failed to drop table: %w", err)
	}

	// Create 0_quick_links table
	// Schema matches banquet_links collection
	_, err = db.Exec(`CREATE TABLE "0_quick_links" (
		"original_url" TEXT,
		"scheme" TEXT,
		"user" TEXT,
		"host" TEXT,
		"path" TEXT,
		"explore_link" TEXT,
		"datasetpath" TEXT,
		"columnset" TEXT,
		"query" TEXT
	);`)
	if err != nil {
		return "", fmt.Errorf("failed to create table 0_quick_links: %w", err)
	}

	// Fetch records from PocketBase
	// We use "1=1" as filter to get all records
	records, err := app.FindRecordsByFilter("banquet_links", "1=1", "", 1000, 0)
	if err != nil {
		// If collection doesn't exist or error, we still return the valid (empty) DB path
		// logging the error might be useful though.
		fmt.Printf("Warning: failed to fetch banquet_links: %v\n", err)
		return homePath, nil
	}

	if len(records) == 0 {
		return homePath, nil
	}

	tx, err := db.Begin()
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}

	stmt, err := tx.Prepare(`INSERT INTO "0_quick_links" (original_url, scheme, user, host, path, explore_link, datasetpath, columnset, query) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		return "", fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, r := range records {
		_, err = stmt.Exec(
			r.GetString("original_url"),
			r.GetString("scheme"),
			r.GetString("user"),
			r.GetString("host"),
			r.GetString("path"),
			r.GetString("explore_link"),
			r.GetString("datasetpath"),
			r.GetString("columnset"),
			r.GetString("query"),
		)
		if err != nil {
			tx.Rollback()
			return "", fmt.Errorf("failed to insert record: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	return homePath, nil
}
