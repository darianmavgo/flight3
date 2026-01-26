package main

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darianmavgo/sqliter/sqliter"
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

	// Initialize dummy TableWriter
	tpl := getMockTemplate()
	tw := sqliter.NewTableWriter(tpl, sqliter.DefaultConfig())

	if err := handleBanquet(e, tw); err != nil {
		t.Fatalf("handleBanquet failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// Verify JSON
	// Verify HTML Output
	body := rec.Body.String()
	if !strings.Contains(body, "<table>") {
		t.Error("Expected HTML output containing <table>")
	}

	// Check content
	if !strings.Contains(body, "alpha") {
		t.Errorf("Expected body to contain 'alpha, got: %s", body)
	}
	if !strings.Contains(body, "beta") {
		t.Errorf("Expected body to contain 'beta', got: %s", body)
	}
	if !strings.Contains(body, "gamma") {
		t.Errorf("Expected body to contain 'gamma', got: %s", body)
	}
}

func getMockTemplate() *template.Template {
	t := template.New("root")
	t.New("head.html").Parse(`<html><body><table>`)
	t.New("foot.html").Parse(`</table></body></html>`)
	t.New("row.html").Parse(`{{range .}}<td>{{.}}</td>{{end}}`)
	return t
}

func TestBanquetDirectoryListing(t *testing.T) {
	// 1. Setup Temp Dir Structure
	tmpDir, err := os.MkdirTemp("", "pb_test_listing")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create files/dirs to list
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "subfolder"), 0755)

	// 2. Setup App
	testApp, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer testApp.Cleanup()

	// 3. Create 'rclone_remotes' collection
	if err := ensureCollections(testApp); err != nil {
		t.Fatal(err)
	}

	// 4. Register 'r2-auth' remote (mocked as local)
	remoteCollection, _ := testApp.FindCollectionByNameOrId("rclone_remotes")
	remote := core.NewRecord(remoteCollection)
	remote.Set("name", "r2-auth")
	remote.Set("type", "local")
	// On real S3, URL might be .../bucket-name/.
	// But here for 'local', we point root to tmpDir.
	remote.Set("config", map[string]interface{}{"root": tmpDir})
	if err := testApp.Save(remote); err != nil {
		t.Fatal(err)
	}

	// 5. Mock Request for "/https:/r2-auth@localhost/" (Simulating user's URL structure)
	// The User requested: http://127.0.0.1:8090/https:/r2-auth@d8dc...r2.cloudflarestorage.com/
	// We'll use localhost as host for simplicity, user alias is 'r2-auth'.
	// Path is root (trailing slash).

	reqURL := "/https:/r2-auth@localhost/"
	req := httptest.NewRequest(http.MethodGet, reqURL, nil)
	rec := httptest.NewRecorder()

	e := &core.RequestEvent{
		App: testApp,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	}

	// 6. Invoke Handler
	tpl := getMockTemplate()
	tw := sqliter.NewTableWriter(tpl, sqliter.DefaultConfig())

	// This is expected to FAIL currently (likely 500 or 404 or empty)
	// We want to assert what *should* happen (200 OK with listing)
	err = handleBanquet(e, tw)

	// Logging result for debugging during TDD
	t.Logf("Status: %d", rec.Code)
	t.Logf("Body: %s", rec.Body.String())
	if err != nil {
		t.Logf("Error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for directory listing, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "file1.txt") {
		t.Errorf("Listing missing 'file1.txt'")
	}
	if !strings.Contains(body, "subfolder") {
		t.Errorf("Listing missing 'subfolder'")
	}
}
