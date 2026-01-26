package tests

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase"
)

// TestScanBullshitLinks verifies that no records match the "bullshit" criteria
// (user=local, host=localhost, or explore_link containing https:/local@localhost)
func TestScanBullshitLinks(t *testing.T) {
	// Setup Logger
	logger := log.New(os.Stdout, "BS_CHECK: ", log.LstdFlags)

	// Connect to DB directly
	workDir, _ := os.Getwd()
	projectRoot, _ := filepath.Abs(filepath.Join(workDir, ".."))
	dataDir := filepath.Join(projectRoot, "user_settings", "pb_data")

	if _, err := os.Stat(filepath.Join(dataDir, "data.db")); os.IsNotExist(err) {
		t.Skip("Database not found, skipping scan.")
	}

	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: dataDir,
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}

	// 1. Scan for bad fields
	badFieldsFilter := "user='local' || host='localhost'"
	records, err := app.FindRecordsByFilter("banquet_links", badFieldsFilter, "id", 100, 0)
	if err != nil {
		t.Fatal(err)
	}

	foundBS := 0
	for _, rec := range records {
		logger.Printf("FOUND BS RECORD: %s (User: %s, Host: %s)", rec.Id, rec.GetString("user"), rec.GetString("host"))
		foundBS++
	}

	// 2. Scan for bad Explore Links matching https:/local@localhost
	// Filter substring search not always easy in PB filter syntax depending on implementation
	// Client-side filter is safer for detailed substring check
	allRecords, err := app.FindRecordsByFilter("banquet_links", "1=1", "id", 1000, 0)
	if err != nil {
		t.Fatal(err)
	}

	for _, rec := range allRecords {
		link := rec.GetString("explore_link")
		if strings.Contains(link, "https:/local") || strings.Contains(link, "@localhost") {
			logger.Printf("FOUND BS LINK: %s -> %s", rec.Id, link)
			foundBS++
		}
	}

	if foundBS > 0 {
		t.Fatalf("Test Failed: Found %d records containing bullshit strings (local/localhost/etc).", foundBS)
	} else {
		t.Log("PASSED: No bullshit links found.")
	}
}
