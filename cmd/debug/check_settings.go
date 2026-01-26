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
		// List all collections
		collections, err := app.FindAllCollections(core.CollectionTypeBase)
		if err != nil {
			return err
		}

		log.Println("Collections:")
		found := false
		for _, c := range collections {
			log.Printf("- %s", c.Name)
			if c.Name == "app_settings" {
				found = true
			}
		}

		if found {
			// Check content
			rec, err := app.FindFirstRecordByData("app_settings", "key", "serve_folder")
			if err == nil {
				log.Printf("Found serve_folder: %s", rec.GetString("value"))
			} else {
				log.Println("serve_folder key not found in app_settings")
			}
		}

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
