package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func main() {
	// Source: user_settings/pb_data
	// Destination: pb_data (current)

	wd, _ := os.Getwd()
	sourceDataDir := "/Users/darianhickman/Documents/user_settings/pb_data" // Absolute path from user
	if _, err := os.Stat(sourceDataDir); err != nil {
		// Try relative to parent if it's there
		sourceDataDir = filepath.Join(wd, "..", "user_settings", "pb_data")
	}

	destDataDir := filepath.Join(wd, "pb_data")

	log.Printf("Importing from: %s", sourceDataDir)
	log.Printf("Importing to:   %s", destDataDir)

	// 1. Open Source App (Headless)
	sourceApp := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: sourceDataDir,
	})
	if err := sourceApp.Bootstrap(); err != nil {
		log.Fatalf("Failed to bootstrap source app: %v", err)
	}

	// 2. Open Destination App (Headless)
	destApp := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: destDataDir,
	})
	if err := destApp.Bootstrap(); err != nil {
		log.Fatalf("Failed to bootstrap destination app: %v", err)
	}

	// 3. Collections to import
	collections := []string{"rclone_remotes", "mksqlite_configs", "data_pipelines", "banquet_links"}

	for _, collName := range collections {
		log.Printf("Processing collection: %s", collName)

		// Check if collection exists in both
		sourceColl, err := sourceApp.FindCollectionByNameOrId(collName)
		if err != nil {
			log.Printf("  Skip: Collection %s not found in source", collName)
			continue
		}

		destColl, err := destApp.FindCollectionByNameOrId(collName)
		if err != nil {
			log.Printf("  Skip: Collection %s not found in destination", collName)
			continue
		}

		// Fetch all records from source
		records, err := sourceApp.FindRecordsByFilter(sourceColl.Id, "1=1", "", 0, 0)
		if err != nil {
			log.Printf("  Error fetching records: %v", err)
			continue
		}

		log.Printf("  Found %d records to upsert", len(records))

		for _, srcRec := range records {
			// Try to find by ID in destination
			destRec, err := destApp.FindRecordById(destColl.Id, srcRec.Id)
			if err != nil {
				// Create new if not found
				destRec = core.NewRecord(destColl)
				destRec.Id = srcRec.Id
			}

			// Copy all fields
			for k, v := range srcRec.PublicExport() {
				// Skip system fields that manage IDs or auth
				if k == "id" || k == "collectionId" || k == "collectionName" {
					continue
				}
				destRec.Set(k, v)
			}

			// Default 'enabled' to true for remotes if it's a bool field and currently false/unset
			if collName == "rclone_remotes" && !destRec.GetBool("enabled") {
				destRec.Set("enabled", true)
			}

			// Copy standard fields manually
			destRec.Set("created", srcRec.Get("created"))
			destRec.Set("updated", srcRec.Get("updated"))

			if err := destApp.Save(destRec); err != nil {
				log.Printf("    Failed to save record %s: %v", srcRec.Id, err)
			} else {
				// log.Printf("    Upserted %s", srcRec.Id)
			}
		}
	}

	log.Println("Import complete!")
}
