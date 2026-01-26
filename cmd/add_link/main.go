package main

import (
	"log"
	"os"
	"path/filepath"

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
		log.Println("Adding specific test link...")

		coll, err := app.FindCollectionByNameOrId("banquet_links")
		if err != nil {
			return err
		}

		// The URL the user was struggling with
		fullLink := "http://127.0.0.1:8090/https:/r2-auth@https:/d8dc30936fb37cbd74552d31a709f6cf.r2.cloudflarestorage.com/test-mksqlite"

		// Hardcoded parsed values for correctness
		rec := core.NewRecord(coll)
		rec.Set("original_url", "https://r2-auth@https:/d8dc30936fb37cbd74552d31a709f6cf.r2.cloudflarestorage.com/test-mksqlite")
		rec.Set("scheme", "https")
		rec.Set("user", "r2-auth")
		rec.Set("host", "d8dc30936fb37cbd74552d31a709f6cf.r2.cloudflarestorage.com")
		rec.Set("path", "/test-mksqlite")
		rec.Set("explore_link", fullLink)

		if err := app.Save(rec); err != nil {
			log.Printf("Failed to save: %v", err)
		} else {
			log.Println("Successfully added link.")
		}

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
