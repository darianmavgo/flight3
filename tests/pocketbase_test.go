package tests

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func TestPocketbaseBasic(t *testing.T) {
	// Basic placeholder call
	t.Log("Testing pocketbase import")
	app := pocketbase.New()
	if app == nil {
		t.Error("Failed to create pocketbase app")
	}
}

func TestPocketbaseSyncEdit(t *testing.T) {
	// 1. Setup paths
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// We assume test is run from project root, but if run from 'tests/' dir, adjust.
	// We'll try to locate sample_data relative to common anchors.
	fixturePath := filepath.Join(wd, "../sample_data/fixture_links.db")
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		fixturePath = filepath.Join(wd, "sample_data/fixture_links.db") // Try relative to root
	}

	// Create a temp copy of the fixture to avoid modifying the original
	tempDBPath := filepath.Join(os.TempDir(), "fixture_links_test.db")
	t.Logf("Using temp DB: %s", tempDBPath)

	// Copy file in a separate block to ensure handles are closed
	func() {
		src, err := os.Open(fixturePath)
		if err != nil {
			t.Fatalf("Failed to open fixture: %v", err)
		}
		defer src.Close()

		dst, err := os.Create(tempDBPath)
		if err != nil {
			t.Fatalf("Failed to create temp DB file: %v", err)
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			t.Fatalf("Failed to copy fixture: %v", err)
		}
		// Explicit close (though defer works for the func scope)
		dst.Close()
	}()

	// 2. Initialize PocketBase with the temp DB

	app := pocketbase.NewWithConfig(pocketbase.Config{
		DBConnect: func(dbPath string) (*dbx.DB, error) {
			// If request is for the main data.db, use our fixture
			if strings.Contains(dbPath, "data.db") {
				return dbx.Open("sqlite", tempDBPath)
			}
			// For aux/logs, use a memory DB or separate temp file to avoid locking our fixture
			return dbx.Open("sqlite", ":memory:")
		},
		// Disable default data dir to prevent other side effects, though we are overriding DBConnect
		DefaultDataDir: os.TempDir(),
	})

	// Bootstrap the app (initializes Dao, etc.)
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("Failed to bootstrap app: %v", err)
	}

	// 3. Define and Save 'sample_links' collection
	// This triggers core/collection_record_table_sync.go logic to create the SQLite table
	collName := "sample_links"
	existingColl, err := app.FindCollectionByNameOrId(collName)
	var collection *core.Collection

	if err == nil {
		collection = existingColl
	} else {
		collection = core.NewBaseCollection(collName)
		// Add a 'url' text field
		collection.Fields.Add(&core.TextField{Name: "url"})
		// Add a 'name' text field (optional, but good for context)
		collection.Fields.Add(&core.TextField{Name: "name"})

		if err := app.Save(collection); err != nil {
			t.Fatalf("Failed to save collection (sync table): %v", err)
		}
	}

	// 4. Ensure we have at least 3 rows
	// Simplified count check via Dao
	records, err := app.FindRecordsByFilter(collName, "1=1", "id", 100, 0)
	if err != nil {
		t.Fatalf("Failed to list records: %v", err)
	}

	for i := len(records); i < 3; i++ {
		rec := core.NewRecord(collection)
		rec.Set("url", "http://placeholder.com")
		rec.Set("name", "Placeholder")
		if err := app.Save(rec); err != nil {
			t.Fatalf("Failed to create placeholder record %d: %v", i, err)
		}
		records = append(records, rec)
	}

	// 5. Update the third row
	targetRow := records[2] // 0-indexed, so index 2 is the 3rd row
	t.Logf("Updating 3rd row (ID: %s) to say 'hello darian'", targetRow.Id)

	targetRow.Set("name", "hello darian") // Setting 'name' col as implicit "say" target
	// Also set URL just in case that's what was meant
	// targetRow.Set("url", "hello darian")

	if err := app.Save(targetRow); err != nil {
		t.Fatalf("Failed to update 3rd row: %v", err)
	}

	// 6. Verify
	refetched, err := app.FindRecordById(collName, targetRow.Id)
	if err != nil {
		t.Fatalf("Failed to refetch record: %v", err)
	}

	if val := refetched.GetString("name"); val != "hello darian" {
		t.Errorf("Verification failed. Expected 'hello darian', got '%s'", val)
	} else {
		t.Log("Successfully verified 3rd row update!")
	}
}
