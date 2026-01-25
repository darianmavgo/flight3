package main

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/darianmavgo/banquet"
)

func main() {
	reqURI := "/https:/r2-auth@d8dc30936fb37cbd74552d31a709f6cf.r2.cloudflarestorage.com/test-mksqlite/sample_data/21mb.csv"

	fmt.Printf("Original: %s\n", reqURI)

	// Normalize if it's a direct nested URL (missing second slash or leading slash)
	// Browser/Router might have normalized /https:/ to /https:/
	if strings.Contains(reqURI, "https:/") && !strings.Contains(reqURI, "https://") {
		reqURI = strings.Replace(reqURI, "https:/", "https://", 1)
	} else if strings.Contains(reqURI, "http:/") && !strings.Contains(reqURI, "http://") {
		reqURI = strings.Replace(reqURI, "http:/", "http://", 1)
	}

	if strings.HasPrefix(reqURI, "/") && (strings.HasPrefix(reqURI, "/http") || strings.Contains(reqURI, "/https")) {
		trimmed := strings.TrimPrefix(reqURI, "/")
		if strings.HasPrefix(trimmed, "http") {
			reqURI = trimmed
		}
	}

	fmt.Printf("Normalized: %s\n", reqURI)

	b, err := banquet.ParseNested(reqURI)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Path: %s\n", b.Path)
	if b.User != nil {
		fmt.Printf("User: %s\n", b.User.Username())
	} else {
		fmt.Println("User: <nil>")
	}
	fmt.Printf("Host: %s\n", b.Host)

	// Compare with net/url
	fmt.Println("--- net/url ---")
	u, err := url.Parse(reqURI)
	if err != nil {
		fmt.Printf("net/url Error: %v\n", err)
	} else {
		fmt.Printf("Scheme: %s\n", u.Scheme)
		fmt.Printf("Host: %s\n", u.Host)
		if u.User != nil {
			fmt.Printf("User: %s\n", u.User.Username())
		}
	}
}
