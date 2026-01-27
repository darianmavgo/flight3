package tests

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/darianmavgo/flight3/internal/flight"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pocketbase/pocketbase"
)

// TestIndexDirectoryRemote verifies that the IndexDirectory method correctly
// indexes a remote directory into a SQLite database with the standard tb0 schema.
func TestIndexDirectoryRemote(t *testing.T) {
	// 1. Setup Test Environment
	testRoot := filepath.Join("..", "test_output", "index_directory_test")
	os.RemoveAll(testRoot)
	defer os.RemoveAll(testRoot)

	pbDataDir := filepath.Join(testRoot, "pb_data")
	os.MkdirAll(pbDataDir, 0755)

	// 2. Initialize PocketBase
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: pbDataDir,
	})

	if err := app.Bootstrap(); err != nil {
		t.Fatalf("Failed to bootstrap PocketBase: %v", err)
	}

	// 3. Initialize Rclone Manager
	cacheDir := filepath.Join(pbDataDir, "cache")
	if err := flight.InitRclone(cacheDir); err != nil {
		t.Fatalf("Failed to initialize rclone: %v", err)
	}

	rcloneManager := flight.GetRcloneManager()
	if rcloneManager == nil {
		t.Fatal("Rclone manager is nil")
	}

	// 4. Ensure collections exist
	if err := flight.EnsureCollections(app); err != nil {
		t.Fatalf("Failed to ensure collections: %v", err)
	}

	// 5. Find a real remote to test with
	// We'll use the first enabled S3 remote from the database
	remoteRecord, err := app.FindFirstRecordByFilter(
		"rclone_remotes",
		"enabled = true && type = 's3'",
	)
	if err != nil {
		t.Skip("No enabled S3 remote found in database, skipping remote directory test")
	}

	t.Logf("Testing with remote: %s (type: %s)",
		remoteRecord.GetString("name"),
		remoteRecord.GetString("type"))

	// 6. Get VFS for the remote
	vfs, err := rcloneManager.GetVFS(remoteRecord)
	if err != nil {
		t.Fatalf("Failed to get VFS: %v", err)
	}

	// 7. Test indexing the root directory
	remotePath := "/"
	cachePath := filepath.Join(cacheDir, "test_index.db")

	err = rcloneManager.IndexDirectory(vfs, remotePath, cachePath)
	if err != nil {
		t.Fatalf("IndexDirectory failed: %v", err)
	}

	// 8. Verify the SQLite database was created
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Fatal("Cache database was not created")
	}

	// 9. Open and verify the database schema
	db, err := sql.Open("sqlite3", cachePath)
	if err != nil {
		t.Fatalf("Failed to open cache database: %v", err)
	}
	defer db.Close()

	// Verify table exists
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='tb0'").Scan(&tableName)
	if err != nil {
		t.Fatalf("Table tb0 not found: %v", err)
	}

	// Verify columns
	rows, err := db.Query("PRAGMA table_info(tb0)")
	if err != nil {
		t.Fatalf("Failed to get table info: %v", err)
	}
	defer rows.Close()

	expectedColumns := map[string]bool{
		"path":      false,
		"name":      false,
		"size":      false,
		"extension": false,
		"mod_time":  false,
		"is_dir":    false,
	}

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue interface{}

		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			t.Fatalf("Failed to scan column info: %v", err)
		}

		if _, exists := expectedColumns[name]; exists {
			expectedColumns[name] = true
		}
	}

	for col, found := range expectedColumns {
		if !found {
			t.Errorf("Expected column %s not found in tb0", col)
		}
	}

	// 10. Verify data was inserted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM tb0").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count rows: %v", err)
	}

	if count == 0 {
		t.Error("No entries were indexed (this may be expected for an empty remote)")
	} else {
		t.Logf("Successfully indexed %d entries", count)
	}

	// 11. Verify a sample row has the expected structure
	if count > 0 {
		var path, name, size, extension, modTime, isDir string
		err = db.QueryRow("SELECT path, name, size, extension, mod_time, is_dir FROM tb0 LIMIT 1").
			Scan(&path, &name, &size, &extension, &modTime, &isDir)
		if err != nil {
			t.Fatalf("Failed to read sample row: %v", err)
		}

		t.Logf("Sample entry: path=%s, name=%s, size=%s, ext=%s, is_dir=%s",
			path, name, size, extension, isDir)

		// Basic validation
		if name == "" {
			t.Error("Entry name is empty")
		}
		if isDir != "0" && isDir != "1" {
			t.Errorf("is_dir should be '0' or '1', got: %s", isDir)
		}
	}
}

// TestIndexDirectoryLocal verifies IndexDirectory works with local filesystem
// by using the mksqlite filesystem converter directly
func TestIndexDirectoryLocal(t *testing.T) {
	// 1. Setup Test Environment
	testRoot := filepath.Join("..", "test_output", "index_directory_local_test")
	os.RemoveAll(testRoot)
	defer os.RemoveAll(testRoot)

	pbPublicDir := filepath.Join(testRoot, "pb_public")
	os.MkdirAll(pbPublicDir, 0755)

	// Create some test files
	os.WriteFile(filepath.Join(pbPublicDir, "test1.txt"), []byte("content1"), 0644)
	os.WriteFile(filepath.Join(pbPublicDir, "test2.csv"), []byte("a,b,c\n1,2,3"), 0644)
	os.Mkdir(filepath.Join(pbPublicDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(pbPublicDir, "subdir", "nested.json"), []byte("{}"), 0644)

	// 2. Use ConvertToSQLite which handles local directories via mksqlite
	cachePath := filepath.Join(testRoot, "local_index.db")

	err := flight.ConvertToSQLite(pbPublicDir, cachePath)
	if err != nil {
		t.Fatalf("ConvertToSQLite failed: %v", err)
	}

	// 3. Verify the database
	db, err := sql.Open("sqlite3", cachePath)
	if err != nil {
		t.Fatalf("Failed to open cache database: %v", err)
	}
	defer db.Close()

	// Verify table exists
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='tb0'").Scan(&tableName)
	if err != nil {
		t.Fatalf("Table tb0 not found: %v", err)
	}

	// Count entries
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM tb0").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count rows: %v", err)
	}

	// We expect at least 3 files + 1 directory = 4 entries
	if count < 4 {
		t.Errorf("Expected at least 4 entries, got %d", count)
	}

	// Verify we can find our test files
	var foundTxt, foundCsv, foundDir bool
	rows, err := db.Query("SELECT name, is_dir FROM tb0")
	if err != nil {
		t.Fatalf("Failed to query entries: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name, isDir string
		if err := rows.Scan(&name, &isDir); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}

		switch name {
		case "test1.txt":
			foundTxt = true
			if isDir != "0" {
				t.Error("test1.txt should not be marked as directory")
			}
		case "test2.csv":
			foundCsv = true
			if isDir != "0" {
				t.Error("test2.csv should not be marked as directory")
			}
		case "subdir":
			foundDir = true
			if isDir != "1" {
				t.Error("subdir should be marked as directory")
			}
		}
	}

	if !foundTxt {
		t.Error("test1.txt not found in index")
	}
	if !foundCsv {
		t.Error("test2.csv not found in index")
	}
	if !foundDir {
		t.Error("subdir not found in index")
	}

	t.Logf("Successfully verified local directory indexing with %d entries", count)
}
