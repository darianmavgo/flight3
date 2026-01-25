package main
import (
"fmt"
"net/url"
)
func main() {
	u, err := url.Parse("https://r2-auth@d8dc30936fb37cbd74552d31a709f6cf.r2.cloudflarestorage.com/test-mksqlite/sample_data/21mb.csv")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Scheme: %s\n", u.Scheme)
	fmt.Printf("Host: %s\n", u.Host)
	fmt.Printf("Path: %s\n", u.Path)
	if u.User != nil {
		fmt.Printf("User: %s\n", u.User.Username())
	} else {
		fmt.Println("User is NIL")
	}
}
