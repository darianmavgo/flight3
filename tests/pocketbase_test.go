package tests

import (
	"io"
	"os"
	"path/filepath"
	"testing"

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

	// Ensure we have a clean test structure
	testRoot := filepath.Join(wd, "../test_output", "pocketbase_test")
	os.RemoveAll(testRoot)

	pbDataDir := filepath.Join(testRoot, "pb_data")
	os.MkdirAll(pbDataDir, 0755)

	fixturePath := filepath.Join(wd, "../pb_public/sample_data/fixture_links.db")
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		fixturePath = filepath.Join(wd, "pb_public/sample_data/fixture_links.db")
	}

	tempDBPath := filepath.Join(pbDataDir, "data.db")

	// Copy fixture to data.db
	func() {
		src, err := os.Open(fixturePath)
		if err != nil {
			t.Fatalf("Failed to open fixture: %v", err)
		}
		defer src.Close()

		dst, err := os.Create(tempDBPath)
		if err != nil {
			t.Fatalf("Failed to create test DB file: %v", err)
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			t.Fatalf("Failed to copy fixture: %v", err)
		}
	}()

	// 2. Initialize PocketBase with the standard pb_data structure
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: pbDataDir,
	})

	// Bootstrap the app
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("Failed to bootstrap app: %v", err)
	}

	// 3. Define and Save 'sample_links' collection
	collName := "sample_links"
	existingColl, err := app.FindCollectionByNameOrId(collName)
	var collection *core.Collection

	if err == nil {
		collection = existingColl
	} else {
		collection = core.NewBaseCollection(collName)
		collection.Fields.Add(&core.TextField{Name: "url"})
		collection.Fields.Add(&core.TextField{Name: "name"})

		if err := app.Save(collection); err != nil {
			t.Fatalf("Failed to save collection: %v", err)
		}
	}

	// 4. Ensure we have at least 3 rows
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
	targetRow := records[2]
	t.Logf("Updating 3rd row (ID: %s) to say 'hello darian'", targetRow.Id)

	targetRow.Set("name", "hello darian")

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
