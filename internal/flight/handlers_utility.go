package flight

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// PathValidationResponse represents the result of path validation
type PathValidationResponse struct {
	Valid         bool     `json:"valid"`
	ExpandedPath  string   `json:"expanded_path"`
	Segments      []string `json:"segments"`
	ValidSegments []string `json:"valid_segments"`
	BreakPoint    string   `json:"break_point,omitempty"`
	ErrorMessage  string   `json:"error_message,omitempty"`
}

// HandleValidatePath validates a file path and returns detailed breakdown
func HandleValidatePath(e *core.RequestEvent) error {
	path := e.Request.URL.Query().Get("path")
	if path == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "path parameter is required",
		})
	}

	response := validatePath(path)
	return e.JSON(http.StatusOK, response)
}

func validatePath(path string) PathValidationResponse {
	// Expand tilde
	expandedPath := path
	if strings.HasPrefix(path, "~") {
		if homeDir, err := os.UserHomeDir(); err == nil {
			expandedPath = strings.Replace(path, "~", homeDir, 1)
		}
	}

	// Split into segments
	segments := strings.Split(strings.Trim(expandedPath, "/"), "/")

	// Check full path first
	if _, err := os.Stat(expandedPath); err == nil {
		return PathValidationResponse{
			Valid:         true,
			ExpandedPath:  expandedPath,
			Segments:      segments,
			ValidSegments: segments,
		}
	}

	// Find where path breaks
	validSegments := []string{}
	var breakPoint string

	for i := range segments {
		testPath := "/" + filepath.Join(segments[:i+1]...)
		if _, err := os.Stat(testPath); err == nil {
			validSegments = segments[:i+1]
		} else {
			breakPoint = strings.Join(segments[i:], "/")
			break
		}
	}

	if len(validSegments) == 0 {
		return PathValidationResponse{
			Valid:        false,
			ExpandedPath: expandedPath,
			Segments:     segments,
			ErrorMessage: "The entire path does not exist. Tip: Check if you're in the right directory or if the file was moved.",
		}
	}

	return PathValidationResponse{
		Valid:         false,
		ExpandedPath:  expandedPath,
		Segments:      segments,
		ValidSegments: validSegments,
		BreakPoint:    breakPoint,
		ErrorMessage:  fmt.Sprintf("Path breaks at: %s", breakPoint),
	}
}

// PathSegment represents a parsed segment of a path
type PathSegment struct {
	Text   string `json:"text"`
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
	Type   string `json:"type,omitempty"` // "directory", "file", "banquet_table"
}

// ParsePathResponse represents the result of path parsing
type ParsePathResponse struct {
	Original string        `json:"original"`
	Expanded string        `json:"expanded"`
	Segments []PathSegment `json:"segments"`
}

// HandleParsePath parses a path into structured segments
func HandleParsePath(e *core.RequestEvent) error {
	urlParam := e.Request.URL.Query().Get("url")
	if urlParam == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "url parameter is required",
		})
	}

	response := parsePath(urlParam)
	return e.JSON(http.StatusOK, response)
}

func parsePath(path string) ParsePathResponse {
	response := ParsePathResponse{
		Original: path,
		Segments: []PathSegment{},
	}

	// Check for Banquet URL (contains semicolon for table)
	var tableName string
	if idx := strings.Index(path, ";"); idx != -1 {
		tableName = path[idx+1:]
		path = path[:idx]
	}

	// Expand tilde
	expanded := path
	homeDir, _ := os.UserHomeDir()
	if strings.HasPrefix(path, "~") && homeDir != "" {
		expanded = strings.Replace(path, "~", homeDir, 1)

		// Add tilde as first segment
		response.Segments = append(response.Segments, PathSegment{
			Text:   "~",
			Path:   homeDir,
			Exists: true,
			Type:   "directory",
		})
	}

	response.Expanded = expanded

	// Parse path segments
	parts := strings.Split(strings.Trim(expanded, "/"), "/")
	currentPath := ""

	for i, part := range parts {
		if part == "" {
			continue
		}

		if currentPath == "" {
			currentPath = "/" + part
		} else {
			currentPath = filepath.Join(currentPath, part)
		}

		// Check if exists
		info, err := os.Stat(currentPath)
		exists := err == nil
		segmentType := "directory"
		if exists && !info.IsDir() {
			segmentType = "file"
		}

		// Don't duplicate home directory for tilde paths
		if strings.HasPrefix(path, "~") && i == 0 {
			continue
		}

		response.Segments = append(response.Segments, PathSegment{
			Text:   part,
			Path:   currentPath,
			Exists: exists,
			Type:   segmentType,
		})
	}

	// Add table segment if present
	if tableName != "" {
		response.Segments = append(response.Segments, PathSegment{
			Text:   tableName,
			Path:   "",   // Tables don't have filesystem paths
			Exists: true, // Assume exists, would need DB query to verify
			Type:   "banquet_table",
		})
	}

	return response
}

