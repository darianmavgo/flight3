package main

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"os"
	"path/filepath"

	"github.com/darianmavgo/banquet"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func main() {
	// 1. Locate Data Dir (Assumes running from Project Root)
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	dataDir := filepath.Join(workDir, "user_settings", "pb_data")

	// 2. Initialize PocketBase (Headless)
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: dataDir,
	})

	// 3. Define the Seeding Logic
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		log.Println("Starting seeding...")

		collName := "banquet_links"

		// A. Ensure Collection Exists
		collection, err := app.FindCollectionByNameOrId(collName)
		if err != nil {
			log.Println("Creating 'banquet_links' collection...")
			collection = core.NewBaseCollection(collName)
			collection.Fields.Add(&core.TextField{Name: "original_url"})
			collection.Fields.Add(&core.TextField{Name: "scheme"})
			collection.Fields.Add(&core.TextField{Name: "user"})
			collection.Fields.Add(&core.TextField{Name: "host"})
			collection.Fields.Add(&core.TextField{Name: "path"})
			collection.Fields.Add(&core.URLField{Name: "explore_link"})
			if err := app.Save(collection); err != nil {
				return fmt.Errorf("failed to create collection: %v", err)
			}
		}

		// B. Generate Links
		log.Println("Generating links...")

		// Base templates
		baseHost := "d8dc30936fb37cbd74552d31a709f6cf.r2.cloudflarestorage.com"
		authUser := "r2-auth"

		// Known paths from codebase or reasonable guests
		paths := []string{
			"/",
			"/test-mksqlite/",
			"/test-mksqlite/sample_data/",
			"/test-mksqlite/sample_data/21mb.csv",
			"/bucket-a/",
			"/bucket-b/file.txt",
		}

		for i := 0; i < 50; i++ {
			// Cycle through paths or generate variants
			path := paths[i%len(paths)]
			if i >= len(paths) {
				path = fmt.Sprintf("/generated-path-%d/file.csv", i)
			}

			// Construct Banquet URL (Inner URL)
			// https://r2-auth@host/path
			// innerURL := fmt.Sprintf("https://%s@%s%s", authUser, baseHost, path)

			// Construct Request URL (Outer URL)
			// http://127.0.0.1:8090/https:/r2-auth@host/path
			// Note logic in main.go expects "/https:/..."
			requestURL := fmt.Sprintf("http://127.0.0.1:8090/https:/%s@%s%s", authUser, baseHost, path)

			// Parse
			// We use the Inner URL for parsing logic simulation, or valid banquet parser
			// The parser expects the encoded form usually found in path.
			// Let's use the logic found in debug_parsing.go to be safe:
			// Normalize "https:/" -> "https://"

			// Actually, we want to store the PARSED parts.
			// Let's use banquet.ParseNested on the inner URL purely (or the request URI part)

			// Simulating what the server sees: "/https:/..."
			serverSeenURI := fmt.Sprintf("/https:/%s@%s%s", authUser, baseHost, path)

			// Normalize as main.go does
			normalizedURI := serverSeenURI
			if strings.Contains(normalizedURI, "https:/") && !strings.Contains(normalizedURI, "https://") {
				normalizedURI = strings.Replace(normalizedURI, "https:/", "https://", 1)
			}
			if strings.HasPrefix(normalizedURI, "/") {
				normalizedURI = strings.TrimPrefix(normalizedURI, "/")
			}

			// Now parse
			b, err := banquet.ParseNested(normalizedURI)
			if err != nil {
				// Fallback with net/url if banquet fails (as main.go does patch)
				log.Printf("Banquet parse failed for %s: %v", normalizedURI, err)
			}

			// Patch from main.go
			if u, err := url.Parse(normalizedURI); err == nil {
				if u.Scheme != "" && b.Scheme == "" {
					b.Scheme = u.Scheme
				}
				if u.Host != "" && b.Host == "" {
					b.Host = u.Host
				}
				if u.User != nil && b.User == nil {
					b.User = u.User
				}
			}

			// Create Record
			rec := core.NewRecord(collection)
			rec.Set("original_url", normalizedURI) // Store inner part? Or request URL? User said "banquet urls"
			// I'll store the full requestable URL in explore_link, and inner in original_url

			rec.Set("scheme", b.Scheme)
			if b.User != nil {
				rec.Set("user", b.User.Username())
			}
			rec.Set("host", b.Host)
			rec.Set("path", b.Path) // or b.DataSetPath?
			rec.Set("explore_link", requestURL)

			if err := app.Save(rec); err != nil {
				log.Printf("Failed to save record %d: %v", i, err)
			}
		}

		log.Println("Seeding complete. Exiting.")
		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
