package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/darianmavgo/flight3/internal/flight"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootPathServeFolder(t *testing.T) {
	// Setup test directory
	tempDir, err := os.MkdirTemp("", "flight3-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a dummy file in the serve folder
	testFile := filepath.Join(tempDir, "test.csv")
	err = os.WriteFile(testFile, []byte("a,b\n1,2"), 0644)
	require.NoError(t, err)

	pbDataDir, err := os.MkdirTemp("", "flight3-pb-*")
	require.NoError(t, err)
	defer os.RemoveAll(pbDataDir)

	testApp, err := tests.NewTestApp(pbDataDir)
	require.NoError(t, err)
	defer testApp.Cleanup()

	// 1. Ensure collections exist
	err = flight.EnsureCollections(testApp)
	require.NoError(t, err)

	// 2. Configure serve_folder in app_settings
	record, err := testApp.FindCollectionByNameOrId("app_settings")
	require.NoError(t, err)

	newRecord := core.NewRecord(record)
	newRecord.Set("key", "serve_folder")
	newRecord.Set("value", tempDir)
	err = testApp.Save(newRecord)
	require.NoError(t, err)

	// 3. Request root path
	// _ = httptest.NewRequest("GET", "/", nil)
	// We need to trigger the routing. In a real PB app, it happens via OnServe.
	// For testing, we can simulate the handler or use a full serve event.

	// Let's use a simpler approach: test the ServeFolder helper first
	baseDir := flight.GetServeFolder(testApp)
	assert.Equal(t, tempDir, baseDir)

	// Real integration test would need Echo router setup
}
