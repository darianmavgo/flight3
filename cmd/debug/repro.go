package main

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/darianmavgo/banquet"
)

func main() {
	// The problematic URL path from user request
	// Browser URL: http://127.0.0.1:8090/https:/r2-auth@https:/d8dc30936fb37cbd74552d31a709f6cf.r2.cloudflarestorage.com/test-mksqlite
	// The server sees the RequestURI part:
	reqURI := "/https:/r2-auth@https:/d8dc30936fb37cbd74552d31a709f6cf.r2.cloudflarestorage.com/test-mksqlite"

	fmt.Printf("Input: %s\n", reqURI)

	// Simulation of main.go logic
	trimmed := reqURI
	if strings.Contains(reqURI, "https:/") && !strings.Contains(reqURI, "https://") {
		trimmed = strings.Replace(trimmed, "https:/", "https://", 1)
	} else if strings.Contains(reqURI, "http:/") && !strings.Contains(reqURI, "http://") {
		trimmed = strings.Replace(trimmed, "http:/", "http://", 1)
	}

	if strings.HasPrefix(trimmed, "/") && (strings.HasPrefix(trimmed, "/http") || strings.Contains(trimmed, "/https")) {
		trimmed = strings.TrimPrefix(trimmed, "/")
	}

	// PROPOSED FIX: Remove scheme after @
	if idx := strings.LastIndex(trimmed, "@"); idx != -1 {
		// Check what's after @
		after := trimmed[idx+1:]
		if strings.HasPrefix(after, "https:/") {
			// Handle https:// or https:/
			if strings.HasPrefix(after, "https://") {
				trimmed = trimmed[:idx+1] + after[8:]
			} else {
				trimmed = trimmed[:idx+1] + after[7:]
			}
		} else if strings.HasPrefix(after, "http:/") {
			if strings.HasPrefix(after, "http://") {
				trimmed = trimmed[:idx+1] + after[7:]
			} else {
				trimmed = trimmed[:idx+1] + after[6:]
			}
		}
	}

	fmt.Printf("Pre-processed (Fixed): %s\n", trimmed)

	b, err := banquet.ParseNested(trimmed)
	if err != nil {
		fmt.Printf("Banquet Error: %v\n", err)
	} else {
		fmt.Printf("Banquet Host: '%s'\n", b.Host) // Expecting this to be 'https' based on error
		fmt.Printf("Banquet User: %v\n", b.User)
		fmt.Printf("Banquet Path: '%s'\n", b.Path)
	}

	u, err := url.Parse(trimmed)
	if err != nil {
		fmt.Printf("net/url Error: %v\n", err)
	} else {
		fmt.Printf("URL Host: '%s'\n", u.Host)
		fmt.Printf("URL User: %v\n", u.User)
	}
}
