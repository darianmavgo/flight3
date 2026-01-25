package tests
import (
"fmt"
"strings"
"github.com/darianmavgo/banquet"
)
func ExampleBanquetParsing() {
	url := "/https:/r2-auth@d8dc30936fb37cbd74552d31a709f6cf.r2.cloudflarestorage.com/test-mksqlite/sample_data/21mb.csv"
	if strings.Contains(url, "https:/") && !strings.Contains(url, "https://") {
		url = strings.Replace(url, "https:/", "https://", 1)
	}
	if strings.HasPrefix(url, "/") {
		url = strings.TrimPrefix(url, "/")
	}
	fmt.Printf("Normalized: %s\n", url)
	b, err := banquet.ParseNested(url)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Scheme: %s\n", b.URL.Scheme)
	fmt.Printf("Host: %s\n", b.URL.Host)
	fmt.Printf("Path: %s\n", b.URL.Path)
	if b.URL.User != nil {
		fmt.Printf("User: %s\n", b.URL.User.Username())
	}
	// Output:
	// Normalized: https://r2-auth@d8dc30936fb37cbd74552d31a709f6cf.r2.cloudflarestorage.com/test-mksqlite/sample_data/21mb.csv
	// Scheme: https
	// Host: d8dc30936fb37cbd74552d31a709f6cf.r2.cloudflarestorage.com
	// Path: /test-mksqlite/sample_data/21mb.csv
	// User: r2-auth
}
