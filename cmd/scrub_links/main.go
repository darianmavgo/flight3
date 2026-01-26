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
		log.Println("Scrubbing local links...")

		// Find links that look "local" (user=local OR host=localhost)
		records, err := app.FindRecordsByFilter("banquet_links", "user='local' || host='localhost'", "id", 500, 0)
		if err != nil {
			return err
		}

		for _, rec := range records {
			path := rec.GetString("path")
			// Ensure path starts with /
			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}

			// Scrub "bullshit" fields
			rec.Set("original_url", path) // Just the path
			rec.Set("scheme", "")
			rec.Set("user", "")
			rec.Set("host", "")

			// Update explore_link to simplified version
			// http://127.0.0.1:8090<path>
			exploreLink := "http://127.0.0.1:8090" + path
			rec.Set("explore_link", exploreLink)

			// datasetpath cleanup?
			// previously: localhost/Contacts.html
			// New: just Contacts.html? or /Contacts.html?
			// User didn't specify, but "localhost" is "bullshit".
			rec.Set("datasetpath", path)

			if err := app.Save(rec); err != nil {
				log.Printf("Failed to scrub record %s: %v", rec.Id, err)
			} else {
				log.Printf("Scrubbed %s -> %s", path, exploreLink)
			}
		}

		log.Println("Scrub complete.")
		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
