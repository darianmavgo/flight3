package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/darianmavgo/sqliter/sqliter"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"

	_ "github.com/rclone/rclone/backend/local"
)

func TestBanquetCaching(t *testing.T) {
	// Setup temp dir
	tmpDir, err := os.MkdirTemp("", "pb_test_cache")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Eval symlinks
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	csvPath := filepath.Join(tmpDir, "data.csv")
	if err := os.WriteFile(csvPath, []byte("id,val\n1,one"), 0644); err != nil {
		t.Fatal(err)
	}

	testApp, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer testApp.Cleanup()

	if err := ensureCollections(testApp); err != nil {
		t.Fatal(err)
	}

	// Setup Remote & Pipeline
	// Use explicit root = tmpDir logic
	rCol, _ := testApp.FindCollectionByNameOrId("rclone_remotes")
	remote := core.NewRecord(rCol)
	remote.Set("name", "local_cache_test")
	remote.Set("type", "local")
	// Important: Rclone expects local paths to be absolute or relative to CWD.
	// We pass tmpDir as root.
	remote.Set("config", map[string]interface{}{"root": tmpDir})
	testApp.Save(remote)

	pCol, _ := testApp.FindCollectionByNameOrId("data_pipelines")
	pipeline := core.NewRecord(pCol)
	pipeline.Set("name", "cache-pipe")
	pipeline.Set("rclone_remote", remote.Id)
	pipeline.Set("rclone_path", "data.csv") // Relative to root
	pipeline.Set("cache_ttl", 5)            // 5 minutes TTL
	testApp.Save(pipeline)

	// Helper to make request
	makeReq := func() {
		req := httptest.NewRequest(http.MethodGet, "/banquet/cache-pipe/tb0", nil)
		rec := httptest.NewRecorder()
		e := &core.RequestEvent{
			App: testApp,
			Event: router.Event{
				Request:  req,
				Response: rec,
			},
		}

		// Dummy Tw
		tpl := getMockTemplate()
		tw := sqliter.NewTableWriter(tpl, sqliter.DefaultConfig())

		if err := handleBanquet(e, tw, tpl); err != nil {
			t.Fatalf("handleBanquet failed: %v", err)
		}
		if rec.Code != 200 {
			t.Fatalf("Expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	}

	// 1. First Request (Generates Cache)
	makeReq()

	cacheDir := filepath.Join(testApp.DataDir(), "cache")
	cacheFile := filepath.Join(cacheDir, pipeline.Id+".db")

	info1, err := os.Stat(cacheFile)
	if err != nil {
		t.Fatalf("Cache file %s not created", cacheFile)
	}
	modTime1 := info1.ModTime()

	// Wait a bit to ensure potential new timestamp would be different (at least 1s if FS resolution low)
	time.Sleep(1100 * time.Millisecond)

	// 2. Second Request (Should Hit Cache)
	makeReq()

	info2, err := os.Stat(cacheFile)
	if err != nil {
		t.Fatal(err)
	}
	modTime2 := info2.ModTime()

	if !modTime1.Equal(modTime2) {
		t.Errorf("Cache was regenerated! T1: %v, T2: %v", modTime1, modTime2)
	} else {
		t.Logf("Cache hit verified. T1: %v == T2: %v", modTime1, modTime2)
	}

	// 3. Force Expiration (Set ModTime to past)
	// TTL is 5 mins. Set time to 6 mins ago.
	oldTime := time.Now().Add(-6 * time.Minute)
	if err := os.Chtimes(cacheFile, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	// 4. Third Request (Should Regenerate)
	makeReq()

	info3, err := os.Stat(cacheFile)
	if err != nil {
		t.Fatal(err)
	}
	modTime3 := info3.ModTime()

	if modTime3.Equal(oldTime) {
		t.Error("Cache should have been regenerated, but timestamp matches old forced time")
	} else {
		t.Logf("Cache regeneration verified. New Time: %v", modTime3)
	}
	// 4. Test Cache Key Conversion & Stability (Adhoc)
	t.Run("AdhocCacheKeyStability", func(t *testing.T) {
		// Mock app manually? We need to call handleBanquet.
		// We'll reuse the setup but need to ensure 'handleBanquet' uses the logic we changed.
		// We use a predefined remote 'local_cache_test' which we made in main test body.
		// That remote has name="local_cache_test".
		// We need to request a URL that parses to user=local_cache_test.

		// URL 1: Standard
		// /http:/local_cache_test@localhost/data.csv
		req1 := httptest.NewRequest(http.MethodGet, "/http:/local_cache_test@localhost/data.csv", nil)
		rec1 := httptest.NewRecorder()
		e1 := &core.RequestEvent{App: testApp, Event: router.Event{Request: req1, Response: rec1}}

		// Dummy Tw
		tpl := getMockTemplate()
		tw := sqliter.NewTableWriter(tpl, sqliter.DefaultConfig())

		if err := handleBanquet(e1, tw, tpl); err != nil {
			t.Fatalf("Req1 failed: %v", err)
		}

		// Find the cache files generated
		// We expect BOTH a .csv and a .db file
		csvFiles1, _ := filepath.Glob(filepath.Join(cacheDir, "adhoc-*.csv"))
		if len(csvFiles1) == 0 {
			t.Fatal("No source .csv cache file generated for adhoc request")
		}
		dbFiles1, _ := filepath.Glob(filepath.Join(cacheDir, "adhoc-*.db"))
		if len(dbFiles1) == 0 {
			t.Fatal("No converted .db cache file generated for adhoc request")
		}

		cacheFileCSV := csvFiles1[0]
		cacheFileDB := dbFiles1[0]

		infoCSV1, _ := os.Stat(cacheFileCSV)
		infoDB1, _ := os.Stat(cacheFileDB)

		t.Logf("Generated Cache: %s and %s", filepath.Base(cacheFileCSV), filepath.Base(cacheFileDB))

		// Wait a bit to ensure potential modifications would have new timestamps
		time.Sleep(1 * time.Second)

		// Reset
		// We want to verify that a DIFFERENT url (scheme/query diff) reuses the SAME file
		// AND does not trigger a re-download (modification).

		// URL 2: Different Scheme, Added Query
		// /https:/local_cache_test@localhost/data.csv?foo=bar&new=param
		req2 := httptest.NewRequest(http.MethodGet, "/https:/local_cache_test@localhost/data.csv?foo=bar&new=param", nil)
		rec2 := httptest.NewRecorder()
		e2 := &core.RequestEvent{App: testApp, Event: router.Event{Request: req2, Response: rec2}}

		// Dummy Tw
		tpl = getMockTemplate()
		tw = sqliter.NewTableWriter(tpl, sqliter.DefaultConfig())

		if err := handleBanquet(e2, tw, tpl); err != nil {
			t.Fatalf("Req2 failed: %v", err)
		}

		// Check existence again
		csvFiles2, _ := filepath.Glob(filepath.Join(cacheDir, "adhoc-*.csv"))
		dbFiles2, _ := filepath.Glob(filepath.Join(cacheDir, "adhoc-*.db"))

		if len(csvFiles2) != 1 || len(dbFiles2) != 1 {
			t.Errorf("Cache file count mismatch. Expected 1 each. Got CSV: %d, DB: %d", len(csvFiles2), len(dbFiles2))
		}

		if csvFiles2[0] != cacheFileCSV {
			t.Errorf("CSV Filename changed! OLD: %s, NEW: %s", cacheFileCSV, csvFiles2[0])
		}
		if dbFiles2[0] != cacheFileDB {
			t.Errorf("DB Filename changed! OLD: %s, NEW: %s", cacheFileDB, dbFiles2[0])
		}

		// Verify ModTime (No re-download)
		infoCSV2, _ := os.Stat(csvFiles2[0])
		infoDB2, _ := os.Stat(dbFiles2[0])

		if !infoCSV1.ModTime().Equal(infoCSV2.ModTime()) {
			t.Error("Source CSV file was re-downloaded/modified! Timestamp changed.")
		} else {
			t.Log("Source CSV cache hit verified (Timestamp unchanged).")
		}

		if !infoDB1.ModTime().Equal(infoDB2.ModTime()) {
			t.Error("DB file was re-converted/modified! Timestamp changed.")
		} else {
			t.Log("DB cache hit verified (Timestamp unchanged).")
		}

		t.Log("Cache key stability verified: Scheme and Query ignored.")
	})
}
