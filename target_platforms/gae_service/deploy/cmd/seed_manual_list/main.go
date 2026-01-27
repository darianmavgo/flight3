package main

import (
	"fmt"
	"log"
	"net/url"
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
		log.Println("Seeding manual list to banquet_links...")

		coll, err := app.FindCollectionByNameOrId("banquet_links")
		if err != nil {
			return err
		}

		host := "d8dc30936fb37cbd74552d31a709f6cf.r2.cloudflarestorage.com"
		user := "r2-auth"
		bucket := "test-mksqlite"

		files := []string{
			"sample_data/",
			"Apps_GoogleDownload_Darian.Device_takeout-20251014T200156Z-1-007_Takeout_Drive_trading_crisis-winners_TZA_6_years_data.csv",
			"BERKSHIRE HATHAWAY INC.html",
			"COURSE SCHEDULE Syllabus - Fall 2023 Calc 2 Schedule.csv",
			"COURSE SCHEDULE Syllabus.xlsx",
			"ECISBillStatement.pdf",
			"Expenses.csv.db",
			"Expenses.csv",
			"History.xlsx",
			"History2.xlsx",
			"app.yaml",
			"request_headers_sample_chrome.txt",
			"websites.csv",
		}

		for _, filename := range files {
			cleanName := strings.TrimSpace(filename)

			// ENCODE the filename for URL path
			encodedName := url.PathEscape(cleanName)
			path := fmt.Sprintf("/%s/%s", bucket, encodedName)

			// Inner URL (Sanitized style: no double scheme)
			originalURL := fmt.Sprintf("https://%s@%s%s", user, host, path)

			// Outer Link
			// http://127.0.0.1:8090/https:/r2-auth@HOST/PATH
			exploreLink := fmt.Sprintf("http://127.0.0.1:8090/https:/%s@%s%s", user, host, path)

			dsPath := fmt.Sprintf("%s%s", host, path)

			rec := core.NewRecord(coll)
			rec.Set("original_url", originalURL)
			rec.Set("scheme", "https")
			rec.Set("user", user)
			rec.Set("host", host)
			rec.Set("path", path)
			rec.Set("explore_link", exploreLink)
			rec.Set("datasetpath", dsPath)
			rec.Set("columnset", "")
			rec.Set("query", "")

			if err := app.Save(rec); err != nil {
				log.Printf("Failed to save %s: %v", cleanName, err)
			} else {
				log.Printf("Added: %s", cleanName)
			}
		}

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
