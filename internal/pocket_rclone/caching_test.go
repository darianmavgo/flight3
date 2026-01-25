package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

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
		if err := handleBanquet(e); err != nil {
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
}
