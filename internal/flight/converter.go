package flight

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ConvertToSQLite converts a source file to SQLite database using mksqlite CLI
func ConvertToSQLite(sourcePath, destPath string) error {
	log.Printf("[CONVERTER] Converting %s -> %s", sourcePath, destPath)

	// Detect file type from extension
	ext := strings.ToLower(filepath.Ext(sourcePath))

	switch ext {
	case ".db", ".sqlite", ".sqlite3":
		// Already SQLite, just copy
		return copyFile(sourcePath, destPath)
	case ".csv", ".xlsx", ".xls", ".json", ".txt", ".html", ".htm":
		// Supported formats - proceed with conversion
	default:
		return fmt.Errorf("unsupported file type: %s", ext)
	}

	// Use mksqlite CLI to convert
	log.Printf("[CONVERTER] Using mksqlite for conversion")

	// Run mksqlite command
	// mksqlite -i input.csv -o output.db
	cmd := exec.Command("mksqlite", "-i", sourcePath, "-o", destPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mksqlite conversion failed: %w\nOutput: %s", err, string(output))
	}

	log.Printf("[CONVERTER] Conversion successful")
	return nil
}

// copyFile copies a file from src to dst (for already-SQLite files)
func copyFile(src, dst string) error {
	// Simple implementation - could be optimized
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
