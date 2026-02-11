package tests

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestE2EBinary builds the flight3 binary and runs it against a real port
func TestE2EBinary(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// 1. Build Binary
	projectRoot, _ := os.Getwd()
	for filepath.Base(projectRoot) != "flight3" {
		projectRoot = filepath.Dir(projectRoot)
	}

	binPath := filepath.Join(projectRoot, "bin", "flight3")
	cmdBuild := exec.Command("go", "build", "-o", binPath, "./cmd/flight/main.go")
	cmdBuild.Dir = projectRoot
	if out, err := cmdBuild.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build binary: %v\n%s", err, out)
	}
	t.Logf("Built binary at %s", binPath)

	// 2. Setup Test Data Dir
	testDataDir := filepath.Join(projectRoot, "test_output", "e2e_binary_test")
	os.RemoveAll(testDataDir)
	os.MkdirAll(testDataDir, 0755)

	// 3. Start Server
	port := "8099" // Use non-standard port to avoid conflicts
	cmdRun := exec.Command(binPath, "serve", "--http=127.0.0.1:"+port, "--dir="+filepath.Join(testDataDir, "pb_data"))
	cmdRun.Dir = projectRoot

	// Capture output for debugging
	// cmdRun.Stdout = os.Stdout
	// cmdRun.Stderr = os.Stderr

	if err := cmdRun.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer func() {
		if cmdRun.Process != nil {
			cmdRun.Process.Kill()
		}
	}()

	// 4. Wait for it to be up
	baseURL := "http://127.0.0.1:" + port
	t.Logf("Waiting for server at %s...", baseURL)

	ready := false
	for i := 0; i < 30; i++ {
		resp, err := http.Get(baseURL + "/api/health")
		if err == nil && resp.StatusCode == 200 {
			ready = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if !ready {
		t.Fatalf("Server failed to come up within 15 seconds")
	}

	// 5. Run Verification

	// A. Check Home SQLite Path
	resp, err := http.Get(baseURL + "/sqliter/home")
	if err != nil {
		t.Fatalf("Failed to get home: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected 200 OK from /sqliter/home, got %d: %s", resp.StatusCode, body)
	}

	// B. Check default static file (should be empty until we put something)
	// We can assume admin UI loads at /_/
	respAdmin, err := http.Get(baseURL + "/_/")
	if err != nil {
		t.Fatalf("Failed to get admin: %v", err)
	}
	if respAdmin.StatusCode != 200 {
		t.Errorf("Expected 200 OK from /_/, got %d", respAdmin.StatusCode)
	}

	t.Log("E2E Binary Test Passed")
}
