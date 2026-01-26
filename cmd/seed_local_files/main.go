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
	sampleDataDir := filepath.Join(workDir, "sample_data")

	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: dataDir,
	})

	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		log.Println("Seeding local files to banquet_links...")

		// 1. Ensure 'local' remote exists
		remoteColl, err := app.FindCollectionByNameOrId("rclone_remotes")
		if err != nil {
			return err
		}

		remoteName := "local"
		existingRemote, _ := app.FindFirstRecordByData("rclone_remotes", "name", remoteName)
		if existingRemote == nil {
			log.Println("Creating 'local' remote pointing to sample_data...")
			rec := core.NewRecord(remoteColl)
			rec.Set("name", remoteName)
			rec.Set("type", "local") // Rclone type
			// Local config usually just needs keys, but for my custom logic in main.go, I support "root" in config
			// main.go: handled "root" key specially.
			// However, standard Rclone "local" backend doesn't use "root" key in connection string the same way...
			// Wait, main.go logic:
			// 	if v, ok := configMap["root"]; ok { rootPath = ... }
			//  connStr := ... + ":" + rootPath
			// For local, config is usually empty, and root is the path.
			// So yes, my main.go 'root' logic supports this abstraction perfectly.
			rec.Set("config", map[string]interface{}{
				"root": sampleDataDir,
			})
			if err := app.Save(rec); err != nil {
				return fmt.Errorf("failed to save local remote: %v", err)
			}
		}

		// 2. Add Links
		linkColl, err := app.FindCollectionByNameOrId("banquet_links")
		if err != nil {
			return err
		}

		files := []string{
			"./Contacts.html",
			"./sample.xlsx",
			"./sample.csv",
			"./fixture_links.db",
			"./20mb.xlsx",
			"./test.html",
			"./bq.html",
			"./21mb.csv",
			"./demo_chrome",
			"./demo_chrome/twitter.architecture.jpeg",
			"./demo_chrome/request_headers_postman.txt",
			"./demo_chrome/Affirmations.pdf",
			"./demo_chrome/ContentType.xml",
			"./demo_chrome/ECISBillStatement.pdf",
			"./demo_chrome/desktop.ini",
			"./demo_chrome/request_headers_sample_curl.txt",
			"./demo_chrome/t8.shakespeare.txt",
		}

		host := "localhost" // Arbitrary for local?
		// User alias matches the remote name "local" so automatic resolution works
		user := "local"

		for _, rawPath := range files {
			// Strip leading .
			cleanPath := strings.TrimPrefix(rawPath, ".")

			// Ensure it starts with /
			if !strings.HasPrefix(cleanPath, "/") {
				cleanPath = "/" + cleanPath
			}

			// Encode path components (simple way: split, encode, join)
			// But since these are known safe strings mostly, url.PathEscape is good.
			// We want to escape segments, not slashes.
			parts := strings.Split(cleanPath, "/")
			encodedParts := make([]string, len(parts))
			for i, p := range parts {
				if p == "" {
					continue
				} // root / -> empty first part
				encodedParts[i] = url.PathEscape(p)
			}
			// Rejoin
			encodedPath := strings.Join(encodedParts, "/")
			if strings.HasPrefix(cleanPath, "/") && !strings.HasPrefix(encodedPath, "/") {
				encodedPath = "/" + encodedPath
			}

			// Inner URL
			// usage: https://local@localhost/file.csv
			originalURL := fmt.Sprintf("https://%s@%s%s", user, host, encodedPath)

			// Explore Link
			exploreLink := fmt.Sprintf("http://127.0.0.1:8090/https:/%s@%s%s", user, host, encodedPath)

			dsPath := fmt.Sprintf("%s%s", host, encodedPath)

			// Check if exists? Skip duplicates? user didn't specify. I'll just add.

			rec := core.NewRecord(linkColl)
			rec.Set("original_url", originalURL)
			rec.Set("scheme", "https")
			rec.Set("user", user)
			rec.Set("host", host)
			rec.Set("path", encodedPath)
			rec.Set("explore_link", exploreLink)
			rec.Set("datasetpath", dsPath)
			rec.Set("columnset", "")
			rec.Set("query", "")

			if err := app.Save(rec); err != nil {
				log.Printf("Failed to save %s: %v", rawPath, err)
			} else {
				log.Printf("Added local: %s", rawPath)
			}
		}

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
