package flight

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/darianmavgo/banquet"
)

// GenCacheKey generates a cache key based on the banquet request and remote config.
// The auth alias "b.User" might matter for disambiguation.
// Deliberately not including scheme since file could be pulled via s3 or https in some situations.
// Includes remoteConfigHash to ensure cache isolation per credential set.
func GenCacheKey(b *banquet.Banquet, remoteConfigHash string) string {
	userInfo := ""
	if b.User != nil {
		userInfo = b.User.String()
	}
	parts := []string{userInfo, b.Hostname(), b.DataSetPath, remoteConfigHash}
	// Filter out empty parts
	var filtered []string
	for _, p := range parts {
		if p != "" {
			filtered = append(filtered, p)
		}
	}
	return strings.Join(filtered, "-")
}

// ValidateCache checks if cached SQLite file is still valid based on TTL
func ValidateCache(cachePath string, ttlMinutes float64) (bool, error) {
	// Check file existence
	info, err := os.Stat(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // Cache doesn't exist
		}
		return false, err // Some other error
	}

	// Get file modification time
	modTime := info.ModTime()

	// Calculate age in minutes
	age := time.Since(modTime).Minutes()

	// Check if still valid
	if age > ttlMinutes {
		return false, nil // Cache expired
	}

	return true, nil // Cache is valid
}

// GetCachePath returns the full path to a cache file
func GetCachePath(dataDir, cacheKey string) string {
	return filepath.Join(dataDir, "cache", cacheKey+".db")
}
