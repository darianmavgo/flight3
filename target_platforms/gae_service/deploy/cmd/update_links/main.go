package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/darianmavgo/banquet"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func main() {
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	dataDir := filepath.Join(workDir, "user_settings", "pb_data")

	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: dataDir,
	})

	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		log.Println("Updating banquet_links schema and data...")

		coll, err := app.FindCollectionByNameOrId("banquet_links")
		if err != nil {
			return err
		}

		// 1. Add New Fields if missing
		updates := false
		if coll.Fields.GetByName("datasetpath") == nil {
			coll.Fields.Add(&core.TextField{Name: "datasetpath"})
			updates = true
		}
		if coll.Fields.GetByName("columnset") == nil {
			coll.Fields.Add(&core.TextField{Name: "columnset"})
			updates = true
		}
		if coll.Fields.GetByName("query") == nil {
			coll.Fields.Add(&core.TextField{Name: "query"})
			updates = true
		}

		if updates {
			if err := app.Save(coll); err != nil {
				return err
			}
			log.Println("Schema updated.")
		}

		// 2. Update Records
		// Increase limit to cover 50+ records
		// Sort by id for determinism
		records, err := app.FindRecordsByFilter("banquet_links", "1=1", "id", 200, 0)
		if err != nil {
			return err
		}

		for _, rec := range records {
			originalURL := rec.GetString("original_url")
			if originalURL == "" {
				continue
			}

			// Parse with Banquet logic
			// Need to sanitize similar to main.go if needed, assume original_url stores the sanitized inner or cleaned one?
			// seed script stored "normalizedURI".

			b, err := banquet.ParseNested(originalURL)
			if err != nil {
				log.Printf("Failed to parse %s: %v", originalURL, err)
				continue
			}

			// Extract
			datasetPath := b.DataSetPath
			columnSet := b.ColumnPath

			// Query: user likely wants the raw query string "foo=bar" or similar if present?
			// Banquet parse usually strips query from Path?
			// Let's check b.URL.RawQuery if b.URL is populated.
			var qry string
			if b.URL != nil {
				qry = b.URL.RawQuery
			} else {
				// Fallback to manual check if b.URL is nil (ParseNested sometimes populates it)
				if idx := strings.Index(originalURL, "?"); idx != -1 {
					qry = originalURL[idx+1:]
				}
			}

			// Update fields
			rec.Set("datasetpath", datasetPath)
			rec.Set("columnset", columnSet)
			rec.Set("query", qry)

			if err := app.Save(rec); err != nil {
				log.Printf("Failed to update record %s: %v", rec.Id, err)
			}
		}

		log.Println("Update complete.")
		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
