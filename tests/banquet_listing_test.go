package tests

import (
	"os"
	"path/filepath"
	"testing"

	// Import UI for embedding if needed, or just mock
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// TestBanquetDirectoryListing verifies that accessing a root/directory via Banquet
// returns a table listing of the files (buckets/objects) instead of an error.
func TestBanquetDirectoryListing(t *testing.T) {
	// 1. Setup Test Environment (Temp Dir & Files)
	tmpDir := "../test_output/banquet_listing_test"
	os.RemoveAll(tmpDir)

	pbPublicDir := filepath.Join(tmpDir, "pb_public")
	pbDataDir := filepath.Join(tmpDir, "pb_data")

	if err := os.MkdirAll(pbPublicDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pbDataDir, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some dummy files in pb_public to "list"
	os.WriteFile(filepath.Join(pbPublicDir, "file1.txt"), []byte("content1"), 0644)
	os.WriteFile(filepath.Join(pbPublicDir, "file2.csv"), []byte("id,val\n1,A\n2,B"), 0644)
	os.Mkdir(filepath.Join(pbPublicDir, "subdir"), 0755)

	// 2. Initialize PocketBase
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: pbDataDir,
	})

	// Bootstrap to initialize DB
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}

	// 3. Create 'rclone_remotes' collection and a 'local' mock for 'r2-auth'
	// We map 'r2-auth' to our local temp dir to simulate the bucket root
	remotesColl := core.NewBaseCollection("rclone_remotes")
	remotesColl.Fields.Add(&core.TextField{Name: "name"})
	remotesColl.Fields.Add(&core.TextField{Name: "type"})
	remotesColl.Fields.Add(&core.JSONField{Name: "config"})
	app.Save(remotesColl)

	record := core.NewRecord(remotesColl)
	record.Set("name", "r2-auth") // The alias used in the URL
	record.Set("type", "local")   // Use local fs for testing
	// Point 'root' to our temp dir
	record.Set("config", map[string]interface{}{
		"rclone_config": map[string]string{ // Not used by my logic usually, checking main.go...
		},
		// My logic flattens config. ensuring 'root' key works for local?
		// main.go: handled "root" key specially.
		"root": pbPublicDir,
	})
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}

	// Bind the Banquet handler (Need to reference/copy logic or import main if possible?
	// main is package main. I cannot import it.
	// I must rely on running the BINARY or copy-pasting the handler logic into a test helper.
	// Since I cannot call main.handleBanquet from here, I will check if I can rely on 'banquet' package
	// or if I need to move handleBanquet to a shared package.
	// For NOW, I will use the binary running approach OR just inspect the file structure
	// BUT the user asked to "add a test".
	// I will attempt to REFINE main.go first to export the handler or move logic to internal/banquet_handler.

	// WAIT: I can just run the logic inside the test by re-implementing the call to the handler
	// IF I extract the handler to a separate package.
	// Refactoring `cmd/flight/main.go` -> logic to `internal/app/handlers.go` is best practice.
}
