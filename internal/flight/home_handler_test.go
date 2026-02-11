package flight

import (
	"database/sql"
	"os"
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	_ "modernc.org/sqlite"
)

func TestEnsureHomeSQLite(t *testing.T) {
	// Setup temp dir
	tempDir, err := os.MkdirTemp("", "flight_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a minimal PocketBase app
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: tempDir,
	})

    // Initialize app components
    if err := app.Bootstrap(); err != nil {
        t.Fatal(err)
    }

	// Create 'banquet_links' collection
    collection := core.NewBaseCollection("banquet_links")
    collection.Fields.Add(&core.TextField{Name: "original_url"})
    collection.Fields.Add(&core.TextField{Name: "scheme"})
    collection.Fields.Add(&core.TextField{Name: "user"})
    collection.Fields.Add(&core.TextField{Name: "host"})
    collection.Fields.Add(&core.TextField{Name: "path"})
    collection.Fields.Add(&core.URLField{Name: "explore_link"})
    collection.Fields.Add(&core.TextField{Name: "datasetpath"})
    collection.Fields.Add(&core.TextField{Name: "columnset"})
    collection.Fields.Add(&core.TextField{Name: "query"})
    if err := app.Save(collection); err != nil {
        t.Fatal(err)
    }

	// Insert a record
    record := core.NewRecord(collection)
    record.Set("original_url", "http://example.com")
    record.Set("host", "example.com")
    if err := app.Save(record); err != nil {
        t.Fatal(err)
    }

	// Run EnsureHomeSQLite
	path, err := EnsureHomeSQLite(app)
	if err != nil {
		t.Fatalf("EnsureHomeSQLite failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); err != nil {
		t.Errorf("File not found: %s", path)
	}

	// Verify content
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var count int
	err = db.QueryRow(`SELECT count(*) FROM "0_quick_links"`).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}

	if count != 1 {
		t.Errorf("Expected 1 row, got %d", count)
	}

    var host string
    err = db.QueryRow(`SELECT host FROM "0_quick_links" LIMIT 1`).Scan(&host)
    if err != nil {
        t.Fatal(err)
    }
    if host != "example.com" {
        t.Errorf("Expected host 'example.com', got '%s'", host)
    }
}
