package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestHandleBanquet(t *testing.T) {
	// Setup temp dir for test data (source files)
	tmpDir, err := os.MkdirTemp("", "pb_test_rclone")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Eval symlinks
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy CSV file
	csvPath := filepath.Join(tmpDir, "data.csv")
	csvContent := "id,name,value\n1,alpha,100\n2,beta,200\n3,gamma,300"
	if err := os.WriteFile(csvPath, []byte(csvContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Setup PocketBase
	testApp, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer testApp.Cleanup()

	// Initialize collections
	if err := ensureCollections(testApp); err != nil {
		t.Fatal(err)
	}

	// Create Rclone Remote (Local)
	remoteCollection, _ := testApp.FindCollectionByNameOrId("rclone_remotes")
	remote := core.NewRecord(remoteCollection)
	remote.Set("name", "local_test")
	remote.Set("type", "local")
	remote.Set("config", map[string]interface{}{"root": tmpDir})
	if err := testApp.Save(remote); err != nil {
		t.Fatal(err)
	}

	// Pipeline Record
	pipelineCollection, _ := testApp.FindCollectionByNameOrId("data_pipelines")
	pipeline := core.NewRecord(pipelineCollection)
	pipeline.Set("name", "test-pipeline")
	pipeline.Set("rclone_remote", remote.Id)
	pipeline.Set("rclone_path", "data.csv") // Relative to root
	pipeline.Set("cache_ttl", 10)
	if err := testApp.Save(pipeline); err != nil {
		t.Fatal(err)
	}

	// Mock Request
	// URL: /banquet/test-pipeline/tb0
	// We need to register the handler first on the app router
	// But testApp doesn't expose the router easily for ServeHTTP?
	// Actually tests.NewTestApp returns *TestApp which has logic.
	// Typically we manually invoke handler or use testApp.Router?
	// In v0.23, e.Router is available in OnServe.

	// We can manually construct RequestEvent and call handleBanquet.
	// But handleBanquet needs *core.RequestEvent which wraps echo.Context (or Go net/http).
	// PB v0.23 uses Go 1.22 mux? No, it uses its own router wrapper.

	// Let's try to mock the RequestEvent.
	// core.NewRequestEvent(app, response, request) ?

	req := httptest.NewRequest(http.MethodGet, "/banquet/test-pipeline/tb0", nil)
	rec := httptest.NewRecorder()

	// We need to bootstrap the app partially to have dataDir?
	// testApp creates temp data dir.

	// Construct event
	// core.RequestEvent struct has App, Response, Request.
	// Response is http.ResponseWriter.

	// Ideally we assume handleBanquet signature matches what we need.
	// func handleBanquet(app *pocketbase.PocketBase, e *core.RequestEvent) error

	// Wait, core.RequestEvent might not be public instantiable easily with all fields populated?
	// It is public.
	e := &core.RequestEvent{
		App: testApp,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	}
	// Also internal Router is needed for PathValue matching?
	// handleBanquet uses `reqURI := e.Request.RequestURI`. It does NOT use `e.Request.PathValue`.
	// It only uses `e.Request` and `e.Response`.

	// Wait, handleBrowse uses PathValue. handleBanquet logic I wrote uses `banquet.ParseNested(reqURI)`.
	// So logic should hold.

	if err := handleBanquet(e); err != nil {
		t.Fatalf("handleBanquet failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// Verify JSON
	var result []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 rows, got %d", len(result))
	}

	// Check first row
	row1 := result[0]
	if row1["name"] != "alpha" {
		t.Errorf("Expected row1.name = alpha, got %v", row1["name"])
	}
}
