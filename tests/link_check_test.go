package tests

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase"
)

// TestCheckAllBanquetLinks iterates through the 'banquet_links' collection
// and makes an HTTP GET request to each 'explore_link'.
// Results are logged to 'logs/link_check_results.log'.
//
// Run this test with:
// go test -v -run TestCheckAllBanquetLinks ./tests
//
// Note: This test assumes the 'flight' server is RUNNING separately on port 8090.
// It acts as an integration test against the live server.
func TestCheckAllBanquetLinks(t *testing.T) {
	// 1. Setup Logging
	logDir := "../logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatal(err)
	}
	logFile, err := os.Create(filepath.Join(logDir, "link_check_results.log"))
	if err != nil {
		t.Fatal(err)
	}
	defer logFile.Close()

	logger := log.New(logFile, "", log.LstdFlags)
	logger.Println("Starting Banquet Link Check...")
	fmt.Println("Starting Banquet Link Check... (logging to logs/link_check_results.log)")

	// 2. Connect to PocketBase DB (Direct DB Access)
	// We read the DB directly to get the list of links, so we don't depend on an API to listing them.
	// This requires access to the data.db file.

	// Locate DB
	workDir, _ := os.Getwd() // tests/
	projectRoot, _ := filepath.Abs(filepath.Join(workDir, ".."))
	dataDir := filepath.Join(projectRoot, "user_settings", "pb_data")

	// Verify DB exists
	if _, err := os.Stat(filepath.Join(dataDir, "data.db")); os.IsNotExist(err) {
		t.Fatalf("Database not found at %s. Ensure project is setup.", dataDir)
	}

	// Initialize Headless app JUST to read DB
	// We use a separate data dir logic or copy?
	// Accessing SQLite while server is running in WAL mode is generally safe for Valid Readers.
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: dataDir,
	})

	// Bootstrap to init DB connection
	// Note: Bootstrap might try to migrations? No, just init.
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("Failed to bootstrap app reader: %v", err)
	}

	// 3. Fetch Links
	records, err := app.FindRecordsByFilter("banquet_links", "1=1", "id", 1000, 0)
	if err != nil {
		t.Fatalf("Failed to fetch banquet_links: %v", err)
	}

	logger.Printf("Found %d links to check.", len(records))

	// 4. Check Each Link
	passed := 0
	failed := 0

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	for _, rec := range records {
		link := rec.GetString("explore_link")
		if link == "" {
			logger.Printf("SKIP: Record %s has no explore_link", rec.Id)
			continue
		}

		logger.Printf("CHECK: %s (ID: %s)", link, rec.Id)

		resp, err := client.Get(link)
		if err != nil {
			logger.Printf("FAIL: Error fetching %s: %v", link, err)
			failed++
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			logger.Printf("PASS: %s [%d]", link, resp.StatusCode)
			passed++
		} else {
			logger.Printf("FAIL: %s [%d]", link, resp.StatusCode)
			failed++
		}
	}

	summary := fmt.Sprintf("Check Complete. Passed: %d, Failed: %d", passed, failed)
	logger.Println(summary)
	t.Log(summary)

	if failed > 0 {
		t.Errorf("Some links failed. See logs/link_check_results.log for details.")
	}
}
