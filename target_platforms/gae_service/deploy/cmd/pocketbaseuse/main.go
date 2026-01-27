package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func main() {
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DBConnect: func(dbPath string) (*dbx.DB, error) {
			// Redirect "data.db" connections to our sample_data/fixture_links.db
			// We check for "data.db" suffix because PocketBase constructs the path like ".../data.db"
			if strings.HasSuffix(dbPath, "data.db") {
				// We assume the app is run from the project root.
				// If not, we might need a more robust way to find sample_data.
				// For now, look for sample_data in the current working directory.

				wd, err := os.Getwd()
				if err != nil {
					return nil, err
				}

				// Construct path to existing fixture
				customPath := filepath.Join(wd, "sample_data", "fixture_links.db")

				// Verify it exists (optional, but good for debugging)
				if _, err := os.Stat(customPath); err == nil {
					log.Printf("Using custom database: %s", customPath)
					return dbx.Open("sqlite", customPath)
				} else {
					log.Printf("Custom database not found at %s, falling back to default %s", customPath, dbPath)
				}
			}

			// Default behavior for other DBs (like logs.db) or fallback
			return dbx.Open("sqlite", dbPath)
		},
	})

	// Register a hook to migrate existing non-PocketBase tables if needed
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// Check if 'links' collection already exists
		collectionName := "links"
		sourceTableName := "test_links"

		_, err := app.FindCollectionByNameOrId(collectionName)
		if err == nil {
			// Collection exists
			return nil
		}

		// Check if the source table exists
		var exists int
		err = app.DB().NewQuery("SELECT count(*) FROM sqlite_master WHERE type='table' AND name={:name}").Bind(dbx.Params{"name": sourceTableName}).Row(&exists)
		if err != nil {
			return err
		}

		if exists > 0 {
			log.Printf("Migrating legacy table '%s' to PocketBase collection '%s'...", sourceTableName, collectionName)

			// 1. Rename old table
			tempTableName := "_" + sourceTableName + "_old"
			_, err = app.DB().NewQuery("ALTER TABLE " + sourceTableName + " RENAME TO " + tempTableName).Execute()
			if err != nil {
				return err
			}

			// 2. Create new collection
			collection := core.NewBaseCollection(collectionName)

			// Add 'url' field
			collection.Fields.Add(&core.TextField{Name: "url"}) // Assuming default config is fine

			// Save the collection (this creates the table)
			if err := app.Save(collection); err != nil {
				return err
			}

			// 3. Copy data
			_, err = app.DB().NewQuery(`
				INSERT INTO ` + collectionName + ` (id, created, updated, url)
				SELECT 
					lower(hex(randomblob(15))), -- pseudo-random ID
					datetime('now'),
					datetime('now'),
					url
				FROM ` + tempTableName + `
			`).Execute()
			if err != nil {
				return err
			}

			// Optional: Drop old table
			_, err = app.DB().NewQuery("DROP TABLE " + tempTableName).Execute()
			if err != nil {
				log.Printf("Warning: failed to drop temp table %s: %v", tempTableName, err)
			}

			log.Printf("Migration of '%s' completed.", collectionName)
		} else {
			log.Printf("Table '%s' not found, skipping migration.", sourceTableName)
		}

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
