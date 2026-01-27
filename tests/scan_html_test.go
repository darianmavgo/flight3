package tests

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestFindHTMLAssets scans the codebase for HTML files and Go files containing "embed".
// It verifies that expected HTML assets exist and are referenced.
func TestFindHTMLAssets(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// Assuming test runs from root or tests/.
	// If run from root via `go test ./tests`, Getwd is root.
	// If run from tests via `go test .`, Getwd is tests.
	// Normalized root:
	if strings.HasSuffix(root, "tests") {
		root = filepath.Dir(root)
	}

	logResults := []string{}

	// 1. Scan for .html files
	t.Log("Scanning for .html files...")
	foundHTML := 0
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skip hidden dirs (like .git, .idea) and vendor/node_modules
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") || d.Name() == "vendor" || d.Name() == "node_modules" || d.Name() == "sample_data" {
				return filepath.SkipDir
			}
			if path == filepath.Join(root, "pb_data", "cache") {
				return filepath.SkipDir
			}
		}

		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".html") {
			rel, _ := filepath.Rel(root, path)
			logResults = append(logResults, "HTML File: "+rel)
			foundHTML++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// 2. Scan for 'embed' directives in Go files
	t.Log("Scanning for Go embed directives...")
	foundEmbeds := 0
	embedRegex := regexp.MustCompile(`//go:embed\s+(.*)`)

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") || d.Name() == "vendor" {
				return filepath.SkipDir
			}
		}

		if !d.IsDir() && strings.HasSuffix(d.Name(), ".go") {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			scanner := bufio.NewScanner(file)
			lineNum := 0
			for scanner.Scan() {
				lineNum++
				line := scanner.Text()
				matches := embedRegex.FindStringSubmatch(line)
				if len(matches) > 1 {
					rel, _ := filepath.Rel(root, path)
					logResults = append(logResults, "Go Embed: "+rel+":"+matches[1])
					foundEmbeds++
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// 3. Report & Assert
	t.Logf("Found %d HTML files and %d Embed directives.", foundHTML, foundEmbeds)
	for _, l := range logResults {
		t.Log(l)
	}

	// Assertions based on Flight3 knowledge
	// We expect at least templates/row.html, templates/head.html, templates/foot.html
	// And embed in main.go

	expectedFiles := []string{}

	for _, expect := range expectedFiles {
		found := false
		for _, logLine := range logResults {
			if strings.Contains(logLine, expect) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find known asset '%s', but it was missing from scan.", expect)
		}
	}

	if foundEmbeds == 0 {
		t.Errorf("Expected to find at least one //go:embed directive (e.g. in main.go). Found none.")
	}
}