// CacheStats represents cache statistics
type CacheStats struct {
	FileCount      int    `json:"file_count"`
	TotalSizeBytes int64  `json:"total_size_bytes"`
	TotalSizeMB    string `json:"total_size_mb"`
	CacheDirectory string `json:"cache_directory"`
}

// HandleCacheStats returns statistics about the conversion cache
func HandleCacheStats(e *core.RequestEvent) error {
	cacheDir := GetCacheDirectory()

	var fileCount int
	var totalSize int64

	filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			fileCount++
			totalSize += info.Size()
		}
		return nil
	})

	stats := CacheStats{
		FileCount:      fileCount,
		TotalSizeBytes: totalSize,
		TotalSizeMB:    fmt.Sprintf("%.2f", float64(totalSize)/(1024*1024)),
		CacheDirectory: cacheDir,
	}

	return e.JSON(http.StatusOK, stats)
}

// HandleCacheClear clears all cached conversions
func HandleCacheClear(e *core.RequestEvent) error {
	cacheDir := GetCacheDirectory()

	var deletedCount int
	filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && path != cacheDir {
			if err := os.Remove(path); err == nil {
				deletedCount++
			}
		}
		return nil
	})

	return e.JSON(http.StatusOK, map[string]interface{}{
		"deleted_count": deletedCount,
		"message":       fmt.Sprintf("Cleared %d cached files", deletedCount),
	})
}

// HandleCacheEntry handles operations on a specific cache entry
func HandleCacheEntry(e *core.RequestEvent) error {
	path := e.Request.URL.Query().Get("path")
	if path == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "path parameter is required",
		})
	}

	// For DELETE requests, remove the cached entry
	if e.Request.Method == http.MethodDelete {
		cacheKey := getCacheKeyForPath(path)
		cacheDir := GetCacheDirectory()
		cachePath := filepath.Join(cacheDir, cacheKey)

		if err := os.Remove(cachePath); err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{
				"error": "cache entry not found",
			})
		}

		return e.JSON(http.StatusOK, map[string]string{
			"message": "cache entry deleted",
		})
	}

	return e.JSON(http.StatusMethodNotAllowed, map[string]string{
		"error": "method not allowed",
	})
}

// GetCacheDirectory returns the cache directory path
func GetCacheDirectory() string {
	// Check if we're using a custom data directory
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		homeDir, _ := os.UserHomeDir()
		dataDir = filepath.Join(homeDir, "Library", "Application Support", "Flight3")
	}

	cacheDir := filepath.Join(dataDir, "pb_data", "cache", "conversions")
	os.MkdirAll(cacheDir, 0755)
	return cacheDir
}

// GetAppDataDirectory returns the root application data directory
func GetAppDataDirectory() string {
	// Check if we're using a custom data directory
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		homeDir, _ := os.UserHomeDir()
		dataDir = filepath.Join(homeDir, "Library", "Application Support", "Flight3")
	}
	os.MkdirAll(dataDir, 0755)
	return dataDir
}

func getCacheKeyForPath(path string) string {
	// Simple cache key generation
	// In production, you'd want to include file modification time
	sanitized := strings.ReplaceAll(path, "/", "_")
	sanitized = strings.ReplaceAll(sanitized, "\\", "_")
	sanitized = strings.ReplaceAll(sanitized, ":", "_")
	return sanitized + ".db"
}

// Helper to write JSON response
func jsonResponse(e *core.RequestEvent, status int, data interface{}) error {
	e.Response.Header().Set("Content-Type", "application/json")
	e.Response.WriteHeader(status)
	return json.NewEncoder(e.Response).Encode(data)
}
