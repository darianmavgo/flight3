package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/darianmavgo/flight3/internal/flight"
	"github.com/pocketbase/pocketbase"
)

func TestRclonePocketBaseIntegration(t *testing.T) {
	// Create persistent test output directory
	projectRoot, _ := os.Getwd()
	// Navigate up if we are in tests subdir
	for filepath.Base(projectRoot) != "flight3" && filepath.Base(projectRoot) != "/" {
		projectRoot = filepath.Dir(projectRoot)
	}
	outputDir := filepath.Join(projectRoot, "test_output", "rclone_integration")
	_ = os.RemoveAll(outputDir) // Soft delete previous run
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create output dir: %v", err)
	}
	// defer os.RemoveAll(outputDir) // Disabled as requested

	tempDir := outputDir

	// Initialize PocketBase with temp directory pointing to pb_data
	pbDataDir := filepath.Join(tempDir, "pb_data")
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: pbDataDir,
	})

	// Bootstrap the app to initialize database
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("Failed to bootstrap PocketBase: %v", err)
	}
	defer app.ResetBootstrapState()

	// Initialize rclone with standard cache path
	cacheDir := filepath.Join(pbDataDir, "cache")
	if err := flight.InitRclone(cacheDir); err != nil {
		t.Fatalf("Failed to initialize rclone: %v", err)
	}

	// Ensure collections
	if err := flight.EnsureCollections(app); err != nil {
		t.Fatalf("Failed to ensure collections: %v", err)
	}

	t.Log("Rclone + PocketBase integration test passed")
}

func TestCacheKeyGeneration(t *testing.T) {
	// This test will be expanded once we have actual banquet objects to test with
	t.Log("Cache key generation test placeholder")
}

func TestCacheValidation(t *testing.T) {
	// Create persistent test output directory
	projectRoot, _ := os.Getwd()
	// Navigate up if we are in tests subdir
	for filepath.Base(projectRoot) != "flight3" && filepath.Base(projectRoot) != "/" {
		projectRoot = filepath.Dir(projectRoot)
	}
	outputDir := filepath.Join(projectRoot, "test_output", "cache_validation") // unique subdir
	_ = os.RemoveAll(outputDir)                                                // Soft delete previous run
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create output dir: %v", err)
	}
	// defer os.RemoveAll(outputDir) // Disabled as requested

	tempDir := outputDir

	cachePath := filepath.Join(tempDir, "test.db")

	// Test non-existent cache
	valid, err := flight.ValidateCache(cachePath, 1440)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if valid {
		t.Error("Expected cache to be invalid (doesn't exist)")
	}

	// Create cache file
	if err := os.WriteFile(cachePath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create cache file: %v", err)
	}

	// Test fresh cache
	valid, err = flight.ValidateCache(cachePath, 1440)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !valid {
		t.Error("Expected cache to be valid (just created)")
	}

	// Test expired cache (0 minute TTL)
	valid, err = flight.ValidateCache(cachePath, 0)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if valid {
		t.Error("Expected cache to be invalid (0 TTL)")
	}

	t.Log("Cache validation test passed")
}

func TestConversion(t *testing.T) {
	// Test file conversion
	// This would require actual test files
	t.Skip("Conversion test requires test data files")
}
