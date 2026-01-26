package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

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
		log.Println("Fixing local links...")

		records, err := app.FindRecordsByFilter("banquet_links", "user='local'", "id", 200, 0)
		if err != nil {
			return err
		}

		for _, rec := range records {
			path := rec.GetString("path")
			// User wants original_url to be just the path: /Contacts.html
			// Ensure path starts with /
			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}

			log.Printf("Updating %s: original_url -> %s", rec.Id, path)
			rec.Set("original_url", path)

			if err := app.Save(rec); err != nil {
				log.Printf("Failed to update record %s: %v", rec.Id, err)
			}
		}

		log.Println("Fix complete.")
		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
